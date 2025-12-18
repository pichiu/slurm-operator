# Source Tree Analysis - slurm-operator

> 生成日期：2025-12-18
> 掃描模式：Exhaustive Scan

## 專案根目錄結構

```
slurm-operator/
├── api/                          # 🎯 CRD API 類型定義
│   └── v1beta1/                  # API 版本 v1beta1
│       ├── *_types.go            # CRD 類型結構
│       ├── *_keys.go             # 短名稱和 GVK 常量
│       ├── *_convert.go          # 版本轉換邏輯
│       ├── base_types.go         # 共享基礎類型
│       ├── well_known.go         # 已知標籤/註釋常量
│       ├── groupversion_info.go  # API 群組版本定義
│       └── zz_generated.deepcopy.go  # 生成的 DeepCopy
│
├── cmd/                          # 🚀 入口點
│   ├── manager/                  # Operator 主程式
│   │   └── main.go               # 控制器管理器入口
│   └── webhook/                  # Webhook 伺服器
│       └── main.go               # Webhook 入口
│
├── config/                       # ⚙️ Kubernetes 清單 (Kubebuilder 生成)
│   ├── crd/
│   │   └── bases/                # CRD YAML 定義
│   ├── rbac/
│   │   └── role.yaml             # ClusterRole 定義
│   └── webhook/
│       └── manifests.yaml        # Webhook 配置
│
├── internal/                     # 🔧 內部套件
│   ├── builder/                  # Kubernetes 資源構建器
│   │   ├── labels/               # 標籤構建器
│   │   ├── metadata/             # 元數據構建器
│   │   ├── scripts/              # 腳本模板
│   │   ├── *_app.go              # 應用程式構建 (controller, worker, login, etc.)
│   │   ├── *_config.go           # ConfigMap 構建
│   │   ├── *_service.go          # Service 構建
│   │   ├── container.go          # Container 構建
│   │   ├── pod_template.go       # Pod 範本構建
│   │   └── servicemonitor.go     # Prometheus ServiceMonitor
│   │
│   ├── clientmap/                # Slurm 客戶端池管理
│   │   └── clientmap.go          # 執行期安全的客戶端映射
│   │
│   ├── controller/               # 🎮 控制器實現
│   │   ├── accounting/           # Accounting (slurmdbd) 控制器
│   │   │   ├── accounting_controller.go
│   │   │   ├── accounting_sync.go
│   │   │   ├── accounting_sync_status.go
│   │   │   └── eventhandler/
│   │   │
│   │   ├── controller/           # Controller (slurmctld) 控制器
│   │   │   ├── controller_controller.go
│   │   │   ├── controller_sync.go
│   │   │   ├── controller_sync_status.go
│   │   │   └── eventhandler/
│   │   │
│   │   ├── nodeset/              # NodeSet (slurmd) 控制器 ⭐ 核心
│   │   │   ├── nodeset_controller.go
│   │   │   ├── nodeset_sync.go
│   │   │   ├── nodeset_sync_status.go
│   │   │   ├── nodeset_history.go    # ControllerRevision 管理
│   │   │   ├── indexes/              # API 索引
│   │   │   ├── podcontrol/           # Pod 生命週期管理
│   │   │   ├── slurmcontrol/         # Slurm 通訊介面
│   │   │   ├── eventhandler/         # Pod/Node/Controller 事件
│   │   │   └── utils/                # 排序和工具
│   │   │
│   │   ├── loginset/             # LoginSet (login nodes) 控制器
│   │   │   ├── loginset_controller.go
│   │   │   └── eventhandler/
│   │   │
│   │   ├── restapi/              # RestApi (slurmrestd) 控制器
│   │   │   ├── restapi_controller.go
│   │   │   └── eventhandler/
│   │   │
│   │   ├── token/                # Token (JWT) 控制器
│   │   │   ├── token_controller.go
│   │   │   └── slurmjwt/         # JWT 生成邏輯
│   │   │
│   │   └── slurmclient/          # Slurm 客戶端管理控制器
│   │       └── slurmclient_controller.go
│   │
│   ├── utils/                    # 🛠️ 工具函數
│   │   ├── config/               # 配置檔案生成
│   │   ├── crypto/               # 密碼學 (SSH 金鑰、雜湊)
│   │   ├── domainname/           # Kubernetes DNS 名稱
│   │   ├── durationstore/        # 持續時間存儲
│   │   ├── historycontrol/       # ControllerRevision 管理
│   │   ├── mathutils/            # 數學工具
│   │   ├── objectutils/          # K8s 物件操作
│   │   ├── podcontrol/           # Pod 控制介面
│   │   ├── podinfo/              # Pod 資訊結構
│   │   ├── podutils/             # Pod 狀態檢查
│   │   ├── reflectutils/         # 反射工具
│   │   ├── refresolver/          # CR 參考解析
│   │   ├── structutils/          # 結構操作
│   │   ├── testutils/            # 測試工具
│   │   └── timestore/            # 時間存儲
│   │
│   └── webhook/                  # 🔒 Webhook 實現
│       ├── accounting_webhook.go
│       ├── controller_webhook.go
│       ├── nodeset_webhook.go
│       ├── loginset_webhook.go
│       ├── restapi_webhook.go
│       ├── token_webhook.go
│       └── pod_binding_webhook.go  # Pod 綁定修改器
│
├── pkg/                          # 📦 公開套件
│   ├── conditions/               # Slurm 節點狀態條件
│   │   └── constants.go          # PodCondition 常量
│   └── taints/                   # 節點污點管理
│       └── taints.go             # Worker 節點污點
│
├── helm/                         # 📊 Helm Charts
│   ├── slurm-operator/           # Operator Chart
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   └── templates/
│   │       ├── operator/         # Operator Deployment
│   │       ├── webhook/          # Webhook Deployment
│   │       ├── cert-manager/     # 證書管理
│   │       └── tests/            # Helm 測試
│   │
│   ├── slurm-operator-crds/      # CRD Chart
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   └── templates/            # CRD YAML 檔案
│   │
│   └── slurm/                    # Slurm 集群 Chart
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── _vendor/              # 第三方整合 (NVIDIA DCGM)
│       └── templates/
│           ├── accounting/       # Accounting CR
│           ├── controller/       # Controller CR
│           ├── nodeset/          # NodeSet CR
│           ├── loginset/         # LoginSet CR
│           ├── restapi/          # RestApi CR
│           ├── secrets/          # 認證密鑰
│           ├── cluster/          # 集群配置
│           └── vendor/           # 供應商整合
│
├── docs/                         # 📚 專案文件
│   ├── concepts/                 # 概念文件
│   │   ├── architecture.md
│   │   ├── nodeset-controller.md
│   │   ├── slurmclient-controller.md
│   │   └── slurm.md
│   ├── usage/                    # 使用指南
│   │   ├── installation.md
│   │   ├── develop.md
│   │   ├── autoscaling.md
│   │   ├── hybrid.md
│   │   └── ... (更多)
│   ├── _static/                  # 靜態資源
│   │   └── images/               # 架構圖
│   └── versioning.md
│
├── test/                         # 🧪 測試
│   └── e2e/                      # 端對端測試
│
├── hack/                         # 🔨 開發腳本
│   ├── kind.yaml                 # Kind 叢集配置
│   └── resources/                # 測試資源
│
├── tools/                        # 🔧 工具配置
├── LICENSES/                     # 📄 授權檔案
├── CHANGELOG/                    # 📝 變更日誌
│
├── go.mod                        # Go 模組定義
├── go.sum                        # 依賴校驗
├── Makefile                      # 構建命令
├── Dockerfile                    # 容器映像
├── docker-bake.hcl               # Docker Buildx 配置
├── PROJECT                       # Kubebuilder 專案配置
├── README.md                     # 專案說明
└── VERSION                       # 版本號
```

