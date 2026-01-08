# NodeSet Partition Deep Dive

本文件詳細分析 Slurm Operator 中 NodeSet Partition 的完整運作機制。

## 目錄

1. [slurm.conf Reconcile 觸發條件及完整流程](#1-slurmconf-reconcile-觸發條件及完整流程)
2. [完整的 Partition 建立流程](#2-完整的-partition-建立流程)
3. [slurmctld Reconfigure 機制](#3-slurmctld-reconfigure-機制)

---

## 1. slurm.conf Reconcile 觸發條件及完整流程

### 1.1 觸發條件

slurm.conf 的更新是由 **Controller Reconciler** 負責。以下事件會觸發 Controller Reconcile：

| 事件類型 | 觸發方式 | 程式碼位置 |
|----------|----------|------------|
| Controller CR 變更 | `.For(&Controller{})` | `controller_controller.go:109` |
| 擁有的 StatefulSet 變更 | `.Owns(&StatefulSet{})` | `controller_controller.go:110` |
| 擁有的 Service 變更 | `.Owns(&Service{})` | `controller_controller.go:111` |
| 擁有的 ConfigMap 變更 | `.Owns(&ConfigMap{})` | `controller_controller.go:112` |
| 擁有的 Secret 變更 | `.Owns(&Secret{})` | `controller_controller.go:113` |
| Accounting CR 變更 | `.Watches(&Accounting{})` | `controller_controller.go:114` |
| NodeSet CR 變更 | `.Watches(&NodeSet{})` | `controller_controller.go:115` |
| Secret 變更 (Auth) | `.Watches(&Secret{})` | `controller_controller.go:116` |

### 1.2 相關程式碼段落

#### SetupWithManager - Watch 設定

**檔案:** `internal/controller/controller/controller_controller.go:105-121`

```go
func (r *ControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        Named(ControllerName).
        For(&slinkyv1beta1.Controller{}).
        Owns(&appsv1.StatefulSet{}).
        Owns(&corev1.Service{}).
        Owns(&corev1.ConfigMap{}).
        Owns(&corev1.Secret{}).
        Watches(&slinkyv1beta1.Accounting{}, eventhandler.NewAccountingEventHandler(r.Client)).
        Watches(&slinkyv1beta1.NodeSet{}, eventhandler.NewNodeSetEventHandler(r.Client)).
        Watches(&corev1.Secret{}, eventhandler.NewSecretEventHandler(r.Client)).
        WithOptions(controller.Options{
            MaxConcurrentReconciles: maxConcurrentReconciles,
        }).
        Complete(r)
}
```

#### NodeSet Event Handler

**檔案:** `internal/controller/controller/eventhandler/eventhandler_nodeset.go:70-82`

```go
func (e *NodesetEventHandler) enqueueRequest(ctx context.Context, obj client.Object, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
    nodeset, ok := obj.(*slinkyv1beta1.NodeSet)
    if !ok {
        return
    }

    controller, err := e.refResolver.GetController(ctx, nodeset.Spec.ControllerRef)
    if err != nil {
        return
    }

    objectutils.EnqueueRequest(q, controller)
}
```

### 1.3 完整 Reconcile 流程

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. 觸發事件 (NodeSet/Controller/Accounting 變更)                │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 2. Event Handler 將 Controller 加入 Reconcile 隊列              │
│    [eventhandler_nodeset.go:70-82]                              │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 3. ControllerReconciler.Reconcile() 被呼叫                      │
│    [controller_controller.go:77-103]                            │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 4. Sync() 執行 syncSteps                                        │
│    [controller_sync.go:27-127]                                  │
│                                                                 │
│    syncSteps:                                                   │
│    ├─ "Service"      → BuildControllerService()                 │
│    ├─ "Config"       → BuildControllerConfig()  ← slurm.conf    │
│    ├─ "StatefulSet"  → BuildController()                        │
│    └─ "ServiceMonitor" → BuildControllerServiceMonitor()        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 5. BuildControllerConfig() 產生 slurm.conf                      │
│    [controller_config.go:29-152]                                │
│                                                                 │
│    - 讀取 Accounting 設定                                       │
│    - 讀取所有 NodeSets (GetNodeSetsForController)               │
│    - 讀取 ConfigFiles, Scripts                                  │
│    - 呼叫 buildSlurmConf() 產生內容                             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 6. SyncObject() 更新 ConfigMap                                  │
│    [objectutils/patch.go:23-202]                                │
│                                                                 │
│    - 比較新舊 ConfigMap                                         │
│    - 如果有變更則 Patch                                         │
└─────────────────────────────────────────────────────────────────┘
```

### 1.4 Config Sync Step 詳細程式碼

**檔案:** `internal/controller/controller/controller_sync.go:56-74`

```go
{
    Name: "Config",
    Sync: func(ctx context.Context, controller *slinkyv1beta1.Controller) error {
        var object *corev1.ConfigMap
        var err error
        if controller.Spec.External {
            object, err = r.builder.BuildControllerConfigExternal(controller)
        } else {
            object, err = r.builder.BuildControllerConfig(controller)
        }
        if err != nil {
            return fmt.Errorf("failed to build: %w", err)
        }
        if err := objectutils.SyncObject(r.Client, ctx, object, true); err != nil {
            return fmt.Errorf("failed to sync object (%s): %w", klog.KObj(object), err)
        }
        return nil
    },
},
```

---

## 2. 完整的 Partition 建立流程

### 2.1 重要概念

**Partition 不是獨立的 Kubernetes 資源**，它只是 slurm.conf 中的一行設定：

```
PartitionName=<name> Nodes=<name> [額外設定]
```

### 2.2 API 定義

**檔案:** `api/v1beta1/nodeset_types.go:114-124`

```go
// NodeSetPartition defines the Slurm partition configuration for the NodeSet.
type NodeSetPartition struct {
    // Enabled will create a partition for this NodeSet.
    // +default:=true
    Enabled bool `json:"enabled"`

    // Config is added to the NodeSet's partition line.
    // Ref: https://slurm.schedmd.com/slurmd.html#OPT_conf-%3Cnode-parameters%3E
    // +optional
    Config string `json:"config,omitzero"`
}
```

### 2.3 Partition 產生邏輯

**檔案:** `internal/builder/controller_config.go:252-279`

```go
if len(nodesetList.Items) > 0 {
    conf.AddProperty(config.NewPropertyRaw("#"))
    conf.AddProperty(config.NewPropertyRaw("### COMPUTE & PARTITION ###"))
}
for _, nodeset := range nodesetList.Items {
    name := nodeset.Name
    template := nodeset.Spec.Template.PodSpecWrapper
    if template.Hostname != "" {
        name = strings.Trim(template.Hostname, "-")
    }

    // 永遠產生 NodeSet 行
    nodesetLine := []string{
        fmt.Sprintf("NodeSet=%v", name),
        fmt.Sprintf("Feature=%v", name),
    }
    nodesetLineRendered := strings.Join(nodesetLine, " ")
    conf.AddProperty(config.NewPropertyRaw(nodesetLineRendered))

    // 檢查 partition.enabled
    partition := nodeset.Spec.Partition
    if !partition.Enabled {
        continue  // 跳過 Partition
    }

    // partition.enabled = true → 產生 Partition 行
    partitionLine := []string{
        fmt.Sprintf("PartitionName=%v", name),
        fmt.Sprintf("Nodes=%v", name),
        partition.Config,  // optional 額外設定
    }
    partitionLineRendered := strings.Join(partitionLine, " ")
    conf.AddProperty(config.NewPropertyRaw(partitionLineRendered))
}
```

### 2.4 完整流程圖

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. 使用者建立 NodeSet CR (partition.enabled = true)                         │
│                                                                             │
│    apiVersion: slinky.slurm.net/v1beta1                                     │
│    kind: NodeSet                                                            │
│    metadata:                                                                │
│      name: compute                                                          │
│    spec:                                                                    │
│      controllerRef:                                                         │
│        name: slurm                                                          │
│      partition:                                                             │
│        enabled: true                                                        │
│        config: "Default=YES MaxTime=24:00:00"                               │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ 2. NodesetEventHandler.Create() 被觸發                                      │
│    [eventhandler_nodeset.go:35-41]                                          │
│                                                                             │
│    func (e *NodesetEventHandler) Create(...) {                              │
│        e.enqueueRequest(ctx, evt.Object, q)                                 │
│    }                                                                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ 3. enqueueRequest() 找到 Controller 並排隊                                   │
│    [eventhandler_nodeset.go:70-82]                                          │
│                                                                             │
│    controller, _ := e.refResolver.GetController(ctx, nodeset.Spec.ControllerRef)│
│    objectutils.EnqueueRequest(q, controller)                                │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ 4. Controller Reconcile 執行 "Config" Step                                  │
│    [controller_sync.go:56-74]                                               │
│                                                                             │
│    object, _ := r.builder.BuildControllerConfig(controller)                 │
│    objectutils.SyncObject(r.Client, ctx, object, true)                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ 5. BuildControllerConfig() 讀取所有 NodeSets                                 │
│    [controller_config.go:39]                                                │
│                                                                             │
│    nodesetList, _ := b.refResolver.GetNodeSetsForController(ctx, controller)│
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ 6. buildSlurmConf() 產生 NodeSet 和 Partition 行                             │
│    [controller_config.go:252-279]                                           │
│                                                                             │
│    for _, nodeset := range nodesetList.Items {                              │
│        // 產生 NodeSet 行                                                   │
│        conf.AddProperty("NodeSet=compute Feature=compute")                  │
│                                                                             │
│        if !partition.Enabled { continue }                                   │
│                                                                             │
│        // 產生 Partition 行                                                 │
│        conf.AddProperty("PartitionName=compute Nodes=compute Default=YES...") │
│    }                                                                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ 7. ConfigMap 被更新                                                          │
│    [objectutils/patch.go:77-87]                                             │
│                                                                             │
│    obj.Data = o.Data                                                        │
│    c.Patch(ctx, oldObj, patch)                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────────────┐
│ 8. slurm.conf 內容                                                          │
│                                                                             │
│    ### COMPUTE & PARTITION ###                                              │
│    NodeSet=compute Feature=compute                                          │
│    PartitionName=compute Nodes=compute Default=YES MaxTime=24:00:00         │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.5 產生的 slurm.conf 範例

| `partition.enabled` | `partition.config` | slurm.conf 內容 |
|---------------------|-------------------|-----------------|
| `true` | `""` (空) | `PartitionName=foo Nodes=foo ` |
| `true` | `"Default=YES"` | `PartitionName=foo Nodes=foo Default=YES` |
| `true` | `"MaxTime=24:00:00 Default=YES"` | `PartitionName=foo Nodes=foo MaxTime=24:00:00 Default=YES` |
| `false` | (任何值) | (不產生 PartitionName 行) |

---

## 3. slurmctld Reconfigure 機制

### 3.1 Reconfigure Sidecar 工作原理

Controller Pod 中有一個 `reconfigure` sidecar container，負責監控 slurm.conf 變更並觸發 slurmctld 重新載入。

**檔案:** `internal/builder/scripts/reconfigure.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

SLURM_DIR="/etc/slurm"
INTERVAL="5"

function getHash() {
    echo "$(find "$SLURM_DIR" -type f -exec sha256sum {} \; | sort -k2 | sha256sum)"
}

function reconfigure() {
    echo "[$(date)] Reconfiguring Slurm..."
    until scontrol reconfigure; do
        echo "[$(date)] Failed to reconfigure, try again..."
        sleep 2
    done
    echo "[$(date)] SUCCESS"
}

function main() {
    local lastHash=""
    local newHash=""

    echo "[$(date)] Start '$SLURM_DIR' polling"
    while true; do
        newHash="$(getHash)"
        if [ "$newHash" != "$lastHash" ]; then
            reconfigure
            lastHash="$newHash"
        fi
        sleep "$INTERVAL"
    done
}
main
```

### 3.2 Reconfigure Container 定義

**檔案:** `internal/builder/controller_app.go:297-312`

```go
func (b *Builder) reconfigureContainer(container slinkyv1beta1.ContainerWrapper) corev1.Container {
    opts := ContainerOpts{
        base: corev1.Container{
            Name: "reconfigure",
            Command: []string{
                "tini",
                "-g",
                "--",
                "bash",
                "-c",
                reconfigureScript,
            },
            RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
            VolumeMounts: []corev1.VolumeMount{
                {Name: slurmEtcVolume, MountPath: slurmEtcDir, ReadOnly: true},
                {Name: slurmAuthSocketVolume, MountPath: slurmctldAuthSocketDir, ReadOnly: true},
            },
        },
        // ...
    }
    // ...
}
```

### 3.3 完整 Reconfigure 流程

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. ConfigMap 被更新 (slurm.conf)                                │
│    [由 Controller Reconciler 觸發]                              │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 2. Kubernetes 將 ConfigMap 同步到 Pod Volume                    │
│    [kubelet sync period, 可能需要 1-2 分鐘]                     │
│                                                                 │
│    /etc/slurm/slurm.conf 被更新                                 │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 3. Reconfigure Sidecar 偵測到變更                               │
│    [每 5 秒檢查一次 /etc/slurm 的 hash]                         │
│                                                                 │
│    newHash="$(getHash)"                                         │
│    if [ "$newHash" != "$lastHash" ]; then                       │
│        reconfigure                                              │
│    fi                                                           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 4. 執行 scontrol reconfigure                                    │
│                                                                 │
│    until scontrol reconfigure; do                               │
│        sleep 2                                                  │
│    done                                                         │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 5. slurmctld 重新載入設定                                       │
│                                                                 │
│    - 新的 NodeSet 定義生效                                      │
│    - 新的 Partition 定義生效                                    │
│    - 其他設定變更生效                                           │
└─────────────────────────────────────────────────────────────────┘
```

### 3.4 ConfigMap Propagation 延遲

Kubernetes ConfigMap 更新到 Pod 的延遲取決於：

| 因素 | 說明 |
|------|------|
| kubelet sync period | 預設 1 分鐘 |
| ConfigMap cache TTL | 預設與 sync period 相同 |
| 掛載方式 | subPath 掛載不會自動更新 |

**預設情況下，ConfigMap 更新可能需要 1-2 分鐘才會同步到 Pod。**

### 3.5 診斷指令

```bash
# 1. 檢查 reconfigure sidecar 的 logs
kubectl logs <controller-pod> -c reconfigure -f

# 2. 檢查 Pod 內的 slurm.conf 是否已更新
kubectl exec <controller-pod> -c slurmctld -- cat /etc/slurm/slurm.conf | grep -E "NodeSet|Partition"

# 3. 檢查 ConfigMap 的內容
kubectl get configmap <controller>-config -o jsonpath='{.data.slurm\.conf}' | grep -E "NodeSet|Partition"

# 4. 手動觸發 reconfigure
kubectl exec <controller-pod> -c slurmctld -- scontrol reconfigure

# 5. 檢查 slurmctld 是否已載入新的 partition
kubectl exec <controller-pod> -c slurmctld -- scontrol show partition
```

### 3.6 強制立即同步

如果需要立即同步，可以：

```bash
# 方法 1: 刪除 Pod 讓它重建（會載入最新的 ConfigMap）
kubectl delete pod <controller-pod>

# 方法 2: 等待 reconfigure sidecar 自動偵測（最多 1-2 分鐘）
kubectl logs <controller-pod> -c reconfigure -f
```

---

## 附錄：相關檔案索引

| 檔案 | 說明 |
|------|------|
| `api/v1beta1/nodeset_types.go` | NodeSet API 定義，包含 Partition 結構 |
| `api/v1beta1/controller_types.go` | Controller API 定義 |
| `internal/controller/controller/controller_controller.go` | Controller Reconciler 主程式 |
| `internal/controller/controller/controller_sync.go` | Controller Sync 步驟 |
| `internal/controller/controller/eventhandler/eventhandler_nodeset.go` | NodeSet Event Handler |
| `internal/builder/controller_config.go` | slurm.conf 產生邏輯 |
| `internal/builder/controller_app.go` | Controller Pod 定義，包含 reconfigure sidecar |
| `internal/builder/scripts/reconfigure.sh` | Reconfigure 腳本 |
| `internal/utils/objectutils/patch.go` | SyncObject 函數 |
