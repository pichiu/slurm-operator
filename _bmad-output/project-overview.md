# Project Overview - slurm-operator

> 生成日期：2025-12-18
> 文件語言：繁體中文 (台灣)

## 專案摘要

**slurm-operator** 是由 [SchedMD](https://schedmd.com/) 開發的官方 Kubernetes Operator，用於在 Kubernetes 上部署和管理 [Slurm](https://slurm.schedmd.com/) 高效能運算 (HPC) 叢集。

| 屬性 | 值 |
|------|-----|
| **專案名稱** | slurm-operator |
| **完整名稱** | Kubernetes Operator for Slurm Clusters |
| **組織** | SlinkyProject / SchedMD |
| **儲存庫** | https://github.com/SlinkyProject/slurm-operator |
| **授權** | Apache License 2.0 |
| **版本** | 1.0.0 |
| **Slurm 版本** | 25.11 |

## 專案目的

此專案結合了 Kubernetes 和 Slurm 兩種工作負載管理器的優勢：

- **Kubernetes** 擅長調度無限期運行、資源需求模糊、單節點、鬆散策略的工作負載，並可無限擴展資源池
- **Slurm** 擅長快速調度有限期運行、資源需求明確、多節點、嚴格策略的工作負載，資源池已知

Operator 實現了自訂控制器和 CRD，用於 Slurm 叢集的完整生命週期管理（創建、升級、優雅關閉）。

## 技術堆疊

| 類別 | 技術 | 版本 |
|------|------|------|
| **語言** | Go | 1.25.0 |
| **框架** | Kubebuilder | v4 |
| **執行時** | controller-runtime | v0.22.4 |
| **Kubernetes** | client-go | v0.34.3 |
| **測試** | Ginkgo/Gomega | v2.23.4 / v1.37.0 |
| **容器** | Distroless | nonroot |
| **部署** | Helm | v3.19.0 |

## 架構類型

- **儲存庫類型**: Monolith（單一內聚程式碼庫）
- **專案類型**: Backend（Kubernetes Operator）
- **架構模式**: Kubernetes Operator Pattern
- **API 群組**: `slinky.slurm.net`
- **API 版本**: `v1beta1`

## 核心功能

### Custom Resource Definitions (CRDs)

| CRD | 短名稱 | 用途 |
|-----|--------|------|
| **Controller** | slurmctld | Slurm 控制平面管理 |
| **NodeSet** | slurmd, nss | 計算節點集合管理 |
| **LoginSet** | sackd, lss | 登入節點集合管理 |
| **Accounting** | slurmdbd | 會計資料庫管理 |
| **RestApi** | slurmrestd | REST API 服務管理 |
| **Token** | jwt | JWT 令牌生命週期管理 |

### 主要特性

1. **Controller (slurmctld)**
   - 自動配置檔案變更偵測
   - 零停機時間重新配置
   - 持久化狀態存儲支援

2. **NodeSets (slurmd)**
   - 同質計算節點集合
   - 工作負載感知的擴縮容
   - Slurm 節點狀態同步 (Idle, Allocated, Mixed, Down, Drain)
   - 支援縮放至零
   - 主機名稱解析支援

3. **LoginSets**
   - SSH 存取和使用者管理
   - SSSD 整合
   - 支援縮放至零

4. **混合支援**
   - 支援部分元件在 Kubernetes 外部運行

## 程式碼統計

| 指標 | 數量 |
|------|------|
| Go 檔案 (internal/) | 202 |
| 測試檔案 | 105 |
| API 類型檔案 | 22 |
| Go 依賴 | ~750 |
| CRD 數量 | 6 |
| 控制器數量 | 7 |
| Webhook 數量 | 7 |
| Helm Charts | 3 |

## 相容性

| 軟體 | 最低版本 |
|------|----------|
| Kubernetes | v1.29 |
| Slurm | 25.11 |
| Cgroup | v2 |

## 快速開始

```bash
# 安裝 cert-manager
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --set 'crds.enabled=true' \
  --namespace cert-manager --create-namespace

# 安裝 slurm-operator 和 CRDs
helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace

# 安裝 Slurm 叢集
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace
```

## 專案結構快速參考

```
slurm-operator/
├── api/v1beta1/      # CRD 類型定義
├── cmd/              # 入口點 (manager, webhook)
├── internal/         # 內部套件 (controllers, webhooks, builders, utils)
├── pkg/              # 公開套件 (conditions, taints)
├── config/           # Kubernetes 清單 (CRDs, RBAC, webhooks)
├── helm/             # Helm charts (3 個 charts)
├── docs/             # 專案文件
├── test/             # 測試 (e2e)
└── hack/             # 開發腳本
```

## 相關連結

- [官方文件](https://slinky.schedmd.com/)
- [GitHub 儲存庫](https://github.com/SlinkyProject/slurm-operator)
- [Slurm 官網](https://slurm.schedmd.com/)
- [SchedMD 支援](https://support.schedmd.com/)
