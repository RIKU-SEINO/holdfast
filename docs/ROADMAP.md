# holdfast — 全体ロードマップ（チャレンジ・ハンドブック）

> 希少リソースを排他確保する**汎用リース基盤**を、作って・壊して・運用するための全体設計とフェーズ計画。
> 日々の進め方・約束・コマンドは [`/CLAUDE.md`](../CLAUDE.md) を参照。本書は「全体像と各 Phase の詳細」を持つ。

---

## 1. 正体 — これは「予約」ではなく、リースのプリミティブ

ドメイン名（座席・在庫…）は捨てる。残るのはこれだけ：

> **「希少リソースを N 単位、TTL 付きで排他確保し、フェンシングトークン付きで冪等にコミット／解放する」**

座席もチケットも在庫も車両も、ライブラリは知らない。全部この一つの部品の応用。

### 同じプリミティブの応用例

| 応用 | Resource（確保対象） | Units | フェンシングが効く瞬間 |
|---|---|---|---|
| 座席 / チケット予約 | 公演 × 席 | 席数 | 期限切れホールドの「復活」を弾く |
| 在庫引当 | SKU | 在庫数 | 二重引当を弾く |
| 配車ディスパッチ | エリア × 車両 | 1（= mutex） | 古いノードの二重アサインを弾く |
| レート制限 / クォータ | `api:user:42` | トークン数 | —（TTL で自然回復） |
| ジョブの排他取得 | `job:id` | 1 | 止まった旧ワーカーの書き戻しを弾く |

`Units=1` なら分散 mutex、`Units=N` なら分散 semaphore。この「カウント付き」が、ただのロックとの差。

### 設計判断：どこまで汎用にするか（抽象の高さ）

さらに下げて**ただの分散 mutex/lease** にすると etcd や Redlock の再発明になり差別化が消える。逆に**「席」に縛る**と応用が狭い。だから高さを**「カウント付きリソースのリース（semaphore 相当）」**に固定する。単なるロックより一段リッチで、在庫・座席・レート制限・プール上限に自然に効く。この“汎用すぎず・ドメインすぎない”高さを自分で選んで言語化したこと自体が、設計の主張になる。

---

## 2. 境界とルール

価値はほぼ全部、**「ライブラリは何を約束し、何を呼び出し側に委ねるか」**を言い切れるかに乗る。

### 持つもの / 持たないもの

| 持つ（ライブラリが正しさを保証） | 持たない（呼び出し側の責務） |
|---|---|
| ① リース（TTL 付き確保） | リソースが「何か」の意味（席・在庫・車…） |
| ② フェンシングトークン | 価格・決済・UI・認証・カタログ |
| ③ 冪等キー（安全なリトライ） | 待合室 / 流入制御ポリシー |
| ④ 枠を超えて確保しない不変条件 | どのバックエンド（Store）を使うか |

### いちばん美しい主張（CORE）

ライブラリは**ストレージを持たない**。コアは保存先を知らず、「アトミックな条件付き書き込みができる」という最小契約 = `Store` にだけ依存する。すると**ライブラリ全体の正しさが「Store がアトミックな compare-and-set を提供する」一点に還元される**。Postgres なら行ロック／条件付き UPDATE が、Redis なら Lua／WATCH が、自作 Raft なら複製ログが、その CAS を担う。**同じコア・同じテスト・差し替え可能な正しさの土台**——これが背骨。

### 3 つのルール

1. **契約から書く**：API と適合テストを先に固め、実装を後から各バックエンドで通す。
2. **壊して確かめる**：並行で殴り、ノードを落とし、分断する。二重確保は異常時にしか出ない。
3. **問いに答える**：各フェーズの「設計の問い」に自分の言葉で答える。そのまま LT になる。

---

## 3. 設計の核 — コア API と、包み方 (C)

コアは 2 つの面を持つ。約束する面（`Leaser`）と、持ち込む面（`Store`）。正しさは後者のアトミック性に集約する。

