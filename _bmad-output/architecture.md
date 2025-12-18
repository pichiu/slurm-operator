# Architecture Document - slurm-operator

> 生成日期：2025-12-18
> 掃描模式：Exhaustive Scan

## 1. 執行摘要

slurm-operator 是一個 Kubernetes Operator，採用標準的 Operator 模式實現，使用 Kubebuilder v4 框架構建。它提供 6 個 Custom Resource Definitions (CRDs) 和 7 個控制器來管理 Slurm HPC 叢集的完整生命週期。

## 2. 技術堆疊

| 類別 | 技術 | 版本 | 說明 |
|------|------|------|------|
| 語言 | Go | 1.25.0 | 主要開發語言 |
| 框架 | Kubebuilder | v4 | Operator 腳手架 |
| 控制器 | controller-runtime | v0.22.4 | 控制器框架 |
| K8s API | client-go | v0.34.3 | Kubernetes 客戶端 |
| Slurm | slurm-client | v1.0.0 | Slurm API 客戶端 |
| 測試 | Ginkgo/Gomega | v2/v1 | BDD 測試框架 |
| 監控 | Prometheus | - | 指標收集 |
| 部署 | Helm | v3.19.0 | Chart 打包 |
| 容器 | Distroless | nonroot | 基礎映像 |

## 3. 架構模式

### 3.1 Kubernetes Operator Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     slurm-operator                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ Controller  │  │  NodeSet    │  │  LoginSet   │         │
│  │ Reconciler  │  │  Reconciler │  │  Reconciler │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ Accounting  │  │   RestApi   │  │    Token    │         │
│  │ Reconciler  │  │  Reconciler │  │  Reconciler │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│  ┌─────────────────────────────────────────────────┐       │
│  │              SlurmClient Reconciler              │       │
│  └─────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Slurm Cluster                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  slurmctld  │  │   slurmd    │  │  slurmdbd   │         │
│  │ (Controller)│  │  (NodeSet)  │  │ (Accounting)│         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│  ┌─────────────┐  ┌─────────────┐                          │
│  │ slurmrestd  │  │    login    │                          │
│  │  (RestApi)  │  │  (LoginSet) │                          │
│  └─────────────┘  └─────────────┘                          │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 控制器架構

每個控制器遵循相同的模式：

```go
type XxxReconciler struct {
    client.Client                           // K8s API 客戶端
    Scheme        *runtime.Scheme           // 類型註冊
    EventRecorder record.EventRecorder      // 事件記錄
    DurationStore *durationstore.DurationStore // 重試延遲
}

func (r *XxxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. 獲取 CR
    // 2. 檢查刪除/Finalizer
    // 3. 同步子資源 (Pod, Service, ConfigMap, Secret)
    // 4. 更新狀態
    // 5. 返回結果
}

func (r *XxxReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&slinkyv1beta1.Xxx{}).
        Owns(&corev1.Pod{}).
        Watches(&slinkyv1beta1.Controller{}, handler.EnqueueRequestsFromMapFunc(...)).
        Complete(r)
}
```

## 4. 元件架構

### 4.1 CRD 資源關係

```
                    ┌─────────────┐
                    │  Controller │
                    │  (slurmctld)│
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
    ┌──────────┐     ┌──────────┐     ┌──────────┐
    │  NodeSet │     │ LoginSet │     │  RestApi │
    │ (slurmd) │     │  (login) │     │(slurmrestd)
    └──────────┘     └──────────┘     └──────────┘
          │
          │ (optional)
          ▼
    ┌──────────┐
    │Accounting│
    │(slurmdbd)│
    └──────────┘

    ┌──────────┐
    │  Token   │ (獨立，參考 jwtHs256KeyRef)
    │  (JWT)   │
    └──────────┘
```

### 4.2 資源所有權

| 父資源 | 擁有的子資源 |
|--------|-------------|
| Controller | StatefulSet, Service, ConfigMap, Secret, ServiceMonitor |
| NodeSet | Pod, Service, ControllerRevision, PodDisruptionBudget |
| LoginSet | Deployment, Service, ConfigMap, Secret |
| Accounting | StatefulSet, Service, ConfigMap, Secret |
| RestApi | Deployment, Service, ConfigMap, Secret |
| Token | Secret |

### 4.3 事件處理流程

```
┌──────────────────────────────────────────────────────────┐
│                    Event Source                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ Watch: CR   │  │ Watch: Pod  │  │Watch: Secret│       │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘       │
└─────────┼────────────────┼────────────────┼──────────────┘
          │                │                │
          ▼                ▼                ▼
┌──────────────────────────────────────────────────────────┐
│                   Event Handlers                          │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ EnqueueRequestForObject / EnqueueRequestsFromMapFunc│ │
│  └─────────────────────────────────────────────────────┘ │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│                     Work Queue                            │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ Rate-Limited Queue with Exponential Backoff         │ │
│  └─────────────────────────────────────────────────────┘ │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│                   Reconciler                              │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ Reconcile() → Sync() → UpdateStatus()               │ │
│  └─────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

## 5. 控制器詳細設計

### 5.1 NodeSet 控制器 (核心)

NodeSet 控制器是最複雜的控制器，負責管理 Slurm 計算節點。

**關鍵特性：**
- **Expectations 追蹤**: 使用 `UIDTrackingControllerExpectations` 追蹤預期 Pod 建立/刪除
- **Slurm 狀態同步**: 從 Slurm 守護程式同步節點狀態 (Idle, Allocated, Down, Drain)
- **歷史版本管理**: 使用 ControllerRevision 管理配置版本
- **工作負載保護**: PodDisruptionBudget 防止中斷運行中的作業

**事件處理器：**
```go
// Pod 事件
PodEventHandler:
  - Create: expectations.CreationObserved()
  - Update: 檢查標籤/狀態變更
  - Delete: expectations.DeletionObserved()

