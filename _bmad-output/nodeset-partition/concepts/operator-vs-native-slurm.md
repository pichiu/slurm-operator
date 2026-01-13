# slurm-operator 與原生 Slurm 的架構差異

> 本文深入分析 slurm-operator 與原生 Slurm（scontrol）在 Partition 管理上的根本差異，幫助你理解設計權衡並選擇適合的管理方式。

## 快速參考

| 面向 | 原生 Slurm (scontrol) | slurm-operator |
|------|----------------------|----------------|
| **通訊方式** | 直接 RPC 到 slurmctld | 透過 ConfigMap + reconfigure |
| **生效時間** | 即時（毫秒級） | 延遲（1-2 分鐘） |
| **變更模式** | 命令式（Imperative） | 宣告式（Declarative） |
| **狀態持久性** | 記憶體 + state 檔案 | Kubernetes etcd |
| **觸發機制** | 直接觸發內部元件 | 間接透過 `scontrol reconfigure` |

## 核心架構差異

### 原生 Slurm 的 RPC 機制

當你執行 `scontrol create partition` 時，會發生以下流程：

```mermaid
sequenceDiagram
    autonumber
    participant Admin as 管理員
    participant SC as scontrol
    participant API as libslurm API
    participant RPC as RPC 層
    participant CTLD as slurmctld

    Admin->>SC: scontrol create partition PartitionName=test
    SC->>SC: 解析參數
    SC->>API: slurm_create_partition(&part_msg)
    API->>RPC: 序列化 + 發送 RPC
    Note over RPC: REQUEST_CREATE_PARTITION (3003)
    RPC->>CTLD: TCP 傳輸

    rect rgb(230, 255, 230)
        Note over CTLD: slurmctld 內部處理
        CTLD->>CTLD: validate_super_user()
        CTLD->>CTLD: lock_slurmctld()
        CTLD->>CTLD: update_part() - 建立 Partition
        CTLD->>CTLD: build_part_bitmap()
        CTLD->>CTLD: gs_reconfig() - Gang Scheduler
        CTLD->>CTLD: select_g_reconfigure()
        CTLD->>CTLD: unlock_slurmctld()
    end

    CTLD-->>RPC: SLURM_SUCCESS
    RPC-->>API: 回應結果
    API-->>SC: 返回狀態碼
    SC-->>Admin: 完成（毫秒內）
```

**關鍵特點**：

- 直接透過 `libslurm` API 發送 RPC 訊息（訊息類型 3003）
- slurmctld 直接在記憶體中建立 `part_record_t` 結構
- 同步觸發多個內部元件（Gang Scheduler、Select Plugin 等）
- 整個過程在毫秒內完成

### slurm-operator 的間接機制

相對地，slurm-operator 必須透過「配置檔案」來間接控制 Slurm：

```mermaid
sequenceDiagram
    autonumber
    participant User as 使用者
    participant K8S as Kubernetes API
    participant RC as Controller Reconciler
    participant BLD as Builder
    participant CM as ConfigMap
    participant KL as kubelet
    participant RS as Reconfigure Sidecar
    participant CTLD as slurmctld

    User->>K8S: kubectl apply NodeSet
    K8S->>RC: NodeSet Watch 事件
    RC->>RC: Reconcile() 被觸發
    RC->>BLD: BuildControllerConfig()
    BLD->>BLD: 產生 slurm.conf 內容
    BLD->>CM: Patch ConfigMap

    rect rgb(255, 245, 230)
        Note over CM,KL: ConfigMap 傳播延遲
        CM->>CM: 內容更新
        Note over KL: 等待 kubelet sync period
        KL->>KL: 偵測 ConfigMap 變更
        KL->>RS: 更新 Volume 內容
    end

    rect rgb(230, 245, 255)
        Note over RS,CTLD: Reconfigure 流程
        RS->>RS: 每 5 秒檢查 hash
        RS->>RS: 偵測到變更
        RS->>CTLD: scontrol reconfigure
        CTLD->>CTLD: 重新載入整份 slurm.conf
    end

    Note over User,CTLD: 總延遲：1-2 分鐘
```

**關鍵差異**：

- 無法直接呼叫 `libslurm` API（Kubernetes controller 不是 C 程式）
- 只能透過修改 `slurm.conf` ConfigMap 來間接控制
- 需要經過 kubelet 同步週期（預設 1 分鐘）
- 最終透過 `scontrol reconfigure` 觸發重載

## 為什麼有這些差異？

### 1. 技術架構限制

```mermaid
flowchart TB
    subgraph native["原生 Slurm 生態"]
        C["C 語言程式"]
        LIB["libslurm.so"]
        RPC["Slurm RPC 協議"]
        C --> LIB --> RPC
    end

    subgraph k8s["Kubernetes 生態"]
        GO["Go 語言程式"]
        CR["controller-runtime"]
        API["Kubernetes API"]
        GO --> CR --> API
    end

    native ~~~ k8s

    style native fill:#e8f5e9
    style k8s fill:#e3f2fd
```