```go
// 汎用リース・プリミティブのコア。対象が「何か」も、保存先も知らない。
type AcquireRequest struct {
    Resource       string        // 確保対象のキー。"seat:A12" でも "rate:user:42" でも、意味はアプリが決める
    Units          int           // 確保する単位数（1 なら mutex、N なら semaphore 相当）
    TTL            time.Duration // リースの寿命
    IdempotencyKey string        // リトライを安全に（メッシュの retry 含む）
}
type Lease struct {
    ID      string
    Token   uint64    // 単調増加するフェンシングトークン
    Expires time.Time
}

// ライブラリが「約束する」面。
type Leaser interface {
    Acquire(ctx, req AcquireRequest) (Lease, error)            // ErrExhausted / ErrConflict
    Commit(ctx, leaseID string, token uint64) (Receipt, error) // 古い token/期限切れは拒否
    Release(ctx, leaseID string, token uint64) error
}

// 呼び出し側が「持ち込む」面。正しさは全てここのアトミック性に還元される。
type Store interface {
    Claim(ctx, p Placement) error                       // 原子的に Units を確保。枠が尽きていれば ErrExhausted
    Commit(ctx, leaseID string, token uint64) (Receipt, error)
    Release(ctx, leaseID string, token uint64) error
    Reap(ctx, now time.Time) (int, error)               // 期限切れリースの回収
}
```

> **実装メモ**：Phase 0 では簡単のため `Store` interface 1 つに `Acquire/Commit/Release/Reap` を持たせて始める（`memory` 実装がそれを満たす）。`Leaser` ファサード（冪等オーケストレーションを `Store` の上に重ねる層）は Phase 3 で導入する。

### パッケージング (C)：コアを作ってから、サービスで包む

```
消費者（応用の back / 別言語クライアント）
      ↓ SDK 経由（gRPC）
薄いサービス（gRPC API・多言語 SDK）        ← Phase 3〜
      ↓ import
コア：Leaser（リース・プリミティブ）          ← storage-agnostic
      ↓ Store interface（最小契約：アトミック CAS）
  ┌───────────┬───────────┬───────────┐
  in-memory      Postgres        自作 Raft
```

storage-agnostic なので、そのまま `import` で組み込んでも、薄いサーバで包んで多言語から叩いてもよい。Envoy・Raft・Terraform を載せる Phase もそのまま全部活きる。

---

## 4. リポジトリ構成と、使う側の体験

### 「なんだかなあ」の正体：Envoy・IaC は応用例のものではない（重要）

Envoy・Terraform・k8s・可観測性は「応用例アプリのためのインフラ」**ではない**。**holdfast 自身をサービスとして運用するための層**だ。holdfast サーバは Raft でクラスタを組む“分散インフラそのもの”——だから自分のノード間 mTLS（Envoy）、自分のクラスタを立てる Terraform、自分のリース操作の SLO/トレースが要る。これらは holdfast に first-class で同梱する（`deploy/`）。**応用例は薄いまま、ただ holdfast を呼ぶだけ。**

### リポジトリ／モジュール構成

```
holdfast/                         ← リポ A：コア + サーバ + 運用層（monorepo）
├── go.mod                        module .../holdfast（コア：依存ほぼ 0）
├── holdfast.go                   Leaser / Store / Lease（契約）
├── conformance/                  適合テスト一式（再利用可能なテストキット）
├── store/
│   ├── memory/                   コアに同梱・依存 0
│   ├── postgres/  go.mod         ← 別モジュール（pq 依存をコアに漏らさない）
│   └── raft/      go.mod         ← 別モジュール（hashicorp/raft を隔離）
├── proto/                        gRPC スキーマ
├── cmd/holdfastd/                サーバ本体（コアを gRPC で包む）
└── deploy/  helm/ envoy/ terraform/   holdfast 自身を運用する層

holdfast-examples/                ← リポ B：薄い応用例（別 go.mod）
└── seat-booking/  web/(React) → back/(holdfast SDK 利用) → holdfast
holdfast-sdk-{ts,py}/             ← リポ C：クライアント SDK（バックエンド言語向け）
```

### なぜ分けるのか

- **コアと各バックエンドを別モジュール**に → `import holdfast` だけなら pq も raft も降ってこない。`go.mod` はライブラリの公開契約。軽く保つこと自体が良い設計のサイン。
- **応用例は別リポ／別モジュール** → Web フレームワークや React のビルド依存をコアの `go.mod` に混ぜない。
- **コア + サーバ + deploy は同居** → 一緒にバージョンが動き、適合テストが全バックエンドを一度に検証できる。

### 使う側の 2 モード

