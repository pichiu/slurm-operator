# 自動擴展 (Autoscaling)

## TL;DR

slurm-operator 可以根據 Slurm 指標自動擴展 NodeSet Pod。本指南說明如何使用 KEDA 配置自動擴展，讓 Slurm 分割區根據待處理作業數量自動增減節點數量，實現資源的彈性調配。

---

## Translation

slurm-operator 可以配置為根據 Slurm 指標 (Metrics) 自動擴展 NodeSet Pod。本指南討論如何使用 [KEDA] 配置自動擴展。

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [自動擴展](#自動擴展-autoscaling)
  - [目錄](#目錄)
  - [開始之前](#開始之前)
    - [相依套件](#相依套件)
      - [驗證 KEDA Metrics API Server 正在運行](#驗證-keda-metrics-api-server-正在運行)
  - [自動擴展](#自動擴展-1)
    - [NodeSet Scale 子資源](#nodeset-scale-子資源)
    - [KEDA ScaledObject](#keda-scaledobject)

<!-- mdformat-toc end -->

## 開始之前

在嘗試自動擴展 NodeSet 之前，Slinky 應該已完全部署到 Kubernetes 叢集，且 Slurm 作業應該能夠執行。

### 相依套件

自動擴展需要 Slinky 未包含的額外服務。請依照文件安裝 [Prometheus]、[Metrics Server] 和 [KEDA]。

Prometheus 會安裝用於回報指標並透過 Grafana 檢視的工具。Metrics Server 是 `kubectl top` 等工具回報 CPU 和記憶體使用量所需的。建議使用 KEDA 進行自動擴展，因為它比標準的水平 Pod 自動擴展器 ([HPA]) 提供更好的可用性改進。

安裝 Prometheus Helm chart，執行以下指令：

```sh
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack \
  --set 'installCRDs=true' \
  --namespace prometheus --create-namespace
```

安裝 KEDA Helm chart，執行以下指令：

```sh
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda \
  --namespace keda --create-namespace
```

安裝 Slurm Helm chart，執行以下指令：

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set 'controller.metrics.enabled=true' \
  --set 'controller.metrics.serviceMonitor.enabled=true' \
  --namespace slurm --create-namespace
```

#### 驗證 KEDA Metrics API Server 正在運行

```sh
$ kubectl get apiservice -l app.kubernetes.io/instance=keda
NAME                              SERVICE                                AVAILABLE   AGE
v1beta1.external.metrics.k8s.io   keda/keda-operator-metrics-apiserver   True        22h
```

[KEDA] 提供 HPA 所需的 metrics apiserver，用於根據 Slurm 的自訂指標進行擴展。可以使用 [Prometheus Adapter] 等替代方案，但 KEDA 除了包含 metrics apiserver 外，還提供可用性增強和 HPA 改進。

## 自動擴展

自動擴展 NodeSet 允許 Slurm 分割區 (Partitions) 根據 CPU 和記憶體使用量進行擴展和收縮。使用 Slurm 指標，NodeSet 還可以根據 Slurm 特定資訊進行擴展，如待處理作業數量或分割區中最大待處理作業的大小。有許多方法可以配置自動擴展。根據執行的作業類型和叢集中可用的資源，嘗試不同的組合。

### NodeSet Scale 子資源

在 Kubernetes 中擴展資源需要 Deployment 和 StatefulSet 等資源支援 [scale 子資源][scale subresource]。NodeSet 自訂資源也是如此。

Scale 子資源提供標準介面來觀察和控制資源的副本數量。對於 NodeSet，它允許 Kubernetes 和相關服務控制作為 NodeSet 一部分運行的 `slurmd` 副本數量。

要手動擴展 NodeSet，請使用 `kubectl scale` 指令。在此範例中，NodeSet (nss) `slurm-worker-radar` 被擴展到 1。

```sh
$ kubectl scale -n slurm nss/slurm-worker-radar --replicas=1
nodeset.slinky.slurm.net/slurm-worker-radar scaled

$ kubectl get pods -o wide -n slurm -l app.kubernetes.io/instance=slurm-worker-radar
NAME                   READY   STATUS    RESTARTS   AGE     IP            NODE          NOMINATED NODE   READINESS GATES
slurm-worker-radar-0   1/1     Running   0          2m48s   10.244.4.17   kind-worker   <none>           <none>
```

這對應到 Slurm 分割區 `radar`。

```sh
$ kubectl exec -n slurm statefulset/slurm-controller -- sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
radar        up   infinite      1   idle kind-worker
```

NodeSet 可以擴展到零。在這種情況下，沒有 `slurmd` 副本在運行，所有排程到該分割區的作業將保持在待處理狀態。

```sh
$ kubectl scale nss/slurm-worker-radar -n slurm --replicas=0
nodeset.slinky.slurm.net/slurm-worker-radar scaled
```

為了讓 NodeSet 按需擴展，需要部署自動擴展器。KEDA 允許資源從 0\<->1 擴展，並建立 HPA 根據 Prometheus 等觸發器進行擴展。

### KEDA ScaledObject

KEDA 使用自訂資源 [ScaledObject] 來監控和擴展資源。它會自動建立根據外部觸發器（如 Prometheus）擴展所需的 HPA。透過 Slurm 指標，NodeSet 可以根據從 Slurm restapi 收集的資料進行擴展。

此範例 [ScaledObject] 會監控分割區 `radar` 的待處理作業數量，並擴展 NodeSet `slurm-worker-radar`，直到滿足閾值或達到 `maxReplicaCount`。

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: scale-radar
spec:
  scaleTargetRef:
    apiVersion: slinky.slurm.net/v1beta1
    kind: NodeSet
    name: slurm-worker-radar
  idleReplicaCount: 0
  minReplicaCount: 1
  maxReplicaCount: 3
  triggers:
    - type: prometheus
      metricType: Value
      metadata:
        serverAddress: http://prometheus-kube-prometheus-prometheus.prometheus:9090
        query: slurm_partition_jobs_pending{partition="radar"}
        threshold: '5'
```

> [!NOTE]
> Prometheus 觸發器使用 `metricType: Value` 而非預設的 `AverageValue`。`AverageValue` 透過將閾值除以當前副本數來計算副本數。

請查看 [ScaledObject] 文件以獲取允許選項的完整清單。

在此場景中，ScaledObject `scale-radar` 會從 Prometheus 查詢 Slurm 指標 `slurm_partition_pending_jobs`，並帶有標籤 `partition="radar"`。

當觸發器上有活動（至少一個待處理作業）時，KEDA 會將 NodeSet 擴展到 `minReplicaCount`，然後讓 HPA 處理向上擴展到 `maxReplicaCount` 或向下縮減到 `minReplicaCount`。在可配置的時間內沒有觸發器活動後，KEDA 會將 NodeSet 縮減到 `idleReplicaCount`。請參閱 [KEDA] 文件中關於 [idleReplicaCount] 的更多範例。

> [!NOTE]
> 由於 HPA 控制器工作方式的限制，`idleReplicaCount` 唯一支援的值是 0。

要驗證 KEDA ScaledObject，請在沒有副本的 NodeSet 上將其套用到叢集的適當命名空間。

```sh
$ kubectl scale nss/slurm-worker-radar -n slurm --replicas=0
nodeset.slinky.slurm.net/slurm-worker-radar scaled
```

等待 Slurm 回報分割區沒有節點。

```sh
$ slurm@slurm-controller-0:/tmp$ sinfo -p radar
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
radar        up   infinite      0    n/a
```

使用 `kubectl` 將 ScaledObject 套用到正確的命名空間，並驗證 KEDA 和 HPA 資源已建立。

```sh
$ kubectl apply -f scaledobject.yaml -n slurm
scaledobject.keda.sh/scale-radar created

$ kubectl get -n slurm scaledobjects
NAME           SCALETARGETKIND                     SCALETARGETNAME        MIN   MAX   TRIGGERS     AUTHENTICATION   READY   ACTIVE   FALLBACK   PAUSED    AGE
scale-radar    slinky.slurm.net/v1beta1.NodeSet    slurm-worker-radar    1     5     prometheus                    True    False    Unknown    Unknown   28s

$ kubectl get -n slurm hpa
NAME                    REFERENCE                      TARGETS       MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-scale-radar    NodeSet/slurm-worker-radar    <unknown>/5   1         5         0          32s
```

一旦建立 [ScaledObject] 和 HPA，啟動一些作業來測試 `NodeSet` scale 子資源是否有相應的擴展。

```sh
$ sbatch --wrap "sleep 30" --partition radar --exclusive
```

NodeSet 會因為觸發器上的活動而擴展到 `minReplicaCount`。一旦待處理作業數量超過配置的 `threshold`（向分割區提交更多獨占作業），將建立更多副本來處理額外的需求。在超過 `threshold` 之前，NodeSet 將保持在 `minReplicaCount`。

> [!NOTE]
> 此範例僅適用於單節點作業，除非 `threshold` 設定為 1。在這種情況下，只要有待處理作業，HPA 就會繼續向上擴展 NodeSet，直到達到 `maxReplicaCount`。

在預設的 `coolDownPeriod` 5 分鐘內沒有觸發器活動後，KEDA 會將 NodeSet 縮減到 0。

<!-- Links -->

[hpa]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
[idlereplicacount]: https://keda.sh/docs/concepts/scaling-deployments/#idlereplicacount
[keda]: https://keda.sh/docs/
[metrics server]: https://github.com/kubernetes-sigs/metrics-server
[prometheus]: https://prometheus-operator.dev/docs/getting-started/introduction/
[prometheus adapter]: https://github.com/kubernetes-sigs/prometheus-adapter
[scale subresource]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource
[scaledobject]: https://keda.sh/docs/concepts/scaling-deployments/

---

## Explanation

### 為何需要自動擴展？

在 HPC 環境中，工作負載通常有明顯的波動：
- **尖峰時段**：大量作業等待執行
- **離峰時段**：資源閒置浪費

自動擴展可以：
- 在需要時自動增加運算節點
- 在閒置時減少資源消耗
- 優化成本和資源利用率

### KEDA vs 原生 HPA

| 特性 | HPA | KEDA |
|-----|-----|------|
| 支援縮減到 0 | 否 | 是 |
| 自訂指標 | 需要 Adapter | 內建支援 |
| Prometheus 整合 | 需要額外配置 | 原生支援 |
| 配置複雜度 | 較高 | 較低 |

---

## Practical Example

### 完整的自動擴展配置範例

```yaml
# scaledobject.yaml
# 自動擴展 radar 分割區的 NodeSet
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: scale-radar               # ScaledObject 名稱
  namespace: slurm                # 部署到 slurm 命名空間
spec:
  scaleTargetRef:
    apiVersion: slinky.slurm.net/v1beta1
    kind: NodeSet                 # 擴展目標類型
    name: slurm-worker-radar      # 擴展目標名稱
  idleReplicaCount: 0             # 閒置時縮減到 0
  minReplicaCount: 1              # 有活動時最少 1 個副本
  maxReplicaCount: 10             # 最多 10 個副本
  cooldownPeriod: 300             # 冷卻期 300 秒（5 分鐘）
  triggers:
    - type: prometheus
      metricType: Value
      metadata:
        # Prometheus 服務位址
        serverAddress: http://prometheus-kube-prometheus-prometheus.prometheus:9090
        # 查詢待處理作業數量
        query: slurm_partition_jobs_pending{partition="radar"}
        # 每 5 個待處理作業增加一個副本
        threshold: '5'
```

### 測試自動擴展

```bash
# 1. 確認 NodeSet 當前為 0 副本
kubectl get nss -n slurm slurm-worker-radar

# 2. 提交作業觸發擴展
kubectl exec -n slurm deployment/slurm-login-slinky -- \
  sbatch --wrap "sleep 60" --partition radar

# 3. 觀察 ScaledObject 狀態
kubectl get scaledobjects -n slurm -w

# 4. 觀察 Pod 變化
kubectl get pods -n slurm -l app.kubernetes.io/instance=slurm-worker-radar -w

# 5. 查看 HPA 狀態
kubectl get hpa -n slurm
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| KEDA 未偵測到指標 | 確認 Prometheus 查詢語法正確，可先在 Grafana 測試 |
| 擴展太慢 | 調整 `pollingInterval`（預設 30 秒） |
| 縮減到 0 後無法擴展 | 檢查 KEDA operator 日誌 |
| 指標值一直是 unknown | 確認 ServiceMonitor 正確配置 |

### 小技巧

1. **先測試查詢**：在 Prometheus UI 中測試 PromQL 查詢
2. **設定合理閾值**：根據作業特性設定 threshold
3. **監控冷卻期**：調整 `cooldownPeriod` 避免頻繁擴縮
4. **考慮作業大小**：大型作業可能需要不同的擴展策略

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `helm install keda kedacore/keda -n keda --create-namespace` | 安裝 KEDA |
| `kubectl get scaledobjects -n slurm` | 查看 ScaledObject |
| `kubectl get hpa -n slurm` | 查看 HPA |
| `kubectl scale nss/<name> -n slurm --replicas=N` | 手動擴展 NodeSet |
| `kubectl describe scaledobject <name> -n slurm` | 查看 ScaledObject 詳細狀態 |
| `sbatch --wrap "sleep 30" --partition <partition>` | 提交測試作業 |
