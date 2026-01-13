# KEDA Scale-to-Zero 深度分析：多 NodeSet 情境

> 產生日期：2026-01-13 | 版本：v1.0

## 目錄

- [概述](#概述)
- [核心問題](#核心問題)
- [架構解析](#架構解析)
- [Scale-to-Zero 機制詳解](#scale-to-zero-機制詳解)
- [多 NodeSet 場景分析](#多-nodeset-場景分析)
- [Job 進來時的完整流程](#job-進來時的完整流程)
- [程式碼佐證](#程式碼佐證)
- [潛在問題與限制](#潛在問題與限制)
- [最佳實踐建議](#最佳實踐建議)

---

## 概述

本文深入分析當 KEDA 將 Partition 中的多個 NodeSet 都 scale 到 0 時，有新 Job 進來會發生什麼事。

### 關鍵發現

| 情境 | 結果 |
|------|------|
| 所有 NodeSet = 0 replicas | Partition 存在但無可用節點，Job 進入 PENDING |
| Job 提交後 | Prometheus 記錄 pending jobs，KEDA 檢測到後觸發 scale-up |
| Scale-up 延遲 | 約 1-3 分鐘（取決於 Prometheus scrape interval + KEDA polling + Pod 啟動時間）|
| 多 NodeSet 同時 scale-up | **可能發生**，取決於 ScaledObject 配置 |

---

## 核心問題

當一個 Partition 包含多個 NodeSet，且全部都 scale 到 0 時：

```
┌─────────────────────────────────────────────────────────────┐
│                    Partition: compute                        │
├─────────────────────────────────────────────────────────────┤
│  NodeSet: compute-cpu      │  replicas: 0 (scaled to zero)  │
│  NodeSet: compute-gpu      │  replicas: 0 (scaled to zero)  │
│  NodeSet: compute-highmem  │  replicas: 0 (scaled to zero)  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    sinfo -p compute
                    PARTITION  NODES  STATE
                    compute    0      n/a     ← 無任何可用節點
```

**問題**：當使用者提交 `sbatch -p compute job.sh` 時會發生什麼？

---

## 架構解析

### 完整系統架構

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                               │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐        │
│  │  ScaledObject   │     │  ScaledObject   │     │  ScaledObject   │        │
│  │  (compute-cpu)  │     │  (compute-gpu)  │     │ (compute-highmem)│        │
│  └────────┬────────┘     └────────┬────────┘     └────────┬────────┘        │
│           │                       │                       │                  │
│           └───────────────────────┼───────────────────────┘                  │
│                                   │                                          │
│                                   ▼                                          │
│                        ┌─────────────────────┐                               │
│                        │    KEDA Operator    │                               │
│                        │  (polls every 30s)  │                               │
│                        └──────────┬──────────┘                               │
│                                   │                                          │
│                    ┌──────────────┴──────────────┐                           │
│                    │     Prometheus Query         │                           │
│                    │  slurm_partition_jobs_       │                           │
│                    │  pending{partition="compute"}│                           │
│                    └──────────────┬──────────────┘                           │
│                                   │                                          │
│                                   ▼                                          │
│                        ┌─────────────────────┐                               │
│                        │     Prometheus      │                               │
│                        │  (scrapes metrics)  │                               │
│                        └──────────┬──────────┘                               │
│                                   │                                          │
│                    ┌──────────────┴──────────────┐                           │
│                    │      ServiceMonitor         │                           │
│                    │  /metrics/jobs              │                           │
│                    │  /metrics/partitions        │                           │
│                    └──────────────┬──────────────┘                           │
│                                   │                                          │
│                                   ▼                                          │
│              ┌────────────────────────────────────────┐                      │
│              │          Controller Pod                │                      │
│              │  ┌─────────────┐  ┌─────────────────┐ │                      │
│              │  │  slurmctld  │  │   slurmrestd    │ │                      │
│              │  │             │  │  (port 6820)    │ │                      │
│              │  └─────────────┘  └─────────────────┘ │                      │
│              └────────────────────────────────────────┘                      │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Metrics 來源

**檔案**：`internal/builder/controller_servicemonitor.go:60-66`

```go
defaultEndpoints := []monitoringv1.Endpoint{
    newMetricsEndpoint("/metrics/jobs"),       // ← Job metrics 包含 pending count
    newMetricsEndpoint("/metrics/nodes"),
    newMetricsEndpoint("/metrics/partitions"), // ← Partition metrics
    newMetricsEndpoint("/metrics/scheduler"),
}
```

Prometheus 會定期抓取這些 endpoints，其中 `/metrics/jobs` 提供類似以下的 metrics：

```prometheus
# HELP slurm_partition_jobs_pending Number of pending jobs per partition
# TYPE slurm_partition_jobs_pending gauge
slurm_partition_jobs_pending{partition="compute"} 5
slurm_partition_jobs_pending{partition="debug"} 0
```

---

## Scale-to-Zero 機制詳解

### KEDA ScaledObject 配置

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: scale-compute-cpu
  namespace: slurm
spec:
  scaleTargetRef:
    apiVersion: slinky.slurm.net/v1beta1
    kind: NodeSet
    name: compute-cpu
  idleReplicaCount: 0      # ← 必須為 0 才能 scale-from-zero
  minReplicaCount: 1       # ← scale-up 時的最小值
  maxReplicaCount: 10
  cooldownPeriod: 300      # ← 5 分鐘無活動後 scale 到 0
  triggers:
    - type: prometheus
      metricType: Value
      metadata:
        serverAddress: http://prometheus:9090
        query: slurm_partition_jobs_pending{partition="compute"}
        threshold: '1'     # ← 有任何 pending job 就觸發
```

### Scale-to-Zero 流程

```
Time 0:00 - 所有 Jobs 完成
     │
     ▼
Time 0:00~5:00 - Cooldown Period (預設 300 秒)
     │
     │  KEDA 持續檢查：
     │  - slurm_partition_jobs_pending = 0
     │  - 無活動持續 5 分鐘
     │
     ▼
Time 5:00 - KEDA 觸發 Scale-to-Zero
     │
     │  KEDA → NodeSet.spec.replicas = 0
     │
     ▼
Time 5:01 - NodeSet Controller 處理
     │
     │  檔案：nodeset_sync.go:542-568
     │  syncNodeSet() 檢測到：
     │    len(pods) > nodeset.Spec.Replicas (0)
     │
     ▼
Time 5:02 - doPodScaleIn() 執行
     │
     │  1. 標記 pods 為 cordon
     │  2. Drain Slurm nodes
     │  3. 等待 jobs 完成
     │  4. 刪除 pods
     │
     ▼
Time 5:30~6:00 - 所有 Pods 已刪除
     │
     │  sinfo -p compute
     │  PARTITION  NODES  STATE
     │  compute    0      n/a
     │
     ▼
     此時 Partition 仍存在於 slurm.conf
     但無任何可用節點
```

### NodeSet Controller Scale-In 程式碼

**檔案**：`internal/controller/nodeset/nodeset_sync.go:542-568`

```go
// syncNodeSet will reconcile NodeSet pod replica counts.
// Pods will be:
//   - Scaled out when: `replicaCount < replicasWant"
//   - Scaled in when: `replicaCount > replicasWant"
func (r *NodeSetReconciler) syncNodeSet(
    ctx context.Context,
    nodeset *slinkyv1beta1.NodeSet,
    pods []*corev1.Pod,
    hash string,
) error {
    logger := log.FromContext(ctx)

    // 取得目標 replica 數量（KEDA 設定的 0）
    replicaCount := int(ptr.Deref(nodeset.Spec.Replicas, 0))

    // 計算差異
    diff := len(pods) - replicaCount

    if diff < 0 {
        // 需要 scale-out（建立更多 pods）
        diff = -diff
        logger.V(2).Info("Too few NodeSet pods", "need", replicaCount, "creating", diff)
        return r.doPodScaleOut(ctx, nodeset, pods, diff, hash)
    }

    if diff > 0 {
        // 需要 scale-in（刪除 pods）← 當 replicas=0 時走這條路
        logger.V(2).Info("Too many NodeSet pods", "need", replicaCount, "deleting", diff)
        podsToDelete, podsToKeep := nodesetutils.SplitActivePods(pods, diff)
        return r.doPodScaleIn(ctx, nodeset, podsToDelete, podsToKeep)
    }

    // 已達到目標數量
    logger.V(2).Info("Processing NodeSet pods", "replicas", replicaCount)
    return r.doPodProcessing(ctx, nodeset, pods, hash)
}
```

---

## 多 NodeSet 場景分析

### 場景設定

假設有一個 `compute` partition 包含三個 NodeSet：

```yaml
# NodeSet 1: compute-cpu
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: compute-cpu
spec:
  replicas: 0  # 被 KEDA scale 到 0
  partition:
    enabled: true
    config: "State=UP"

# NodeSet 2: compute-gpu
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: compute-gpu
spec:
  replicas: 0  # 被 KEDA scale 到 0
  partition:
    enabled: true
    config: "State=UP"

# NodeSet 3: compute-highmem
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: compute-highmem
spec:
  replicas: 0  # 被 KEDA scale 到 0
  partition:
    enabled: true
    config: "State=UP"
```

### 生成的 slurm.conf

**檔案**：`internal/builder/controller_config.go:252-279`

```go
for _, nodeset := range nodesetList.Items {
    name := nodeset.Name

    // 生成 NodeSet 行
    nodesetLine := []string{
        fmt.Sprintf("NodeSet=%v", name),
        fmt.Sprintf("Feature=%v", name),
    }
    conf.AddProperty(config.NewPropertyRaw(nodesetLineRendered))

    // 生成 Partition 行
    partition := nodeset.Spec.Partition
    if !partition.Enabled {
        continue
    }
    partitionLine := []string{
        fmt.Sprintf("PartitionName=%v", name),
        fmt.Sprintf("Nodes=%v", name),
        partition.Config,
    }
    conf.AddProperty(config.NewPropertyRaw(partitionLineRendered))
}
```

**生成結果**：

```conf
# slurm.conf (自動生成)

### COMPUTE & PARTITION ###
NodeSet=compute-cpu Feature=compute-cpu
PartitionName=compute-cpu Nodes=compute-cpu State=UP

NodeSet=compute-gpu Feature=compute-gpu
PartitionName=compute-gpu Nodes=compute-gpu State=UP

NodeSet=compute-highmem Feature=compute-highmem
PartitionName=compute-highmem Nodes=compute-highmem State=UP
```

> ⚠️ **重要發現**：在 slurm-operator 中，每個 NodeSet 預設會建立**自己的 Partition**（1:1 關係），而不是多個 NodeSet 共享一個 Partition。

### 如果要多個 NodeSet 共享同一個 Partition

需要使用 Helm Chart 的 `partitions` 區塊：

```yaml
# values.yaml
nodesets:
  compute-cpu:
    partition:
      enabled: false  # 關閉自動建立
  compute-gpu:
    partition:
      enabled: false
  compute-highmem:
    partition:
      enabled: false

partitions:
  compute:
    enabled: true
    nodesets:
      - compute-cpu
      - compute-gpu
      - compute-highmem
    configMap:
      State: UP
      Default: "YES"
```

---

## Job 進來時的完整流程

### 時序圖

```
┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────┐ ┌────────────┐ ┌─────────┐
│   User   │ │ slurmctld│ │slurmrestd │ │Prom. │ │    KEDA    │ │ NodeSet │
└────┬─────┘ └────┬─────┘ └─────┬─────┘ └──┬───┘ └─────┬──────┘ └────┬────┘
     │            │             │          │           │             │
     │ sbatch job.sh            │          │           │             │
     │──────────►│             │          │           │             │
     │            │             │          │           │             │
     │            │ Job enters  │          │           │             │
     │            │ PENDING     │          │           │             │
     │            │ (no nodes)  │          │           │             │
     │            │             │          │           │             │
     │            │         GET /metrics/jobs         │             │
     │            │◄────────────┼──────────│           │             │
     │            │             │          │           │             │
     │            │ pending=1   │          │           │             │
     │            │─────────────┼────────►│           │             │
     │            │             │          │           │             │
     │            │             │          │  Query    │             │
     │            │             │          │  pending  │             │
     │            │             │          │◄──────────│             │
     │            │             │          │           │             │
     │            │             │          │  Result=1 │             │
     │            │             │          │──────────►│             │
     │            │             │          │           │             │
     │            │             │          │           │ Scale to    │
     │            │             │          │           │ minReplicas │
     │            │             │          │           │────────────►│
     │            │             │          │           │             │
     │            │             │          │           │   Create    │
     │            │             │          │           │    Pod      │
     │            │             │          │           │◄────────────│
     │            │             │          │           │             │
     │            │      slurmd registers  │           │             │
     │            │◄───────────────────────┼───────────┼─────────────│
     │            │             │          │           │             │
     │            │ Job starts  │          │           │             │
     │            │ RUNNING     │          │           │             │
     │            │             │          │           │             │
     │ Job output │             │          │           │             │
     │◄───────────│             │          │           │             │
     │            │             │          │           │             │
```

### 詳細時間線

```
T+0:00    使用者提交 Job
          $ sbatch -p compute-cpu job.sh
          │
          ▼
T+0:00    slurmctld 接收 Job
          - Job 進入 PENDING 狀態
          - Reason: "Nodes required for job are DOWN, DRAINED or reserved"
          - 或 "PartitionNodeLimit" 如果沒有節點

          $ squeue
          JOBID  PARTITION    NAME   STATE   REASON
          123    compute-cpu  job.sh PENDING (Resources)
          │
          ▼
T+0:01    slurmrestd 暴露 metrics
          - /metrics/jobs 端點返回：
            slurm_partition_jobs_pending{partition="compute-cpu"} 1
          │
          ▼
T+0:30    Prometheus 抓取 metrics
          (預設 scrape interval 30 秒)
          - 儲存 slurm_partition_jobs_pending = 1
          │
          ▼
T+1:00    KEDA 查詢 Prometheus
          (預設 polling interval 30 秒)
          - Query: slurm_partition_jobs_pending{partition="compute-cpu"}
          - Result: 1
          - Threshold: 1
          - Decision: Scale to minReplicaCount (1)
          │
          ▼
T+1:01    KEDA 更新 NodeSet
          - PATCH NodeSet/compute-cpu
          - spec.replicas: 0 → 1
          │
          ▼
T+1:02    NodeSet Controller Reconcile

          檔案：nodeset_sync.go:542-568

          replicaCount := 1  // 目標
          diff := len(pods) - replicaCount  // 0 - 1 = -1

          // diff < 0，需要 scale-out
          return r.doPodScaleOut(ctx, nodeset, pods, 1, hash)
          │
          ▼
T+1:03    doPodScaleOut() 執行

          檔案：nodeset_sync.go:570-654

          1. 計算需要建立的 Pod 數量
          2. 生成 Pod manifest
          3. 呼叫 CreateNodeSetPod()
          │
          ▼
T+1:05    Pod 建立
          - Pod: compute-cpu-0
          - Status: Pending → ContainerCreating
          │
          ▼
T+1:30    Pod Running
          - kubelet 排程 Pod 到 worker node
          - Container 啟動
          - slurmd 進程啟動
          │
          ▼
T+1:45    slurmd 註冊到 slurmctld
          - slurmd 透過 configless 取得設定
          - 向 slurmctld 回報節點狀態
          - 節點狀態：IDLE
          │
          ▼
T+2:00    Job 開始執行
          - slurmctld 分配 Job 到新節點
          - Job 狀態：PENDING → RUNNING

          $ squeue
          JOBID  PARTITION    NAME   STATE   NODELIST
          123    compute-cpu  job.sh RUNNING compute-cpu-0
```

### 關鍵延遲分析

| 階段 | 預估時間 | 說明 |
|------|----------|------|
| Job 提交 → Prometheus 抓取 | 0-30s | 取決於 scrape interval |
| Prometheus 抓取 → KEDA 檢測 | 0-30s | 取決於 KEDA polling interval |
| KEDA 更新 → Controller reconcile | 0-5s | Controller watch 延遲 |
| Pod 建立 → Pod Running | 10-30s | Image pull + container start |
| slurmd 啟動 → 節點註冊 | 5-15s | slurmd 初始化 + 網路延遲 |
| **總計** | **15-110s** | 最佳情況 ~15s，最差情況 ~2分鐘 |

---

## 程式碼佐證

### 1. NodeSet Scale Subresource 定義

**檔案**：`api/v1beta1/nodeset_types.go:297`

```go
// +kubebuilder:subresource:scale:specpath=".spec.replicas",statuspath=".status.replicas",selectorpath=".status.selector"
```

這個 annotation 告訴 Kubernetes：
- NodeSet 支援 scale subresource
- KEDA 可以透過標準 scale API 修改 `spec.replicas`

### 2. doPodScaleOut() - Pod 建立邏輯

**檔案**：`internal/controller/nodeset/nodeset_sync.go:570-654`

```go
// doPodScaleOut handles scaling-out NodeSet pods.
// NodeSet pods should be uncordoned and undrained, and new pods created.
func (r *NodeSetReconciler) doPodScaleOut(
    ctx context.Context,
    nodeset *slinkyv1beta1.NodeSet,
    pods []*corev1.Pod,
    numCreate int,
    hash string,
) error {
    logger := log.FromContext(ctx)
    key := objectutils.KeyFunc(nodeset)

    // 先 uncordon 現有的 pods
    uncordonFn := func(i int) error {
        pod := pods[i]
        return r.syncPodUncordon(ctx, nodeset, pod)
    }
    if _, err := utils.SlowStartBatch(len(pods), utils.SlowStartInitialBatchSize, uncordonFn); err != nil {
        return err
    }

    // 限制單次建立數量（防止 thundering herd）
    numCreate = mathutils.Clamp(numCreate, 0, burstReplicas) // burstReplicas = 250

    // 計算已使用的 ordinals
    usedOrdinals := set.New[int]()
    for _, pod := range pods {
        usedOrdinals.Insert(nodesetutils.GetOrdinal(pod))
    }

    // 建立新的 Pod manifests
    podsToCreate := make([]*corev1.Pod, numCreate)
    ordinal := 0
    for i := range numCreate {
        for usedOrdinals.Has(ordinal) {
            ordinal++
        }
        pod, err := r.newNodeSetPod(r.Client, ctx, nodeset, ordinal, hash)
        if err != nil {
            return err
        }
        usedOrdinals.Insert(ordinal)
        podsToCreate[i] = pod
    }

    // 設定 expectations
    if err := r.expectations.ExpectCreations(logger, key, numCreate); err != nil {
        return err
    }

    // 使用 slow-start 批次建立 Pods
    // 避免同時建立大量 Pods 導致 API server 過載
    successfulCreations, err := utils.SlowStartBatch(numCreate, utils.SlowStartInitialBatchSize, func(index int) error {
        pod := podsToCreate[index]
        if err := r.podControl.CreateNodeSetPod(ctx, nodeset, pod); err != nil {
            if apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause) {
                return nil
            }
            return err
        }
        return nil
    })

    // 處理建立失敗的情況
    if skippedPods := numCreate - successfulCreations; skippedPods > 0 {
        logger.V(2).Info("Slow-start failure. Skipping creation of pods",
            "podsSkipped", skippedPods)
        for range skippedPods {
            r.expectations.CreationObserved(logger, key)
        }
    }

    return err
}
```

### 3. doPodScaleIn() - Pod 刪除邏輯

**檔案**：`internal/controller/nodeset/nodeset_sync.go:674-743`

```go
// doPodScaleIn handles scaling-in NodeSet pods.
// Certain NodeSet pods should be cordoned and drained, and defunct pods
// deleted after being fully drained.
func (r *NodeSetReconciler) doPodScaleIn(
    ctx context.Context,
    nodeset *slinkyv1beta1.NodeSet,
    podsToDelete, podsToKeep []*corev1.Pod,
) error {
    logger := log.FromContext(ctx)
    key := objectutils.KeyFunc(nodeset)

    // 先 uncordon 要保留的 pods
    uncordonFn := func(i int) error {
        pod := podsToKeep[i]
        return r.syncPodUncordon(ctx, nodeset, pod)
    }
    if _, err := utils.SlowStartBatch(len(podsToKeep), utils.SlowStartInitialBatchSize, uncordonFn); err != nil {
        return err
    }

    // 處理 PVC retention policy
    fixPodPVCsFn := func(i int) error {
        pod := podsToDelete[i]
        if matchPolicy, err := r.podControl.PodPVCsMatchRetentionPolicy(ctx, nodeset, pod); err != nil {
            return err
        } else if !matchPolicy {
            if err := r.podControl.UpdatePodPVCsForRetentionPolicy(ctx, nodeset, pod); err != nil {
                return err
            }
        }
        return nil
    }
    if _, err := utils.SlowStartBatch(len(podsToDelete), utils.SlowStartInitialBatchSize, fixPodPVCsFn); err != nil {
        return err
    }

    numDelete := mathutils.Clamp(len(podsToDelete), 0, burstReplicas)

    // 設定 deletion expectations
    if err := r.expectations.ExpectDeletions(logger, key, getPodKeys(podsToDelete)); err != nil {
        return err
    }

    // 批次處理刪除
    _, err := utils.SlowStartBatch(numDelete, utils.SlowStartInitialBatchSize, func(index int) error {
        pod := podsToDelete[index]
        podKey := kubecontroller.PodKey(pod)

        // processCondemned 會：
        // 1. Cordon pod
        // 2. Drain Slurm node
        // 3. 等待 drain 完成
        // 4. 刪除 pod
        if err := r.processCondemned(ctx, nodeset, podsToDelete, index); err != nil {
            r.expectations.DeletionObserved(logger, key, podKey)
            if !apierrors.IsNotFound(err) {
                return err
            }
        }

        // 確認 Slurm node 已經 drained
        if isDrained, err := r.slurmControl.IsNodeDrained(ctx, nodeset, pod); !isDrained || err != nil {
            r.expectations.DeletionObserved(logger, key, podKey)
            if err != nil {
                return err
            }
        }
        return nil
    })

    return err
}
```

### 4. Partition 配置生成

**檔案**：`internal/builder/controller_config.go:252-279`

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

    // NodeSet 定義（總是生成）
    nodesetLine := []string{
        fmt.Sprintf("NodeSet=%v", name),
        fmt.Sprintf("Feature=%v", name),
    }
    nodesetLineRendered := strings.Join(nodesetLine, " ")
    conf.AddProperty(config.NewPropertyRaw(nodesetLineRendered))

    // Partition 定義（只有 enabled 時才生成）
    partition := nodeset.Spec.Partition
    if !partition.Enabled {
        continue  // ← 如果 partition.enabled=false，跳過
    }
    partitionLine := []string{
        fmt.Sprintf("PartitionName=%v", name),
        fmt.Sprintf("Nodes=%v", name),
        partition.Config,
    }
    partitionLineRendered := strings.Join(partitionLine, " ")
    conf.AddProperty(config.NewPropertyRaw(partitionLineRendered))
}
```

---

## 潛在問題與限制

### 問題 1：多個 ScaledObject 同時觸發

**情境**：如果多個 NodeSet 使用相同的 Prometheus query（例如都監控同一個 partition 的 pending jobs），則所有 NodeSet 可能同時 scale up。

```yaml
# ScaledObject for compute-cpu
triggers:
  - type: prometheus
    metadata:
      query: slurm_partition_jobs_pending{partition="compute"}  # 共用 partition

# ScaledObject for compute-gpu
triggers:
  - type: prometheus
    metadata:
      query: slurm_partition_jobs_pending{partition="compute"}  # 共用 partition
```

**結果**：當有 1 個 pending job 時，兩個 NodeSet 都會 scale up，即使只需要其中一個。

**解決方案**：

1. **每個 NodeSet 使用獨立的 Partition**（推薦）
2. 使用更精確的 metric query（如果 Slurm 有提供）
3. 使用 KEDA 的 `activationThreshold` 減少誤觸發

### 問題 2：Scale-from-Zero 延遲

**情境**：從 0 scale 到 1 需要等待：
- Prometheus scrape interval (0-30s)
- KEDA polling interval (0-30s)
- Pod scheduling + startup (10-60s)
- slurmd registration (5-15s)

**影響**：使用者提交 Job 後需要等待 1-2 分鐘才能開始執行。

**解決方案**：

1. 縮短 Prometheus scrape interval（但會增加 slurmctld 負載）
2. 縮短 KEDA polling interval（但會增加 Prometheus 負載）
3. 維持 `minReplicaCount: 1` 永遠保持一個節點
4. 使用 warm pool（保持少量 idle 節點）

### 問題 3：Partition 仍存在但無節點

**情境**：所有 NodeSet scale 到 0 後，Partition 仍存在於 slurm.conf 中，但沒有可用節點。

```bash
$ sinfo -p compute-cpu
PARTITION    AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute-cpu  up     infinite      0    n/a
```

**影響**：
- Job 可以提交，但會一直 PENDING
- 使用者可能不知道為什麼 Job 不執行
- 需要等待 scale-up 才能執行

**解決方案**：

1. 在文件中說明這個行為
2. 設定 Job timeout
3. 提供 CLI 工具讓使用者查詢當前可用資源

### 問題 4：idleReplicaCount 只能是 0

**KEDA 限制**：`idleReplicaCount` 只支援設為 0。

```yaml
spec:
  idleReplicaCount: 0   # ← 只能是 0
  minReplicaCount: 1    # ← scale-up 時的最小值
```

**原因**：KEDA 的 scale-from-zero 機制是透過 HPA pausing 實現的，HPA 無法直接管理 0 replicas。

**影響**：無法設定「idle 時保持 1 個節點」的行為。

**解決方案**：

1. 使用 `minReplicaCount: 1` + 較長的 `cooldownPeriod`
2. 不使用 KEDA，改用自訂 controller

---

## 最佳實踐建議

### 1. ScaledObject 配置建議

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: scale-compute-cpu
spec:
  scaleTargetRef:
    apiVersion: slinky.slurm.net/v1beta1
    kind: NodeSet
    name: compute-cpu

  # Scale-from-zero 設定
  idleReplicaCount: 0
  minReplicaCount: 1
  maxReplicaCount: 10

  # 冷卻時間（避免頻繁 scale up/down）
  cooldownPeriod: 300        # 5 分鐘

  # Polling 間隔
  pollingInterval: 15        # 15 秒（加快反應速度）

  triggers:
    - type: prometheus
      metricType: Value      # 使用絕對值而非平均值
      metadata:
        serverAddress: http://prometheus:9090
        # 使用特定 partition 的 metric
        query: slurm_partition_jobs_pending{partition="compute-cpu"}
        threshold: '1'

  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 300  # 5 分鐘穩定期
          policies:
            - type: Percent
              value: 50
              periodSeconds: 60
        scaleUp:
          stabilizationWindowSeconds: 0    # 立即 scale up
          policies:
            - type: Percent
              value: 100
              periodSeconds: 15
```

### 2. 每個 NodeSet 使用獨立 Partition

```yaml
# values.yaml
nodesets:
  compute-cpu:
    replicas: 0
    partition:
      enabled: true
      config: "State=UP MaxTime=7-00:00:00"

  compute-gpu:
    replicas: 0
    partition:
      enabled: true
      config: "State=UP MaxTime=1-00:00:00 Gres=gpu:4"
```

### 3. 監控建議

設定以下告警：

```yaml
# Prometheus alert rules
groups:
  - name: slurm-scaling
    rules:
      - alert: JobPendingTooLong
        expr: |
          slurm_partition_jobs_pending > 0
          and
          time() - slurm_job_submit_time > 300
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Job pending for more than 5 minutes"

      - alert: NodeSetScaleUpFailed
        expr: |
          kube_nodeset_status_replicas < kube_nodeset_spec_replicas
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "NodeSet failed to scale up"
```

### 4. 測試 Scale-from-Zero

```bash
#!/bin/bash
# test-scale-from-zero.sh

# 1. 確認所有 NodeSet 都是 0 replicas
kubectl get nodeset -n slurm -o wide

# 2. 提交測試 Job
kubectl exec -n slurm deployment/slurm-controller -- \
  sbatch --wrap "sleep 60" -p compute-cpu

# 3. 觀察 scaling 過程
watch -n 5 'kubectl get pods,nodeset -n slurm'

# 4. 檢查 Job 狀態
kubectl exec -n slurm deployment/slurm-controller -- \
  squeue -l
```

---

## 結論

### 核心行為總結

1. **Scale-to-Zero**：KEDA 在 cooldown period 後將 NodeSet replicas 設為 0
2. **Partition 保持存在**：即使沒有節點，Partition 仍定義在 slurm.conf 中
3. **Job PENDING**：提交的 Job 會進入 PENDING 狀態等待資源
4. **自動 Scale-up**：Prometheus metrics 觸發 KEDA，NodeSet 自動 scale up
5. **延遲時間**：從 Job 提交到執行約需 1-2 分鐘

### 多 NodeSet 注意事項

- 每個 NodeSet 需要獨立的 ScaledObject
- 建議每個 NodeSet 使用獨立的 Partition
- 共享 Partition 時需注意 metric query 的精確性

### 關鍵程式碼參考

| 功能 | 檔案 | 行數 |
|------|------|------|
| Scale subresource 定義 | `api/v1beta1/nodeset_types.go` | 297 |
| 主要 sync 邏輯 | `internal/controller/nodeset/nodeset_sync.go` | 47-103 |
| Replica 比較 | `internal/controller/nodeset/nodeset_sync.go` | 542-568 |
| Pod Scale-out | `internal/controller/nodeset/nodeset_sync.go` | 570-654 |
| Pod Scale-in | `internal/controller/nodeset/nodeset_sync.go` | 674-743 |
| Partition 配置生成 | `internal/builder/controller_config.go` | 252-279 |
| ServiceMonitor 定義 | `internal/builder/controller_servicemonitor.go` | 60-66 |

---

## 相關文件

- [autoscaling.md](../../docs/usage/autoscaling.md) - 官方 Autoscaling 文件
- [deep-dive-partition.md](./deep-dive-partition.md) - Partition 管理深度分析
- [deep-dive-nodeset-storage.md](./deep-dive-nodeset-storage.md) - NodeSet 儲存管理
