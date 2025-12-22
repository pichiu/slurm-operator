# Slurm 叢集的 Kubernetes Operator

在 [Kubernetes] 上執行 [Slurm]，由 [SchedMD] 開發。這是一個 [Slinky] 專案。

## TL;DR

Slurm Operator 讓您可以在 Kubernetes 上部署和管理 Slurm HPC 叢集。它結合了 Kubernetes 的彈性擴展能力與 Slurm 強大的 HPC 工作負載排程功能，提供 NodeSet（工作節點）、LoginSet（登入節點）管理，支援混合式部署、自動擴展和零停機配置更新。

---

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [Slurm 叢集的 Kubernetes Operator](#slurm-叢集的-kubernetes-operator)
  - [目錄](#目錄)
  - [概述](#概述)
    - [Slurm 叢集](#slurm-叢集)
  - [功能特色](#功能特色)
    - [控制器](#控制器)
    - [NodeSets](#nodesets)
    - [LoginSets](#loginsets)
    - [混合式支援](#混合式支援)
    - [Slurm](#slurm)
  - [相容性](#相容性)
  - [快速開始](#快速開始)
  - [升級](#升級)
    - [1.Y 版本](#1y-版本)
    - [0.Y 版本](#0y-版本)
  - [文件](#文件)
  - [支援與開發](#支援與開發)
  - [授權](#授權)

<!-- mdformat-toc end -->

## 概述

[Slurm] 和 [Kubernetes] 是最初為不同類型工作負載設計的工作負載管理器 (Workload Manager)。概括來說：Kubernetes 擅長排程通常執行無限期時間、可能具有模糊資源需求、在單一節點上、具有寬鬆策略的工作負載，但可以無限擴展其資源池以滿足需求；Slurm 擅長快速排程執行有限時間、具有明確定義的資源需求和拓撲、在多個節點上、具有嚴格策略的工作負載，但其資源池是已知的。

本專案結合了兩種工作負載管理器的優點，統一在 Kubernetes 上。它包含一個 [Kubernetes] Operator，用於部署和管理 [Slurm] 叢集的某些元件。此儲存庫實作了[自訂控制器 (Custom Controllers)][custom-controllers]和[自訂資源定義 (Custom Resource Definitions, CRDs)][crds]，設計用於 Slurm 叢集的生命週期（建立、升級、優雅關閉）。

!["Slurm Operator 架構"](../docs/_static/images/architecture-operator.svg)

如需更多架構說明，請參閱[架構][architecture]文件。

### Slurm 叢集

Slurm 叢集非常靈活，可以以各種方式配置。我們的 Slurm Helm chart 提供了一個高度可自訂的參考實作，並嘗試公開 Slurm 提供的所有功能。

!["Slurm 架構"](../docs/_static/images/architecture-slurm.svg)

如需更多關於 Slurm 的資訊，請參閱 [Slurm][slurm-docs] 文件。

## 功能特色

### 控制器

Slurm 控制平面 (Control-plane) 負責將 Slurm 工作負載排程到其工作節點並管理其狀態。

Slurm 配置檔的變更會被自動偵測，Slurm 叢集會無縫重新配置，控制平面零停機時間。

> [!NOTE]
> kubelet 的 `configMapAndSecretChangeDetectionStrategy` 和 `syncFrequency` 設定直接影響 Pod 掛載的 ConfigMap 和 Secret 何時更新。預設情況下，kubelet 處於 `Watch` 模式，輪詢頻率為 60 秒。

### NodeSets

一組同質性的 Slurm 工作節點 (Slurm Workers，運算節點)，被委派執行 Slurm 工作負載。

Operator 會在需要縮減、升級或以其他方式處理節點故障時，考慮 Slurm 節點上執行中的工作負載。Slurm 節點在因縮減或升級而最終終止之前，會被標記為[排空 (drain)][slurm-drain]。

Slurm 節點狀態（例如 Idle、Allocated、Mixed、Down、Drain、Not Responding 等）會透過 Pod 條件套用到每個 NodeSet Pod；每個 NodeSet Pod 包含反映其自身 Slurm 節點狀態的 Pod 狀態。

Operator 支援 NodeSet 縮減到零 (Scale to Zero)，將資源縮減到零個副本。因此，任何同樣支援縮減到零的水平 Pod 自動擴展器 (Horizontal Pod Autoscaler, HPA) 都可以與 NodeSets 最佳搭配。

NodeSets 可以透過主機名稱解析。這使得登入 Pod 和工作節點 Pod 之間可以基於主機名稱的解析，實現使用可預測主機名稱（例如 `cpu-1-0`、`gpu-2-1`）的直接 Pod 對 Pod 通訊。

### LoginSets

一組同質性的 Slurm 登入節點（提交節點、跳板主機），透過 SSSD 管理使用者身分。

Operator 支援 LoginSet 縮減到零，將資源縮減到零個副本。因此，任何同樣支援縮減到零的水平 Pod 自動擴展器 (HPA) 都可以與 LoginSets 最佳搭配。

### 混合式支援

有時 Slurm 叢集的部分但非全部元件在 Kubernetes 中。Operator 及其 CRDs 設計為支援這些使用情境。

### Slurm

Slurm 是一個功能完整的 HPC 工作負載管理器。以下是一些功能亮點：

- [**帳務 (Accounting)**][slurm-accounting]：收集每個作業和作業步驟執行的帳務資訊。
- [**分割區 (Partitions)**][slurm-arch]：具有資源集合和限制（例如作業大小限制、作業時間限制、允許的使用者）的作業佇列。
- [**預約 (Reservations)**][slurm-reservations]：為選定使用者和/或選定帳戶執行的作業預留資源。
- [**作業相依性 (Job Dependencies)**][slurm-dependency]：延遲作業開始直到滿足指定的相依性。
- [**作業容器 (Job Containers)**][slurm-containers]：執行非特權 OCI 容器套件的作業。
- [**MPI**][slurm-mpi]：啟動平行 MPI 作業，支援各種 MPI 實作。
- [**優先權 (Priority)**][slurm-priority]：在提交時和持續基礎上（例如隨著作業老化）為作業分配優先權。
- [**搶佔 (Preemption)**][slurm-preempt]：停止一個或多個低優先權作業以讓高優先權作業執行。
- [**QoS**][slurm-qos]：影響排程優先權、搶佔和資源限制的策略集合。
- [**公平分享 (Fairshare)**][slurm-fairshare]：根據歷史使用量在使用者和帳戶之間公平分配資源。
- [**節點健康檢查 (Node Health Check)**][slurm-healthcheck]：透過腳本定期檢查節點健康狀況。

## 相容性

| 軟體 | 最低版本 |
| :--------- | :----------------------------------------------------------------------: |
| Kubernetes | [v1.29](https://kubernetes.io/blog/2023/12/13/kubernetes-v1-29-release/) |
| Slurm      | [25.11](https://www.schedmd.com/slurm-version-25-11-0-is-now-available/) |
| Cgroup     | [v2](https://docs.kernel.org/admin-guide/cgroup-v2.html) |

## 快速開始

安裝 [cert-manager] 及其 CRDs：

```sh
# 新增 jetstack Helm 儲存庫
helm repo add jetstack https://charts.jetstack.io
# 更新儲存庫
helm repo update
# 安裝 cert-manager，啟用 CRDs
helm install cert-manager jetstack/cert-manager \
  --set 'crds.enabled=true' \
  --namespace cert-manager --create-namespace
```

安裝 slurm-operator 及其 CRDs：

```sh
# 安裝 Slurm Operator CRDs
helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds
# 安裝 Slurm Operator
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace
```

安裝 Slurm 叢集：

```sh
# 安裝 Slurm 叢集
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace
```

如需更多說明，請參閱[安裝指南][installation]。

## 升級

Slinky 版本表示為 **X.Y.Z**，其中 **X** 是主要版本 (Major)，**Y** 是次要版本 (Minor)，**Z** 是修補版本 (Patch)，遵循[語意化版本 (Semantic Versioning)][semver] 術語。

請參閱[版本管理][versioning]以獲取更多詳細資訊。

### 1.Y 版本

較新的 [CRDs] 可能會引入重大變更。要在 `v1.Y` 版本之間升級（例如 `v1.0.Z` => `v1.1.Z`），請先升級 slurm-operator-crds chart，然後再升級 slurm-operator chart。任何 Slurm charts 將透過 CRD 轉換自動處理；不需要進一步操作。仍建議升級 Slurm charts 以使用新功能。

```bash
# 升級 CRDs
helm upgrade slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds
# 升級 Operator
helm upgrade slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky
```

要使用新的 Slinky CRD 功能，請查看 CRDs 和 Slurm chart 的變更。適當更新您的 values.yaml 並升級 chart。

```sh
# 升級 Slurm 叢集
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm
```

### 0.Y 版本

現有的 [CRDs] 可能會引入重大變更。要在 `v0.Y` 版本之間升級（例如 `v0.1.Z` => `v0.2.Z`），請解除安裝所有 Slinky charts 並刪除 Slinky CRDs，然後像正常一樣安裝新版本。

```bash
# 解除安裝所有 Slinky 元件
helm --namespace=slurm uninstall slurm
helm --namespace=slinky uninstall slurm-operator
helm uninstall slurm-operator-crds
```

如果 CRDs 不是透過 `slurm-operator-crds` Helm chart 安裝的：

```bash
# 手動刪除 CRDs
kubectl delete customresourcedefinitions.apiextensions.k8s.io accountings.slinky.slurm.net
kubectl delete customresourcedefinitions.apiextensions.k8s.io clusters.slinky.slurm.net # 已廢棄
kubectl delete customresourcedefinitions.apiextensions.k8s.io loginsets.slinky.slurm.net
kubectl delete customresourcedefinitions.apiextensions.k8s.io nodesets.slinky.slurm.net
kubectl delete customresourcedefinitions.apiextensions.k8s.io restapis.slinky.slurm.net
kubectl delete customresourcedefinitions.apiextensions.k8s.io tokens.slinky.slurm.net
```

## 文件

專案文件位於此儲存庫的 docs 目錄中。

Slinky 文件可以在[這裡][slinky-docs]找到。

**繁體中文文件**位於 `docs-zhTW/` 目錄中。

## 支援與開發

歡迎功能請求、程式碼貢獻和錯誤報告！

Github/Gitlab 提交的 issues 和 PRs/MRs 將盡力處理。

SchedMD 官方問題追蹤器在 <https://support.schedmd.com/>。

如需安排演示或聯繫，請[聯繫 SchedMD][contact-schedmd]。

## 授權

Copyright (C) SchedMD LLC.

根據 [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0) 授權，您不得在不遵守授權的情況下使用本專案。

除非適用法律要求或書面同意，否則根據本授權分發的軟體是以「原樣」基礎分發的，不附帶任何明示或暗示的保證或條件。請參閱授權以了解管理權限和限制的特定語言。

---

## Explanation

### Slurm Operator 是什麼？

Slurm Operator 是一個 Kubernetes Operator，它允許您：

| 功能 | 說明 |
|-----|------|
| **部署 Slurm** | 在 Kubernetes 上部署完整的 Slurm HPC 叢集 |
| **管理生命週期** | 自動化 Slurm 元件的建立、升級和關閉 |
| **彈性擴展** | 支援 NodeSet 和 LoginSet 的自動擴展（包括縮減到零） |
| **混合式部署** | 支援部分元件在 K8s 內、部分在外部的配置 |
| **零停機更新** | 配置變更時自動重新載入，無需重啟 |

### 為何選擇 Slurm on Kubernetes？

- **最佳化資源利用**：結合 K8s 的彈性擴展與 Slurm 的精確資源管理
- **統一基礎設施**：在同一平台上執行 HPC 和雲原生工作負載
- **降低營運成本**：利用 K8s 生態系統的工具和最佳實踐
- **混合彈性**：可以逐步遷移現有 HPC 工作負載

---

## Practical Example

### 完整安裝流程

```bash
# 1. 安裝 cert-manager（用於 TLS 憑證管理）
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --set 'crds.enabled=true' \
  --namespace cert-manager --create-namespace

# 等待 cert-manager 就緒
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s

# 2. 安裝 Slurm Operator
helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace

# 等待 Operator 就緒
kubectl wait --for=condition=Available deployment/slurm-operator -n slinky --timeout=120s

# 3. 安裝 Slurm 叢集
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace

# 4. 驗證安裝
kubectl get pods -n slinky
kubectl get pods -n slurm
```

### 測試 Slurm 叢集

```bash
# 進入登入 Pod
kubectl exec -it -n slurm deployment/slurm-login-slinky -- bash

# 執行 Slurm 指令
sinfo                           # 查看叢集狀態
srun hostname                   # 執行簡單指令
sbatch --wrap="sleep 60"        # 提交批次作業
squeue                          # 查看作業佇列
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| cert-manager 未安裝 | Operator 的 Webhook 需要 cert-manager 簽署憑證 |
| CRDs 未安裝 | 先執行 `helm install slurm-operator-crds` |
| 版本不相容 | 確認 Kubernetes >= v1.29，Slurm >= 25.11 |
| 升級順序錯誤 | 先升級 CRDs，再升級 Operator，最後升級 Slurm |

### 小技巧

1. **使用 values.yaml**：複雜配置建議使用檔案而非命令列參數
2. **監控 Pod 狀態**：定期檢查所有 Slurm Pod 的健康狀態
3. **查看文件**：詳細配置選項請參考 `docs-zhTW/` 目錄中的繁體中文文件
4. **測試環境先行**：在測試環境驗證配置後再套用到正式環境

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `helm install cert-manager jetstack/cert-manager --set 'crds.enabled=true' -n cert-manager --create-namespace` | 安裝 cert-manager |
| `helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds` | 安裝 Operator CRDs |
| `helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator -n slinky --create-namespace` | 安裝 Operator |
| `helm install slurm oci://ghcr.io/slinkyproject/charts/slurm -n slurm --create-namespace` | 安裝 Slurm 叢集 |
| `kubectl get pods -n slurm` | 檢查 Slurm Pod 狀態 |
| `kubectl exec -it -n slurm deployment/slurm-login-slinky -- sinfo` | 執行 Slurm 指令 |

<!-- links -->

[architecture]: ./concepts/architecture.md
[cert-manager]: https://cert-manager.io/docs/installation/helm/
[contact-schedmd]: https://www.schedmd.com/slurm-resources/contact-schedmd/
[crds]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions
[custom-controllers]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-controllers
[installation]: ./installation.md
[kubernetes]: https://kubernetes.io/
[schedmd]: https://schedmd.com/
[semver]: https://semver.org/
[slinky]: https://slinky.ai/
[slinky-docs]: https://slinky.schedmd.com/
[slurm]: https://slurm.schedmd.com/overview.html
[slurm-accounting]: https://slurm.schedmd.com/accounting.html
[slurm-arch]: https://slurm.schedmd.com/quickstart.html#arch
[slurm-containers]: https://slurm.schedmd.com/containers.html
[slurm-dependency]: https://slurm.schedmd.com/sbatch.html#OPT_dependency
[slurm-docs]: ./concepts/slurm.md
[slurm-drain]: https://slurm.schedmd.com/scontrol.html#OPT_DRAIN
[slurm-fairshare]: https://slurm.schedmd.com/fair_tree.html
[slurm-healthcheck]: https://slurm.schedmd.com/slurm.conf.html#OPT_HealthCheckProgram
[slurm-mpi]: https://slurm.schedmd.com/mpi_guide.html
[slurm-preempt]: https://slurm.schedmd.com/preempt.html
[slurm-priority]: https://slurm.schedmd.com/priority_multifactor.html
[slurm-qos]: https://slurm.schedmd.com/qos.html
[slurm-reservations]: https://slurm.schedmd.com/reservations.html
[versioning]: ./versioning.md