| モード①：組み込み（import） | モード②：サービス（gRPC） |
|---|---|
| `holdfast.New(pgStore).Acquire(…)` をプロセス内で呼ぶ。**自分の Postgres を Store として持ち込む**。holdfast クラスタは立てない。ロックの“正しさ”だけ借りる。 | holdfast クラスタ（Raft 3 ノード）を `deploy/` で立て、**どの言語からでも SDK で叩く**（etcd を使う感覚）。自前ストレージ不要、holdfast 自身が複製で正しさを保証。 |

```go
// モード①：組み込み。Store を持ち込み、コアの正しさだけ使う。
l := holdfast.New(pg.Open(db))
lease, _ := l.Acquire(ctx, holdfast.AcquireRequest{Resource:"seat:A12", Units:1, TTL:2*time.Minute})
// …決済など…
receipt, _ := l.Commit(ctx, lease.ID, lease.Token) // 古い token は弾かれる
```

### SDK と FE は別レイヤー — FE は holdfast を直接叩かない

- **❶ SDK は出す**：ただし「サービス消費者 = **バックエンド**」向けのクライアント SDK（Go はネイティブ、TS/Python は proto から生成）。これは first-class。
- **❷ React は出さない**：応用例のまま。しかも**ブラウザは holdfast を直接呼ばない**——リースを握り・フェンシングトークンを提示し・冪等キーを管理するのはバックエンドの仕事。ブラウザは不安定（タブを閉じる）で信頼境界の外。座席予約 UI は「自分の back/」を叩き、back/ が holdfast SDK を使う。コーディネーション基盤を公開ネットに晒さない、は DB/etcd と同じ原則。

Phase 0–2 は主に**モード①**（コアと各 Store を作る）。Phase 3 以降の Envoy・IaC・SLO は、**モード②の holdfast クラスタ自身を運用する**話になる。

---

## 5. フェーズ詳細

各 Phase は `ゴール / 要件(MUST・STRETCH) / 設計の問い / 完了条件(DoD) / 落とし穴` の形式。**DoD を満たすまで次に進まない。**

---

### Phase 0 — 契約から作る（API・適合テスト・参照実装）

**ゴール**：コードより先に「API 契約」と「どのバックエンドも満たすべき性質テスト」を書く。それを通す最小の in-memory 実装を置き、React の例アプリで API の使い心地を検証する。
**新技術**：Go / 契約駆動 / 適合テスト / React 例アプリ

**要件**

- **MUST** `Leaser` と `Store` の API 契約を定義。エラー（`ErrExhausted` / `ErrConflict`）も型で。
- **MUST** 適合テスト（conformance suite）を先に書く：枠を超えて確保しない・commit は冪等・TTL 失効・古い token 拒否。
- **MUST** テストを通す in-memory Store（参照実装）を実装する。
- **MUST** React 例アプリ：リソースと残枠を並べた盤面、Acquire/Commit ボタン、空き枠のライブ表示。
- **STRETCH** 「例アプリを先に書いて、書きづらかったら API を直す」ドッグフーディングを 1 周回す。

**設計の問い**

- ライブラリが**約束すること**と**委ねること**の線は、型のどこに現れている？
- `Store` の最小契約は何か？ これ以上削ると正しさが壊れる一線はどこ？
- 冪等キーの意味論：同じキーで 2 回 `Acquire` したら、結果は何であるべき？

**完了条件 (DoD)**

- 適合テストが in-memory バックエンドで緑。
- 例アプリから端から端まで動き、API の形が「クライアントを書いた結果」で決まっている。

**落とし穴**

- 誰も使わない API を先に作り込む（→ 例アプリを先に書くことで回避）。
- Store に機能を盛りすぎてバックエンド実装の難度を上げる。ドメイン語彙をコアに漏らす。

---

### Phase 1 — 競合を作り、in-memory をアプリ層で守る

**ゴール**：参照実装（in-memory Store）に並行で殴りかかり、枠の二重確保を再現する。Go の並行プリミティブで Claim をアトミックにする。
**新技術**：Go 並行処理 / アプリ層の排他制御

**要件**

