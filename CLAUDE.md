# CLAUDE.md — holdfast

> このファイルは Claude Code がリポジトリの文脈を理解するためのものです。
> **最初に読んで、ここに書かれた設計・不変条件・進め方の約束を守ってください。**
>
> **全体像・各 Phase の詳細（要件 / 設計の問い / DoD / 落とし穴 / レイヤー地図 / 教材対応）は [`docs/ROADMAP.md`](docs/ROADMAP.md) に全部あります。** 本ファイルは日々の作業用の要約。Phase に取りかかる前に ROADMAP の該当 Phase を必ず参照すること。
> **各 Phase の事前読書・クイズ・学習メモは [`docs/learning-notes/`](docs/learning-notes/) にあります。** `phase-N.md` を開き、クイズ①を解いてから手を動かすこと。回答・ハマりメモ・ミスログも同ファイルに書き込む。

---

## 0. まず置き換えるもの

- モジュールパス：`github.com/RIKU-SEINO/holdfast`（置換済み）。
- 現在地：**Phase 1（Go 並行 — mutex で MemoryStore を並行安全にする）**。
- 次の増分：`go test -race` で壊れるテストを書く → mutex で直す → `go test -race` を緑にする。

---

## 1. プロジェクト概要

**holdfast** は、ドメインに依存しない**汎用リース・プリミティブ**のライブラリ。

> 「希少リソースを **N 単位、TTL 付きで排他確保**し、**フェンシングトークン**付きで**冪等にコミット／解放**する」

座席予約・在庫引当・配車・レート制限・ジョブ排他取得・コネクションプールは、すべてこの**一つの部品の応用**にすぎない。holdfast 自身は「席」も「在庫」も知らない。

- `Units=1` なら分散 mutex、`Units=N` なら分散 semaphore。
- 抽象の高さは意図的に「**カウント付きリソースのリース**」に固定する。ただの分散ロックまで下げない（etcd/Redlock の再発明になる）、ドメインまで上げない（応用が狭くなる）。

### いちばん大事な設計原則

**ライブラリの正しさは「`Store` がアトミックな compare-and-set を提供する」一点に還元される。**
コアは保存先を知らず、`Store` インターフェースにだけ依存する。`Store` を満たすバックエンド（in-memory / Postgres / Raft）を差し替えても、**同じ適合テスト（conformance suite）**が全部を検証する。

---

## 2. 現在のコア契約（Phase 0 時点）

ルートパッケージ `holdfast`（`holdfast.go`）が契約。**ロジックは持たず、型と interface だけ**。

```go
type RegisterRequest struct { Resource string; Capacity int }
type AcquireRequest  struct { Resource string; Units int; TTL time.Duration; IdempotencyKey string }
type CommitRequest   struct { LeaseID string; Token uint64 }
type ReleaseRequest  struct { LeaseID string; Token uint64 }
type Lease           struct { ID string; Token uint64; Expires time.Time }
type Receipt         struct { LeaseID string }

var ErrUnknownResource = errors.New("holdfast: unknown resource")
var ErrExhausted       = errors.New("holdfast: no units available")
var ErrConflict        = errors.New("holdfast: stale token or unknown lease")

// バックエンドが満たすべき唯一の契約。
type Store interface {
    Register(ctx context.Context, req RegisterRequest, now time.Time) error
    Acquire(ctx context.Context, req AcquireRequest, now time.Time) (Lease, error)
    Commit(ctx context.Context, req CommitRequest) (Receipt, error)
    Release(ctx context.Context, req ReleaseRequest) error
    Reap(ctx context.Context, now time.Time) (int, error)
}
```

将来（Phase 3 以降）、冪等オーケストレーションを担う `Leaser` ファサードを `Store` の上に重ねる予定。**今はまだ作らない。**

---

## 3. 不変条件（絶対に壊さない）

これらは適合テストで保証され、どのバックエンドでも成り立つ必要がある：

1. **枠を超えて確保できない**（合計 `Units` が capacity を超えない）。
2. **フェンシング**：`Commit` / `Release` は、最新でない `token` を必ず `ErrConflict` で弾く。
3. **冪等**：同じ `IdempotencyKey` の `Acquire` は同じ `Lease` を返す（重複確保しない）。※ Phase 0 後半で導入。
4. **TTL 失効**：期限切れリースは `Reap` で回収され、枠が戻る。※ Phase 0 後半で導入。
5. **並行安全**：`go test -race` がクリーン。※ Phase 1 から。

新しいバックエンドや機能を足すときも、**この 5 つを満たし続けること**。

---

## 4. アーキテクチャと、このリポの範囲

```
holdfast/  (このリポ = コア + サーバ + 運用層 / monorepo)
├── holdfast.go            契約（型 + Store interface）
├── conformance/           再利用可能な適合テスト（全バックエンド共通）
├── store/
│   ├── memory/            参照実装（依存 0）
│   ├── postgres/  (Phase 2 で誕生・別 go.mod)
│   └── raft/      (Phase 5 で誕生・別 go.mod)
├── proto/                 (Phase 3)
├── cmd/holdfastd/         (Phase 3：コアを gRPC で包むサーバ)
└── deploy/                (Phase 6：helm / envoy / terraform — holdfast 自身を運用)
```

**別リポ**（このリポには入れない）：

- `holdfast-examples/`（座席予約など。薄い応用例。React は**自分の back/ 経由で**holdfast を呼ぶ。ブラウザから holdfast を直接叩かない）
- `holdfast-sdk-{ts,py}/`（バックエンド言語向けクライアント SDK。別ツールチェーン）

