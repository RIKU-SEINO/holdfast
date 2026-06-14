# STUDY GUIDE — holdfast

> ROADMAP.md が「何を作るか」なら、このファイルは「どう学ぶか」。
> Phase に入る前にここを読み、クイズを解いてから手を動かす。

---

## 学習の進め方（全 Phase 共通）

1. **事前読書**（1〜2 時間）→ 2. **クイズ①**（読んだ直後） → 3. **実装** → 4. **クイズ②**（実装後）→ 5. **振り返り**

- 「全部読んでから作る」は NG。章の途中で手を動かす。
- クイズは声に出して答える。言語化できなければ理解が浅い。
- 「わからなくて当然」な概念は、実装でハマってから読み直すと 10 倍頭に入る。
- TS/Node との比較メモを手元に残すと後で資産になる。

---

## Phase 0 — Go 基礎 + コア契約（完了）

### 事前読書

| 素材 | 何を得る |
|---|---|
| [A Tour of Go](https://go.dev/tour) — Basics・Methods and interfaces | Go の型システム・interface の構造的部分型 |
| 『プログラミング言語 Go』（Donovan & Kernighan）ch.1–2 | TS との書き方の差（`:=`・ゼロ値・`error` 返し） |

### クイズ①（読後）

1. Go の interface は「宣言して実装します」と書かなくてもよい。TS の `implements` と何が違う？
2. `var x int` のとき `x` の値は何？ TS で言う `undefined` とどう違う？
3. `errors.New("foo") == errors.New("foo")` は `true` か `false` か？ なぜ？
4. Go に例外（try/catch）がない。エラーをどう伝える？

### クイズ②（実装後）

1. `Store` interface を満たすのに `*MemoryStore` と `MemoryStore` のどちらを使った？ ポインタが必要な理由は？
2. `holdfast.ErrExhausted` を package-level var にした理由を「呼び出し側が `==` で比較できるから」以外の言葉で説明せよ。
3. map の値（struct）を直接 `s.resources[key].used++` と書けないのはなぜ？

---

## Phase 1 — Go 並行（進行中）

### 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [Go Tour: Concurrency](https://go.dev/tour/concurrency) | goroutine・channel・`sync.WaitGroup` の基本 | ★★★ |
| [Go Blog: The Go Memory Model](https://go.dev/ref/mem) — Happens-before だけ | 「なぜ race が壊れるか」の根拠 | ★★★ |
| 『Concurrency in Go』（Cox-Buday）ch.1–3 | mutex / channel の使い分け、race detector の読み方 | ★★☆ |
| [Go Blog: Share Memory By Communicating](https://go.dev/blog/codewalks) | Go の設計哲学（channel 優先） | ★☆☆ |

> **TS との対比**：Promise / async-await は「非同期 I/O の待機」、goroutine は「並行実行スレッド」。別の問題を解いている。

### クイズ①（読後）

1. goroutine はスレッドか？ OS スレッドと何が違う？
2. `go test -race` は何を検知する？ "data race" とは何か 1 文で定義せよ。
3. `sync.Mutex` の `Lock()`/`Unlock()` が「happens-before」を保証するとはどういう意味？
4. channel と mutex、どちらを使うべき場面を 1 つずつ挙げよ。
5. `defer mu.Unlock()` はなぜ慣用的か？ `defer` なしで書くとどのリスクがある？

### クイズ②（実装後）

1. `-race` で落ちたとき、出力の `DATA RACE` セクションには何が書いてあった？ どの行同士が競合していた？
2. Mutex を追加する前と後で、テストの実行時間はどう変わった？ なぜ？
3. `sync.RWMutex` を使うとしたら `MemoryStore` のどのメソッドで `RLock` を使える？ 今の実装で使えない理由は？

---

## Phase 2 — Postgres バックエンド

### 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| 『データ指向アプリケーションデザイン』（DDIA）ch.7 — トランザクション | ACID・分離レベル・lost update・write skew | ★★★ |
| [Postgres Docs: Transaction Isolation](https://www.postgresql.org/docs/current/transaction-iso.html) | Postgres の実際の挙動 | ★★★ |
| [pgx README](https://github.com/jackc/pgx) | Go の Postgres ドライバの使い方 | ★★☆ |
| DDIA ch.3 — ストレージエンジン（B-Tree の節のみ） | インデックスがロックとどう絡むか | ★☆☆ |

### クイズ①（読後）

1. ACID の「I（Isolation）」は何を保証する？ 完全な分離（Serializable）のコストは？
2. Read Committed と Repeatable Read の違いを「ファントム読み」の例で説明せよ。
3. 楽観ロック vs 悲観ロック：holdfast の `Acquire` にどちらが向く？ 理由は？
4. `SELECT FOR UPDATE` は何をロックする？ どの分離レベルで意味を持つ？
5. `holdfast.ErrConflict` を Postgres で実装するとき、どの SQL エラーコードをハンドルする？

### クイズ②（実装後）

1. 同じ conformance suite が in-memory と Postgres の両方で通った。何がそれを可能にしているか？
2. Postgres バックエンドでトークンの単調増加を保証するには `nextToken` の代わりに何を使った？
3. `pgx` のコネクションプールを使った。プールサイズをいくつにしたか？ その判断根拠は？

---

## Phase 3 — gRPC + SDK

### 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [gRPC Basics: Go](https://grpc.io/docs/languages/go/basics/) | proto 定義 → コード生成 → サーバ実装の流れ | ★★★ |
| [Protocol Buffers Language Guide (proto3)](https://protobuf.dev/programming-guides/proto3/) | フィールド番号・後方互換ルール | ★★★ |
| [Go Blog: Contexts and structs](https://go.dev/blog/context-and-structs) | `context.Context` の正しい使い方 | ★★☆ |
| 『SRE ブック』ch.8 — Release Engineering（graceful shutdown の節のみ） | drain の意味、SIGTERM の受け取り方 | ★★☆ |
| [Kubernetes: Configure Liveness, Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) | `/livez` vs `/readyz` の使い分け | ★★☆ |

### クイズ①（読後）

1. REST と gRPC の最大の違いを「契約」の観点から説明せよ。
2. proto のフィールド番号を変えると何が壊れる？ フィールドを削除するときの安全な手順は？
3. `context.WithTimeout` でタイムアウトを設定したとき、サーバ側はそれをどう受け取る？
4. SIGTERM を受けてから SIGKILL までの猶予時間（`terminationGracePeriodSeconds`）に何をすべきか？
5. `/livez` に DB の疎通確認を入れるとどんな問題が起きる？

### クイズ②（実装後）

1. gRPC の `status.Code` と `holdfast.ErrExhausted` をどうマッピングした？ `codes.ResourceExhausted` か `codes.Unavailable` か、その判断理由は？
2. 処理中の RPC を drain するために `grpc.Server.GracefulStop()` を使ったとき、それが完了するまで何が起きているか？
3. 生成した TypeScript SDK の `Acquire` 呼び出しは、タイムアウト時にどのエラーを投げる？

---

## Phase 4 — Envoy サイドカー

### 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [Envoy Getting Started](https://www.envoyproxy.io/docs/envoy/latest/start/quick-start/run-envoy) | xDS・listener・cluster の概念 | ★★★ |
| [Envoy Docs: Circuit breaking](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/circuit_breaking) | half-open の仕組み | ★★★ |
| 『Designing Distributed Systems』（Burns）ch.2 — Sidecar Pattern | なぜアプリと分離するか | ★★☆ |
| [Envoy Docs: Retry semantics](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#x-envoy-retry-on) | retry 予算・retry storm | ★★☆ |

### クイズ①（読後）

1. サイドカーパターンの利点を「アプリを変更せずに〇〇を追加できる」という形で 3 つ挙げよ。
2. circuit breaker の 3 状態（closed / open / half-open）を図なしで説明せよ。
3. retry storm とは何か？ jitter はどう防ぐ？
4. bulkhead は何を「隔壁」で守っている？ Envoy でどう設定する？

### クイズ②（実装後）

1. `holdfast.Acquire` は冪等か？ 冪等でない場合、Envoy の retry はなぜ危険か？
2. circuit breaker が open になったとき、クライアントには何が返る？ アプリでどうハンドルする？

---

## Phase 5 — Raft バックエンド

### 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| DDIA ch.8 — 分散システムの問題 | ネットワーク分断・クロック・プロセス停止 | ★★★ |
| DDIA ch.9 — 一貫性と合意 | 線形化可能性・2PC・Raft の位置づけ | ★★★ |
| [Raft Paper](https://raft.github.io/raft.pdf)（Ongaro & Ousterhout）§1–5 | リーダー選出・ログ複製の仕組み | ★★★ |
| [Raft Visualization](https://raft.github.io/) | 直感的理解（必ずインタラクションせよ） | ★★★ |
| [hashicorp/raft README](https://github.com/hashicorp/raft) | Go 実装の使い方 | ★★☆ |

> **この Phase の前に**: 非同期レプリケーション（leader-follower）を先に自作する。「合意なしのレプリカ」の何が怖いかを体で知ってから Raft に進む。

### クイズ①（読後）

1. 「線形化可能性（linearizability）」を 1 文で定義せよ。「直列化可能性（serializability）」と何が違う？
2. Raft でリーダーが死んだとき、何が起きる？ quorum とは何個以上か？
3. フェンシングトークンが「止まっていた古いリーダーの書き込み」を防ぐ仕組みを説明せよ。
4. ログインデックスがフェンシングトークンとして使える理由は？
5. split brain はいつ起きる？ Raft はなぜ split brain を起こさないか？

### クイズ②（実装後）

1. 非同期レプリカで conformance suite を通したとき、どのテストが落ちた？ なぜ？
2. Raft FSM の `Apply` メソッドで `Acquire` を実装したとき、「何をコミットログに書くか」を説明せよ。
3. 適合テストが in-memory / Postgres / Raft の全部で通る。これが「Store の契約が正しい」ことを意味する理由は？

---

## Phase 6 — Terraform + k8s でクラスタ構築

### 事前読書（クラウドインフラ）

> **この Phase が「クラウドインフラ」の本丸**。AWS CDK/CloudFormation の経験が Terraform を理解する土台になる。

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [Terraform Getting Started (AWS)](https://developer.hashicorp.com/terraform/tutorials/aws-get-started) — resource / variable / output | CDK との対比で IaC を理解 | ★★★ |
| [Kubernetes Basics Interactive Tutorial](https://kubernetes.io/docs/tutorials/kubernetes-basics/) | Pod / Deployment / Service の関係 | ★★★ |
| [k8s Docs: PodDisruptionBudget](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) | rolling update と可用性の保証 | ★★★ |
| [Dockerfile multi-stage builds](https://docs.docker.com/build/building/multi-stage/) | builder と runner を分けるなぜ | ★★☆ |
| [distroless](https://github.com/GoogleContainerTools/distroless) | scratch より使いやすい最小イメージ | ★★☆ |
| [Terraform Docs: Remote Backend (S3)](https://developer.hashicorp.com/terraform/language/backend/s3) | state の共有とロック | ★★☆ |
| [k8s Docs: HPA](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) | スケーリング指標の選び方 | ★☆☆ |

> **CDK/CFn との対比**（あなたの経験が活きる）
> - Terraform の `resource` ≈ CDK の `new Construct(this, ...)`
> - Terraform の `tfstate` ≈ CloudFormation の stack state
> - `terraform plan` ≈ `cdk diff`
> - `terraform apply` ≈ `cdk deploy`
> - 最大の違い：Terraform は**宣言的 HCL**、CDK は**手続き的 TypeScript**。「どう作るか」を書くのが CDK、「何が欲しいか」だけ書くのが Terraform。

### クイズ①（読後）

1. Terraform の state ファイルは何を記録している？ 壊れるとどうなる？
2. `terraform plan` が「差分なし」を返すのはどういう状態か？
3. k8s の `Pod` と `Deployment` の違いは？ なぜ Pod を直接作らないのか？
4. `resources.requests` と `limits` を同じ値に設定すると QoS class は何になる？ なぜそれが保証になる？
5. multi-stage Dockerfile で builder と runner を分ける理由を「攻撃面」と「イメージサイズ」の両方から説明せよ。
6. `PodDisruptionBudget` の `maxUnavailable: 1` は何を保証する？ `minAvailable` との違いは？

### クイズ②（実装後）

1. `terraform destroy` を実行する前に確認すべきことを 3 つ挙げよ。
2. rolling update 中に 5xx が出なかったとしたら、それを可能にした k8s の仕組みを順番に説明せよ。
3. Raft クラスタを k8s StatefulSet でデプロイした。なぜ Deployment ではなく StatefulSet か？

---

## Phase 7 — 分散トレーシング + SLO + カオス

### 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| 『SRE ブック』（Google）ch.4 — SLO、ch.6 — モニタリング | SLI/SLO/error budget の定義 | ★★★ |
| 『SRE Workbook』ch.5 — Alerting on SLOs | burn rate アラートの設計 | ★★★ |
| [OpenTelemetry Docs: Go](https://opentelemetry.io/docs/languages/go/) | trace / metric / log の三本柱 | ★★★ |
| [Prometheus Docs: histogram_quantile](https://prometheus.io/docs/practices/histograms/) | p99 の計算方法、bucket の切り方 | ★★☆ |
| [Chaos Engineering](https://principlesofchaos.org/) — Principles | 「何を壊すか」の設計思想 | ★★☆ |

### クイズ①（読後）

1. SLI・SLO・SLA の違いを 1 行ずつで。
2. error budget とは何か？ 「30 日間 99.9% SLO」なら月何分使える？
3. burn rate が「1」を超えるとはどういう意味か？ burn rate 2 なら error budget は何日で尽きる？
4. 短期ウィンドウ（1h）と長期ウィンドウ（6h）の 2 段構えアラートは、1 段のみと何が違う？
5. CPU 使用率が「悪い SLI」な理由を「ユーザー体験」の言葉で説明せよ。
6. `sum(rate(errors[5m])) / sum(rate(requests[5m]))` は何を計算している？

### クイズ②（実装後）

1. カオス試験で pod kill をしたとき SLO が割れたか？ 割れたなら何を直した？
2. 1 トレースで「Envoy → gRPC サーバ → Postgres」の内訳が見えた。遅延が一番大きかった区間はどこか？
3. burn rate アラートが鳴ったとき、最初に見るダッシュボードのパネルはどれか？ なぜ？

---

## 参考書籍まとめ

| 書籍 | Phase | 一言メモ |
|---|---|---|
| 『プログラミング言語 Go』Donovan & Kernighan | 0–1 | Go の教科書。辞書として手元に置く |
| 『Concurrency in Go』Cox-Buday | 1 | goroutine / channel / race の決定版 |
| 『データ指向アプリケーションデザイン』Kleppmann (DDIA) | 2・5・7 | 分散システム全体の地図。特に ch.7・8・9 |
| 『SRE ブック』Google | 3・7 | 運用の設計思想。graceful shutdown と SLO の根拠 |
| 『SRE Workbook』Google | 7 | burn rate アラートの具体的設計が ch.5 にある |
| Raft Paper (Ongaro & Ousterhout 2014) | 5 | 論文だが読みやすい。30 ページで Raft の全体 |
| 『Designing Distributed Systems』Burns | 4–5 | サイドカー / レプリカ / バッチパターンの設計 |
