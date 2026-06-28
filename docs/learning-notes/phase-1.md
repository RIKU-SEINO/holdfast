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

> 回答： 厳密にはOSスレッドとは異なる。OS スレッドは、1プロセスの中で動く実行単位であり、同じメモリ空間を共有する。一方で、goroutineは、N個のOSスレッドに対して、M個（N << M）の goroutineがGoランタイムにより割り当てられる。gouroutineは同じメモリ空間を共有することができるが、思想としてはチャンネルを経由してメモリ共有を行うべきである。

> **正答例**：goroutine は OS スレッドそのものではない。Go ランタイムが N 個の OS スレッド上に M 個（N ≪ M）の goroutine を載せる **M:N スケジューリング**を行う。OS スレッドが各 MB 級の固定スタックを持ち、切り替えにカーネル（syscall）を伴うのに対し、goroutine は**初期スタック約 2KB で動的に伸縮**し、切り替えも**ユーザー空間で完結**する。この軽さゆえに数万〜数十万の goroutine を同時に走らせられる。メモリ空間は共有するが、設計思想としては channel を介した受け渡しを優先する（share memory by communicating）。

> **FB**（8/10）：M:N とランタイム管理、channel 思想は的確。ただし「なぜ M ≫ N が成立するのか」の*原因*（軽量スタック・ユーザー空間切り替え）が抜けている。M:N は結果であって、それを可能にするコストの低さこそが本質。

**Q2. `go test -race` は何を検知する？ "data race" とは何か 1 文で定義せよ。**

> 回答： data raceとは、あるメモリ位置に対して、排他制御が行われていない書き込み処理と、排他制御されていない書き込み処理もしくは排他制御されていない読み取り処理が並行して行われることで、処理間でのデータの同期が行われないことである。`go test -race` はtestスイートを実行する中でこのようなdata raceを検知することができる。

> **正答例**：data race とは、**2 つの goroutine が同一のメモリ位置に並行してアクセスし、少なくとも一方が書き込みであり、それらのアクセスが happens-before 関係で順序付けられていない**状態（Go メモリモデルの定義）。`go test -race` は実行中にメモリアクセスを計装し、happens-before で順序付けられないアクセス対を検出した時点で `DATA RACE` を報告する。なお race detector は「実際に踏んだ」パスしか検出できない（静的解析ではない）点に注意。

> **FB**（7/10）：「同一メモリ・並行・少なくとも一方が書き込み」は捉えている。だが定義の主語を「**排他制御されていない**」にしているのが不正確。正式な条件は「**happens-before で順序付けられていない**」。排他制御（mutex）はその順序を作る*手段の一つ*にすぎず、channel や atomic でも順序は作れる。「ロックしてない＝race」ではなく「順序保証がない＝race」。この区別は Q3 に直結する。

**Q3. `sync.Mutex` の `Lock()`/`Unlock()` が「happens-before」を保証するとはどういう意味？**

> 回答：あるメモリ位置に対する書き込み/読み込み処理に対して、sync.MutexによるLock()/Unlock()を適用することで、その区間の中で入るgoroutineの数を1にすることができるため、そのメモリ位置に対して処理を加える他のgoroutineが非同期的に実行されることを防ぎ、スレッドセーフな状態であることを保証することである。

> **正答例**：happens-before の保証とは、**相互排他とは別の「メモリ可視性の順序保証」**を指す。具体的には、同じ mutex について **ある goroutine の `n` 回目の `Unlock()` は、別の goroutine の `n+1` 回目の `Lock()` より happens-before する**。これにより、Unlock の前に行った書き込みは、後で Lock を取った goroutine から**必ず見える**ことが保証される。CPU キャッシュやコンパイラ／CPU による命令の並べ替えがあっても、この同期点を跨いだ可視性は守られる。これが mutex が data race を消す本当の理由。

