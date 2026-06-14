# Phase 5 — Raft バックエンド

## 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| DDIA ch.8 — 分散システムの問題 | ネットワーク分断・クロック・プロセス停止 | ★★★ |
| DDIA ch.9 — 一貫性と合意 | 線形化可能性・2PC・Raft の位置づけ | ★★★ |
| [Raft Paper](https://raft.github.io/raft.pdf)（Ongaro & Ousterhout）§1–5 | リーダー選出・ログ複製の仕組み | ★★★ |
| [Raft Visualization](https://raft.github.io/) | 直感的理解（必ずインタラクションせよ） | ★★★ |
| [hashicorp/raft README](https://github.com/hashicorp/raft) | Go 実装の使い方 | ★★☆ |

> **この Phase の前に**: 非同期レプリケーション（leader-follower）を先に自作する。「合意なしのレプリカ」の何が怖いかを体で知ってから Raft に進む。

---

## クイズ①（読後）

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

## 学習メモ

### ハマりメモ

### 設計メモ

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| | | | |

---

## クイズ②（実装後）

**Q1. 非同期レプリカで conformance suite を通したとき、どのテストが落ちた？ なぜ？**

> 回答：

**Q2. Raft FSM の `Apply` メソッドで `Acquire` を実装したとき、「何をコミットログに書くか」を説明せよ。**

> 回答：

**Q3. 適合テストが in-memory / Postgres / Raft の全部で通る。これが「Store の契約が正しい」ことを意味する理由は？**

> 回答：
