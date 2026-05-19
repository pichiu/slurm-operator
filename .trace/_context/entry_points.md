# Entry Points — slurm-operator

## 兩個二進位進入點

### 1. Controller Manager（`cmd/manager/main.go`）

**啟動路徑**：
```
main()
 ├── parseFlags()           # 解析 CLI flags
 ├── ctrl.NewManager()      # 建立 controller-runtime manager
 │   ├── scheme 註冊:
 │   │   ├── clientgoscheme (core k8s types)
 │   │   ├── monitoringv1 (Prometheus ServiceMonitor)
 │   │   └── slinkyv1beta1 (自定義 CRD)
 │   └── 設定: metrics, cache, health probe, leader election
 ├── clientmap.NewClientMap()          # 初始化 Slurm client map
 ├── controller.NewReconciler(...).SetupWithManager(mgr)  # Controller CRD
 ├── restapi.NewReconciler(...).SetupWithManager(mgr)     # RestApi CRD
 ├── accounting.NewReconciler(...).SetupWithManager(mgr)  # Accounting CRD
 ├── nodeset.NewReconciler(...).SetupWithManager(mgr)     # NodeSet CRD
 ├── loginset.NewReconciler(...).SetupWithManager(mgr)    # LoginSet CRD
 ├── slurmclient.NewReconciler(...).SetupWithManager(mgr) # SlurmClient (virtual)
 ├── token.NewReconciler(...).SetupWithManager(mgr)       # Token CRD
 ├── mgr.AddHealthzCheck("healthz", healthz.Ping)         # /healthz
 ├── mgr.AddReadyzCheck("readyz", healthz.Ping)           # /readyz
 └── mgr.Start(ctrl.SetupSignalHandler())                 # 主迴圈
```

#### CLI Flags
| Flag | 預設 | 說明 |
|------|------|------|
| `--health-addr` | `:8081` | 健康檢查端點 |
| `--metrics-addr` | `:8080` | Prometheus metrics 端點 |
| `--leader-elect` | `false` | 啟用 leader election |
| `--leader-elect-namespace` | `""` | Leader election namespace |
| `--metrics-secure` | `false` | Metrics 使用 TLS |
| `--enable-http2` | `false` | 啟用 HTTP/2（預設關閉以避免 CVE） |
| `--namespaces` | `""` | 限制 watch 的 namespace（逗號分隔） |
| `--propagated-node-conditions` | `""` | 傳播到 Slurm drain reason 的 Node condition 型別 |
| `--nodeset-workers` | `1` | NodeSet controller 並發 worker 數 |

#### 重要初始化細節
- `clientMap` 是一個執行緒安全的 map，儲存每個 `Controller` CR 對應的 Slurm REST API client
- HTTP/2 預設禁用，避免 Stream Cancellation CVE (GHSA-qppj-fm5r-hxr3, GHSA-4374-p667-p6c8)
- `propagatedNodeConditions` 讓 Kubernetes Node conditions（如硬體故障）能傳播成 Slurm drain reason

---

### 2. Webhook Server（`cmd/webhook/main.go`）

**啟動路徑**：
```
main()
 ├── parseFlags()           # 解析 CLI flags
 ├── ctrl.NewManager()      # 建立 manager（只含 webhook server）
 ├── mgr.AddHealthzCheck / AddReadyzCheck
 ├── ControllerWebhook.SetupWebhookWithManager(mgr)    # 驗證 Controller CR
 ├── RestapiWebhook.SetupWebhookWithManager(mgr)       # 驗證 RestApi CR
 ├── AccountingWebhook.SetupWebhookWithManager(mgr)    # 驗證 Accounting CR
 ├── NodeSetWebhook.SetupWebhookWithManager(mgr)       # 驗證 NodeSet CR
 ├── LoginSetWebhook.SetupWebhookWithManager(mgr)      # 驗證 LoginSet CR
 ├── TokenWebhook.SetupWebhookWithManager(mgr)         # 驗證 Token CR
 ├── PodBindingWebhook.SetupWebhookWithManager(mgr)    # Mutate pod/binding（注入 topology）
 └── mgr.Start(ctrl.SetupSignalHandler())
```

#### CLI Flags（Webhook）
| Flag | 預設 | 說明 |
|------|------|------|
| `--server-addr` | `:9443` | Webhook 監聽地址 |
| `--health-addr` | `:8081` | 健康檢查端點 |
| `--metrics-addr` | `0` | Metrics（預設停用） |
| `--leader-elect` | `false` | Leader election |
| `--namespaces` | `""` | 限制 watch namespace |

#### Webhook 清單
| Webhook | 型別 | 路徑 | 觸發時機 |
|---------|------|------|----------|
| `ControllerWebhook` | Validator | `/validate-slinky-slurm-net-v1beta1-controller` | Controller create/update |
| `RestapiWebhook` | Validator | `/validate-slinky-slurm-net-v1beta1-restapi` | RestApi create/update |
| `AccountingWebhook` | Validator | `/validate-slinky-slurm-net-v1beta1-accounting` | Accounting create/update |
| `NodeSetWebhook` | Validator | `/validate-slinky-slurm-net-v1beta1-nodeset` | NodeSet create/update |
| `LoginSetWebhook` | Validator | `/validate-slinky-slurm-net-v1beta1-loginset` | LoginSet create/update |
| `TokenWebhook` | Validator | `/validate-slinky-slurm-net-v1beta1-token` | Token create/update |
| `PodBindingWebhook` | Mutator | `/mutate--v1-binding` | Pod binding（排程時） |

---

## Scheme 初始化（`init()` in `cmd/manager/main.go`）

```go
utilruntime.Must(clientgoscheme.AddToScheme(scheme))    // 核心 k8s 型別
utilruntime.Must(monitoringv1.AddToScheme(scheme))       // Prometheus ServiceMonitor
utilruntime.Must(slinkyv1beta1.AddToScheme(scheme))      // Slinky CRDs
```

API group: `slinky.slurm.net` / 版本: `v1beta1`（`api/v1beta1/groupversion_info.go`）

---

## Controller 啟動細節

每個 Controller 在 `NewReconciler()` 時會建立：
- `builder.*Builder` — 對應 CRD 的 Kubernetes 物件建構器
- `refresolver.RefResolver` — 解析 `controllerRef` 指向的 CR
- `historycontrol.HistoryControlInterface` — ControllerRevision 管理（僅 NodeSet）
- `podcontrol.PodControlInterface` — Pod 建立/刪除（僅 NodeSet）
- `slurmcontrol.SlurmControlInterface` — Slurm API 操作（僅 NodeSet）
- `events.EventRecorder` — Kubernetes Event 發送

NodeSet controller 額外啟動一個 backoff GC goroutine，週期清理失敗 pod 的 backoff 紀錄：
```go
go wait.Until(failedPodsBackoff.GC, BackoffGCInterval, ctx.Done())
// BackoffGCInterval = 1 minute, backoff max = 15 minutes
```
