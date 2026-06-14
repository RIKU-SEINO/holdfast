# 学習メモ — holdfast

> 自分専用の記録。実装中に気づいたこと・ハマったこと・設計の問いへの答えを書き溜める。
> クイズの回答欄は `> 回答：` の後に記入する。

---

## Go の癖メモ（TS と違うところ）

### `:=` と `=` の使い分け

```go
err := store.Register(...)   // 新規宣言（:= で err を生やす）
_, err = store.Acquire(...)  // 再代入（err はもう存在する → = だけ）
```

> **TS との違い**：TS は `let err` で宣言してから何度でも `=` で代入できる。Go の `:=` は「宣言 + 代入」が同時に起きるので、同スコープで 2 回使うとコンパイルエラー。
> ただし、左辺に**新しい変数が 1 つでもあれば** `:=` を使える（`lease, err :=`）。

---

### interface は「構造的部分型」

```go
// TS: implements を書く
class MemoryStore implements Store { ... }

// Go: 書かなくてよい。メソッドが揃っていれば自動で満たす
type MemoryStore struct { ... }
func (s *MemoryStore) Acquire(...) { ... }  // ← これだけで Store を満たす
```

> コンパイラが「このメソッド群を持っているか」を自動でチェックする。明示的な宣言は不要。

---

### ポインタレシーバ（`*MemoryStore` vs `MemoryStore`）

```go
func (s *MemoryStore) Acquire(...) { ... }  // ポインタレシーバ
func (s MemoryStore) Acquire(...)  { ... }  // 値レシーバ
```

- `*MemoryStore`：フィールドへの**変更が呼び出し元に反映される**
- `MemoryStore`：**コピーが渡る**ので変更が捨てられる
- MemoryStore は `leases` / `resources` map を持ち書き換えるので **`*MemoryStore` 一択**

---

### ゼロ値

```go
var x int      // x == 0（undefined ではない）
var s string   // s == ""
var b bool     // b == false
var p *int     // p == nil
```

> TS の `undefined` と違い、Go の変数は**必ず何かの値を持つ**。これを前提にしないとバグる。

---

### エラーハンドリング（例外なし）

```go
// Go: エラーは戻り値で返す
lease, err := store.Acquire(ctx, req, now)
if err != nil {
    return holdfast.Lease{}, err
}

// TS: throw / try-catch
```

> `error` は最後の戻り値が慣習。複数の戻り値が自然に書けるのが Go の強み。

---

### package-level エラー変数

```go
// 悪い例：インラインで作ると == 比較ができない
return errors.New("holdfast: no units available")  // 毎回別のポインタが生まれる

// 良い例：package-level で定義する
var ErrExhausted = errors.New("holdfast: no units available")
// 呼び出し側: if err == holdfast.ErrExhausted { ... }
```

> `errors.New("foo") == errors.New("foo")` は **`false`**。
> Go のポインタ比較なので、同じ文字列でも別の変数は別のアドレスを持つ。

---

### map の struct 更新（コピー・修正・書き戻し）

```go
// コンパイルエラー（map から取得した struct は addressable でない）
s.resources[key].used += units

// 正しい書き方
state := s.resources[key]   // コピー
state.used += units          // コピーを修正
s.resources[key] = state    // 書き戻す
```

> TS の `obj[key].field = ...` と違い、Go の map 値はアドレスを取れないため直接書き換えできない。

---

### `if _, ok := m[key]; ok` パターン

```go
if _, exists := s.resources[req.Resource]; exists {
    return nil  // すでに登録済み
}
```

> 読み方：`s.resources[req.Resource]` を取得して `_`（値）と `exists`（存在フラグ）に分解。
> TS の `if (key in obj)` に近いが、Go では取得と存在確認を**同時に**行う。
> `;` より前が「初期化文」、後が「条件式」。

---

### `defer` の慣用句

```go
mu.Lock()
defer mu.Unlock()  // 関数を抜けるときに必ず実行される
```

> `defer` なしだと `return` を忘れたり panic が起きたときに Unlock が漏れる。
> Go では `defer mu.Unlock()` が mutex のデファクト標準。

---

## 設計メモ

### `Capacity` はどこに置くか（Phase 0 での問い）

**最初の案**：`AcquireRequest` に `Capacity` を置く
**問題**：`Acquire` するたびに「最大 N 個まで」を渡すのは不自然。リソースの「枠」は事前に決めておくもの。

