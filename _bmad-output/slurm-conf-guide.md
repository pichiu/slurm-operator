# slurm.conf 配置指南

本文件說明 Slurm Operator 如何生成與管理 slurm.conf，以及如何透過 Helm 或 CRD 自訂配置。

---

## 概覽：Operator 如何生成 slurm.conf

Slurm Operator **自動生成** slurm.conf，由以下來源組合：

```
┌─────────────────────────────────────────────────────────────┐
│                      slurm.conf 結構                         │
├─────────────────────────────────────────────────────────────┤
│  1. GENERAL          ← 自動生成（ClusterName, SlurmUser...） │
│  2. LOGGING          ← 自動生成                              │
│  3. PLUGINS          ← 自動生成 + 條件化                     │
│  4. ACCOUNTING       ← 根據 Accounting CR 條件生成           │
│  5. PROLOG/EPILOG    ← 根據 Script Refs 條件生成             │
│  6. COMPUTE NODES    ← 根據 NodeSet CR 自動生成              │
│  7. PARTITIONS       ← 根據 NodeSet + Helm partitions 生成   │
│  8. EXTRA CONFIG     ← 用戶自定義（extraConf）               │
└─────────────────────────────────────────────────────────────┘
```

---

## 自動生成的 slurm.conf 範例

```ini
#
### GENERAL ###
ClusterName=mycluster
SlurmUser=slurm
SlurmctldHost=slinky-controller
StateSaveLocation=/var/spool/slurmctld
SlurmctldPidFile=/var/run/slurmctld.pid
SlurmdPidFile=/var/run/slurmd.pid
SlurmctldPort=6817
SlurmdPort=6818
SlurmdSpoolDir=/var/spool/slurmd
ReturnToService=2
#
### LOGGING ###
SlurmctldLogFile=/var/log/slurm/slurmctld.log
SlurmdLogFile=/var/log/slurm/slurmd.log
#
### PLUGINS & PARAMETERS ###
AuthType=auth/slurm
AuthAltTypes=auth/jwt
AuthAltParameters=jwt_key=/etc/slurm/jwt_hs256.key
CredType=cred/slurm
#
### PROCESS TRACKING ###
ProctrackType=proctrack/cgroup
TaskPlugin=task/cgroup,task/affinity
#
### SCHEDULING ###
SelectType=select/cons_tres
SelectTypeParameters=CR_CPU_Memory
SchedulerType=sched/backfill
#
### ACCOUNTING ###
AccountingStorageType=accounting_storage/slurmdbd
AccountingStorageHost=slinky-accounting
AccountingStoragePort=6819
AccountingStorageTRES=gres/gpu
#
### COMPUTE NODES ###
NodeSet=compute Feature=compute
NodeSet=gpu Feature=gpu
#
### PARTITIONS ###
PartitionName=compute Nodes=compute State=UP Default=YES MaxTime=UNLIMITED
PartitionName=gpu Nodes=gpu State=UP MaxTime=7-00:00:00 PriorityTier=100
#
### EXTRA CONFIG ###
MinJobAge=2
SchedulerParameters=defer,max_rpc_cnt=0
```

---

## 三種配置層級

| 層級 | 配置文件 | 用途 | 分隔符 |
|------|----------|------|--------|
| **Controller** | slurm.conf | 全局配置 | 換行 `\n` |
| **NodeSet** | slurmd `--conf` | 節點級配置 | 空格 |
| **Accounting** | slurmdbd.conf | 資料庫配置 | 換行 `\n` |

---

## 配置方式一：extraConf（原始字串）

### Controller extraConf

```yaml
# values.yaml
controller:
  extraConf: |
    MinJobAge=2
    MaxNodeCount=2048
    SlurmctldDebug=debug2
    SchedulerParameters=defer,max_rpc_cnt=0
    DefMemPerCPU=1024
    MaxMemPerCPU=8192
```

### NodeSet extraConf（空格分隔）

```yaml
nodesets:
  gpu:
    slurmd:
      extraConf: "Features=gpu,tesla Weight=10 Gres=gpu:nvidia:2"
```

### Accounting extraConf

```yaml
accounting:
  extraConf: |
    CommitDelay=1
    DebugLevel=debug2
    PurgeJobAfter=12month
    PurgeEventAfter=1month
```

---

## 配置方式二：extraConfMap（結構化）

### Controller extraConfMap

```yaml
controller:
  extraConfMap:
    MinJobAge: 2
    MaxNodeCount: 2048
    SlurmctldDebug: debug2
    # 列表會自動轉換為逗號分隔
    DebugFlags:
      - time
      - trace_events
    SchedulerParameters:
      - defer
      - max_rpc_cnt=0
```