- **MUST** 同じリソースの枠を奪い合う並行負荷で二重確保を再現（`go test -race` + 並列 Acquire）。
- **MUST** `sync.Mutex` / channel の単一所有 / リソースごとの細粒度ロックで `Claim` をアトミックにする。
- **MUST** 適合テストの並行性プロパティが in-memory で緑になる。
- **STRETCH** 確保探索は並列、枠の確定だけ直列、という構造でスループットを上げる。

**設計の問い**

- リソース全体を 1 つの Mutex で守る vs リソースごとに守る——スループットとデッドロックのトレードオフは？
- Mutex と channel、この `Claim` ではどちらが素直に書ける？
- ロック自体が壁になったら、何を測ってどう緩める？

**完了条件 (DoD)**

- 高並行でも二重確保ゼロ、かつ単一ロックでスループットを殺していない。`-race` クリーン。

**落とし穴**

- グローバル Mutex で全リクエストを直列化。ロック順序不一致でデッドロック。

---

### Phase 2 — Postgres バックエンドを実装する

**ゴール**：同じ適合テストを、プロセスを跨いでも通るバックエンドで満たす。DB の行ロックと分離レベルで、Claim のアトミック性を DB に担わせる。
**新技術**：トランザクション / 行ロック / 分離レベル

**要件**

- **MUST** 悲観ロック：`SELECT … FOR UPDATE` で対象行を確保してから Claim。
- **MUST** ワークキュー化：`FOR UPDATE SKIP LOCKED` で、複数ワーカーが互いをブロックせず空き枠を引く。
- **MUST** 楽観ロック：`version` 列＋条件付き `UPDATE` でも同じ正しさを実装し、悲観版と比較。
- **MUST** Phase 0 の適合テストを**無改造で Postgres バックエンドに通す**。
- **STRETCH** 分離レベルを切り替え、現れる異常の差を観察する。

> **低レイヤーの核（CORE）：枠を壊す異常は「write skew」**
> 二者がそれぞれ「まだ枠が空いている」と読んで確認してから同時に確保すると、各トランザクション単体は矛盾なく見えるのに全体は枠超過（二重確保）になる——これが write skew。`FOR UPDATE` で読む行を物理的に押さえるか、`SERIALIZABLE` で DB に検知・中断させるかで防ぐ。「なぜ `REPEATABLE READ` では防ぎきれない異常があるのか」を実際にぶつけるのが肝。（DDIA 7 章）

**設計の問い**

- 悲観 vs 楽観：競合が多い／少ないでどちらが有利？ リトライ前提なのはどちら？
- `SKIP LOCKED` はなぜ「空き枠を配る」のに効く？ 普通の `FOR UPDATE` だと何が起きる？
- 行ロックを握ったまま外部 API（決済・通知）を呼ぶと、何が壊れる？

**完了条件 (DoD)**

- 複数プロセスから競合させても二重確保ゼロ（悲観・楽観の両方）。serialization failure を検知して安全にリトライ。

**落とし穴**

- ロック保持中に外部呼び出しを挟み保持時間が爆発。楽観ロックの衝突リトライを書かず lost update。

---

### Phase 3 — コアをサービスで包む（gRPC + 多言語 SDK）

**ゴール**：storage-agnostic なコアを薄い gRPC サービスで包む。これが (C) の「サービス化」。etcd / Chubby 型のリースサービスになり、多言語クライアントから叩ける。
**新技術**：gRPC / Protocol Buffers / SDK

**要件**

- **MUST** コアを薄い gRPC サービスで包む。proto で `Acquire/Commit/Release` を定義。
- **MUST** 別言語の SDK を生成（例：React 例アプリ用の TypeScript、または Python）。
- **MUST** 冪等キーを API に通し、再送が安全であることを保証する。
- **MUST** graceful shutdown：SIGTERM で新規受付を止め、処理中 RPC を待ってから終了（drain）する。
- **MUST** `/livez`（自分が生きているか）と `/readyz`（Store に繋がっているか）を分ける。
- **MUST** `context.Context` による timeout を全 RPC に通す。サーバの read/write deadline も設定する。
- **STRETCH** 空き枠のライブ更新を server streaming で配信し、React 盤面に反映。

**設計の問い**

- 「組み込み（import）」と「サービス（gRPC）」で、API 契約は同じに保てる？ どこがズレる？
- 後方互換なスキーマ進化とは？ フィールド番号の再利用・削除で何が壊れる？
- Acquire は同期呼び出しか、キュー投入の非同期か。リアルタイム性と信頼性のどちらを取る？
- `/livez` に Store の健全性チェックを入れると何が起きる？（→ Phase 6 の落とし穴に直結）