**決定**：`RegisterRequest` に `Capacity` を置き、`Register()` を `Store` に追加
```go
store.Register(ctx, holdfast.RegisterRequest{Resource: "seats", Capacity: 100}, now)
store.Acquire(ctx, holdfast.AcquireRequest{Resource: "seats", Units: 1, TTL: 5*time.Minute}, now)
```

> 「登録（容量の宣言）」と「確保（使用）」を分けることで意図が明確になる。

---

### `LeaseID` の生成方法

**最初の案**：`IdempotencyKey` を LeaseID に使う
**問題**：`IdempotencyKey` が空のリクエストが複数来ると衝突する

**決定**：`fmt.Sprintf("%s-%d", resource, token)` でトークンから生成
```go
token := s.nextToken
leaseId := fmt.Sprintf("%s-%d", req.Resource, token)
s.nextToken++
```

> `token` をキャプチャしてから `nextToken` をインクリメントする順序が重要。
> 逆にすると返す `Lease.Token` とレコードに保存した `token` がズレる（実際にハマった）。

---

### `IdempotencyKey` を Phase 0 で実装しなかった理由

「同じキーで再送したら同じ Lease を返す」という冪等性は、TTL 管理が安定してから実装すべき。
Phase 0 の優先事項は「基本的な Acquire / Commit / Release / Reap が正しく動くこと」。

---

### `validateLease()` ヘルパーを抽出した理由

`Commit` と `Release` が全く同じチェック（leaseID 存在確認 + token 一致確認）を行う。
DRY の観点から `validateLease()` に切り出した。

```go
func (s *MemoryStore) validateLease(leaseID string, token uint64) (leaseRecord, error) {
    lease, exists := s.leases[leaseID]
    if !exists || lease.token != token {
        return leaseRecord{}, holdfast.ErrConflict
    }
    return lease, nil
}
```

---

### TTL の「厳密に後」セマンティクス

```go
func (l leaseRecord) isExpired(now time.Time) bool {
    return now.After(l.expires)  // 厳密に後（== は期限切れではない）
}
```

テストでは `leaseExpires.Add(time.Nanosecond)` を渡して「ちょうど 1ns 後」を確認。
`now == expires` のとき**まだ有効**とするのが自然（期限の瞬間は猶予を含む）。

---

## ミスログ（Phase 0）

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| 1 | 同スコープで `err :=` を 2 回使った | `:=` の意味を誤解 | 2 回目は `=` |
| 2 | `AcquireRequest` に `Capacity` を置いた | 「確保」と「登録」を混同 | 容量は `Register` で先に宣言 |
| 3 | `leaseID string` のフィールドを小文字にした | exported の概念が抜けた | パッケージ外から使うフィールドは大文字 |
| 4 | `s.resources[key].used += units` | map の struct は addressable でない | コピー→修正→書き戻し |
| 5 | `nextToken` をインクリメントしてから `token` に代入 | 順序ミス | `token := s.nextToken` → `s.nextToken++` の順 |
| 6 | import パスを `holdfast/memory` にした | ディレクトリ構造を確認せず | 正しくは `holdfast/store/memory` |
| 7 | `package memory` にした（テストファイル） | 外部テストパッケージの慣習を知らなかった | `_test` サフィックスで外部テスト |
| 8 | `testExhausted` で `Capacity` を書き忘れた | RegisterRequest の変更後にテストを更新し忘れ | 型変更後は全テストを確認 |
| 9 | `errors.New(...)` をインラインに書いた | package-level にすべきと気づかなかった | 呼び出し側が `==` で比較するものは package-level に |

---

## クイズ回答欄

> 各 Phase の実装前後に回答を記入する。「言語化できるか」が理解の指標。

---

### Phase 0 クイズ①（読後）

**Q1. Go の interface は「宣言して実装します」と書かなくてもよい。TS の `implements` と何が違う？**

> 回答：

**Q2. `var x int` のとき `x` の値は何？ TS で言う `undefined` とどう違う？**

> 回答：

**Q3. `errors.New("foo") == errors.New("foo")` は `true` か `false` か？ なぜ？**

> 回答：

**Q4. Go に例外（try/catch）がない。エラーをどう伝える？**

> 回答：

---

### Phase 0 クイズ②（実装後）

**Q1. `Store` interface を満たすのに `*MemoryStore` と `MemoryStore` のどちらを使った？ ポインタが必要な理由は？**

> 回答：`*MemoryStore`。フィールド（leases・resources・nextToken）を変更するメソッドがあるため、値レシーバだと変更がコピーに閉じて呼び出し元に反映されない。

**Q2. `holdfast.ErrExhausted` を package-level var にした理由を「呼び出し側が `==` で比較できるから」以外の言葉で説明せよ。**

