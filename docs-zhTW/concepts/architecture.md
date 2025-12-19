# 架構 (Architecture)

## TL;DR

Slinky `slurm-operator` 是一個遵循 Kubernetes Operator 模式的控制器，負責管理容器化的 Slurm 叢集。它透過 Kubernetes API 和 Slurm REST API 來協調自訂資源 (Custom Resources)，支援混合式部署和自動擴展功能。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [架構](#架構-architecture)
  - [目錄](#目錄)
  - [概述](#概述)
  - [Operator](#operator)
  - [Slurm](#slurm)
    - [混合式部署](#混合式部署)
    - [自動擴展](#自動擴展)
  - [目錄結構](#目錄結構)
    - [`api/`](#api)
    - [`cmd/`](#cmd)
    - [`config/`](#config)
    - [`docs/`](#docs)
    - [`hack/`](#hack)
    - [`helm/`](#helm)
    - [`internal/`](#internal)
    - [`internal/controller/`](#internalcontroller)
    - [`internal/webhook/`](#internalwebhook)

<!-- mdformat-toc end -->

## 概述

本文件描述 Slinky `slurm-operator` 的高階架構。

## Operator

下圖從通訊角度說明 Operator (操作器) 的運作方式。

<img src="../../docs/_static/images/architecture-operator.svg" alt="Slurm Operator 架構" width="100%" height="auto" />

`slurm-operator` 遵循 Kubernetes [Operator 模式][operator-pattern]。

> Operator (操作器) 是 Kubernetes 的軟體擴充套件，它利用自訂資源 (Custom Resources) 來管理應用程式及其元件。Operator 遵循 Kubernetes 原則，特別是控制迴圈 (Control Loop)。

`slurm-operator` 針對每個需要管理的自訂資源定義 (Custom Resource Definition, CRD) 都有一個對應的控制器 (Controller)。每個控制器都有一個控制迴圈，用於調和 (Reconcile) 自訂資源 (Custom Resource, CR) 的狀態。

通常，Operator 只關注 Kubernetes API 回報的資料。但在我們的情況下，我們也關注 Slurm API 回報的資料，這會影響 `slurm-operator` 如何調和特定的自訂資源。

## Slurm

下圖從通訊角度說明容器化 Slurm 叢集 (Cluster) 的架構。

<img src="../../docs/_static/images/architecture-slurm.svg" alt="Slurm 叢集架構" width="100%" height="auto" />

關於 Slurm 的更多資訊，請參閱 [Slurm 文件][slurm]。

### 混合式部署

以下混合式部署圖僅為範例。混合式配置有許多不同的設定方式。核心重點是：slurmd 可以在裸機 (Bare-metal) 上執行，並且仍然可以加入您的容器化 Slurm 叢集；您的 Slurm 叢集所需或想要的外部服務（例如 AD/LDAP、NFS、MariaDB）不必在 Kubernetes 中才能與您的 Slurm 叢集正常運作。

<img src="../../docs/_static/images/architecture-slurm-hybrid.svg" alt="混合式 Slurm 叢集架構" width="100%" height="auto" />

### 自動擴展

Kubernetes 支援資源自動擴展 (Autoscaling)。在 Slurm 的情境中，當您的 Kubernetes 和 Slurm 叢集有工作負載波動時，自動擴展 Slurm 工作節點 (Workers) 會非常有用。

<img src="../../docs/_static/images/architecture-autoscale.svg" alt="自動擴展架構" width="100%" height="auto" />

請參閱[自動擴展指南][autoscaling]以獲取更多資訊。

## 目錄結構

本專案遵循以下慣例：

- [Golang][golang-layout]
- [operator-sdk]
- [Kubebuilder]

### `api/`

包含自訂 Kubernetes API 定義。這些會成為自訂資源定義 (CRDs) 並安裝到 Kubernetes 叢集中。

### `cmd/`

包含要編譯成二進位指令的程式碼。

### `config/`

包含用於 [kustomize] 部署的 YAML 設定檔。

### `docs/`

包含專案文件。

### `hack/`

包含開發和 Kubebuilder 相關檔案。這包括一個 kind.sh 腳本，可用於建立具有本地測試所需所有先決條件的 kind 叢集。

### `helm/`

包含 [Helm] 部署，包括 values.yaml 等設定檔。

Helm 是將此專案安裝到 Kubernetes 叢集的建議方法。

### `internal/`

包含內部使用的程式碼。此程式碼無法被外部匯入。

### `internal/controller/`

包含控制器 (Controllers)。

每個控制器以其管理的自訂資源定義 (CRD) 命名。

### `internal/webhook/`

包含 Webhook。

每個 Webhook 以其管理的自訂資源定義 (CRD) 命名。

<!-- Links -->

[autoscaling]: ../usage/autoscaling.md
[golang-layout]: https://go.dev/doc/modules/layout
[helm]: https://helm.sh/
[kubebuilder]: https://book.kubebuilder.io/
[kustomize]: https://kustomize.io/
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[operator-sdk]: https://sdk.operatorframework.io/
[slurm]: ./slurm.md

---

## Explanation

### 什麼是 Kubernetes Operator？

Kubernetes Operator 是一種軟體設計模式，它擴展了 Kubernetes 的功能，讓您可以用 Kubernetes 原生的方式來管理複雜的應用程式。Operator 使用「控制迴圈」持續監控資源的實際狀態，並將其調整為期望狀態。

### Slurm Operator 的角色

`slurm-operator` 扮演著橋樑的角色，它：
1. **監控 Kubernetes**：追蹤 Pod、NodeSet 等資源的狀態
2. **監控 Slurm**：透過 REST API 取得 Slurm 節點和作業的資訊
3. **協調兩者**：確保 Kubernetes 資源與 Slurm 叢集狀態保持同步

---

## Practical Example

### 查看 Operator 狀態

```bash
# 檢查 slurm-operator Pod 是否正常運行
# -n slinky: 指定命名空間為 slinky
# --selector: 使用標籤選擇器篩選 Pod
kubectl get pods -n slinky --selector='app.kubernetes.io/instance=slurm-operator'

# 查看 Operator 的日誌
# -f: 持續追蹤日誌輸出
kubectl logs -n slinky -l app.kubernetes.io/instance=slurm-operator -f

# 列出所有 Slinky 相關的自訂資源定義
kubectl get crds | grep slinky
```

### 查看 CRD 和 CR

```bash
# 列出所有 NodeSet 自訂資源
kubectl get nodesets -n slurm

# 查看特定 NodeSet 的詳細資訊
kubectl describe nodeset slurm-worker-slinky -n slurm

# 列出所有 Cluster 資源
kubectl get clusters.slinky.slurm.net -n slurm
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| Operator Pod 無法啟動 | 確認 cert-manager 已正確安裝並運行 |
| CRD 未找到 | 執行 `helm install slurm-operator-crds` 安裝 CRDs |
| 控制器無法連接 Slurm | 檢查 slurmrestd 服務是否正常運行 |
| 資源狀態不同步 | 檢查 Operator 日誌中是否有錯誤訊息 |

### 小技巧

1. **使用 Helm 安裝**：這是最推薦的安裝方式，能確保所有元件正確配置
2. **監控日誌**：定期檢查 Operator 日誌有助於發現問題
3. **理解 CRD**：熟悉 NodeSet、Cluster 等 CRD 的 spec 和 status 欄位

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `kubectl get pods -n slinky` | 查看 Operator Pod 狀態 |
| `kubectl get crds \| grep slinky` | 列出 Slinky CRDs |
| `kubectl get nodesets -n slurm` | 列出所有 NodeSet |
| `kubectl describe nodeset <name> -n slurm` | 查看 NodeSet 詳細資訊 |
| `kubectl logs -n slinky -l app.kubernetes.io/instance=slurm-operator` | 查看 Operator 日誌 |
