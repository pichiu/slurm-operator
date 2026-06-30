# slurm-operator 專案總覽

## 一句話總結

**slurm-operator** 是由 SchedMD（Slurm 原廠）開發的 Kubernetes Operator，讓 HPC workload manager Slurm 以容器化方式運行於 Kubernetes 叢集中，透過 CRD 管理 Slurm 所有元件的生命週期，使組織能在同一基礎設施上同時運行傳統 HPC 工作負載與雲端原生應用。

---

## 技術棧總覽

| 類別 | 技術 | 版本 | 用途 |
|------|------|------|------|
| Runtime | Go | 1.26.4 | 主要開發語言（更新於 2026-06-30） |
| Operator Framework | controller-runtime | 0.23.3 | Kubernetes reconcile loop 框架 |
| API Extension | Kubernetes CRD | v1beta1 | 自定義資源（slinky.slurm.net） |
| Kubernetes | k8s.io/api | 0.35.2 | Kubernetes API 型別 |
| Slurm API | slurm-client | v1.1.0-rc1（0dac744） | Slurm REST API（v0044）客戶端（更新於 2026-06-30） |
| JWT | golang-jwt/jwt | v5 | Slurm auth/jwt 簽章 |
| Monitoring | prometheus-operator | 0.89.0 | ServiceMonitor CRD |
| Database 整合 | mariadb-operator | v0.38.1 | Accounting 資料庫（slurmdbd） |
| 容器化 | Docker Buildx | 28.5.2+ | 映像建置與多平台發布 |
| 部署 | Helm v3 | 3.20.2 | Kubernetes 安裝套件 |
| 測試 | Ginkgo/Gomega | v2 | BDD 單元/整合測試 |
| E2E 測試 | sigs.k8s.io/e2e-framework | v0.6.0 | End-to-end 測試 |
| Linting | golangci-lint | — | 程式碼品質 |
| TLS | cert-manager | — | Webhook 憑證（外部依賴） |
| 版本 | 1.2.0-rc1 | — | 目前版本 |

---

## 相容性需求

| 軟體 | 最低版本 |
|------|----------|
| Kubernetes | v1.29 |
| Slurm | 25.11 |
| Cgroup | v2 |

---

## 關鍵指令速查

### 開發建置

```bash
# 建置 container images
make build-images

# 建置 Helm charts
make build-chart

# 產生 CRD YAML（從 Go type 定義）
make manifests

# 產生 DeepCopy 等程式碼
make generate

# 執行所有測試
make test

# 執行 linting
make lint

# 查看所有可用指令
make help
```

### 安裝（Helm）

```bash
# 1. 安裝 cert-manager（前置條件）
helm install cert-manager oci://quay.io/jetstack/charts/cert-manager \
  --namespace cert-manager --create-namespace --set crds.enabled=true

# 2. 安裝 CRDs
helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds

# 3. 安裝 Operator
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace

# 4. 安裝 Slurm 叢集
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace
```

### 升級

```bash
# v1.Y 升級（支援 CRD conversion，不需要重裝）
helm upgrade slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds \
  --version $SLINKY_VERSION
helm upgrade slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace slinky --version $SLINKY_VERSION
```

### 驗證

```bash
# 確認 operator 運行狀態
kubectl -n slinky get pods
# NAME                                      READY   STATUS
# slurm-operator-xxx                        1/1     Running
# slurm-operator-webhook-xxx                1/1     Running

# 查看 NodeSet 狀態
kubectl get nodesets -n slurm

# 查看 Slurm 叢集 pods
kubectl -n slurm get pods
```

---

## 文件地圖

| 文件 | 說明 |
|------|------|
| [INDEX.md](./INDEX.md) | 本文件：專案總覽與速查 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 系統架構、元件關係、設計決策 |
| [DATA_MODEL.md](./DATA_MODEL.md) | CRD 資料模型、欄位說明 |
| [API_SURFACE.md](./API_SURFACE.md) | CRD API 參考、Webhook 說明 |
| [DEV_GUIDE.md](./DEV_GUIDE.md) | 開發者上手指南、測試、貢獻流程 |
| [CODEBASE_MAP.md](./CODEBASE_MAP.md) | 程式碼地圖、目錄結構、快速導航 |
| [DISCOVERY_LOG.md](./DISCOVERY_LOG.md) | 探索紀錄、技術債、待解問題 |
| [TRACE_META.md](./TRACE_META.md) | Trace metadata（增量更新用） |

### 官方文件

| 文件 | 說明 |
|------|------|
| [docs/concepts/architecture.md](../docs/concepts/architecture.md) | 官方架構說明 |
| [docs/installation.md](../docs/installation.md) | 官方安裝指南 |
| [docs/usage/](../docs/usage/) | 使用指南（GPU、autoscaling、hybrid 等） |

---

## 專案術語表

| 術語 | 說明 |
|------|------|
| **Slinky** | 本計畫的品牌名稱，由 SchedMD 主導，NVIDIA 支援 |
| **Controller** | Slurm 控制器（slurmctld）的 CRD，管理整個 Slurm 叢集的控制平面 |
| **NodeSet** | Slurm 計算節點（slurmd）群組的 CRD，類似 StatefulSet 或 DaemonSet |
| **LoginSet** | Slurm 登入節點（submit node）群組的 CRD |
| **Accounting** | Slurm 會計子系統（slurmdbd）的 CRD |
| **RestApi** | Slurm REST API daemon（slurmrestd）的 CRD |
| **Token** | Slurm JWT Token 管理的 CRD |
| **slurmctld** | Slurm 主控制 daemon |
| **slurmd** | Slurm 計算節點 daemon |
| **slurmdbd** | Slurm 資料庫 daemon（負責 accounting） |
| **slurmrestd** | Slurm REST API daemon |
| **Configless** | Slurm 設定集中模式，slurmd 從 slurmctld 取得設定，不需要 NFS |
| **auth/slurm** | 以共享加密 key 取代 MUNGE 的 Slurm 認證插件 |
| **auth/jwt** | JWT 認證，用於 REST API（slurmrestd）通訊 |
| **Dynamic Nodes** | slurmd 動態節點模式，不需要預先在 slurm.conf 定義 |
| **Drain** | Slurm 節點狀態：標記為不接受新 job，但現有 job 繼續執行 |
| **ScalingMode** | NodeSet 的縮放模式：StatefulSet（固定副本數）或 DaemonSet（每 node 一個 pod） |
| **ScheduledUpdate** | NodeSet UpdateStrategy 新類型（2026-06-30）：透過 Slurm MAINT reservation 在排程時間窗口內更新 pod |
| **OversubscribeNode** | NodeSet 選項（2026-06-30）：允許多個 NodeSet pod 共用同一 Kubernetes Node（移除 anti-affinity），不建議生產使用 |
| **ClientMap** | 執行緒安全的 Slurm REST API client 映射表，key 為 Controller CR 的 namespace/name |
| **PodBindingWebhook** | Mutating webhook，在 Pod 被排程時注入 Slurm topology annotation |
| **WorkloadDisruptionProtection** | 透過 PodDisruptionBudget 保護正在執行 Slurm job 的 Pod 不被驅逐 |
| **SyncSteps** | operator 內部的 pipeline 模式，每個 controller 的 sync 邏輯由一系列有序步驟組成 |
| **Hostlist** | Slurm 主機列表格式，如 `node[1-10]`，由 `hostlist` 函式庫展開 |