> 回答：

**Q3. map の値（struct）を直接 `s.resources[key].used++` と書けないのはなぜ？**

> 回答：map から取り出した struct の値はアドレスを持たないため（addressable でない）、直接フィールドを書き換えられない。コピーして修正してから書き戻す必要がある。

---

### Phase 1 クイズ①（読後）

**Q1. goroutine はスレッドか？ OS スレッドと何が違う？**

> 回答：

**Q2. `go test -race` は何を検知する？ "data race" とは何か 1 文で定義せよ。**

> 回答：

**Q3. `sync.Mutex` の `Lock()`/`Unlock()` が「happens-before」を保証するとはどういう意味？**

> 回答：

**Q4. channel と mutex、どちらを使うべき場面を 1 つずつ挙げよ。**

> 回答：

**Q5. `defer mu.Unlock()` はなぜ慣用的か？ `defer` なしで書くとどのリスクがある？**

> 回答：

---

### Phase 1 クイズ②（実装後）

**Q1. `-race` で落ちたとき、出力の `DATA RACE` セクションには何が書いてあった？ どの行同士が競合していた？**

> 回答：

**Q2. Mutex を追加する前と後で、テストの実行時間はどう変わった？ なぜ？**

> 回答：

**Q3. `sync.RWMutex` を使うとしたら `MemoryStore` のどのメソッドで `RLock` を使える？ 今の実装で使えない理由は？**

> 回答：

---

### Phase 2 クイズ①（読後）

**Q1. ACID の「I（Isolation）」は何を保証する？ 完全な分離（Serializable）のコストは？**

> 回答：

**Q2. Read Committed と Repeatable Read の違いを「ファントム読み」の例で説明せよ。**

> 回答：

**Q3. 楽観ロック vs 悲観ロック：holdfast の `Acquire` にどちらが向く？ 理由は？**

> 回答：

**Q4. `SELECT FOR UPDATE` は何をロックする？ どの分離レベルで意味を持つ？**

> 回答：

**Q5. `holdfast.ErrConflict` を Postgres で実装するとき、どの SQL エラーコードをハンドルする？**

> 回答：

---

### Phase 2 クイズ②（実装後）

**Q1. 同じ conformance suite が in-memory と Postgres の両方で通った。何がそれを可能にしているか？**

> 回答：

**Q2. Postgres バックエンドでトークンの単調増加を保証するには `nextToken` の代わりに何を使った？**

> 回答：

**Q3. `pgx` のコネクションプールを使った。プールサイズをいくつにしたか？ その判断根拠は？**

> 回答：

---

### Phase 3 クイズ①（読後）

**Q1. REST と gRPC の最大の違いを「契約」の観点から説明せよ。**

> 回答：

**Q2. proto のフィールド番号を変えると何が壊れる？ フィールドを削除するときの安全な手順は？**

> 回答：

**Q3. `context.WithTimeout` でタイムアウトを設定したとき、サーバ側はそれをどう受け取る？**

> 回答：

**Q4. SIGTERM を受けてから SIGKILL までの猶予時間（`terminationGracePeriodSeconds`）に何をすべきか？**

> 回答：

**Q5. `/livez` に DB の疎通確認を入れるとどんな問題が起きる？**

> 回答：

---

### Phase 3 クイズ②（実装後）

**Q1. gRPC の `status.Code` と `holdfast.ErrExhausted` をどうマッピングした？**

> 回答：

**Q2. 処理中の RPC を drain するために `grpc.Server.GracefulStop()` を使ったとき、それが完了するまで何が起きているか？**

> 回答：

**Q3. 生成した TypeScript SDK の `Acquire` 呼び出しは、タイムアウト時にどのエラーを投げる？**

> 回答：

---

### Phase 4 クイズ①（読後）

**Q1. サイドカーパターンの利点を「アプリを変更せずに〇〇を追加できる」という形で 3 つ挙げよ。**

> 回答：

**Q2. circuit breaker の 3 状態（closed / open / half-open）を図なしで説明せよ。**

> 回答：

**Q3. retry storm とは何か？ jitter はどう防ぐ？**

> 回答：

**Q4. bulkhead は何を「隔壁」で守っている？ Envoy でどう設定する？**

> 回答：

---

### Phase 4 クイズ②（実装後）

**Q1. `holdfast.Acquire` は冪等か？ 冪等でない場合、Envoy の retry はなぜ危険か？**

> 回答：

**Q2. circuit breaker が open になったとき、クライアントには何が返る？ アプリでどうハンドルする？**

> 回答：

---