- **原生 Slurm**：使用 C 語言撰寫，透過 `libslurm` 共享庫與 daemon 通訊
- **slurm-operator**：使用 Go 語言撰寫，遵循 Kubernetes controller 模式
- Go 程式無法直接呼叫 C 的 `libslurm` API（除非使用 cgo，但會增加複雜性）

### 2. Kubernetes 的設計哲學

Kubernetes 推崇**宣告式配置**（Declarative Configuration）：

```mermaid
flowchart LR
    subgraph imperative["命令式 (Imperative)"]
        I1["scontrol create partition..."]
        I2["scontrol update partition..."]
        I3["scontrol delete partition..."]
    end

    subgraph declarative["宣告式 (Declarative)"]
        D1["定義期望狀態"]
        D2["Reconciler 比較"]
        D3["自動收斂"]
        D1 --> D2 --> D3
    end

    style imperative fill:#fff3e0
    style declarative fill:#e3f2fd
```

**宣告式的優點**：

- GitOps 友善：配置可以版本控制
- 自動修復：Reconciler 持續確保狀態一致
- 可審計：所有變更都有記錄

**代價**：

- 失去即時性
- 無法做細粒度的動態調整

### 3. 安全邊界考量

```mermaid
flowchart TD
    subgraph k8s["Kubernetes 叢集"]
        OP["slurm-operator<br/>（無特權）"]
        CM["ConfigMap"]
        POD["Controller Pod"]
    end

    subgraph slurm["Slurm 運行時"]
        CTLD["slurmctld<br/>（Pod 內）"]
        CONF["/etc/slurm/slurm.conf"]
    end

    OP -->|"只能修改"| CM
    CM -->|"Volume Mount"| CONF
    CONF -->|"slurmctld 讀取"| CTLD

    style OP fill:#ffcdd2
    style CTLD fill:#c8e6c9
```

slurm-operator 作為 Kubernetes controller，受限於：

- 無法直接存取 slurmctld 的 Unix socket
- 無法發送 RPC 到 slurmctld
- 只能透過 Kubernetes 原生機制（ConfigMap）傳遞配置

## slurm-operator 的侷限

### 侷限一覽

```mermaid
flowchart TD
    subgraph limitations["slurm-operator 侷限"]
        L1["無法即時變更<br/>必須等 ConfigMap 同步"]
        L2["無法增量修改節點<br/>不支援 +/- 語法"]
        L3["無法直接觸發內部元件<br/>如 Gang Scheduler"]
        L4["必須整份 slurm.conf 重載<br/>無法只更新單一 Partition"]
        L5["Partition 必須綁定 NodeSet<br/>或使用 extraConf"]
    end

    style L1 fill:#ffcdd2
    style L2 fill:#ffcdd2
    style L3 fill:#ffcdd2
    style L4 fill:#ffcdd2
    style L5 fill:#ffcdd2
```

### 詳細說明

| 侷限 | 原因 | 實際影響 |
|------|------|----------|
| **無法即時變更** | ConfigMap propagation 需 1-2 分鐘 | 緊急操作（如 drain partition）無法即時生效 |
| **無法增量修改** | slurm.conf 是完整配置，非增量 | 每次變更都需重建完整節點列表 |
| **無法獨立建立 Partition** | Partition 透過 NodeSet 管理 | 無法建立跨多個 NodeSet 的 Partition |
| **無法直接觸發** | 只能透過 `scontrol reconfigure` | 無法精確控制觸發哪些元件 |
| **重載成本較高** | `scontrol reconfigure` 重載整份配置 | 大型叢集可能有效能影響 |

### 對比：節點增量修改

**原生 Slurm**：支援 `+/-` 語法

```bash
# 增加節點到 partition
scontrol update partitionname=compute nodes=+node[005-010]

# 從 partition 移除節點
scontrol update partitionname=compute nodes=-node[001-002]
```

**slurm-operator**：必須指定完整列表

```yaml
# 無法使用 +/- 語法，必須在 NodeSet replicas 調整
spec:
  replicas: 10  # 從 8 改為 10
```

### 對比：觸發的內部元件

**原生 `scontrol create partition`** 直接觸發：

| 元件 | 函數 | 時機 |
|------|------|------|
| 位圖管理 | `build_part_bitmap()` | 同步 |
| TRES 計算 | `_calc_part_tres()` | 同步 |
| Gang Scheduler | `gs_reconfig()` | 同步 |
| Select Plugin | `select_g_reconfigure()` | 同步 |
| 狀態儲存 | `schedule_part_save()` | 非同步 |
| 作業排程 | `queue_job_scheduler()` | 非同步 |

**slurm-operator** 透過 `scontrol reconfigure`：

| 元件 | 說明 |
|------|------|
| 全部 | 重新讀取整份 slurm.conf，觸發所有相關元件 |

無法選擇性觸發特定元件。

## slurm-operator 可用的更新手段

slurm-operator 只能透過以下方式更新 Slurm 配置：