// Node 事件
NodeEventHandler:
  - 監視污點/標籤/註釋變更
  - 處理拓撲線同步

// Controller 事件
ControllerEventHandler:
  - 將所有參考此 Controller 的 NodeSet 加入佇列
```

### 5.2 Controller 控制器

管理 slurmctld StatefulSet 和相關配置。

**關鍵特性：**
- **配置檔案管理**: 透過 ConfigMap 掛載自訂配置
- **持久化存儲**: 支援 PVC 用於狀態恢復
- **Metrics 整合**: Prometheus ServiceMonitor

### 5.3 SlurmClient 控制器

管理 Slurm 客戶端連線池。

**關鍵特性：**
- **ClientMap**: 執行期安全的客戶端池
- **事件傳播**: 透過 channel 傳播 Slurm 節點狀態變更

## 6. Webhook 架構

### 6.1 驗證 Webhooks

| Webhook | 資源 | 驗證規則 |
|---------|------|----------|
| ControllerWebhook | Controller | ClusterName 長度/格式、不可變欄位檢查 |
| NodeSetWebhook | NodeSet | UpdateStrategy、PVC 保留策略 |
| AccountingWebhook | Accounting | 框架實現 |
| LoginSetWebhook | LoginSet | 框架實現 |
| RestapiWebhook | RestApi | 框架實現 |
| TokenWebhook | Token | 框架實現 |

### 6.2 修改 Webhook

| Webhook | 資源 | 修改邏輯 |
|---------|------|----------|
| PodBindingWebhook | pods/binding | 複製節點拓撲線註釋到 Pod |

## 7. Builder 模式

### 7.1 資源構建流程

```go
builder := builder.New(client)

// 構建 Pod 範本
podTemplate := builder.buildPodTemplate(PodTemplateOpts{
    Key:      key,
    Metadata: metadata,
    base:     baseSpec,
    merge:    mergeSpec,
})

// 構建服務
service, err := builder.BuildService(ServiceOpts{
    Key:      key,
    Headless: true,
    Port:     6817,
}, owner)
```

### 7.2 標籤管理

```go
labels := labels.NewBuilder().
    WithApp(labels.ControllerApp).
    WithInstance(instance).
    WithComponent(labels.ControllerComp).
    WithPartOf("slurm").
    WithManagedBy("slurm-operator").
    Build()
```

## 8. 部署架構

### 8.1 Helm Charts 層次

```
┌─────────────────────────────────────────────────────────────┐
│                   slurm-operator-crds                        │
│  (CRD 定義，必須最先安裝)                                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     slurm-operator                           │
│  ┌─────────────────────┐  ┌─────────────────────┐          │
│  │   Operator Deploy   │  │   Webhook Deploy    │          │
│  └─────────────────────┘  └─────────────────────┘          │
│  ┌─────────────────────┐  ┌─────────────────────┐          │
│  │    RBAC 權限        │  │   cert-manager PKI  │          │
│  └─────────────────────┘  └─────────────────────┘          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                         slurm                                │
│  ┌─────────────────────┐  ┌─────────────────────┐          │
│  │   Controller CR     │  │    NodeSet CR(s)    │          │
│  └─────────────────────┘  └─────────────────────┘          │
│  ┌─────────────────────┐  ┌─────────────────────┐          │
│  │   Accounting CR     │  │    LoginSet CR      │          │
│  └─────────────────────┘  └─────────────────────┘          │
│  ┌─────────────────────┐  ┌─────────────────────┐          │
│  │    RestApi CR       │  │ Secrets/ConfigMaps  │          │
│  └─────────────────────┘  └─────────────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### 8.2 運行時元件

| 元件 | 類型 | 副本 | 端口 |
|------|------|------|------|
| slurm-operator | Deployment | 1 | 8080 (metrics), 8081 (health) |
| slurm-operator-webhook | Deployment | 1 | 9443 (webhook), 8081 (health) |
| slurmctld | StatefulSet | 1 | 6817 |
| slurmd | Pod (NodeSet) | N | 6818, 22 |
| slurmdbd | StatefulSet | 1 | 6819 |
| slurmrestd | Deployment | 1 | 6820 |
| login | Deployment | N | 22 |

## 9. 安全架構

### 9.1 認證機制

- **Slurm Auth**: `auth/slurm` 共享金鑰
- **JWT HS256**: JWT 令牌簽名
- **SSSD**: 使用者目錄服務整合

### 9.2 RBAC 權限

**Operator ClusterRole:**
- 完整 CRUD: CR、ConfigMap、Secret、Service、Pod、Deployment、StatefulSet
- 讀取: Node
- 更新: Pod/status、Node

**Webhook ClusterRole:**
- CR 驗證操作
- Pod/binding 修改

## 10. 監控和可觀測性

### 10.1 Prometheus 指標

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    endpoints:
      - path: /metrics/jobs
      - path: /metrics/nodes
      - path: /metrics/partitions
      - path: /metrics/scheduler
```

### 10.2 健康檢查

- `/healthz`: 存活探針
- `/readyz`: 就緒探針

## 11. 設計決策

| 決策 | 理由 |
|------|------|
| 使用 Kubebuilder v4 | 標準化 Operator 開發 |
| Distroless 基礎映像 | 最小化攻擊面 |
| Expectations 模式 | 避免競態條件 |
| Builder 模式 | 可維護的資源構建 |
| cert-manager 整合 | 自動化 TLS 管理 |
