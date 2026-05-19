# Core Logic — slurm-operator

## 核心抽象：雙 API 調和（Dual-API Reconciliation）

slurm-operator 的「心臟」是 **NodeSet controller**，其獨特之處在於需要同時調和兩個 API 的狀態：
1. **Kubernetes API** — Pod/Node/Service 狀態
2. **Slurm REST API** — slurm node 狀態（IDLE/ALLOCATED/DRAIN/DOWN 等）

---

## 核心設計模式

### 1. Pipeline Pattern（SyncSteps）

`internal/syncsteps/syncsteps.go` 定義了一個泛型 pipeline：

```go
type Step[T client.Object] struct {
    Name        string
    SyncFn      func(context.Context, T) error
    StopOnError bool   // 失敗時是否停止後續步驟
}

func Sync[T client.Object](ctx, recorder, obj, steps []Step[T]) error
```

所有 controller 都用此模式組織 sync 步驟，各步驟獨立失敗，只有 `StopOnError=true` 的步驟失敗時才停止。

NodeSet 的 sync pipeline（`nodeset_sync.go:sync()`）共 11 步：
```
ClusterWorkerService → ClusterWorkerPDB → SSHConfig → NodeTaint
    → RefreshNodeCache(StopOnError) → SlurmDeadline → Cordon
    → NodeSetPods → SlurmNodeRecords → SlurmNodes → SlurmTopology
```

### 2. Interface Segregation（Strategy Pattern）

SlurmControl 以 Interface 定義（`slurmcontrol/slurmcontrol.go`）：

```go
type SlurmControlInterface interface {
    RefreshNodeCache(ctx, nodeset)
    UpdateNodeWithPodInfo(ctx, nodeset, pod)
    UpdateNodeTopology(ctx, nodeset, pod, topologySpec)
    MakeNodeDrain(ctx, nodeset, pod, reason)
    MakeNodeUndrain(ctx, nodeset, pod, reason)
    IsNodeDrain(ctx, nodeset, pod)
    IsNodeDrained(ctx, nodeset, pod)
    IsNodeDownForUnresponsive(ctx, nodeset, pod)
    IsNodeReasonOurs(ctx, nodeset, pod)
    CalculateNodeStatus(ctx, nodeset, pods)
    GetNodeDeadlines(ctx, nodeset, pods)
    GetNodesForPods(ctx, nodeset, pods)
    GetDefunctNodesForNodeSet(ctx, nodeset)
    DeleteNode(ctx, nodeset, nodeName)
}
```

`realSlurmControl` 是預設實作，測試時可替換 mock。

### 3. Thread-Safe Client Map

`internal/clientmap/clientmap.go` — 用 `sync.RWMutex` 保護的 Slurm REST API client map：

```go
type ClientMap struct {
    lock    sync.RWMutex
    clients map[string]client.Client  // key: "namespace/name"
}
```

每個 `Controller` CR 對應一個 Slurm client。`Add()` 時自動呼叫 `client.Start(ctx)` goroutine。

### 4. DurationStore（延遲重排模式）

`internal/utils/durationstore/durationstore.go` — 為 reconcile 提供動態 RequeueAfter 機制：

```go
// NodeSet controller 固定排入 30 秒後重排
durationStore.Push(key, 30*time.Second)

// SlurmClient controller 在 JWT 快過期前 12 分鐘重排
durationStore.Push(controllerKey.String(), refresh)  // 12min
```

### 5. ControllerExpectations（Kubernetes 上游模式）

使用 Kubernetes 原生的 `UIDTrackingControllerExpectations` 機制，確保 Pod 的建立/刪除操作在 informer 確認前不重複執行：

```go
r.expectations.ExpectDeletions(logger, key, podKeys)
r.expectations.RaiseExpectations(logger, key, numCreate, 0)
// → 後續 Reconcile 會先檢查 SatisfiedExpectations()
```

### 6. SlowStart Batch（指數退避批次操作）

Pod 建立/刪除採用指數增長批次大小（1, 2, 4, 8...，最多 250）：

```go
// nodeset_sync.go:doPodScale()
utils.SlowStartBatch(numCreate, SlowStartInitialBatchSize, createPodFn)
```

避免大規模 Pod 建立時（如整個叢集重啟）對 API server 造成衝擊。

---

## NodeSet ScalingMode 邏輯

`syncNodeSetPods()` 根據 `scalingMode` 分兩條路徑：

### StatefulSet Mode（預設）

```go
replicaCount = nodeset.Spec.Replicas
diff = len(currentPods) - replicaCount

if diff < 0:  # 需要建立新 Pod
    // 找未使用的 ordinal，建立 newNodeSetPodOrdinal()
    // Pod name: {nodeset-name}-{ordinal-with-padding}
    doPodScale(podsToKeep=current, podsToDelete=nil, podsToCreate=new)

if diff > 0:  # 需要刪除多餘 Pod
    // SplitActivePods() 選出最末尾的 pods
    doPodScale(podsToKeep=trimmed, podsToDelete=excess, podsToCreate=nil)
```

