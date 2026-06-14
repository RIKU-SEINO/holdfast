# Phase 8 — 応用例の実装 + Fly.io デプロイ（別リポ）

> このリポではなく `holdfast-examples/` で作業する。

## 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [Fly.io Docs: Quickstart (Go)](https://fly.io/docs/languages-and-frameworks/golang/) | `flyctl` の基本操作、`fly.toml` の書き方 | ★★★ |
| [Fly.io Docs: Postgres](https://fly.io/docs/postgres/) | Fly.io 上の Postgres の立て方、接続方法 | ★★★ |
| [Fly.io Docs: Secrets](https://fly.io/docs/apps/secrets/) | DB パスワード等の環境変数管理 | ★★☆ |

---

## クイズ①（読後）

**Q1. `fly.toml` の `[http_service]` と `[[services]]` の違いは？**

> 回答：

**Q2. Fly.io の Postgres は holdfast の Store 契約とどこで繋がる？（どのファイルに何を書く？）**

> 回答：

**Q3. `flyctl secrets set DATABASE_URL=...` で設定した値はアプリ側でどう読む？**

> 回答：

**Q4. `/naive/reserve` と `/holdfast/reserve` の実装の差は何か？ 具体的にどのコードが「check」と「use」を分けているか？**

> 回答：

---

## 学習メモ

### TOCTOU 比較デモ メモ

> `/stats` エンドポイントの実装で気づいたこと・設計判断をここに書く

### ハマりメモ

### 設計メモ

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| | | | |

---

## クイズ②（実装後）

**Q1. 応用例で `ErrExhausted` が返ったとき、UI にどう伝えた？ HTTP ステータスコードは何を選んだ？**

> 回答：

**Q2. holdfast の `Acquire → Commit / Release` のサイクルを、応用例のユースケース（座席予約など）の言葉で説明せよ。**

> 回答：

**Q3. ローカルでは in-memory Store、Fly.io では Postgres Store を使い分けた。切り替えをどう実装した？**

> 回答：

**Q4. `/stats` の結果で、naive の過剰確保率は何%だった？ 並行数を増やすと率はどう動いた？**

> 回答：

**Q5. TOCTOU デモを見た人に「なぜ holdfast が必要か」を 2 文で説明するとしたら？**

> 回答：