> **FB**（5/10）：**問いに正面から答えていない。** 回答は「相互排他（同時に1つだけ）」の説明になっているが、問いは「**happens-before**（書き込みの可視性順序）とは何か」。両者は別物。「同時に1つ」だけでは、片方の書き込みがもう片方から見える保証の説明にならない。mutex が race を消す理由は「排他するから」ではなく「Unlock→Lock の happens-before で可視化するから」。Q2 の定義（順序保証がない＝race）と表裏一体。

**Q4. channel と mutex、どちらを使うべき場面を 1 つずつ挙げよ。**

> 回答： mutexは、ある同一のメモリ位置に対する書き込み/読み取り処理に対して排他制御を適用することでスレッドセーフにする場合に用いるべきである。一方で、channelでは、goroutine間で何らかのデータのやり取りをしたい場合や、先行する処理と後続の処理の速度差を吸収するために(バッファ付きのchannelを)用いるべきである。

> **正答例**：**mutex** ＝共有された状態（例：マップやカウンタ）への並行アクセスを保護する場面。状態は「その場所に留まり」、複数 goroutine が読み書きを奪い合う。**channel** ＝データの**所有権を goroutine 間で移譲**する場面、あるいはパイプライン/ワーカープールで処理を流す場面。バッファ付き channel は生産者と消費者の速度差を吸収するキューとしても使える。指針：「状態を守る」なら mutex、「データを渡す」なら channel。

> **FB**（9/10）：使い分けの軸が正確。加点余地は、channel のもう一つの典型「**所有権の移譲**」（あるデータの責任を渡す＝share memory by communicating の核心）。速度差吸収は*バッファ*の効能、所有権移譲は*channel そのもの*の思想、と分けて言えると満点。

**Q5. `defer mu.Unlock()` はなぜ慣用的か？ `defer` なしで書くとどのリスクがある？**

> 回答：deferなしで書くことのリスクは、ロックが永遠に解放されず、デッドロックに陥ることや、不要なメモリが常に確保された状態となりガーベジコレクトされずOOMに陥ることが考えられる。

> **正答例**：`defer mu.Unlock()` が慣用的なのは、**関数を抜けるどの経路でも確実に Unlock が走る**から。手動 Unlock だと、(1) 複数ある早期 `return` のどこかで書き忘れる、(2) 途中で `panic` すると Unlock 行に到達せずロックが握られたまま――といった事故が起きる。`defer` は return でも panic でも必ず実行されるので、これらを構造的に防げる。リスクの本質は「ロックが解放されず、以後その mutex を待つ goroutine が**永久にブロック（デッドロック／スタベーション）**する」こと。

> **FB**（5/10）：「解放されずデッドロック」は正しい。だが**慣用句たる最大の理由（早期 return / panic でも必ず走る）が抜けている**。さらに「メモリが確保されGCされずOOM」は**誤り**：mutex は数バイトの状態にすぎず、握り続けてもメモリ枯渇は起きない。問題はあくまで他 goroutine のブロックであってメモリではない。誤った因果は減点対象。

**Q6. TOCTOU（Time Of Check, Time Of Use）とは何か？ in-memory Store の `Acquire` でどこが「check」でどこが「use」か？**

> 回答： TOCTOUはレースコンディンションの1形態であり、特定のリソースに対して処理可能な状態であるかどうかをチェックするタイミングと、そのリソースに対して実際に処理を行うタイミングの間で、そのリソースに対して変更が加えられ、実際の処理により意図しない結果が引き起こされることである。in-memory Store の `Acquire` において、 `state.isExhausted(units)` の真偽の判定がcheckであり、 `s.leases[leaseId] = leaseRecord{...}` がuseである。 

> **正答例**：TOCTOU は race condition の一形態で、「状態をチェックした時点」と「その判定を前提に処理を実行する時点」の間に他者が状態を書き換え、判定が無効化されることで意図しない結果が出る。`Acquire` では **check ＝ `state.isExhausted(units)`**（枠が空いているかの判定）、**use ＝ `state.used += units; s.resources[req.Resource] = state`**（使用量カウンタの加算と書き戻し）。2 つの goroutine が両方とも isExhausted を false で通過し、両方が used を加算すると、合計が capacity を超える＝**過剰確保（不変条件①違反）**が起きる。

