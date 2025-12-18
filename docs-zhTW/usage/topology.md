# 拓撲 (Topology)

## TL;DR

Operator 可以將拓撲資訊從 Kubernetes 節點傳播到 Slurm 節點。當 NodeSet Pod 在 Kubernetes 節點上執行時，其註解 (Annotations) 會被 Operator 用來更新 Slurm 節點的拓撲資訊。這需要正確配置 `topology.yaml` 和 Kubernetes 節點註解。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [拓撲](#拓撲-topology)
  - [目錄](#目錄)
  - [概述](#概述)
  - [Kubernetes](#kubernetes)
  - [Slurm](#slurm)
  - [範例](#範例)

<!-- mdformat-toc end -->

## 概述

Operator 可以將拓撲 (Topology) 從 Kubernetes 節點傳播到 Slurm 節點（那些作為 NodeSet Pod 執行的節點）。當 NodeSet Pod 在 Kubernetes 節點上執行時，其註解 (Annotations) 會被 Operator 用來更新已註冊 Slurm 節點的拓撲。動態拓撲需要拓撲檔案才能運作。

如果 `topology.yaml` 或 Kubernetes 節點註解配置錯誤，錯誤會在 Operator 日誌中回報。

## Kubernetes

每個 Kubernetes 節點應該用 `topology.slinky.slurm.net/line` 進行註解。其值會被 Operator 透明地用來更新 Slurm 節點拓撲資訊，僅在作為 NodeSet Pod 執行時。

例如，以下 Kubernetes Node 片段已套用 Slinky 拓撲註解。

```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    topology.slinky.slurm.net/line: topo-switch:s0,topo-block:b0
  name: node0
```

## Slurm

Slurm 支援 [topology.yaml]，一個基於 YAML 的配置檔，能夠在同一個 Slurm 叢集中表達一個或多個拓撲配置。

請查看 [Slurm 拓撲指南][topology-guide]。

## 範例

例如，您的 Slurm 叢集有以下 `topology.yaml`。

```yaml
---
- topology: topo-switch
  cluster_default: true
  tree:
    switches:
      - switch: sw_root
        children: s[1-2]
      - switch: s1
        nodes: node[1-2]
      - switch: s2
        nodes: node[3-4]
- topology: topo-block
  cluster_default: false
  block:
    block_sizes:
      - 2
      - 4
    blocks:
      - block: b1
        nodes: node[1-2]
      - block: b2
        nodes: node[3-4]
- topology: topo-flat
  cluster_default: false
  flat: true
```

而您的 Kubernetes 節點被如此註解以匹配 `topology.yaml`。

```yaml
---
apiVersion: v1
kind: Node
metadata:
  annotations:
    topology.slinky.slurm.net/line: topo-switch:s1,topo-block:b1
  name: node1
---
apiVersion: v1
kind: Node
metadata:
  annotations:
    topology.slinky.slurm.net/line: topo-switch:s1,topo-block:b1
  name: node2
---
apiVersion: v1
kind: Node
metadata:
  annotations:
    topology.slinky.slurm.net/line: topo-switch:s2,topo-block:b2
  name: node3
---
apiVersion: v1
kind: Node
metadata:
  annotations:
    topology.slinky.slurm.net/line: topo-switch:s2,topo-block:b2
  name: node4
```

然後當 `slinky-0` NodeSet Pod 被排程到 Kubernetes 節點 `node3` 時，Operator 會更新 Slurm 節點的拓撲以匹配 `topology.slinky.slurm.net/line`。因此在 Slurm 節點的拓撲更新後，Slurm 會回報以下內容。

```sh
$ scontrol show nodes slinky-0 | grep -Eo "NodeName=[^ ]+|[ ]*Comment=[^ ]+|[ ]*Topology=[^ ]+"
NodeName=slinky-0
   Comment={"namespace":"slurm","podName":"slurm-worker-slinky-0","node":"node3"}
   Topology=topo-switch:s2,topo-block:b2
```

<!-- Links -->

[topology-guide]: https://slurm.schedmd.com/topology.html
[topology.yaml]: https://slurm.schedmd.com/topology.yaml.html

---

## Explanation

### 什麼是 Slurm 拓撲？

Slurm 拓撲功能允許排程器考慮叢集的物理網路結構，以：
- 最佳化作業放置
- 減少網路跳躍數
- 提高通訊效率

### 拓撲類型

| 類型 | 說明 |
|-----|------|
| **tree (樹狀)** | 模擬交換器層次結構 |
| **block (區塊)** | 定義節點群組的效能特性 |
| **flat (扁平)** | 所有節點視為同等距離 |

### 動態拓撲的運作方式

1. Kubernetes 節點有拓撲註解
2. NodeSet Pod 被排程到某個 Kubernetes 節點
3. Operator 讀取該節點的拓撲註解
4. Operator 透過 REST API 更新 Slurm 節點的拓撲
5. Slurm 排程器使用更新後的拓撲資訊

---

## Practical Example

### 設定 Kubernetes 節點拓撲註解

```bash
# 為單一節點新增拓撲註解
kubectl annotate node node1 \
  topology.slinky.slurm.net/line="topo-switch:s1,topo-block:b1"

# 為多個節點批次新增註解
for i in 1 2; do
  kubectl annotate node node$i \
    topology.slinky.slurm.net/line="topo-switch:s1,topo-block:b1"
done

for i in 3 4; do
  kubectl annotate node node$i \
    topology.slinky.slurm.net/line="topo-switch:s2,topo-block:b2"
done

# 驗證註解
kubectl get nodes -o custom-columns=\
"NAME:.metadata.name,\
TOPOLOGY:.metadata.annotations.topology\.slinky\.slurm\.net/line"
```

### 配置 Slurm topology.yaml

```yaml
# 在 values.yaml 中配置拓撲
configFiles:
  topology.yaml: |
    ---
    # 樹狀拓撲：模擬交換器結構
    - topology: topo-switch
      cluster_default: true
      tree:
        switches:
          - switch: spine           # 核心交換器
            children: leaf[1-4]     # 葉交換器
          - switch: leaf1
            nodes: node[1-8]        # 連接到 leaf1 的節點
          - switch: leaf2
            nodes: node[9-16]
          - switch: leaf3
            nodes: node[17-24]
          - switch: leaf4
            nodes: node[25-32]

    # 區塊拓撲：定義效能群組
    - topology: topo-block
      cluster_default: false
      block:
        block_sizes:
          - 8
          - 32
        blocks:
          - block: rack1
            nodes: node[1-8]
          - block: rack2
            nodes: node[9-16]
          - block: rack3
            nodes: node[17-24]
          - block: rack4
            nodes: node[25-32]
```

### 驗證拓撲配置

```bash
# 安裝 Slurm 叢集
helm upgrade --install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values-topology.yaml \
  --namespace=slurm --create-namespace

# 等待 NodeSet Pod 就緒
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=slurmd -n slurm --timeout=300s

# 檢查 Slurm 節點的拓撲
kubectl exec -n slurm statefulset/slurm-controller -- \
  scontrol show nodes | grep -E "NodeName|Topology"

# 查看特定節點的詳細資訊
kubectl exec -n slurm statefulset/slurm-controller -- \
  scontrol show node slurm-worker-slinky-0
```

### 排錯

```bash
# 檢查 Operator 日誌中的拓撲相關訊息
kubectl logs -n slinky deployment/slurm-operator | grep -i topology

# 驗證 Kubernetes 節點註解
kubectl describe node node1 | grep -A 5 "Annotations"

# 檢查 NodeSet Pod 所在的 Kubernetes 節點
kubectl get pods -n slurm -o wide | grep slurm-worker
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| 拓撲未更新 | 確認節點註解格式正確 |
| Operator 日誌顯示錯誤 | 檢查 topology.yaml 和註解的對應關係 |
| Slurm 節點顯示空拓撲 | 驗證 NodeSet Pod 已正確排程 |
| 配置不匹配 | 確保註解中的拓撲名稱與 topology.yaml 一致 |

### 註解格式

正確格式：`topo-name:element,topo-name2:element2`

```
# 正確
topology.slinky.slurm.net/line: topo-switch:s1,topo-block:b1

# 錯誤（缺少拓撲名稱）
topology.slinky.slurm.net/line: s1,b1

# 錯誤（格式錯誤）
topology.slinky.slurm.net/line: topo-switch=s1
```

### 小技巧

1. **測試配置**：先在小規模環境驗證拓撲配置
2. **保持一致**：確保所有相關節點都有正確的註解
3. **監控日誌**：觀察 Operator 日誌以確認拓撲更新
4. **使用標籤輔助**：可以結合 Kubernetes 標籤來管理節點群組

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `kubectl annotate node <name> topology.slinky.slurm.net/line=<value>` | 設定節點拓撲註解 |
| `kubectl get nodes -o jsonpath='{.items[*].metadata.annotations}'` | 查看所有節點註解 |
| `scontrol show node <name>` | 查看 Slurm 節點拓撲 |
| `scontrol show topology` | 查看 Slurm 拓撲配置 |
| `kubectl logs -n slinky deployment/slurm-operator \| grep topology` | 查看拓撲相關日誌 |