### DaemonSet Mode

```go
// 從 Kube Node list 計算哪些 node 需要/不需要 pod
nodeToDaemonPods = getNodesToDaemonPods(pods)
for each kube node:
    podsShouldBeOnNode() → nodesNeedingDaemonPods + podsToDelete
doPodScale(...)
```

`NodeShouldRunDaemonPod()` 使用 Kubernetes 內部的 daemon utils 邏輯（直接引用 `k8s.io/kubernetes/pkg/controller/daemon/util`）。

---

## Pod 刪除（processCondemned）的 Slurm-Aware 邏輯

`processCondemned()` 在刪除 pod 前必須確保 Slurm job 安全：

```go
// nodeset_sync.go:processCondemned()
1. cordon pod（pod annotation）
2. slurmControl.MakeNodeDrain(reason)     # Slurm 標為 DRAIN
3. slurmControl.IsNodeDrained()           # 等待 job 完成（DRAIN + 無 job）
   │
   ├── isDrained = true → 刪除 Pod
   └── isDrained = false → requeueAfter = min(jobDeadline, 30s)
                           等待 job 完成後再 reconcile
```

這確保不會中斷正在執行的 Slurm job（workload protection）。

---

## WorkloadDisruptionProtection（PDB）

當 `workloadDisruptionProtection=true`（預設），operator 會建立動態的 PodDisruptionBudget：

- `maxUnavailable` = 目前正在執行 Slurm job 的 Pod 數
- 隨著 Slurm job 完成，PDB 的 `maxUnavailable` 動態調整
- 防止 Kubernetes 在 Slurm job 執行期間強制驅逐 Pod

---

## PinToNode（StatefulSet 模式）

```go
// NodeSetSpec.PinToNode = true
// NodeSet.Status.OrdinalToNode: map[ordinal] = kubeNodeName
```

Pod 一旦排程到某 Kube node，會記錄到 `status.ordinalToNode`。後續 reconcile 使用 `nodeAffinity` 讓 pod 固定在同一個 Kube node，避免 Slurm 重新排程。

重設條件：
- Kube node 不存在
- Pod template 不再匹配固定的 node

---

## JWT Token 管理（Token controller）

`internal/controller/token/slurmjwt/token.go`：

```go
// JWT claims 結構
type Claims struct {
    jwt.RegisteredClaims
    SunStr  string `json:"sun"`   // Slurm Username
    SunLong int64  `json:"sl_ung"` // ⚠️ 未驗證欄位名稱
}
```

Token controller 負責：
1. 從 `JwtKeyRef` 取得 signing key（支援 PEM/JWKS 格式）
2. 產生 Slurm JWT（`sun` claim = username）
3. 寫入 Kubernetes Secret（供其他元件使用）
4. 在 `Token.Spec.Refresh=true` 時自動輪換

SlurmClient controller 也會獨立產生自己的 JWT（用於 REST API 認證）。

---

## Topology 注入機制（PodBindingWebhook）

**唯一的 Mutating Webhook**，在 Pod 被排程時注入拓撲資訊：

```
Kube Scheduler 決定 Pod → Node 的排程
    │
    └── POST /mutate--v1-binding
         │
         └── PodBindingWebhook.Default()
              ├── 取得 Node.Annotations[AnnotationNodeTopologySpec]
              └── Patch Pod.Annotations[AnnotationNodeTopologySpec] = topologySpec
                   └── NodeSet sync 讀取此 annotation
                        └── slurmrestd PUT /node/{name} {topology: topologySpec}
```

這讓 Slurm 能進行拓撲感知排程（如機架感知），而不需要在 `topology.yaml` 中預先列出所有節點。

---

## 外部 Slurm 元件支援（Hybrid）

`Controller.Spec.External=true` 時，Controller CR 代表一個**不在 Kubernetes 中**的 slurmctld：

```go
// controller_sync.go:Sync()
if controller.Spec.External:
    skip Service step
    build external ConfigMap（只有連線資訊，不含 slurm.conf）
    skip StatefulSet step
```

`ExternalConfig` 欄位描述如何連線到外部 slurmctld（host/port），operator 仍會管理 NodeSet pod 並透過 Slurm REST API 與外部叢集溝通。

---

## Slurm node 狀態對映 Kubernetes PodCondition

`pkg/conditions/conditions.go` 定義 Slurm node 狀態到 PodCondition 的對映：

| Slurm State | PodConditionType |
|-------------|-----------------|
| ALLOCATED | `slinky.slurm.net/Allocated` |
| DOWN | `slinky.slurm.net/Down` |
| IDLE | `slinky.slurm.net/Idle` |
| MIXED | `slinky.slurm.net/Mixed` |
| DRAIN | `slinky.slurm.net/Drain` |
| COMPLETING | `slinky.slurm.net/Completing` |
| NOT_RESPONDING | `slinky.slurm.net/NotResponding` |
| ... | ... |

這讓 Kubernetes HPA 或其他工具可透過 Pod conditions 感知 Slurm 工作負載狀態。