> **FB**（7/10）：概念定義と check の指摘は正確。ただし use の指摘が `s.leases[...] = ...`（lease の記録）になっているのが惜しい。**過剰確保を直接引き起こす use は `state.used += units` の書き戻し**。lease 書き込みは結果の記録であって枠オーバーの原因ではない。「どの check に対応する、どの use か」を*対*で捉えるのが TOCTOU の肝。

**Q7. `runtime.Gosched()` を `check` と `use` の間に挟むと、なぜ過剰確保の再現率が上がるのか？**

> 回答： Q6で説明した、TOCTOUが発生しやすくなるからであり、あるリソースに関するcheckが完了した後に他のgoroutineの実行を `runtime.Gosched()` により許すことで、他のgoroutineによりそのリソースの状態が書き換えられ、そのリソースのuse時に意図しない結果が起きやすくなるからである。

> **正答例**：`runtime.Gosched()` は現在の goroutine を一旦スケジューラに譲り、他の実行可能な goroutine を走らせる。これを check と use の間に挟むと、その瞬間に別の goroutine が走って同じ check を通過し state を書き換える可能性が高まる。本来は稀にしか踏まれない check〜use の窓を**人為的に広げ**、過剰確保を高確率で再現できる。バグを「運任せで稀に起きる」状態から「ほぼ確実に観測できる」状態にする実験テクニック。

> **FB**（10/10）：因果が正確。check 後に明示的にスケジューラへ譲る→窓が広がる→他 goroutine が state を書き換える確率上昇、という連鎖を正しく説明できている。

---

## 学習メモ

### 事前学習メモ
- Goの並行処理の基本: https://zenn.dev/y_yuita/articles/de09b33dad9bfb
- goroutineとOSスレッドの違い: https://zenn.dev/7csc/articles/5e69b2daefb827
- data raceとは: https://zenn.dev/nobishii/articles/go-data-corruptions

### TOCTOU 実験メモ

> UnsafeStore の実験結果をここに記録する

**状態：未着手（途中）** — STRETCH として着手予定。設計まで決め、実装・計測はこれから。
- 設計：`toctou_test.go`（`package memory`）に最小の `unsafeStore`（`Register`＋`Acquire` のみ、mutex なし、check↔use 間に `runtime.Gosched()`）を置く。`resourceState`/`isExhausted` はパッケージ内既存型を流用。
- 計測：N 回試行し、capacity=C・goroutine 数 G で「成功数 > C」の回数を数えて発生率を出す。集計は `atomic.Int64`。`G = 2 / 10 / 100` で比較。
- 実行：**`-race` を付けずに**走らせる（UnsafeStore は意図的に race を持つ。測りたいのは data race そのものでなく論理的な過剰確保率）。テスト名を `TestTOCTOU…` にして `-run` で狙い撃ち。
- 残り：実装 → 計測して下表を埋める → クイズ②Q4/Q5 に回答。

| 試行回数 | goroutine 数 | capacity | 過剰確保発生回数 | 発生率 |
|---|---|---|---|---|
| | | | | |

### ハマりメモ

- **テストが一生終わらない（再入デッドロック）。** `Commit`/`Release` の先頭で `s.mu.Lock()` してから、内部で `validateLease` を呼び、その `validateLease` も `s.mu.Lock()` していた。`sync.Mutex` は**再入不可（non-reentrant）**なので、同じ goroutine でもすでに握っている錠前をもう一度 `Lock()` すると「自分が Unlock するのを自分が待つ」状態になり永久にハングする。
  → 直し方：ロック済み前提の内部ヘルパー（`validateLease`）からは `Lock`/`Unlock` を**外す**。「公開メソッドが錠前を取り、小文字の内部メソッドはロック済み前提」という分業にする。