### NodeSet extraConfMap

```yaml
nodesets:
  gpu:
    slurmd:
      extraConfMap:
        Features:
          - gpu
          - tesla
          - cuda
        Weight: 10
        Gres:
          - "gpu:nvidia:2"
```

---

## Partition 配置

### 方式一：透過 NodeSet 自動建立

每個 NodeSet 預設會建立一個同名的 Partition：

```yaml
nodesets:
  compute:
    enabled: true
    replicas: 10
    partition:
      enabled: true  # 預設為 true
      config: |
        State=UP
        Default=YES
        MaxTime=1-00:00:00
```

### 方式二：透過 Helm partitions 區塊（推薦）

可自訂名稱、將多個 NodeSet 關聯到同一個 Partition：

```yaml
# 關閉 NodeSet 自動建立 partition
nodesets:
  compute:
    partition:
      enabled: false
  gpu:
    partition:
      enabled: false
  highmem:
    partition:
      enabled: false

# 自訂 Partition 定義
partitions:
  # 包含所有節點
  all:
    enabled: true
    nodesets:
      - ALL  # 特殊關鍵字
    configMap:
      State: UP
      Default: "YES"
      MaxTime: UNLIMITED

  # 包含多個 NodeSet
  batch:
    enabled: true
    nodesets:
      - compute
      - highmem
    configMap:
      State: UP
      MaxTime: "7-00:00:00"

  # GPU 專用
  gpu-only:
    enabled: true
    nodesets:
      - gpu
    configMap:
      State: UP
      MaxTime: "1-00:00:00"
      PriorityTier: 100
```

---

## 完整 Helm values.yaml 範例

```yaml
# ============================================
# Controller 配置
# ============================================
controller:
  enabled: true

  slurmctld:
    image:
      repository: ghcr.io/slinkyproject/slurmctld
      tag: 25.05-rockylinux9

  # 方式 1: 原始字串（優先）
  extraConf: |
    # 調度器參數
    SchedulerType=sched/backfill
    SchedulerParameters=defer,max_rpc_cnt=0

    # 資源限制
    DefMemPerCPU=1024
    MaxMemPerCPU=8192
    MaxJobCount=50000

    # 作業預設值
    MinJobAge=300
    MaxJobAge=604800

  # 方式 2: 結構化（如果 extraConf 為空則使用）
  extraConfMap:
    DefMemPerCPU: 1024
    MaxMemPerCPU: 8192

# ============================================
# NodeSet 配置
# ============================================
nodesets:
  # 一般計算節點
  compute:
    enabled: true
    replicas: 50
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.05-rockylinux9
      extraConfMap:
        Weight: 1
    partition:
      enabled: false  # 使用下方的 partitions 設定

  # GPU 節點
  gpu:
    enabled: true
    replicas: 8
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.05-rockylinux9
      extraConf: "Features=gpu,tesla,cuda Weight=100 Gres=gpu:nvidia:4"
    partition:
      enabled: false

  # 高記憶體節點
  highmem:
    enabled: true
    replicas: 4
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.05-rockylinux9
      extraConfMap:
        Features:
          - highmem
          - largemem
        Weight: 50
    partition:
      enabled: false

# ============================================
# Partition 配置（自訂名稱 + 多 NodeSet）
# ============================================
partitions:
  # 預設 partition：包含所有節點
  all:
    enabled: true
    nodesets:
      - ALL
    configMap:
      State: UP
      Default: "YES"
      MaxTime: UNLIMITED

  # 一般計算
  batch:
    enabled: true
    nodesets:
      - compute
      - highmem
    configMap:
      State: UP
      MaxTime: "7-00:00:00"
      DefaultTime: "1:00:00"

  # GPU 專用
  gpu:
    enabled: true
    nodesets:
      - gpu
    config: |
      State=UP
      MaxTime=2-00:00:00
      PriorityTier=100
      PreemptMode=REQUEUE
      OverSubscribe=NO

  # 互動式短作業
  interactive:
    enabled: true
    nodesets:
      - compute
      - gpu
    configMap:
      State: UP
      MaxTime: "4:00:00"
      DefaultTime: "30:00"
      PriorityTier: 50

  # 長時間作業
  long:
    enabled: true
    nodesets:
      - compute
      - highmem
    configMap:
      State: UP
      MaxTime: "30-00:00:00"
      DefaultTime: "1-00:00:00"
      PriorityTier: 10

# ============================================
# Accounting 配置
# ============================================
accounting:
  enabled: true

  slurmdbd:
    image:
      repository: ghcr.io/slinkyproject/slurmdbd
      tag: 25.05-rockylinux9

  extraConf: |
    CommitDelay=1
    PurgeJobAfter=12month
    PurgeEventAfter=1month
    PurgeStepAfter=1month
    PurgeResvAfter=1month
    PurgeSuspendAfter=1month
    PurgeTXNAfter=12month
    PurgeUsageAfter=24month
```