**完了条件 (DoD)**

- 同じコアが import でもサービス越しでも使え、適合テストの本質が両方で通る。生成 SDK で React 例アプリが動く。
- SIGTERM 送信時、処理中 RPC が 1 件も切り捨てられない。

**落とし穴**

- chatty な往復でレイテンシ悪化。proto の破壊的変更を無自覚に入れる。
- timeout を設定せず、Store のハングが goroutine を食い尽くす（cascading failure の典型）。
- liveness に Store の健全性チェックを入れて、Store 不調で自分が再起動ループに陥る。

---

### Phase 4 — Envoy サイドカーで信頼性をメッシュへ出す

**ゴール**：holdfast クラスタ（モード②）への通信を Envoy で守る。timeout・retry・サーキットブレーカ・mTLS を Envoy 設定で実現。冪等キーを最初から設計に入れたおかげで、メッシュの自動 retry が安全に効く——という伏線がここで回収される。
**新技術**：Service Mesh / Envoy / 信頼性の責務分離

**要件**

- **MUST** 各 Pod に Envoy をサイドカー注入し、サービス間通信を通す。
- **MUST** Envoy 設定だけで timeout / retry / circuit breaking（接続・リクエスト上限）/ outlier detection を実現。
- **MUST** サービス間を mTLS 化する。
- **STRETCH** bulkhead：upstream（Store / Postgres）ごとに同時接続数を隔離し、1 つの障害が全体を道連れにしない。
- **STRETCH** カナリア／トラフィック分割やリクエストミラーリングを設定で。

> **伏線回収（CORE）：なぜメッシュの retry が安全なのか**
> 一般にメッシュの自動 retry は危険——非冪等な操作を勝手に再送すると二重実行になる。だが holdfast は Phase 0 から冪等キーを契約に持つので、同じキーの再送は同じ結果に畳まれる。「ライブラリの API 設計が、インフラ層(Envoy)の安全性を担保する」——層をまたいだ設計の好例。

**設計の問い**

- アプリに残す責務（冪等性・整合性）と、メッシュに出す責務（再送・遮断）の境界は？
- retry を**アプリとメッシュの両方**に持つと何が起きる？ retry 予算はどこで管理する？
- circuit breaker の half-open は何のためにある？ いつ「もう回復した」と判断する？
- 全クライアントが一斉に retry すると retry storm で下流にとどめを刺す。指数バックオフ＋jitter はどう防ぐ？

**完了条件 (DoD)**

- サービスを過負荷／停止させると Envoy が fail-fast し、復帰後に自動で戻る。
- メッシュが再送しても二重確保が起きないことをテストで示す。

**落とし穴**

- アプリとメッシュの二重 retry で増幅。メッシュが障害を吸収して根本原因を隠す。

---

### Phase 5 — Raft バックエンドを「自分で作る」（合意 & フェンシング）

**ゴール**：単一 DB に縛られず、複製された状態の上でリースを成立させる。Phase 0 から API に居たフェンシングトークンが、ノードを跨いだ瞬間に主役として効いてくる。ここが最深部。
**新技術**：合意（Raft）/ リース / フェンシングトークン

**要件**

- **MUST** まず素朴に **leader-follower の非同期複製**を実装する：書き込みは leader、follower は追従。follower read で古い値（staleness）を実際に観察する。
- **MUST** 次に**合意アルゴリズム**で線形化可能性を達成する。複製ログ（`hashicorp/raft`、または自前 → MIT 6.5840）を土台に Raft Store を実装する。
- **MUST** Phase 0 の適合テストを Raft バックエンドにも**無改造で通す**（3 つ目の通過バックエンド）。
- **MUST** 1 ノード落としても Acquire／Commit が継続し、復帰ノードが catch up する。
- **STRETCH** 分断下でも枠超過（二重確保）しないことを Jepsen/Maelstrom 流の履歴検証（線形化可能性）で示す。

