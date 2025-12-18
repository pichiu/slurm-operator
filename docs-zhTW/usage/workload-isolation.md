# 工作負載隔離 (Workload Isolation)

## TL;DR

在某些環境中，可能需要將執行 Slurm NodeSet 的節點與其他 Kubernetes 工作負載隔離。本指南說明如何使用 Kubernetes 的 Taint（污點）和 Toleration（容忍）機制，以及 Pod 反親和性 (Anti-Affinity) 來實現隔離。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [工作負載隔離](#工作負載隔離-workload-isolation)
  - [目錄](#目錄)
  - [概述](#概述)
  - [先決條件](#先決條件)
  - [Taint 和 Toleration](#taint-和-toleration)
  - [Pod 反親和性](#pod-反親和性)

<!-- mdformat-toc end -->

## 概述

在某些環境中執行 Slinky 時，可能需要將執行 Slurm NodeSet 的節點與其他 Kubernetes 工作負載隔離。

預設情況下，Slurm-operator 不會對執行 NodeSet Pod 的節點新增 Taint (污點)。這可以透過在 Slurm Helm chart 中為特定 NodeSet 設定 `taintKubeNodes` 為 `true`，或部署 `TaintKubeNodes: true` 的 NodeSet CR 來配置。此選項會導致 `nodeset.slinky.slurm.net/worker=:NoExecute` Taint 被套用到任何排程了 NodeSet slurmd 副本的節點。任何沒有與此 Taint 匹配的 Toleration 的 Pod 都會被 Kubernetes 控制器驅逐。

本文件提供如何使用 [Taint 和 Toleration][taints and tolerations] 手動執行此操作的範例，以示範當設定 `taintKubeNodes` 時 Slurm-operator 為 NodeSet Pod 自動執行的操作。

## 先決條件

本指南假設使用者可以存取執行 `slurm-operator` 的功能性 Kubernetes 叢集。請參閱[快速入門指南][quickstart guide]以了解如何在 Kubernetes 叢集上設定 `slurm-operator`。

## Taint 和 Toleration

Taint (污點) 是 Kubernetes 提供的機制，允許節點排斥一組缺乏匹配 Toleration 的 Pod。Toleration (容忍) 是 Kubernetes 提供的機制，允許排程器將 Pod 排程到具有匹配 Taint 的節點上。

對只執行 Slurm Pod 的節點套用 Taint：

```bash
kubectl taint nodes kind-worker2 slinky.slurm.net/slurm:NoExecute
kubectl taint nodes kind-worker3 slinky.slurm.net/slurm:NoExecute
kubectl taint nodes kind-worker4 slinky.slurm.net/slurm:NoExecute
kubectl taint nodes kind-worker5 slinky.slurm.net/slurm:NoExecute
```

確認 Taint 已套用：

```bash
kubectl get nodes -o jsonpath="{range .items[*]}{.metadata.name}:{' '}{range .spec.taints[*]}{.key}={.value}:{.effect},{' '}{end}{'\n'}{end}"

kind-control-plane: node-role.kubernetes.io/control-plane=:NoSchedule,
kind-worker:
kind-worker2: slinky.slurm.net/slurm=:NoExecute
kind-worker3: slinky.slurm.net/slurm=:NoExecute
kind-worker4: slinky.slurm.net/slurm=:NoExecute
kind-worker5: slinky.slurm.net/slurm=:NoExecute
```

接下來，在 `slurm-operator` 元件上配置 Toleration。每個 `slurm-operator` 元件都可以從 `values.yaml` 中設定其 `tolerations`。更新所有元件的 `tolerations` 區段以匹配您在步驟 1 中套用的 `taint`。這需要為 `slurm` 和 `slurm-operator` Helm chart 中的所有元件完成。

```yaml
  # -- Pod 分配的 Toleration。
  # 參考：https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
  tolerations:
    - key: slinky.slurm.net/slurm
      operator: Exists
      effect: NoSchedule
```

## Pod 反親和性

在某些情況下，必須配置[反親和性][anti-affinity]以防止多個 NodeSet Pod (slurmd) 被排程到同一節點。Pod [反親和性][anti-affinity]可以在 NodeSet 的 `affinity` 區段下配置。為確保多個 NodeSet Pod 不能被排程到同一節點，請將以下內容新增到 `affinity` 區段：

```yaml
nodesets:
  slinky:
    ...
    # -- Pod 分配的親和性。
    # 參考：https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
        - topologyKey: kubernetes.io/hostname
          labelSelector:
            matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
              - slurmctld
              - slurmdbd
              - slurmrestd
              - mariadb
              - slurmd
```

在 `values.yaml` 中設定 `affinity` 並套用 Helm chart 後，可以透過執行以下指令在 `NodeSet` 中觀察 `affinity` 區段：

```bash
kubectl describe NodeSet --namespace slurm
```

<!-- links -->

[anti-affinity]: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/
[quickstart guide]: ../installation.md
[taints and tolerations]: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/

---

## Explanation

### Taint 和 Toleration 是什麼？

| 概念 | 說明 |
|-----|------|
| **Taint** | 節點的「標記」，表示該節點有特殊用途 |
| **Toleration** | Pod 的「宣告」，表示它可以容忍特定的 Taint |
| **Effect** | Taint 的影響：NoSchedule、PreferNoSchedule、NoExecute |

### Taint Effect 類型

| Effect | 說明 |
|--------|------|
| **NoSchedule** | 不會將新 Pod 排程到該節點 |
| **PreferNoSchedule** | 盡量不排程，但非強制 |
| **NoExecute** | 不排程且驅逐現有 Pod（無匹配 Toleration） |

### 為何需要工作負載隔離？

1. **效能保證**：HPC 工作負載不受其他服務干擾
2. **資源專用**：確保運算資源完全用於 Slurm 作業
3. **安全隔離**：敏感工作負載與其他服務分離
4. **成本控制**：專用節點可能有不同的計費方式

---

## Practical Example

### 完整的工作負載隔離配置

```bash
# 1. 標記專用於 Slurm 的節點
# 使用標籤識別 Slurm 節點
kubectl label nodes worker-{2..5} role=slurm

# 套用 Taint
kubectl taint nodes -l role=slurm slinky.slurm.net/slurm=:NoExecute

# 驗證
kubectl get nodes -l role=slurm -o custom-columns=\
"NAME:.metadata.name,\
TAINTS:.spec.taints[*].key"
```

### 配置 values.yaml

```yaml
# values-isolation.yaml
# 完整的工作負載隔離配置

# NodeSet 配置
nodesets:
  slinky:
    enabled: true
    replicas: 4

    # 啟用自動 Taint（由 Operator 管理）
    taintKubeNodes: true

    # 配置 Toleration
    tolerations:
      - key: slinky.slurm.net/slurm
        operator: Exists
        effect: NoExecute

    # 配置節點選擇器
    nodeSelector:
      role: slurm

    # 配置反親和性
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
        - topologyKey: kubernetes.io/hostname
          labelSelector:
            matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
              - slurmd

# 控制器也需要 Toleration（如果排程到被 Taint 的節點）
controller:
  tolerations:
    - key: slinky.slurm.net/slurm
      operator: Exists
      effect: NoExecute
```

### 安裝並驗證

```bash
# 安裝 Slurm 叢集
helm upgrade --install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values-isolation.yaml \
  --namespace=slurm --create-namespace

# 驗證 Pod 分布
kubectl get pods -n slurm -o wide

# 確認每個節點只有一個 slurmd Pod
kubectl get pods -n slurm -l app.kubernetes.io/name=slurmd \
  -o custom-columns="POD:.metadata.name,NODE:.spec.nodeName"

# 驗證非 Slurm Pod 無法排程到 Taint 節點
kubectl run test-pod --image=nginx --dry-run=client -o yaml | \
  kubectl apply -f - && \
  kubectl get pod test-pod -o wide
# test-pod 應該排程到沒有 Taint 的節點
kubectl delete pod test-pod
```

### 移除 Taint

```bash
# 移除特定節點的 Taint
kubectl taint nodes kind-worker2 slinky.slurm.net/slurm:NoExecute-

# 批次移除
kubectl taint nodes -l role=slurm slinky.slurm.net/slurm:NoExecute-
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| Pod 無法排程 | 確認 Toleration 與 Taint 匹配 |
| 多個 slurmd 在同一節點 | 配置 Pod 反親和性 |
| 控制器無法啟動 | 為控制器新增 Toleration |
| Taint 被意外移除 | 使用 `taintKubeNodes: true` 讓 Operator 自動管理 |

### Taint 與 Toleration 對應表

| Taint | Toleration |
|-------|-----------|
| `key=value:NoExecute` | `key: key`, `value: value`, `effect: NoExecute` |
| `key:NoExecute` | `key: key`, `operator: Exists`, `effect: NoExecute` |
| `key=:NoSchedule` | `key: key`, `operator: Exists`, `effect: NoSchedule` |

### 小技巧

1. **使用 Operator 管理**：設定 `taintKubeNodes: true` 讓 Operator 自動處理
2. **測試配置**：先在測試環境驗證 Taint/Toleration 配置
3. **監控 Pod 狀態**：定期檢查 Pod 是否正確分布
4. **文件記錄**：記錄哪些節點被 Taint 及其用途

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `kubectl taint nodes <node> key=value:effect` | 新增 Taint |
| `kubectl taint nodes <node> key:effect-` | 移除 Taint |
| `kubectl describe node <node> \| grep Taints` | 查看節點 Taint |
| `kubectl get nodes -o jsonpath='{.items[*].spec.taints}'` | 查看所有 Taint |
| `kubectl describe nodeset <name> -n slurm` | 查看 NodeSet 親和性配置 |
