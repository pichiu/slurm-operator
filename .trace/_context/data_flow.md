# Data Flow — slurm-operator

## 代表性 Use Case：NodeSet Pod 被 Kubernetes Node 驅逐時的處理流程

這是 slurm-operator 最核心的 data flow，展示 Kubernetes API 事件如何驅動 Slurm API 操作。

---

## 完整流程追蹤

### 1. 事件觸發

```
Kubernetes Node 被 cordon（kubectl cordon <node>）
 └── Node.Spec.Unschedulable = true
      └── 觸發 NodeEventHandler（internal/controller/nodeset/eventhandler/eventhandler_node.go）
           └── 將受影響的 NodeSet 加入 reconcile queue
```

**事件處理器注册**（`nodeset_controller.go:SetupWithManager`）：
```go
.Watches(&corev1.Node{}, eventhandler.NewNodeEventHandler(r.Client))
```

### 2. Reconcile 啟動

```go
// nodeset_controller.go:Reconcile()
func (r *NodeSetReconciler) Reconcile(ctx, req) (res ctrl.Result, retErr error)
 ├── r.Sync(ctx, req)          # 委派給 Sync()
 └── durationStore.Pop(req)    # 取得 RequeueAfter（固定 30 秒）
```

### 3. Sync() 主控制邏輯

```go
// nodeset_sync.go:Sync()
1. r.Get(ctx, nodeset)           # 取得最新 NodeSet CR
2. defaults.SetNodeSetDefaults() # 套用預設值
3. r.adoptOrphanRevisions()      # 收養孤兒 ControllerRevision
4. r.getNodeSetRevisions()       # 取得 current/update revision hash
5. r.getNodeSetPods()            # 列出屬於此 NodeSet 的所有 Pod
6. r.expectations.SatisfiedExpectations() # 確認期望狀態已達成
7. r.sync()                      # 主要同步邏輯（含步驟清單）
8. r.syncUpdate()                # Rolling update 邏輯
9. r.truncateHistory()           # 清理舊 ControllerRevision
10. r.syncStatus()               # 更新 NodeSet.Status
```

### 4. sync() 步驟管線

`sync()` 透過 `syncsteps.Sync()` 按順序執行（`internal/syncsteps/syncsteps.go`）：

```
Step 1: ClusterWorkerService      # 確保 headless Service 存在
Step 2: ClusterWorkerPDB          # 確保 PodDisruptionBudget 存在
Step 3: SSHConfig                 # 同步 SSH 相關 ConfigMap/Secret
Step 4: NodeTaint                 # 管理 Kube Node 的 NoExecute taint（deprecated）
Step 5: RefreshNodeCache          # ⚠️ StopOnError=true：強制刷新 Slurm client cache
Step 6: SlurmDeadline             # 從 Slurm job 計算並標注 pod deadline annotation
Step 7: Cordon                    # 同步 Kube node cordon 狀態到 Slurm node drain
Step 8: NodeSetPods               # 管理 Pod 數量（scale up/down, DaemonSet placement）
Step 9: SlurmNodeRecords          # 清理無效的 Slurm node 記錄
Step 10: SlurmNodes               # 刪除已 ready 但 Slurm 中未註冊的 Pod
Step 11: SlurmTopology            # 同步 Kube node 的拓撲資訊到 Slurm node
```

### 5. syncCordon()（節點 cordon 處理）

```go
// nodeset_sync.go:syncCordon()
for each pod:
 ├── 取得 pod 所在的 Kube Node
 ├── node.Spec.Unschedulable == true ？ → Kube node 被 cordon
 ├── !ourReason || slurmNodeIsUnresponsive ？ → 外部設定，不干預
 │
 └── 本 operator 管理的情況：
      ├── nodeIsCordoned → makePodCordonAndDrain(pod, reason)
      │   ├── 套用 cordon annotation 到 pod
      │   └── slurmControl.MakeNodeDrain(nodeset, pod, reason)
      │       └── slurmClient.Update(slurmNode, {State: DRAIN, Reason: "slurm-operator: ..."})
      ├── podIsCordoned → slurmControl.MakeNodeDrain(...)
      └── !podIsCordoned → slurmControl.MakeNodeUndrain(...)
```

drain reason 格式：`"slurm-operator: <reason>"`（`nodeReasonPrefix` 常數）

### 6. SlurmControl → Slurm REST API

