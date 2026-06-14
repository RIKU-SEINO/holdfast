# Phase 4 — Envoy サイドカー

## 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [Envoy Getting Started](https://www.envoyproxy.io/docs/envoy/latest/start/quick-start/run-envoy) | xDS・listener・cluster の概念 | ★★★ |
| [Envoy Docs: Circuit breaking](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/circuit_breaking) | half-open の仕組み | ★★★ |
| 『Designing Distributed Systems』（Burns）ch.2 — Sidecar Pattern | なぜアプリと分離するか | ★★☆ |
| [Envoy Docs: Retry semantics](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#x-envoy-retry-on) | retry 予算・retry storm | ★★☆ |

---

## クイズ①（読後）

**Q1. サイドカーパターンの利点を「アプリを変更せずに〇〇を追加できる」という形で 3 つ挙げよ。**

> 回答：

**Q2. circuit breaker の 3 状態（closed / open / half-open）を図なしで説明せよ。**

> 回答：

**Q3. retry storm とは何か？ jitter はどう防ぐ？**

> 回答：

**Q4. bulkhead は何を「隔壁」で守っている？ Envoy でどう設定する？**

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

**Q1. `holdfast.Acquire` は冪等か？ 冪等でない場合、Envoy の retry はなぜ危険か？**

> 回答：

**Q2. circuit breaker が open になったとき、クライアントには何が返る？ アプリでどうハンドルする？**

> 回答：
