# Phase 7 — 分散トレーシング + SLO + カオス

## 事前読書

| 素材 | 何を得る | 優先度 |
|---|---|---|
| 『SRE ブック』（Google）ch.4 — SLO、ch.6 — モニタリング | SLI/SLO/error budget の定義 | ★★★ |
| 『SRE Workbook』ch.5 — Alerting on SLOs | burn rate アラートの設計 | ★★★ |
| [OpenTelemetry Docs: Go](https://opentelemetry.io/docs/languages/go/) | trace / metric / log の三本柱 | ★★★ |
| [Prometheus Docs: histogram_quantile](https://prometheus.io/docs/practices/histograms/) | p99 の計算方法、bucket の切り方 | ★★☆ |
| [Chaos Engineering](https://principlesofchaos.org/) — Principles | 「何を壊すか」の設計思想 | ★★☆ |

---

## クイズ①（読後）

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

## 学習メモ

### ハマりメモ

### 設計メモ

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| | | | |

---

## クイズ②（実装後）

**Q1. カオス試験で pod kill をしたとき SLO が割れたか？ 割れたなら何を直した？**

> 回答：

**Q2. 1 トレースで「Envoy → gRPC サーバ → Postgres」の内訳が見えた。遅延が一番大きかった区間はどこか？**

> 回答：

**Q3. burn rate アラートが鳴ったとき、最初に見るダッシュボードのパネルはどれか？ なぜ？**

> 回答：