### Phase 5 クイズ①（読後）

**Q1. 「線形化可能性（linearizability）」を 1 文で定義せよ。「直列化可能性（serializability）」と何が違う？**

> 回答：

**Q2. Raft でリーダーが死んだとき、何が起きる？ quorum とは何個以上か？**

> 回答：

**Q3. フェンシングトークンが「止まっていた古いリーダーの書き込み」を防ぐ仕組みを説明せよ。**

> 回答：

**Q4. ログインデックスがフェンシングトークンとして使える理由は？**

> 回答：

**Q5. split brain はいつ起きる？ Raft はなぜ split brain を起こさないか？**

> 回答：

---

### Phase 5 クイズ②（実装後）

**Q1. 非同期レプリカで conformance suite を通したとき、どのテストが落ちた？ なぜ？**

> 回答：

**Q2. Raft FSM の `Apply` メソッドで `Acquire` を実装したとき、「何をコミットログに書くか」を説明せよ。**

> 回答：

**Q3. 適合テストが in-memory / Postgres / Raft の全部で通る。これが「Store の契約が正しい」ことを意味する理由は？**

> 回答：

---

### Phase 6 クイズ①（読後）

**Q1. Terraform の state ファイルは何を記録している？ 壊れるとどうなる？**

> 回答：

**Q2. `terraform plan` が「差分なし」を返すのはどういう状態か？**

> 回答：

**Q3. k8s の `Pod` と `Deployment` の違いは？ なぜ Pod を直接作らないのか？**

> 回答：

**Q4. `resources.requests` と `limits` を同じ値に設定すると QoS class は何になる？ なぜそれが保証になる？**

> 回答：

**Q5. multi-stage Dockerfile で builder と runner を分ける理由を「攻撃面」と「イメージサイズ」の両方から説明せよ。**

> 回答：

**Q6. `PodDisruptionBudget` の `maxUnavailable: 1` は何を保証する？ `minAvailable` との違いは？**

> 回答：

---

### Phase 6 クイズ②（実装後）

**Q1. `terraform destroy` を実行する前に確認すべきことを 3 つ挙げよ。**

> 回答：

**Q2. rolling update 中に 5xx が出なかったとしたら、それを可能にした k8s の仕組みを順番に説明せよ。**

> 回答：

**Q3. Raft クラスタを k8s StatefulSet でデプロイした。なぜ Deployment ではなく StatefulSet か？**

> 回答：

---

### Phase 7 クイズ①（読後）

**Q1. SLI・SLO・SLA の違いを 1 行ずつで。**

> 回答：

**Q2. error budget とは何か？ 「30 日間 99.9% SLO」なら月何分使える？**

> 回答：

**Q3. burn rate が「1」を超えるとはどういう意味か？ burn rate 2 なら error budget は何日で尽きる？**

> 回答：

**Q4. 短期ウィンドウ（1h）と長期ウィンドウ（6h）の 2 段構えアラートは、1 段のみと何が違う？**

> 回答：

**Q5. CPU 使用率が「悪い SLI」な理由を「ユーザー体験」の言葉で説明せよ。**

> 回答：

**Q6. `sum(rate(errors[5m])) / sum(rate(requests[5m]))` は何を計算している？**

> 回答：

---

### Phase 7 クイズ②（実装後）

**Q1. カオス試験で pod kill をしたとき SLO が割れたか？ 割れたなら何を直した？**

> 回答：

**Q2. 1 トレースで「Envoy → gRPC サーバ → Postgres」の内訳が見えた。遅延が一番大きかった区間はどこか？**

> 回答：

**Q3. burn rate アラートが鳴ったとき、最初に見るダッシュボードのパネルはどれか？ なぜ？**

> 回答：

---

### Phase 8 クイズ①（読後）

**Q1. `fly.toml` の `[http_service]` と `[[services]]` の違いは？**

> 回答：

**Q2. Fly.io の Postgres は holdfast の Store 契約とどこで繋がる？**

> 回答：

**Q3. `flyctl secrets set DATABASE_URL=...` で設定した値はアプリ側でどう読む？**

> 回答：

---

### Phase 8 クイズ②（実装後）

**Q1. 応用例で `ErrExhausted` が返ったとき、UI にどう伝えた？ HTTP ステータスコードは何を選んだ？**

> 回答：

**Q2. holdfast の `Acquire → Commit / Release` のサイクルを、応用例のユースケースの言葉で説明せよ。**

> 回答：

**Q3. ローカルでは in-memory Store、Fly.io では Postgres Store を使い分けた。切り替えをどう実装した？**

> 回答：
