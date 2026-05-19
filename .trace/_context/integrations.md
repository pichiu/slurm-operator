# Integrations — slurm-operator

## 外部整合總覽

slurm-operator 整合三個主要外部系統，以及一組可選的 Kubernetes ecosystem 元件。

---

## 1. Slurm REST API（slurm-client）

**最核心的外部整合**。

### 連線設定
```go
// internal/controller/slurmclient/slurmclient_sync.go
config := &slurmclient.Config{
    Server:    "http://{slurmrestd-service}:6820",
    AuthToken: jwt_token,
}
slurmClient, _ := slurmclient.NewClient(config, options)
```

REST API 對應的 Slurm 版本：`v0044`（使用 `slurmapi.V0044*` 型別），對應 Slurm 25.11+ 的 API 版本。

### API 端點使用

| 操作 | HTTP Method | 端點 | 用途 |
|------|------------|------|------|
| 列出節點 | GET | `/slurm/v0044/nodes` | CalculateNodeStatus, GetNodesForPods |
| 取得單一節點 | GET | `/slurm/v0044/node/{name}` | Drain/Undrain 前狀態確認 |
| 更新節點 | POST | `/slurm/v0044/node/{name}` | MakeNodeDrain/Undrain, UpdateNodeTopology |
| 刪除節點 | DELETE | `/slurm/v0044/node/{name}` | DeleteNode（PruneSlurmNodeRecords） |
| 列出 Jobs | GET | `/slurm/v0044/jobs` | GetNodeDeadlines（計算 job 完成時間） |

### 錯誤處理策略

```go
// slurmcontrol.go:tolerateError()
func tolerateError(err error) bool {
    if err == nil { return true }
    errText := err.Error()
    // 404 Not Found：節點不在 Slurm 中（正常情況，如節點尚未啟動）
    // 204 No Content：操作成功但無回應內容
    if errText == http.StatusText(http.StatusNotFound) ||
       errText == http.StatusText(http.StatusNoContent) {
        return true
    }
    return false
}
```

**沒有 circuit breaker**：錯誤直接回傳給 controller，由 controller-runtime 的 rate limiting 重排（exponential backoff）。

**沒有顯式 retry**：依賴 Kubernetes controller 的 reconcile loop 重試機制。

### JWT 認證更新

```go
// SlurmClient controller 在 JWT 過期前自動更新：
lifetime := 15 * time.Minute
refresh  := lifetime * 4 / 5  // = 12 minutes
// 到期前 12 分鐘重排 reconcile，重新產生 JWT
durationStore.Push(controllerKey.String(), refresh)

// 更新現有 client（不重建連線）：
slurmClient.SetServer(server)
slurmClient.SetToken(authToken)
```

### Client 快取（slurm-client 端）

slurm-client 內部有物件快取。NodeSet sync pipeline 的第 5 步（`RefreshNodeCache`）會強制刷新，確保後續步驟看到最新狀態：

```go
// slurmcontrol.go:RefreshNodeCache()
opts := &slurmclient.ListOptions{RefreshCache: true}
slurmClient.List(ctx, nodeList, opts)
```

---

## 2. Kubernetes API Server

主要透過 `sigs.k8s.io/controller-runtime` 的 client 操作。

### 特殊整合：直接引用 k8s.io/kubernetes

```go
// 使用 Kubernetes 內部 controller 工具
import (
    kubecontroller "k8s.io/kubernetes/pkg/controller"
    daemonutils "k8s.io/kubernetes/pkg/controller/daemon/util"
    podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)
```

這讓 NodeSet DaemonSet 模式能精確複製 Kubernetes DaemonSet 的排程邏輯。

### RBAC 需求（controller）

```yaml
# 從 kubebuilder markers 產生，主要資源：
slinky.slurm.net/nodesets: get;list;watch;create;update;patch;delete
slinky.slurm.net/controllers: get;list;watch
pods: get;list;watch;create;update;patch;delete
services: get;list;watch;create;update;patch;delete
nodes: get;list;watch;patch
apps/controllerrevisions: get;list;watch;create;update;patch;delete
policy/poddisruptionbudgets: get;list;watch;create;update;patch;delete
events: get;list;watch;create;update;patch
```

---

## 3. cert-manager（TLS 憑證）

Webhook server 需要 TLS 憑證。預設使用 cert-manager 自動簽發：

```yaml
# helm/slurm-operator/templates/certificate.yaml（推測）
certManager.enabled: true  # 預設啟用
```

