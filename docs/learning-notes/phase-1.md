# Phase 1 — Go 並行

## 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [Go Tour: Concurrency](https://go.dev/tour/concurrency) | goroutine・channel・`sync.WaitGroup` の基本 | ★★★ |
| [Go Blog: The Go Memory Model](https://go.dev/ref/mem) — Happens-before だけ | 「なぜ race が壊れるか」の根拠 | ★★★ |
| 『Concurrency in Go』（Cox-Buday）ch.1–3 | mutex / channel の使い分け、race detector の読み方 | ★★☆ |
| [Go Blog: Share Memory By Communicating](https://go.dev/blog/codewalks) | Go の設計哲学（channel 優先） | ★☆☆ |

> **TS との対比**：Promise / async-await は「非同期 I/O の待機」、goroutine は「並行実行スレッド」。別の問題を解いている。

---

## クイズ①（読後）

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

## 学習メモ

### ハマりメモ

<!-- 実装中に詰まったことをここに書く -->

### 設計メモ

<!-- なぜそう設計したかをここに書く -->

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| | | | |

---

## クイズ②（実装後）

**Q1. `-race` で落ちたとき、出力の `DATA RACE` セクションには何が書いてあった？ どの行同士が競合していた？**

> 回答：

**Q2. Mutex を追加する前と後で、テストの実行時間はどう変わった？ なぜ？**

> 回答：

**Q3. `sync.RWMutex` を使うとしたら `MemoryStore` のどのメソッドで `RLock` を使える？ 今の実装で使えない理由は？**

> 回答：