> **低レイヤーの核（CORE）：ロックを持つことは「安全に書ける」ことではない**
> リース型ロックの罠：保持者が GC ポーズ／分断で長時間止まる間にリースが失効し、別ノードが取得。やがて目覚めた古い保持者が「まだ持っている」と思い込み、枠を確定して二重確保する。解決がフェンシングトークン——Acquire のたびに単調増加する番号を、リソース側（Store）が検証して古い番号を拒否する。このトークンを Phase 0 から API に置いておいたのは、まさにこの瞬間のため。（DDIA 8 章 / Kleppmann のフェンシング論）

**設計の問い**

- リースは壁時計に依存する。クロックのずれ・一時停止で何が崩れる？
- 分断時、誰がリースを持つ？ quorum はどう split brain を防ぐ？
- 「リーダーが生きている」と「リーダーが最新ログを持つ」は同じか？

**完了条件 (DoD)**

- 同じ適合テストが in-memory / Postgres / Raft の 3 バックエンド全部で緑。
- 古いフェンシングトークンの commit が、Store 側で確実に弾かれることをテストで示す。

**落とし穴**

- フェンシング無しで「取れた＝安全」と誤解。壁時計リースで clock skew に踏み抜かれる。quorum を取らず split brain。

---

### Phase 6 — Terraform & k8s で基盤をコード化する

**ゴール**：**holdfast クラスタ自身**（サーバ・Postgres・Envoy・マイグレーション）を、コマンド一発で再現可能にする。これが `deploy/terraform`。利用者が holdfast を「すぐ立てられる」module を宣言的に用意する。
**新技術**：Terraform / Kubernetes / 宣言的インフラ

**要件**

- **MUST** multi-stage の Dockerfile を書く。`CGO_ENABLED=0` の静的バイナリを distroless か scratch に載せ最小イメージにする。
- **MUST** Terraform でクラスタ（ローカルは `kind`、本番志向は EKS）とマネージド Postgres を構築。
- **MUST** サービス / サイドカー注入 / Envoy 設定 / Secret を宣言的に管理。
- **MUST** DB マイグレーションをデプロイの一部（Job / フック）として再現可能に。
- **MUST** 再利用単位を module に切り出す。
- **MUST** `PodDisruptionBudget` で同時退避上限を設定し、rolling update 中も最低限の稼働を保証する。
- **STRETCH** HPA で CPU（またはカスタム指標）に応じてスケール。
- **STRETCH** state を lock 付き remote backend へ。「複数人が apply」前提を体験。

**設計の問い**

- state は誰が・どこに持つ？ apply の権利は誰が握る？
- マイグレーションをデプロイにどう織り込む？ ロールバック時スキーマはどう戻す？
- 手で `kubectl edit` された drift を `plan` はどう見せ、どう戻す？
- `resources.requests` と `limits` を同じ値にすると QoS class はどうなる？ CPU throttling と OOMKilled はなぜ挙動が違う？
- rolling update 中に古い Pod が SIGTERM を受けてから消えるまで、readiness と `terminationGracePeriodSeconds` はどう連携する？（Phase 3 の graceful shutdown が効いてくる）

**完了条件 (DoD)**

- `terraform apply` 一発で、クラスタ＋DB＋サービス＋Envoy がゼロから立つ。`destroy` で綺麗に消える。
- `kubectl rollout` による更新がクライアントから見て無停止（5xx が出ない）。
- Pod を手で delete しても自動復旧し、数秒でトラフィックに復帰する。

**落とし穴**

- state lock 無しの同時 apply で破壊。Secret を state/tfvars に平文。マイグレーションを手運用に残す。
- `readinessProbe` の初期遅延不足で、起動途中の Pod にトラフィックが流れる。
- memory limit が過小で OOMKilled を繰り返す。

---

### Phase 7 — 分散トレーシングと SLO で運用する

**ゴール**：**holdfast 自身**を運用可能にする。FE → Envoy → サーバ → Store（Postgres / Raft）までを 1 トレースで貫き、どこで遅延するかを見える化。holdfast のリース操作に SLO を定義し、カオスでわざと割って気づけることを確かめる。
**新技術**：OpenTelemetry / SLO / カオス試験

**要件**

