# API_SURFACE Part 2 — Status 欄位、HPA、kubectl 操作、Annotation

> 本文件為 [API_SURFACE_part1.md](./API_SURFACE_part1.md) 的續篇。
> **API Group**: `slinky.slurm.net` / **Version**: `v1beta1`

---

## 5. Status 欄位說明

### 5.1 NodeSet.status

| 欄位 | 型別 | `kubectl get` 欄位名 | 說明 |
|------|------|---------------------|------|
| `replicas` | int32 | REPLICAS | 目前運行中的 pod 總數 |
| `updatedReplicas` | int32 | UPDATED | 已更新到最新 template 的 pod 數 |
| `readyReplicas` | int32 | READY | Ready condition 為 True 的 pod 數 |
| `availableReplicas` | int32 | — | 滿足 minReadySeconds 的可用 pod 數 |
| `unavailableReplicas` | int32 | — | 尚不可用的 pod 數 |
| `desired` | int32 | DESIRED | 期望數量（DaemonSet：符合 selector 的 node 數；StatefulSet：`spec.replicas`） |
| `slurmIdle` | int32 | IDLE（-o wide） | Slurm 狀態 IDLE 的節點數（未分配任何 job） |
| `slurmAllocated` | int32 | ALLOCATED（-o wide） | Slurm 狀態 ALLOCATED / MIXED 的節點數 |
| `slurmDown` | int32 | DOWN（-o wide） | Slurm 狀態 DOWN 的節點數（不可用） |
| `slurmDrain` | int32 | DRAIN（-o wide） | Slurm 狀態 DRAIN 的節點數（排空中，不接受新 job） |
| `nodeSetHash` | string | — | 當前 ControllerRevision hash（用於 rolling update 追蹤） |
| `collisionCount` | *int32 | — | ControllerRevision 命名衝突計數 |
| `selector` | string | — | Label selector 字串（供 HPA scale subresource 使用） |
| `ordinalToNode` | map[string]string | — | Pod 序號到 Kubernetes Node 名稱的對應表（StatefulSet 模式） |
| `observedGeneration` | int64 | — | 已處理的最新 generation |

### 5.2 NodeSet.status.conditions

| Condition Type | 說明 | True 條件 |
|---------------|------|-----------|
| `Available` | NodeSet 有足夠可用副本 | `availableReplicas >= desired` |
| `Progressing` | NodeSet 正在更新中 | Rolling update 進行中，或 pod 正在建立 |

### 5.3 LoginSet.status

| 欄位 | 型別 | 說明 |
|------|------|------|
| `replicas` | int32 | 目前 pod 總數 |
| `selector` | string | Label selector（供 HPA 使用） |
| `conditions` | []metav1.Condition | 標準 condition 列表 |

### 5.4 Token.status

| 欄位 | 型別 | 說明 |
|------|------|------|
| `issuedAt` | *metav1.Time | JWT 簽發時間（RFC3339），`kubectl get tokens` 顯示為 IAT 欄位 |
| `conditions` | []metav1.Condition | 標準 condition 列表 |

### 5.5 Controller / Accounting / RestApi.status

均只包含 `conditions []metav1.Condition`，使用標準 Kubernetes Condition type。

---

## 6. Scale Subresource 與 HPA 支援

`NodeSet` 和 `LoginSet` 均已宣告 scale subresource，mapping 如下：

| 路徑 | 說明 |
|------|------|
| `spec.replicas` | spec path（HPA 寫入副本數） |
| `status.replicas` | status path（HPA 讀取當前副本數） |
| `status.selector` | selector path（HPA 計算 pod metric 用） |

### 設定 HPA 自動縮放 NodeSet

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: slurm-workers-hpa
  namespace: slurm
spec:
  scaleTargetRef:
    apiVersion: slinky.slurm.net/v1beta1
    kind: NodeSet
    name: slurm-workers
  minReplicas: 2
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

