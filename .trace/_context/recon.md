# Reconnaissance — slurm-operator

## 專案概述

- **專案名稱**: slurm-operator（Slinky 計畫的一部份）
- **維護者**: SchedMD LLC（Slurm 的主要開發商），由 NVIDIA 支援開發
- **版本**: 1.2.0-rc1（`VERSION` 檔案）
- **授權**: Apache License 2.0
- **語言**: Go 1.26.3
- **定位**: Kubernetes Operator，用於部署與管理 Slurm HPC 工作負載管理器叢集

**一句話摘要**: slurm-operator 讓 Slurm（HPC workload manager）跑在 Kubernetes 上，透過 CRD 管理 Slurm 各元件（slurmctld、slurmd、slurmdbd、slurmrestd）的生命週期，達成 HPC 與雲端原生工作負載的統一排程平台。

---

## 技術棧總覽

| 類別 | 技術 | 版本 | 用途 |
|------|------|------|------|
| Runtime | Go | 1.26.3 | 主要開發語言 |
| Framework | controller-runtime | 0.23.3 | Kubernetes operator 框架 |
| API Extension | Kubernetes CRD | v1beta1 | 自定義資源定義 |
| Kubernetes | k8s.io/api | 0.35.2 | Kubernetes API 型別 |
| Slurm Client | slurm-client | v1.1.0-rc1 | 與 slurmrestd 通訊 |
| JWT | golang-jwt/jwt | v5 | auth/jwt 簽章 |
| Metrics | prometheus-operator/pkg/apis | 0.89.0 | Prometheus ServiceMonitor |
| MariaDB | mariadb-operator | v0.38.1 | Accounting 資料庫整合 |
| Container | Docker / Buildx | 28.5.2+ | 映像建置 |
| Deployment | Helm | v3 | 安裝套件 |
| Testing | ginkgo/gomega | v2 | BDD 測試框架 |
| E2E Testing | sigs.k8s.io/e2e-framework | v0.6.0 | End-to-end 測試 |
| CI/CD | （無 `.github/workflows/` 找到） | — | ⚠️ 未驗證，可能在 GitLab CI |
| Linting | golangci-lint | — | `.golangci.yaml` |
| Pre-commit | pre-commit | — | `.pre-commit-config.yaml` |

---

## 架構模式