- **MUST** OTel トレース伝播：Envoy が `traceparent` を伝播、各サービスがスパンを継ぐ。1 リクエストの内訳が 1 トレースで見える。
- **MUST** RED 指標（Rate / Errors / Duration、レイテンシは histogram）と構造化ログ（`log/slog`）。
- **MUST** SLI / SLO を定義（例：Acquire 成功率 99.9%、p99 < Xms）し error budget を導く。
- **MUST** 負荷（k6 / vegeta、需要スパイク=フラッシュセールを模す）＋ カオス（pod kill・DB 遅延・Raft をパーティション）で SLO 違反を再現。
- **STRETCH** burn rate アラートを実装する（短期ウィンドウ 1h + 長期ウィンドウ 6h の 2 段構えで、急激な error budget 消費を早期検知する）。
- **STRETCH** 「確保競合率」「期限切れ回収の遅れ」専用の SLI を設け、React 側に簡易 SLO ダッシュボードを出す。

**設計の問い**

- 良い SLI はユーザー体験に近い。CPU 使用率が SLI として弱いのはなぜ？
- 平均は tail を隠す。p99 / p999 を見るべき理由と histogram bucket の切り方は？
- アラートを「原因」でなく「症状」で鳴らすとは、具体的にどう設計する？
- burn rate が「1」を超えるとはどういう意味か？ 短期 + 長期の 2 ウィンドウを使う理由は？（SRE Workbook ch.5）

**完了条件 (DoD)**

- 1 トレースで FE→Store までの内訳が見え、遅延箇所を指させる。カオスで意図的に SLO を割り、アラートが検知。

**落とし穴**

- 高カーディナリティラベルでメトリクス破裂。cause-based アラート過多で症状を見逃す。

---

## 6. 巻末 A：同じ契約・別のバックエンド — CAS をどの層で担うか

1 つの適合テストが、下の 3 つ全部を通る。この表を自分の言葉で埋め直せたら理解は本物。

| 関心事 | in-memory（アプリ層） | Postgres（DB 層） | 自作 Raft（合意層） |
|---|---|---|---|
| **Claim の原子性** | Mutex / channel | 行ロック・楽観ロック | 複製ログへの合意 |
| **フェンシング** | 単調カウンタ | token 列の条件付き更新 | ログの index が token になる |
| **耐障害** | 無し（単一プロセス） | DB の HA に依存 | quorum で継続 |
| **一貫性** | プロセス内で自明 | 分離レベル | 線形化可能性 |

メッシュ層(Envoy)は「Store の実装」ではなく、サービス間の信頼性(timeout/retry/mTLS)を担う直交した層として全バックエンドに乗る。

---

## 7. 巻末 B：面接で効く「署名」になる成果物

| 成果物 | なぜ効くか |
|---|---|
| クロスバックエンド適合テスト | 「バックエンドを増やしても同じ性質テストが通る」は、契約設計と実装力を同時に示す。 |
| フェンシングの安全性テスト | 「古いトークンの書き込みを弾く」を実演できると、分散の落とし穴を分かっている証拠になる。 |
| 線形化検証（Jepsen/Maelstrom 流） | 分断下でも枠超過しないことを履歴検証。MIT 6.5840 / Gossip Glomers と地続き。 |
| 「汎用の高さを自分で選んだ」設計判断 | mutex でもドメインアプリでもなく semaphore 相当に置いた理由を語れること自体がシグナル。 |

---

## 8. 巻末 C：本・教材との対応

| テーマ | 対応する本・教材 |
|---|---|
| トランザクション・分離レベル・write skew | *DDIA* 7 章 |
| 分散ロック・フェンシング・リーダーとロック | *DDIA* 8 章 ／ Kleppmann「How to do distributed locking」 |
| 複製・線形化可能性・合意 | *DDIA* 5・9 章 |
| 耐障害パターン（timeout/CB/bulkhead） | *Release It!* |
| SLO・監視・トイル削減 | *SRE Book* |
| Raft を自分で実装（Phase 5 の本丸）・線形化検証 | **MIT 6.5840**（Go ラボ）／ **Fly.io Gossip Glomers** |
| k8s の内部を手で組む | **Kubernetes the Hard Way** |

### この 1 周を終えたら

各フェーズの「設計の問い」への答えを並べると、そのまま濃い LT 群になる。例：「枠を壊す異常はなぜ write skew か」「分散ロックを持つことが安全を意味しない理由（フェンシング）」「冪等キーがなぜメッシュの retry を安全にするのか」「汎用の高さを semaphore 相当に置いた理由」。作って・壊して・説明できる状態が、理解の定着の証拠。