`--set 'certManager.enabled=false'` 時，需要手動提供憑證（secret 掛載到 webhook pod）。

---

## 4. Prometheus（Metrics 監控）

### ServiceMonitor 整合

Controller CR 支援 Prometheus ServiceMonitor（需要安裝 prometheus-operator）：

```yaml
# Controller CR
spec:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
```

Controller builder 會建立/刪除 ServiceMonitor 物件。若 ServiceMonitor CRD 未安裝（`IsNoMatchError`），則優雅地跳過，不報錯。

### Metrics 端點
- Controller Manager：`:8080/metrics`（或 `--metrics-addr`）
- Webhook：預設停用（`--metrics-addr=0`）

---

## 5. MariaDB / MySQL（Accounting 資料庫）

Accounting CR 管理的 slurmdbd 需要 MariaDB/MySQL：

```go
// api/v1beta1/accounting_types.go:StorageConfig
type StorageConfig struct {
    Host           string
    Port           int    // 預設 3306
    Database       string // 預設 "slurm_acct_db"
    Username       string
    PasswordKeyRef corev1.SecretKeySelector  // 從 Secret 取密碼
}
```

支援：
- **mariadb-operator 管理的 MariaDB**（go.mod 中有 `github.com/mariadb-operator/mariadb-operator` 依賴，用於 CRD type 識別）
- **外部資料庫**（任何 Slurm 相容的 MySQL/MariaDB）
- **管理雲端資料庫服務**（如 RDS）

---

## 6. SSSD（System Security Services Daemon）

LoginSet 和 NodeSet（SSH 模式）需要 SSSD 提供使用者身份：

```yaml
# NodeSet 啟用 SSH 時需要
spec:
  ssh:
    enabled: true
    sssdConfRef:
      name: sssd-config
      key: sssd.conf
```

sssd.conf 透過 Secret reference 掛載，operator 負責將其掛載到 login/worker pods。

---

## 7. Pyxis（OCI Container Jobs）

⚠️ 未驗證（僅文件層面）。

`docs/usage/pyxis.md` 描述 Pyxis 整合，Pyxis 是 NVIDIA 開發的 Slurm container plugin。整合方式透過 NodeSet 的 `configFileRefs` 提供 Pyxis 設定，不需要 operator 程式碼變更。

---

## 8. GPU Operator / Device Plugins

⚠️ 未驗證（文件層面）。

GPU 支援透過標準 Kubernetes device plugin 機制（DRA 或 device plugins）：

```yaml
# NodeSet spec 中設定 GPU 資源請求
spec:
  slurmd:
    resources:
      limits:
        nvidia.com/gpu: 8
  extraConfMap:
    Gres: ["gpu:h100:8"]
```

operator 不直接與 GPU operator 溝通，透過標準 Kubernetes 資源請求整合。

---

## 9. SR-IOV Network Device Plugin

⚠️ 未驗證（文件層面）。

`docs/usage/sriov.md` 描述 SR-IOV 高速網路設定，整合方式同 GPU，透過標準 Kubernetes device plugin 機制。

---

## 錯誤處理策略彙整

| 整合 | 404 Not Found | 連線失敗 | 認證失敗 |
|------|--------------|---------|---------|
| Slurm REST API | `tolerateError()` 略過 | 由 reconcile loop 重試 | SlurmClient controller 重新產生 JWT |
| Kubernetes API | 個別處理（物件已刪除） | controller-runtime 重試 | N/A（in-cluster SA） |
| cert-manager | N/A | 安裝失敗時 webhook 無法啟動 | N/A |
| Prometheus | `IsNoMatchError` 略過 | N/A | N/A |
| MariaDB | slurmdbd 自行處理 | slurmdbd 自行重試 | slurmdbd 自行處理 |

---

## 整合架構圖

```
slurm-operator (controller manager)
    │
    ├── Kubernetes API Server ←→ (watches: CRDs, Pods, Nodes, Secrets...)
    │                              (creates/updates: Pods, Services, ConfigMaps...)
    │
    ├── Slurm REST API (slurmrestd)
    │   └── JWT Authentication (via slurm-client)
    │       └── GET /nodes, POST /node/{name}, GET /jobs
    │
    └── cert-manager (TLS for webhook)

slurm-operator-webhook (webhook server)
    │
    ├── Kubernetes API Server ←→ (watches: CRDs, Pods)
    └── (validates CRDs, mutates pod/binding for topology)
```