- **`const` に `:=` は使えない。** `const capacity := 10` は文法エラー。`:=` は「変数の短縮宣言」専用で、定数は `const capacity = 10`（`=`）。コンパイル時に確定する固定値は `const`、実行時に決まる値は `:=`/`var`。
- **`wg.Add` の位置。** ループ前 `Add(goroutines)` でもループ内 `Add(1)`（`go` の前）でも正しい。Wait はループ後に呼ばれ、その時点でカウンタは確定しているため。NG なのは **goroutine の中で `Add(1)`**（Wait と並行に走り早期終了しうる）。
- **`for i := 0; i < n; i++` の現代化。** `i` を使わないなら Go 1.22+ の `for range n` が素直（gopls が "range over int" を提案する）。

### 設計メモ

- **並行テストは conformance に置く。** 「並行下でも枠を超えない」は memory 固有でなく全バックエンドで成り立つべき不変条件①なので、適合テスト（背骨）に入れる。Postgres/Raft でも同じテストが効く。
- **テストの集計は `atomic.Int64`。** 成功数を素の `int++` で数えると**テスト自身が data race を持ち込み**、実装のバグと区別できなくなる。goroutine 内で `_, err :=`（`:=`）にして `err` を goroutine ローカルにするのも同じ理由（`=` だと共有 err への並行書き込みになる）。
- **まず全体で1つの coarse-grained mutex。** 一部フィールドしか書かなくても全体を直列化するためスループットは落ちるが、教育的順序「素直に作る→測る→緩める」に従い、細粒度ロック（リソースごとの錠前）は STRETCH に回す。
- **Read 専用でも保護は要る。** `validateLease` は読み取りだけだが、他 goroutine の Write と並行すれば race。ただし「validateLease 自身が錠前を取る」必要はなく「呼び出し側のロックの傘の下で読む」。＝「読み取りにも保護は要るが、誰が錠前を取るかは別問題」。
- **`UnsafeStore` は `_test.go` に隔離。** わざと不変条件を破る実験用なので、本番ビルドに漏れないよう test-only にする。`conformance.Run` には通さない（必ず落ちるため）。ファイル名は `toctou_test.go`（実験内容を表す名前。`unsafe_test.go` は標準 `unsafe` と紛らわしく避ける）。

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| 1 | `Commit`/`Release` でテストが永久ハング | ロック済みの公開メソッドから、また `Lock` する `validateLease` を呼んだ（再入デッドロック） | `sync.Mutex` は再入不可。内部ヘルパーはロックを取らず「ロック済み前提」にする |
| 2 | クイズ①Q3 で happens-before を「同時に1つ」と説明 | 相互排他と happens-before（可視性順序）を混同 | mutex が race を消すのは「Unlock→Lock の happens-before で書き込みを可視化するから」 |
| 3 | クイズ①Q5 で「ロック保持でOOM」と記述 | 誤った因果 | mutex 保持の害はメモリ枯渇でなく他 goroutine のブロック（デッドロック/スタベーション） |

---

## クイズ②（実装後）

**Q1. `-race` で落ちたとき、出力の `DATA RACE` セクションには何が書いてあった？ どの行同士が競合していた？**

> 回答：

**Q2. Mutex を追加する前と後で、テストの実行時間はどう変わった？ なぜ？**

> 回答：

**Q3. `sync.RWMutex` を使うとしたら `MemoryStore` のどのメソッドで `RLock` を使える？ 今の実装で使えない理由は？**

> 回答：

**Q4. TOCTOU 実験で、goroutine 数を 2 / 10 / 100 と増やすと過剰確保率はどう変わったか？ なぜそうなる？**

> 回答：

**Q5. `UnsafeStore` に `go test -race` をかけると何が出力されるか？ mutex 追加後は出力がどう変わるか？**

> 回答：
