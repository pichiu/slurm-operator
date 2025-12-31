# slurm-operator 專案文件索引

> 生成日期：2025-12-18
> 掃描模式：Exhaustive Scan
> 文件語言：繁體中文 (台灣)

---

## 專案概覽

| 屬性 | 值 |
|------|-----|
| **專案類型** | Kubernetes Operator (Monolith) |
| **主要語言** | Go 1.25.0 |
| **框架** | Kubebuilder v4 + controller-runtime v0.22.4 |
| **API 群組** | `slinky.slurm.net` |
| **API 版本** | `v1beta1` |

### 快速參考

- **技術堆疊**: Go 1.25.0, Kubebuilder v4, controller-runtime, Kubernetes v1.34
- **入口點**: `cmd/manager/main.go` (Operator), `cmd/webhook/main.go` (Webhook)
- **架構模式**: Kubernetes Operator Pattern
- **CRD 數量**: 6 個 (Controller, NodeSet, LoginSet, Accounting, RestApi, Token)

---

## 生成的文件

### 核心文件

| 文件 | 說明 |
|------|------|
| [專案概覽](./project-overview.md) | 專案摘要、技術堆疊、功能概述 |
| [架構文件](./architecture.md) | 系統架構、控制器設計、部署架構 |
| [原始碼樹分析](./source-tree-analysis.md) | 完整目錄結構與註解 |
| [開發指南](./development-guide.md) | 環境設定、構建命令、測試 |
| [資料模型](./data-models.md) | CRD 定義、欄位說明、驗證規則 |

### 使用者文件

| 文件 | 說明 |
|------|------|
| [使用指南](./slurm-usage-guide.md) | 新手入門、基本操作、作業提交 |
| [FAQ](./slurm-faq.md) | 常見問題與解答 |
| [slurm.conf 配置指南](./slurm-conf-guide.md) | slurm.conf 生成機制、extraConf 配置、Partition 設定 |
| [REST API 指南](./rest-api.md) | Slurm REST API 使用指南、JWT 認證、端點說明 |
| [NodeSet API 參考](./nodeset-api-reference.md) | NodeSet CR 完整欄位說明 |
| [Helm NodeSet 管理指南](./helm-nodeset-guide.md) | Helm 管理 NodeSet 的操作指南 |
| [AAA 職責邊界說明](./aaa-responsibilities.md) | Authentication/Accounting 責任邊界與 Bootstrap 指南 |

### 深入解析

| 文件 | 說明 |
|------|------|
| [Partition 管理深入解析](./deep-dive-partition.md) | Partition CRUD 限制、REST API 分析、管理方案評估 |
| [Pyxis 與 NodeSet 深入解析](./deep-dive-pyxis-nodeset.md) | Pyxis 容器化作業、三層架構、Enroot 設定 |
| [Helm Chart 深入解析](./deep-dive-helm.md) | Helm Chart 結構與客製化 |
| [NodeSet 儲存深入解析](./deep-dive-nodeset-storage.md) | NodeSet 儲存配置詳解 |
| [Job 與 Storage 深入解析](./deep-dive-job-storage.md) | Slurm Job 與 K8s Storage 的關係 |

---

## 現有專案文件

### 概念文件

| 文件 | 路徑 |
|------|------|
| 架構概述 | [docs/concepts/architecture.md](../docs/concepts/architecture.md) |
| NodeSet 控制器 | [docs/concepts/nodeset-controller.md](../docs/concepts/nodeset-controller.md) |
| SlurmClient 控制器 | [docs/concepts/slurmclient-controller.md](../docs/concepts/slurmclient-controller.md) |
| Slurm 基礎 | [docs/concepts/slurm.md](../docs/concepts/slurm.md) |

### 使用指南

| 文件 | 路徑 |
|------|------|
| 安裝指南 | [docs/installation.md](../docs/installation.md) |
| 開發指南 | [docs/usage/develop.md](../docs/usage/develop.md) |
| 自動擴展 | [docs/usage/autoscaling.md](../docs/usage/autoscaling.md) |
| 混合部署 | [docs/usage/hybrid.md](../docs/usage/hybrid.md) |
| Jupyter 整合 | [docs/usage/jupyter.md](../docs/usage/jupyter.md) |
| 系統需求 | [docs/usage/system-requirements.md](../docs/usage/system-requirements.md) |
| 拓撲配置 | [docs/usage/topology.md](../docs/usage/topology.md) |
| 工作負載隔離 | [docs/usage/workload-isolation.md](../docs/usage/workload-isolation.md) |
| PyTorch 教學 | [docs/usage/tutorial-pytorch.md](../docs/usage/tutorial-pytorch.md) |
| Pyxis 支援 | [docs/usage/pyxis.md](../docs/usage/pyxis.md) |
| 配置覆蓋 | [docs/usage/override-config-file.md](../docs/usage/override-config-file.md) |

### Helm Chart 文件

| Chart | 路徑 |
|-------|------|
| slurm-operator | [helm/slurm-operator/README.md](../helm/slurm-operator/README.md) |
| slurm-operator-crds | [helm/slurm-operator-crds/README.md](../helm/slurm-operator-crds/README.md) |
| slurm | [helm/slurm/README.md](../helm/slurm/README.md) |

### 其他文件

| 文件 | 路徑 |
|------|------|
| 主要 README | [README.md](../README.md) |
| 變更日誌 | [CHANGELOG.md](../CHANGELOG.md) |
| 版本策略 | [docs/versioning.md](../docs/versioning.md) |

---

## 快速開始

### 1. 安裝先決條件

```bash
# 安裝 cert-manager
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --set 'crds.enabled=true' \
  --namespace cert-manager --create-namespace
```

### 2. 安裝 Operator

```bash
# 安裝 CRDs
helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds

# 安裝 Operator
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace
```

### 3. 部署 Slurm 叢集

```bash
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace
```

---

## AI 輔助開發指南

### 新功能開發

1. 參考 [架構文件](./architecture.md) 了解系統設計
2. 查看 [資料模型](./data-models.md) 了解 CRD 結構
3. 使用 [原始碼樹分析](./source-tree-analysis.md) 定位相關檔案
4. 遵循 [開發指南](./development-guide.md) 的代碼規範

### 擴展 CRD

1. 在 `api/v1beta1/` 添加類型定義
2. 運行 `make generate manifests`
3. 在 `internal/controller/` 實現控制器
4. 在 `internal/webhook/` 添加驗證
5. 更新 Helm charts

### 調試問題

1. 查看控制器日誌: `kubectl logs -n slinky deploy/slurm-operator`
2. 檢查 CR 狀態: `kubectl describe controller <name>`
3. 參考 [NodeSet 控制器](../docs/concepts/nodeset-controller.md) 文件

---

## 文件生成資訊

| 項目 | 值 |
|------|-----|
| 掃描模式 | Exhaustive |
| 掃描時間 | 2025-12-18 |
| Go 檔案數 | 202 |
| 測試檔案數 | 105 |
| 依賴數量 | ~750 |
| CRD 數量 | 6 |
| 控制器數量 | 7 |
| Webhook 數量 | 7 |