```mermaid
flowchart LR
    subgraph methods["更新手段"]
        M1["1. NodeSet CRD<br/>partition.enabled<br/>partition.config"]
        M2["2. Controller CRD<br/>extraConf"]
        M3["3. Helm Values<br/>（轉換為上述）"]
    end

    subgraph result["最終路徑"]
        R1["slurm.conf 更新"]
        R2["ConfigMap Patch"]
        R3["scontrol reconfigure"]
    end

    M1 --> R1
    M2 --> R1
    M3 --> R1
    R1 --> R2 --> R3

    style M1 fill:#e3f2fd
    style M2 fill:#e3f2fd
    style M3 fill:#e3f2fd
    style R3 fill:#fff3e0
```

### 方法 1：NodeSet CRD - Partition 設定

最標準的方式，一個 NodeSet 對應一個 Partition：

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: compute
spec:
  controllerRef:
    name: slurm
  replicas: 8
  partition:
    enabled: true
    config: "Default=YES MaxTime=7-00:00:00 State=UP"
```

**產生的 slurm.conf**：

```conf
NodeSet=compute Feature=compute
PartitionName=compute Nodes=compute Default=YES MaxTime=7-00:00:00 State=UP
```

### 方法 2：Controller CRD - extraConf

適合複雜的 Partition 拓撲或需要跨 NodeSet 的設定：

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: Controller
metadata:
  name: slurm
spec:
  extraConf: |
    # 跨多個 NodeSet 的 Partition
    PartitionName=all Nodes=ALL Default=NO

    # 指定特定 NodeSet 組合
    PartitionName=highprio Nodes=compute,gpu MaxTime=1-00:00:00 Priority=100
```

### 方法 3：Helm Values

透過 Helm 部署時設定：

```yaml
# values.yaml
nodeSets:
  compute:
    replicas: 8
    partition:
      enabled: true
      config: "Default=YES"

  gpu:
    replicas: 2
    partition:
      enabled: true
      config: "MaxTime=1-00:00:00"

controller:
  extraConf: |
    PartitionName=debug Nodes=compute MaxTime=00:30:00
```

## 使用場景建議

根據不同需求選擇適合的管理方式：

```mermaid
flowchart TD
    A{使用場景?} --> B[標準配置管理]
    A --> C[緊急操作]
    A --> D[複雜 Partition 拓撲]
    A --> E[動態調整]

    B --> B1["推薦：slurm-operator<br/>透過 NodeSet CRD"]
    C --> C1["推薦：scontrol<br/>即時生效"]
    D --> D1["推薦：Controller extraConf<br/>突破 1:1 限制"]
    E --> E1["推薦：scontrol<br/>避免重載影響"]

    style B1 fill:#c8e6c9
    style C1 fill:#fff3e0
    style D1 fill:#e3f2fd
    style E1 fill:#fff3e0
```

| 場景 | 推薦方式 | 原因 |
|------|----------|------|
| 標準配置管理 | slurm-operator | 宣告式、GitOps 友善、持久化 |
| 緊急操作 | scontrol | 即時生效 |
| 複雜 Partition 拓撲 | extraConf | 突破 1:1 NodeSet-Partition 限制 |
| 動態調整（作業執行中） | scontrol | 避免重載影響 |

## 變通方案：混合使用

對於需要即時操作的場景，可以結合使用兩種方式：

```bash
# 步驟 1：緊急操作 - 使用 scontrol 即時生效
kubectl exec <controller-pod> -c slurmctld -- \
    scontrol update partitionname=compute state=drain

# 步驟 2：確保持久性 - 同時更新 CRD
kubectl patch nodeset compute --type=merge -p '
spec:
  partition:
    config: "State=DRAIN"
'
```

**注意事項**：

- scontrol 的變更在 Pod 重啟後會遺失
- 務必同步更新 CRD 以確保持久性
- 當 CRD 變更觸發 reconcile 時，會以 CRD 定義為準

## 總結

```mermaid
flowchart LR
    subgraph tradeoff["設計權衡"]
        direction TB
        T1["即時性"] <--> T2["持久性"]
        T3["細粒度控制"] <--> T4["宣告式管理"]
        T5["直接存取"] <--> T6["安全隔離"]
    end

    subgraph native["原生 Slurm"]
        N1["✅ 即時性"]
        N2["✅ 細粒度控制"]
        N3["✅ 直接存取"]
    end

    subgraph operator["slurm-operator"]
        O1["✅ 持久性"]
        O2["✅ 宣告式管理"]
        O3["✅ 安全隔離"]
    end

    style native fill:#e8f5e9
    style operator fill:#e3f2fd
```

slurm-operator 的設計是為了 **Kubernetes 原生整合** 和 **宣告式管理**，這是有意識的架構選擇，而非技術缺陷。理解這些權衡後，你可以：

1. 日常管理使用 slurm-operator（GitOps、可審計、自動修復）
2. 緊急情況使用 scontrol（即時生效）
3. 複雜拓撲使用 extraConf（突破限制）

## 延伸閱讀

- [Partition 基礎概念](./partition-fundamentals.md) - 了解 Partition 與 NodeSet 的關係
- [Reconcile 流程](../architecture/operator-reconcile-flow.md) - 深入了解 Operator 如何更新配置
- [API 能力分析](../management/api-capabilities.md) - 各介面的 CRUD 能力對比
