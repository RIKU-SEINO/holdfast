# Phase 0 — Go 基礎 + コア契約

## 事前読書

| 素材 | 何を得る |
|---|---|
| [A Tour of Go](https://go.dev/tour) — Basics・Methods and interfaces | Go の型システム・interface の構造的部分型 |
| 『プログラミング言語 Go』（Donovan & Kernighan）ch.1–2 | TS との書き方の差（`:=`・ゼロ値・`error` 返し） |

---

## クイズ①（読後）

**Q1. Go の interface は「宣言して実装します」と書かなくてもよい。TS の `implements` と何が違う？**

> 回答： TSではimplementsで実装を明示的に示すことで初めて実装であることが表現されるが、Goのinterfaceはメソッドが揃っていればそれは実装であると自動的に判断される構造的部分型としての特性を持つ。

**Q2. `var x int` のとき `x` の値は何？ TS で言う `undefined` とどう違う？**

> 回答： var x intの場合、0で初期化される。Goでは変数初期化時にゼロ値が入るという特性がある。

**Q3. `errors.New("foo") == errors.New("foo")` は `true` か `false` か？ なぜ？**

> 回答：false。Goではポインタをもとにして等しいかどうかを評価するため。

**Q4. Go に例外（try/catch）がない。エラーをどう伝える？**

> 回答： Goでエラーを伝える基本はマルチリターンである。関数が（Lease, error）のように複数値を返し、呼び出し側が if err != nil でチェックする。

---

## 学習メモ

### Go の癖（TS と違うところ）

#### `:=` と `=` の使い分け

```go
err := store.Register(...)   // 新規宣言（:= で err を生やす）
_, err = store.Acquire(...)  // 再代入（err はもう存在する → = だけ）
```

> **TS との違い**：TS は `let err` で宣言してから何度でも `=` で代入できる。Go の `:=` は「宣言 + 代入」が同時に起きるので、同スコープで 2 回使うとコンパイルエラー。ただし、左辺に**新しい変数が 1 つでもあれば** `:=` を使える（`lease, err :=`）。

#### interface は「構造的部分型」

```go
// TS: implements を書く
class MemoryStore implements Store { ... }

// Go: 書かなくてよい。メソッドが揃っていれば自動で満たす
type MemoryStore struct { ... }
func (s *MemoryStore) Acquire(...) { ... }
```

#### ポインタレシーバ（`*MemoryStore` vs `MemoryStore`）

- `*MemoryStore`：フィールドへの**変更が呼び出し元に反映される**
- `MemoryStore`：**コピーが渡る**ので変更が捨てられる
- MemoryStore は `leases` / `resources` map を持ち書き換えるので `*MemoryStore` 一択

#### ゼロ値

```go
var x int    // x == 0（undefined ではない）
var s string // s == ""
var p *int   // p == nil
```

#### package-level エラー変数

```go
// 悪い例：毎回別のポインタが生まれるので == で比較できない
return errors.New("holdfast: no units available")

// 良い例
var ErrExhausted = errors.New("holdfast: no units available")
// 呼び出し側: if err == holdfast.ErrExhausted { ... }
```

> `errors.New("foo") == errors.New("foo")` は `false`。ポインタ比較なので同じ文字列でも別アドレス。

#### map の struct 更新（コピー・修正・書き戻し）

```go
// コンパイルエラー
s.resources[key].used += units

// 正しい書き方
state := s.resources[key]
state.used += units
s.resources[key] = state
```

#### `if _, ok := m[key]; ok` パターン

```go
if _, exists := s.resources[req.Resource]; exists {
    return nil
}
```

> `;` より前が「初期化文」、後が「条件式」。取得と存在確認を同時に行う。

---

### 設計メモ

#### `Capacity` はどこに置くか

**最初の案**：`AcquireRequest` に `Capacity` を置く  
**問題**：Acquire するたびに「最大 N 個まで」を渡すのは不自然  
**決定**：`RegisterRequest` に `Capacity`、`Register()` を `Store` に追加

#### `LeaseID` の生成方法

**問題**：`IdempotencyKey` を LeaseID に使うと空キーで衝突する  
**決定**：`fmt.Sprintf("%s-%d", resource, token)` でトークンから生成  
**注意**：`token := s.nextToken` → `s.nextToken++` の順を守る（逆にすると token がズレる）

#### `validateLease()` を抽出した理由

`Commit` と `Release` が同じチェック（leaseID 存在確認 + token 一致確認）をするため DRY で抽出。

#### TTL の「厳密に後」セマンティクス

```go
func (l leaseRecord) isExpired(now time.Time) bool {
    return now.After(l.expires)  // now == expires のときはまだ有効
}
```

テストでは `leaseExpires.Add(time.Nanosecond)` で「ちょうど 1ns 後」を確認。

---

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| 1 | 同スコープで `err :=` を 2 回使った | `:=` の意味を誤解 | 2 回目は `=` |
| 2 | `AcquireRequest` に `Capacity` を置いた | 「確保」と「登録」を混同 | 容量は `Register` で先に宣言 |
| 3 | フィールドを小文字（`leaseID`）にした | exported の概念が抜けた | パッケージ外から使うフィールドは大文字 |
| 4 | `s.resources[key].used += units` | map の struct は addressable でない | コピー→修正→書き戻し |
| 5 | `nextToken` をインクリメント後に `token` に代入 | 順序ミス | `token := s.nextToken` → `s.nextToken++` の順 |
| 6 | import パスを `holdfast/memory` にした | ディレクトリ構造を確認せず | 正しくは `holdfast/store/memory` |
| 7 | `package memory` にした（テストファイル） | 外部テストパッケージの慣習を知らなかった | `_test` サフィックスで外部テスト |
| 8 | `testExhausted` で `Capacity` を書き忘れた | RegisterRequest 変更後にテスト更新し忘れ | 型変更後は全テストを確認 |
| 9 | `errors.New(...)` をインラインに書いた | package-level にすべきと気づかなかった | 呼び出し側が `==` で比較するものは package-level に |

---

## クイズ②（実装後）

**Q1. `Store` interface を満たすのに `*MemoryStore` と `MemoryStore` のどちらを使った？ ポインタが必要な理由は？**

> 回答：`*MemoryStore`。フィールド（leases・resources・nextToken）を変更するメソッドがあるため、値レシーバだとコピーに変更が閉じて呼び出し元に反映されない。

**Q2. `holdfast.ErrExhausted` を package-level var にした理由を「呼び出し側が `==` で比較できるから」以外の言葉で説明せよ。**

> 回答： package-level varに統一することで、パッケージ初期化時に1度だけ生成され、それ以降はずっと同じアドレスを使用するため、そのエラーオブジェクトの同一性が担保される。

**Q3. map の値（struct）を直接 `s.resources[key].used++` と書けないのはなぜ？**

> 回答：map から取り出した struct の値はアドレスを持たないため（addressable でない）、直接フィールドを書き換えられない。コピーして修正してから書き戻す。