> **注意**：`scalingMode: DaemonSet` 模式下，`spec.replicas` 欄位無效，HPA **不適用**。

### 手動縮放

```bash
kubectl scale nodeset slurm-workers --replicas=5 -n slurm
kubectl scale loginset slurm-login --replicas=3 -n slurm
```

---

## 7. kubectl 操作範例

### 查看 CRD 資源

```bash
# 使用標準 Kind 名稱
kubectl get controllers,accountings,restapis -n slurm

# 使用短名（shortName）
kubectl get slurmctld,slurmdbd,slurmrestd -n slurm
kubectl get nodesets,nss,slurmd -n slurm
kubectl get loginsets,lss,sackd -n slurm
kubectl get tokens,jwt -n slurm

# NodeSet 基本狀態（DESIRED/REPLICAS/UPDATED/READY）
kubectl get nodesets -n slurm

# NodeSet 詳細狀態含 Slurm 資訊（IDLE/ALLOCATED/DOWN/DRAIN，priority=1 欄位）
kubectl get nodesets -n slurm -o wide

# Token 狀態（USER/IAT 欄位）
kubectl get tokens -n slurm

# 查看特定 NodeSet 的完整狀態
kubectl describe nodeset slurm-workers -n slurm

# 以 JSON 讀取 slurmIdle 欄位
kubectl get nodeset slurm-workers -n slurm \
  -o jsonpath='{.status.slurmIdle}'
```

### 查看 Operator 管理的子資源

```bash
# 查看 NodeSet 管理的所有 pods（StatefulSet 模式）
kubectl get pods -l nodeset.slinky.slurm.net/pod-name=slurm-workers -n slurm

# 查看 NodeSet 管理的 ControllerRevisions（版本歷史）
kubectl get controllerrevisions -l app.kubernetes.io/name=slurm-workers -n slurm

# 查看 PodDisruptionBudget（workloadDisruptionProtection=true 時建立）
kubectl get pdb -n slurm
```

### 除錯

```bash
# 查看 operator controller 日誌
kubectl logs -l app.kubernetes.io/component=manager -n slurm-operator-system

# 查看 webhook 日誌
kubectl logs -l app.kubernetes.io/component=webhook -n slurm-operator-system

# 查看 NodeSet 事件
kubectl get events --field-selector involvedObject.kind=NodeSet -n slurm
```

---

## 8. Well-Known Annotation 操作

### Pod 層級 Annotation

| Annotation | 設定者 | 說明 |
|-----------|--------|------|
| `nodeset.slinky.slurm.net/pod-cordon` | 使用者 | `true` 時觸發 Slurm DRAIN（不驅逐現有 job） |
| `nodeset.slinky.slurm.net/pod-deletion-cost` | 使用者 | int32 字串，負值優先刪除 |
| `nodeset.slinky.slurm.net/pod-deadline` | Operator | RFC3339 時間戳，Slurm job 預計完成時間 |
| `topology.slinky.slurm.net/spec` | PodBindingWebhook | 從 Node 複製來的 topology 字串 |

```bash
# 手動 cordon NodeSet pod（觸發 Slurm DRAIN）
kubectl annotate pod slurm-workers-0 -n slurm \
  nodeset.slinky.slurm.net/pod-cordon=true

# 移除 cordon（恢復 UNDRAIN）
kubectl annotate pod slurm-workers-0 -n slurm \
  nodeset.slinky.slurm.net/pod-cordon-

# 設定 pod 刪除優先順序（縮容時，數值越小越優先刪除）
kubectl annotate pod slurm-workers-3 -n slurm \
  "nodeset.slinky.slurm.net/pod-deletion-cost=-100"

# 查看 operator 設定的 pod deadline
kubectl get pod slurm-workers-0 -n slurm \
  -o jsonpath='{.metadata.annotations.nodeset\.slinky\.slurm\.net/pod-deadline}'
```

