# Phase 6 — Terraform + k8s（kind）でクラスタ構築

## 事前読書

> **この Phase が「クラウドインフラ」の本丸**。AWS CDK/CloudFormation の経験が Terraform を理解する土台になる。

| 素材 | 何を得る | 優先度 |
|---|---|---|
| [Terraform Getting Started](https://developer.hashicorp.com/terraform/tutorials/aws-get-started) — resource / variable / output | CDK との対比で IaC を理解 | ★★★ |
| [Kubernetes Basics Interactive Tutorial](https://kubernetes.io/docs/tutorials/kubernetes-basics/) | Pod / Deployment / Service の関係 | ★★★ |
| [k8s Docs: PodDisruptionBudget](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) | rolling update と可用性の保証 | ★★★ |
| [Dockerfile multi-stage builds](https://docs.docker.com/build/building/multi-stage/) | builder と runner を分けるなぜ | ★★☆ |
| [distroless](https://github.com/GoogleContainerTools/distroless) | scratch より使いやすい最小イメージ | ★★☆ |
| [k8s Docs: HPA](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) | スケーリング指標の選び方 | ★☆☆ |

**CDK/CFn との対比**

| Terraform | CDK/CFn |
|---|---|
| `resource` | `new Construct(this, ...)` |
| `tfstate` | stack state |
| `terraform plan` | `cdk diff` |
| `terraform apply` | `cdk deploy` |
| 宣言的 HCL（「何が欲しいか」） | 手続き的 TS（「どう作るか」） |

---

## クイズ①（読後）

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

## 学習メモ

### ハマりメモ

### 設計メモ

### ミスログ

| # | やらかし | 原因 | 覚えること |
|---|---|---|---|
| | | | |

---

## クイズ②（実装後）

**Q1. `terraform destroy` を実行する前に確認すべきことを 3 つ挙げよ。**

> 回答：

**Q2. rolling update 中に 5xx が出なかったとしたら、それを可能にした k8s の仕組みを順番に説明せよ。**

> 回答：

**Q3. Raft クラスタを k8s StatefulSet でデプロイした。なぜ Deployment ではなく StatefulSet か？**

> 回答：