```go
// internal/controller/nodeset/slurmcontrol/slurmcontrol.go:MakeNodeDrain()
1. r.lookupClient(nodeset)           # 從 clientMap 取得 Slurm client
   └── clientMap.Get(controllerRef)  # thread-safe map
2. slurmClient.Get(slurmNode)        # GET /slurm/v0044/node/{name}
3. 檢查是否已 drain（避免重複操作）
4. slurmClient.Update(slurmNode, {State: DRAIN, Reason: ...})
   └── PUT /slurm/v0044/node/{name}
```

### 7. Slurm Client 認證流程（slurmclient controller）

```go
// internal/controller/slurmclient/slurmclient_sync.go:Sync()
1. 取得 Controller CR
2. r.getRestApiServer() → 找到對應的 RestApi CR
   └── server = "http://{restapi.ServiceFQDN}:6820"
3. r.refResolver.GetSecretKeyRef(controller.AuthJwtRef())  # 取得 JWT signing key
4. slurmjwt.NewToken(signingKey).WithLifetime(15min).NewSignedToken()  # 產生 JWT
5. 設定 refresh 時間（= 15min * 4/5 = 12min）
6. slurmclient.NewClient({Server: server, AuthToken: jwt})
7. clientMap.Add(controllerKey, slurmClient)     # 儲存 client
```

### 8. syncStatus()：同步回 Kubernetes

```go
// nodeset_sync_status.go
1. slurmControl.CalculateNodeStatus()    # 查詢所有 Slurm 節點狀態
2. 更新每個 Pod 的 PodConditions（Idle/Allocated/Down/Drain 等）
3. 更新 NodeSet.Status：
   - Replicas, UpdatedReplicas, ReadyReplicas
   - SlurmIdle, SlurmAllocated, SlurmDown, SlurmDrain
   - Conditions（Available, Progressing 等）
```

---

## 資料轉換層次

```
Kubernetes Event（Node/Pod 變化）
    │ EventHandler
    ▼
Reconcile Request（namespace/name）
    │
    ▼
NodeSet CR（Desired State）
    │ Sync() Pipeline
    ▼
[Step 7: syncCordon]
Kubernetes Node State（cordon/uncordon）
    │ slurmControl.MakeNodeDrain/Undrain
    ▼
Slurm REST API（PUT /node/{name}）
    │
    ▼
Slurm node State（DRAIN/UNDRAIN）
    │ CalculateNodeStatus
    ▼
Pod Conditions + NodeSet.Status（回寫 Kubernetes）
```

---

## Controller Sync Flow（slurmctld 管理）

Controller CR reconcile（`internal/controller/controller/controller_sync.go`）：

```
Step 1: Service     → 建立/更新 slurmctld 的 Kubernetes Service
Step 2: Config      → 建立/更新 ConfigMap（slurm.conf 等設定）
         external=true → 使用 externalConfig 建立 external ConfigMap
         external=false → 產生完整 slurm.conf
Step 3: StatefulSet → 建立/更新 slurmctld StatefulSet（含 slurmctld/reconfigure/logfile containers）
Step 4: ServiceMonitor → 建立/刪除 Prometheus ServiceMonitor
→ syncStatus()
```

---

## Token Controller Sync Flow（JWT 管理）

```go
// internal/controller/token/token_sync.go
1. 取得 Token CR
2. slurmjwt.ParseTokenClaims() → 檢查現有 Token 是否過期
3. 若未過期且未設定 refresh → 直接回傳（immutable secret）
4. slurmjwt.NewToken(signingKey).WithUsername(username).WithLifetime().NewSignedToken()
5. 建立/更新 Kubernetes Secret（含 JWT token value）
```

---

## Pod Binding Webhook（拓撲注入）

```
Pod 被 Kubernetes Scheduler 排程到 Node
    │ POST /mutate--v1-binding
    ▼
PodBindingWebhook.Default(ctx, binding)
    │
    ├── 取得 Pod（確認是 worker pod，labels[app]=worker）
    ├── 取得 Node（binding.Target.Name）
    └── node.Annotations[AnnotationNodeTopologySpec]
         └── patch pod.Annotations[AnnotationNodeTopologySpec] = topologySpec
              │
              └── NodeSet controller 的 syncSlurmTopology() 讀取此 annotation
                   └── slurmControl.UpdateNodeTopology(pod, topologySpec)
                        └── PUT /slurm/v0044/node/{name} {topology: topologySpec}
```