### Node 層級 Annotation

| Annotation | 設定者 | 說明 |
|-----------|--------|------|
| `nodeset.slinky.slurm.net/node-cordon-reason` | 使用者 | Kubernetes node cordon 時顯示的 Slurm drain reason |
| `topology.slinky.slurm.net/spec` | 使用者 / Infra | Slurm Dynamic Topology 字串，由 PodBindingWebhook 複製到 pod |
| `nodeset.slinky.slurm.net/hostname-override` | 使用者 | 覆寫 DaemonSet 模式下的 pod hostname（即 Slurm 節點名稱） |

```bash
# 設定 Kubernetes node cordon 原因（影響 Slurm drain reason 顯示）
kubectl annotate node kube-gpu-node-1 \
  nodeset.slinky.slurm.net/node-cordon-reason="Hardware maintenance scheduled"

# 設定 Slurm Dynamic Topology（PodBindingWebhook 會自動複製到排程到此 node 的 pod）
kubectl annotate node kube-gpu-node-1 \
  "topology.slinky.slurm.net/spec=topo-switch:s2,topo-block:b2"

# DaemonSet 模式：覆寫特定 node 上的 pod hostname（影響 Slurm 節點名稱）
kubectl annotate node kube-gpu-node-1 \
  nodeset.slinky.slurm.net/hostname-override="gpu-node-custom-01"
```

### NodeSet Controller 自動設定的 Label（僅供讀取）

```bash
# 查看 NodeSet pod 的 label
kubectl get pods -n slurm -L \
  nodeset.slinky.slurm.net/pod-hostname,\
  nodeset.slinky.slurm.net/pod-index,\
  nodeset.slinky.slurm.net/pod-protect,\
  nodeset.slinky.slurm.net/scaling-mode
```

| Label | 說明 |
|-------|------|
| `nodeset.slinky.slurm.net/pod-name` | Pod 名稱 |
| `nodeset.slinky.slurm.net/pod-index` | Pod 序號（ordinal） |
| `nodeset.slinky.slurm.net/pod-hostname` | Pod hostname（即 Slurm 節點名稱） |
| `nodeset.slinky.slurm.net/pod-protect` | 是否受 PodDisruptionBudget 保護 |
| `nodeset.slinky.slurm.net/scaling-mode` | `DaemonSet` 或 `StatefulSet` |

---

## 9. 欄位預設值彙整

### NodeSet 欄位預設值

| 欄位 | 預設值 |
|------|--------|
| `spec.replicas` | `1` |
| `spec.scalingMode` | `StatefulSet` |
| `spec.workloadDisruptionProtection` | `true` |
| `spec.updateStrategy.type` | `RollingUpdate` |
| `spec.updateStrategy.rollingUpdate.maxUnavailable` | `"25%"` |
| `spec.persistentVolumeClaimRetentionPolicy.whenDeleted` | `Retain` |
| `spec.persistentVolumeClaimRetentionPolicy.whenScaled` | `Retain` |
| `spec.pruneSlurmNodeRecords` | `Never` |

### Token 欄位預設值

| 欄位 | 預設值 |
|------|--------|
| `spec.refresh` | `true` |

### StorageConfig（Accounting）欄位預設值

| 欄位 | 預設值 |
|------|--------|
| `spec.storageConfig.port` | `3306` |
| `spec.storageConfig.database` | `"slurm_acct_db"` |

---

## 10. 已棄用欄位

| 欄位 | 所在 CRD | 棄用說明 | 取代方式 |
|------|---------|---------|---------|
| `spec.jwtHs256KeyRef` | Controller, Accounting, Token | 使用 HS256 演算法的舊版欄位 | 改用 `spec.jwtKeyRef` |
| `spec.taintKubeNodes` | NodeSet | 計畫在未來版本移除；webhook 會發出 Warning | 改用其他節點隔離機制 |