---

## 產生的 slurm.conf 範例

```ini
#
### GENERAL ###
ClusterName=mycluster
SlurmUser=slurm
SlurmctldHost=slinky-controller
...

#
### COMPUTE NODES ###
NodeSet=compute Feature=compute
NodeSet=gpu Feature=gpu,tesla,cuda
NodeSet=highmem Feature=highmem,largemem

#
### PARTITIONS ###
PartitionName=all Nodes=compute,gpu,highmem State=UP Default=YES MaxTime=UNLIMITED
PartitionName=batch Nodes=compute,highmem State=UP MaxTime=7-00:00:00 DefaultTime=1:00:00
PartitionName=gpu Nodes=gpu State=UP MaxTime=2-00:00:00 PriorityTier=100 PreemptMode=REQUEUE OverSubscribe=NO
PartitionName=interactive Nodes=compute,gpu State=UP MaxTime=4:00:00 DefaultTime=30:00 PriorityTier=50
PartitionName=long Nodes=compute,highmem State=UP MaxTime=30-00:00:00 DefaultTime=1-00:00:00 PriorityTier=10

#
### EXTRA CONFIG ###
SchedulerType=sched/backfill
SchedulerParameters=defer,max_rpc_cnt=0
DefMemPerCPU=1024
MaxMemPerCPU=8192
MaxJobCount=50000
MinJobAge=300
MaxJobAge=604800
```

---

## 常用 slurm.conf 參數參考

### 調度相關

| 參數 | 說明 | 範例 |
|------|------|------|
| `SchedulerType` | 調度演算法 | `sched/backfill` |
| `SchedulerParameters` | 調度參數 | `defer,max_rpc_cnt=0` |
| `SelectType` | 資源選擇類型 | `select/cons_tres` |
| `PriorityType` | 優先級計算 | `priority/multifactor` |

### 資源限制

| 參數 | 說明 | 範例 |
|------|------|------|
| `DefMemPerCPU` | 預設每 CPU 記憶體 (MB) | `1024` |
| `MaxMemPerCPU` | 最大每 CPU 記憶體 (MB) | `8192` |
| `MaxJobCount` | 最大作業數 | `50000` |
| `MaxArraySize` | 最大陣列大小 | `10000` |

### 作業管理

| 參數 | 說明 | 範例 |
|------|------|------|
| `MinJobAge` | 作業完成後保留秒數 | `300` |
| `MaxJobAge` | 作業最大保留秒數 | `604800` |
| `KillWait` | Kill 等待秒數 | `30` |
| `RequeueExit` | 重新排隊的退出碼 | `1-255` |

### Partition 參數

| 參數 | 說明 | 範例 |
|------|------|------|
| `State` | 分區狀態 | `UP`, `DOWN`, `DRAIN` |
| `Default` | 預設分區 | `YES`, `NO` |
| `MaxTime` | 最大執行時間 | `7-00:00:00`, `UNLIMITED` |
| `DefaultTime` | 預設時間 | `1:00:00` |
| `PriorityTier` | 優先層級 | `1-65533` |
| `PreemptMode` | 搶占模式 | `OFF`, `REQUEUE`, `CANCEL` |
| `OverSubscribe` | 超額訂閱 | `NO`, `YES`, `FORCE` |

### 節點參數（NodeSet extraConf）

| 參數 | 說明 | 範例 |
|------|------|------|
| `Features` | 節點特徵標籤 | `gpu,tesla,cuda` |
| `Weight` | 調度權重 | `1-65533` |
| `Gres` | 通用資源 | `gpu:nvidia:4` |

---

## 程式碼位置參考

| 功能 | 檔案路徑 |
|------|----------|
| Controller slurm.conf 生成 | `internal/builder/controller_config.go:155-287` |
| NodeSet --conf 生成 | `internal/builder/worker_app.go:285-332` |
| Accounting slurmdbd.conf 生成 | `internal/builder/accounting_config.go:37-80` |
| Helm extraConf 處理 | `helm/slurm/templates/controller/_helpers.tpl:30-72` |
| Helm partitions 處理 | `helm/slurm/templates/controller/_helpers.tpl:43-70` |

---

## 相關文件

- [NodeSet API 參考](nodeset-api-reference.md)
- [Helm NodeSet 管理指南](helm-nodeset-guide.md)
- [Slurm 使用指南](slurm-usage-guide.md)
- [Slurm FAQ](slurm-faq.md)
- [Helm Deep Dive](deep-dive-helm.md)