## 關鍵目錄說明

### 🎯 API 層 (`api/v1beta1/`)

定義 6 個 CRD 類型：

| CRD | 檔案 | 用途 |
|-----|------|------|
| Controller | `controller_types.go` | Slurm 控制器 (slurmctld) |
| NodeSet | `nodeset_types.go` | 計算節點集合 (slurmd) |
| LoginSet | `loginset_types.go` | 登入節點集合 |
| Accounting | `accounting_types.go` | 會計資料庫 (slurmdbd) |
| RestApi | `restapi_types.go` | REST API (slurmrestd) |
| Token | `token_types.go` | JWT 令牌管理 |

### 🎮 控制器層 (`internal/controller/`)

每個控制器負責：
- **Reconcile Loop**: 調和期望狀態與實際狀態
- **Event Handlers**: 監視相關資源變更
- **Sync Logic**: 同步子資源 (Pods, Services, ConfigMaps)
- **Status Updates**: 更新 CR 狀態

### 🔧 構建器層 (`internal/builder/`)

提供流利的 Builder 模式：
- 構建 Pod 範本、容器、服務
- 管理標籤和元數據
- 生成 Slurm 配置檔案

### 📊 Helm Charts (`helm/`)

三層部署架構：
1. **slurm-operator-crds**: CRD 定義（最先安裝）
2. **slurm-operator**: Operator 和 Webhook
3. **slurm**: Slurm 集群實例

## 入口點

| 元件 | 入口檔案 | 端口 |
|------|----------|------|
| Operator | `cmd/manager/main.go` | 8080 (metrics), 8081 (health) |
| Webhook | `cmd/webhook/main.go` | 9443 (webhook), 8081 (health) |

## 整合點

### Slurm 整合
- `internal/controller/slurmclient/`: 管理 Slurm 客戶端連線
- `internal/controller/nodeset/slurmcontrol/`: Slurm 節點狀態同步
- 使用 `github.com/SlinkyProject/slurm-client` 套件

### Kubernetes 整合
- controller-runtime: 控制器框架
- client-go: Kubernetes API 客戶端
- cert-manager: TLS 證書管理

### 監控整合
- Prometheus ServiceMonitor 支援
- 健康檢查端點

## 檔案統計

| 類型 | 數量 |
|------|------|
| Go 檔案 (內部) | 202 |
| 測試檔案 | 105 |
| API 類型檔案 | 22 |
| Helm 範本 | ~60 |
| 文件檔案 | 22 |

## 重要檔案快速參考

| 用途 | 檔案路徑 |
|------|----------|
| CRD 定義 | `config/crd/bases/slinky.slurm.net_*.yaml` |
| RBAC 權限 | `config/rbac/role.yaml` |
| Operator 入口 | `cmd/manager/main.go` |
| NodeSet 控制器 | `internal/controller/nodeset/nodeset_controller.go` |
| Helm 預設值 | `helm/slurm/values.yaml` |
| 安裝指南 | `docs/installation.md` |
