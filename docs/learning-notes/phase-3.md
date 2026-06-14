# Phase 3 — gRPC + SDK

## 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [gRPC Basics: Go](https://grpc.io/docs/languages/go/basics/) | proto 定義 → コード生成 → サーバ実装の流れ | ★★★ |
| [Protocol Buffers Language Guide (proto3)](https://protobuf.dev/programming-guides/proto3/) | フィールド番号・後方互換ルール | ★★★ |
| [Go Blog: Contexts and structs](https://go.dev/blog/context-and-structs) | `context.Context` の正しい使い方 | ★★☆ |
| 『SRE ブック』ch.8 — Release Engineering（graceful shutdown の節のみ） | drain の意味、SIGTERM の受け取り方 | ★★☆ |
| [Kubernetes: Configure Liveness, Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) | `/livez` vs `/readyz` の使い分け | ★★☆ |

---

## クイズ①（読後）

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

## 学習メモ

### ハマりメモ

### 設計メモ

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| | | | |

---

## クイズ②（実装後）

**Q1. gRPC の `status.Code` と `holdfast.ErrExhausted` をどうマッピングした？ `codes.ResourceExhausted` か `codes.Unavailable` か、その判断理由は？**

> 回答：

**Q2. 処理中の RPC を drain するために `grpc.Server.GracefulStop()` を使ったとき、それが完了するまで何が起きているか？**

> 回答：

**Q3. 生成した TypeScript SDK の `Acquire` 呼び出しは、タイムアウト時にどのエラーを投げる？**

> 回答：