**重要な原則**：

- Envoy・Terraform・k8s・可観測性は「応用例のため」ではなく、**holdfast 自身（Raft クラスタ）を運用するための層**。`deploy/` に同梱する。
- **ディレクトリを先回りで作らない。** 各 Phase で実際に必要になってから生やす。
- コアと各バックエンドは**別モジュール（別 go.mod）**にして依存を隔離する。ただし `postgres` / `raft` の依存が実際に出る Phase 2 / 5 まで分割しない。今はルートに `go.mod` 1 つ。

---

## 5. フェーズ計画（新技術は一度に 1 つだけ）

| Phase | 作るもの | 新しく触る技術 | 状態 |
|---|---|---|---|
| 0 | コア契約 + in-memory 実装 + 適合テスト | Go | **完了** |
| 1 | 競合を作って mutex/channel で守る | Go 並行 | **進行中** |
| 2 | Postgres バックエンド | Postgres / トランザクション | |
| 3 | コアを gRPC サービスで包む + SDK（graceful shutdown・/livez vs /readyz・context timeout 含む） | gRPC / protobuf | |
| 4 | Envoy サイドカー（holdfast 自身の通信、bulkhead・circuit breaker 含む） | Service Mesh | |
| 5 | Raft バックエンド（async 非同期レプリケーション体験 → consensus 実装） | 合意 / フェンシング | |
| 6 | Terraform + k8s で holdfast クラスタを構築（multi-stage Dockerfile・PDB・HPA 含む） | Terraform / k8s | |
| 7 | 分散トレーシング + SLO + カオス試験（burn rate アラート 含む） | OTel / SRE | |
| 8 | **別リポ** `holdfast-examples/` — holdfast を import した応用例 + Fly.io デプロイ | Fly.io / フルスタック | |

> **Phase 3〜7 は holdfast 自体を gRPC サービスとして動かす（学習）。Phase 8 は mode①（import）で使う応用例。k8s は Phase 8 には出てこない。**

各 Phase の **完了条件（DoD）を満たすまで次に進まない**。深掘りは別途の課題ハンドブック（PDF）参照。

---

## 6. 開発環境とコマンド

- **Go 1.26**（Docker Compose のコンテナ内で実行）。ホストには Go を入れていない前提。
- すべての go コマンドは**コンテナの中で**実行する（ツールチェーン統一のため）。

```bash
docker compose up -d                                   # 開発コンテナ起動
docker compose exec dev go test ./...                  # テスト（DoD の基本）
docker compose exec dev go test -race ./...            # 競合検知（Phase 1 以降は必須）
docker compose exec dev go build ./...                 # ビルド
docker compose exec dev go vet ./...                   # 静的検査
docker compose exec dev gofmt -l .                     # 整形漏れチェック（空出力が正）
```

新しい依存を入れるとき（Phase 2 以降）：

```bash
docker compose exec dev go get <module>
docker compose exec dev go mod tidy
```

---

## 7. Claude への進め方の約束（重要）

開発者は **0→1 SaaS の経験者**（Node / React / AWS CDK / CloudFormation / AWS）だが、**Go・k8s・Terraform は初心者**。**作りながら学びたい**。これを踏まえて：

- **学習者向けに進める。** コードを丸投げせず、「なぜそうするか」を簡潔に添える。特に **TS と違う Go の癖**（構造的に満たせば自動で interface 実装になる／例外でなく `error` を返す／ゼロ値）を要所で指摘する。また、開発者に対して常に質問の投げかけや、設計意図など、エッセンスを習得できるようになるためのフィードバックを欠かさない。
- **小さい増分で進め、各ステップで `go test ./...` を緑に保つ。** 大きな変更を一度に出さない。
- **「まず素朴に作って、わざと壊して、直す」教育的順序を守る。** 例：Phase 1 は最初から完璧な並行実装を目指さず、単一スレッド → 並行で `-race` が壊れるのを見せる → mutex で直す。先回りの最適化をしない。
- **一度に 1 つの新技術。** Phase の順序を飛ばして k8s や Terraform を持ち込まない。
- **適合テストを背骨に保つ。** バックエンド固有のテストに崩さない。新バックエンドは `conformance.Run` を通すこと。
- **やらないこと**：
  - コアモジュールの `go.mod` に依存を足さない（コアは標準ライブラリ中心に保つ）。
  - 将来 Phase のディレクトリ・ファイルを先回りで作らない。
  - ドメイン語彙（「席」「在庫」等）をコアに漏らさない。
  - holdfast をブラウザから直接叩く設計にしない（FE は自分の back/ 経由）。
- **変更後は必ずテストを走らせ**、結果（緑/赤）を報告してから次へ。

---

## 8. 用語

- **Lease**：TTL 付きの「確保」。期限切れで自動失効する。
- **Fencing token**：確保ごとに発行される単調増加の番号。リソース側が古い番号を拒否し、「止まっていた古い保持者の遅れた書き込み」による破壊を防ぐ（DDIA 8 章 / Kleppmann）。
- **Conformance suite**：`Store` を受け取り不変条件を検証する、バックエンド非依存のテスト群。
- **Store**：バックエンドが満たす唯一の契約。正しさはこの実装のアトミック性に還元される。
- **モード①（組み込み）**：`holdfast.New(store)` を import してプロセス内で使う。
- **モード②（サービス）**：holdfast クラスタを立て、SDK 経由で叩く（etcd を使う感覚）。Phase 3 以降。