**Kubernetes Operator（Monolith）**，遵循：
- [Golang module layout](https://go.dev/doc/modules/layout)
- [operator-sdk](https://sdk.operatorframework.io/) 慣例
- [Kubebuilder](https://book.kubebuilder.io/) 程式碼產生

Helm chart 分為三個獨立套件（並非 monorepo）：
- `helm/slurm-operator-crds` — CRD 定義
- `helm/slurm-operator` — Operator 本身
- `helm/slurm` — Slurm 叢集 (slurmctld, slurmd, slurmdbd 等)

---

## 目錄結構（3層）

```
slurm-operator/
├── api/                    # CRD 型別定義 (v1beta1)
│   └── v1beta1/            # 6個 CRD：Controller, NodeSet, LoginSet, Accounting, RestApi, Token
├── cmd/                    # 二進位進入點
│   ├── manager/            # slurm-operator 主程式 (controller manager)
│   └── webhook/            # webhook server 主程式
├── config/                 # Kustomize 部署設定
│   ├── crd/bases/          # 產生的 CRD YAML
│   ├── rbac/               # RBAC 設定
│   └── webhook/            # Webhook manifests
├── docs/                   # 官方文件 (Sphinx/RST + Markdown)
│   ├── concepts/           # 架構、NodeSet controller、SlurmClient 等概念說明
│   └── usage/              # 使用指南
├── hack/                   # 開發工具腳本（kind cluster 等）
├── helm/                   # Helm charts
│   ├── slurm/              # Slurm 叢集 chart
│   ├── slurm-operator/     # Operator chart（含 webhook）
│   └── slurm-operator-crds/# CRD chart
├── internal/               # 私有程式碼
│   ├── builder/            # Kubernetes 物件建構器（Pod/Service/ConfigMap 等）
│   │   ├── accountingbuilder/
│   │   ├── common/
│   │   ├── controllerbuilder/
│   │   ├── labels/
│   │   ├── loginbuilder/
│   │   ├── metadata/
│   │   ├── restapibuilder/
│   │   └── workerbuilder/
│   ├── clientmap/          # Slurm REST API client 的執行緒安全映射表
│   ├── controller/         # Reconcile loop 實作
│   │   ├── accounting/     # Accounting (slurmdbd) controller
│   │   ├── controller/     # Controller (slurmctld) controller
│   │   ├── loginset/       # LoginSet (login nodes) controller
│   │   ├── nodeset/        # NodeSet (slurmd workers) controller - 最複雜
│   │   ├── restapi/        # RestApi (slurmrestd) controller
│   │   ├── slurmclient/    # Slurm REST API client 管理
│   │   └── token/          # JWT Token controller
│   ├── defaults/           # CRD 欄位預設值
│   ├── syncsteps/          # 通用的 sync step pipeline
│   ├── utils/              # 雜項工具函式
│   └── webhook/            # Admission webhook 實作
├── pkg/                    # 公開可匯入的程式碼
│   ├── conditions/         # Pod condition 常數（Slurm node 狀態）
│   └── taints/             # Kubernetes taint 輔助函式
├── test/                   # 測試工具
│   └── e2e/                # End-to-end 測試
└── tools/                  # 建置工具
```

---

## CRD 清單（api/v1beta1/）

| CRD | 縮寫 | 代表 Slurm 元件 | 關鍵欄位 |
|-----|------|-----------------|----------|
| `Controller` | `slurmctld` | Slurm controller (slurmctld) | `slurmKeyRef`, `jwtKeyRef`, `accountingRef`, `inplaceReconfigure` |
| `NodeSet` | `nodesets`, `nss`, `slurmd` | Slurm 計算節點群組 (slurmd) | `controllerRef`, `replicas`, `scalingMode`, `partition` |
| `LoginSet` | — | 登入節點群組 | `controllerRef`, `sssdConfRef`, `sshdConfig` |
| `Accounting` | `slurmdbd` | 會計子系統 (slurmdbd) | `slurmKeyRef`, `jwtKeyRef`, `storageConfig` |
| `RestApi` | `slurmrestd` | REST API daemon (slurmrestd) | `controllerRef`, `replicas` |
| `Token` | `tokens`, `jwt` | JWT Token 管理 | `username`, `lifetime`, `jwtKeyRef`, `secretRef` |

---

## 既有文件掃描

### 已找到的文件目錄
- `docs/concepts/architecture.md` — 架構總覽，含 Slurm 特殊功能說明（Configless, auth/slurm, auth/jwt, Dynamic Nodes）
- `docs/concepts/nodeset-controller.md` — NodeSet controller 詳細說明
- `docs/concepts/slurmclient-controller.md` — SlurmClient controller 說明
- `docs/concepts/slurm.md` — Slurm 基礎概念
- `docs/installation.md` — 完整安裝指南
- `docs/usage/autoscaling.md` — HPA 自動縮放
- `docs/usage/hybrid.md` — Hybrid 模式（Kubernetes + bare-metal 混用）
- `docs/usage/jupyter.md` — Jupyter 整合
- `docs/usage/nodeset-operations.md` — NodeSet 操作
- `docs/usage/topology.md` — 拓撲感知排程
- `docs/usage/pyxis.md` — Pyxis OCI container 整合
- `docs/usage/sriov.md` — SR-IOV 網路設定
- `docs/usage/system-requirements.md` — 系統需求
- `docs/usage/tutorial-pytorch.md` — PyTorch 分散式訓練教學
- `docs/usage/workload-isolation.md` — 工作負載隔離
- `docs/versioning.md` — 版本策略

### 文件與程式碼落差
- `docs/concepts/architecture.md` 提到 `internal/webhook/` 目錄，但 webhook 目前在 `internal/webhook/`，**一致**。
- 文件中提到 `auth/jwt` 有 `JwtHs256KeyRef` 欄位，程式碼中標記為 `Deprecated: use JwtKeyRef instead`（`api/v1beta1/controller_types.go`），**文件可能尚未更新此棄用狀態**。
- `TaintKubeNodes` 欄位在 `NodeSetSpec` 標記為 `Deprecated`，webhook 也有警告，但未在現有文件中明確說明。

---

## 相依性亮點

### 核心框架
- `sigs.k8s.io/controller-runtime` — Operator 框架
- `k8s.io/kubernetes` — 直接引用 Kubernetes 內部套件（`k8s.io/kubernetes/pkg/controller`, `pkg/controller/daemon`），用於 NodeSet 的 DaemonSet 調度邏輯
- `github.com/SlinkyProject/slurm-client` — 自家 Slurm REST API client

### 特殊依賴
- `github.com/puttsk/hostlist` — Slurm hostlist 格式展開（如 `node[1-10]`）
- `github.com/mariadb-operator/mariadb-operator` — Accounting 資料庫整合
- `helm.sh/helm/v3` — E2E 測試中使用 Helm API
- `github.com/prometheus-operator/prometheus-operator` — ServiceMonitor CRD

---

## 相容性要求

| 軟體 | 最低版本 |
|------|----------|
| Kubernetes | v1.29 |
| Slurm | 25.11 |
| Cgroup | v2 |

---

## Slurm 必要功能

Operator 假設目標 Slurm 叢集已啟用：
1. **Configless Slurm** — slurmd 從 slurmctld 取得設定，不需要 NFS 共享
2. **auth/slurm** — 以加密 key 取代 MUNGE 認證
   - `use_client_ids` — 允許在無 LDAP 的容器化環境中認證使用者
3. **auth/jwt** — JWT 作為 REST API 認證（`AuthAltType`）
4. **Dynamic Nodes** — slurmd 以動態節點方式啟動，不需預先在 slurm.conf 定義
5. **Dynamic Topology** — 拓撲資訊由 operator 注入，不需預先設定 topology.yaml

---

## 部署架構

兩個獨立的 Kubernetes Deployment：
1. `slurm-operator` — Controller Manager（管理 CRD 調和）
2. `slurm-operator-webhook` — Admission Webhook Server（驗證與預設值注入）

安裝順序：
```
cert-manager → slurm-operator-crds → slurm-operator → slurm (cluster)
```
