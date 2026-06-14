# Phase 2 — Postgres バックエンド

## 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| 『データ指向アプリケーションデザイン』（DDIA）ch.7 — トランザクション | ACID・分離レベル・lost update・write skew | ★★★ |
| [Postgres Docs: Transaction Isolation](https://www.postgresql.org/docs/current/transaction-iso.html) | Postgres の実際の挙動 | ★★★ |
| [pgx README](https://github.com/jackc/pgx) | Go の Postgres ドライバの使い方 | ★★☆ |
| DDIA ch.3 — ストレージエンジン（B-Tree の節のみ） | インデックスがロックとどう絡むか | ★☆☆ |

---

## クイズ①（読後）

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

## 学習メモ

### ハマりメモ

### 設計メモ

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| | | | |

---

## クイズ②（実装後）

**Q1. 同じ conformance suite が in-memory と Postgres の両方で通った。何がそれを可能にしているか？**

> 回答：

**Q2. Postgres バックエンドでトークンの単調増加を保証するには `nextToken` の代わりに何を使った？**

> 回答：

**Q3. `pgx` のコネクションプールを使った。プールサイズをいくつにしたか？ その判断根拠は？**

> 回答：
