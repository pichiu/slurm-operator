# NodeSet API 完整參考

> 最後更新：2025-12-23
> API 版本：`slinky.slurm.net/v1beta1`
> 相關文件：[使用指南](./slurm-usage-guide.md) | [FAQ](./slurm-faq.md) | [Helm 管理指南](./helm-nodeset-guide.md)

---

## 目錄

- [概述](#概述)
- [完整 YAML 範例](#完整-yaml-範例)
- [Spec 欄位詳解](#spec-欄位詳解)
  - [基本設定](#基本設定)
  - [容器設定 (slurmd)](#容器設定-slurmd)
  - [SSH 設定](#ssh-設定)
  - [分區設定 (partition)](#分區設定-partition)
  - [Pod 範本 (template)](#pod-範本-template)
  - [儲存設定 (volumeClaimTemplates)](#儲存設定-volumeclaimtemplates)
  - [更新策略 (updateStrategy)](#更新策略-updatestrategy)
  - [進階設定](#進階設定)
- [Status 欄位說明](#status-欄位說明)
- [常用設定範例](#常用設定範例)

---

## 概述

**NodeSet** 是 Slurm Operator 中用於管理計算節點（slurmd）的 Custom Resource。每個 NodeSet 代表一組同質的 Slurm 計算節點。

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
```

**短名稱**：`nodesets`, `nss`, `slurmd`

```bash
# 以下指令等效
kubectl get nodeset
kubectl get nodesets
kubectl get nss
kubectl get slurmd
```

---

## 完整 YAML 範例

以下是包含所有可設定欄位的完整範例：

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: compute
  namespace: slurm
spec:
  # === 基本設定 ===
  controllerRef:
    name: slinky
    namespace: slurm
  replicas: 3

  # === 容器設定 ===
  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.05-rockylinux9
    resources:
      requests:
        cpu: "1"
        memory: "2Gi"
      limits:
        cpu: "4"
        memory: "8Gi"
    env:
      - name: SLURMD_DEBUG
        value: "verbose"

  # === SSH 設定 ===
  ssh:
    enabled: false
    extraSshdConfig: |
      PermitRootLogin no
    sssdConfRef:
      name: slurm-sssd-conf
      key: sssd.conf

  # === 日誌邊車 ===
  logfile:
    image: busybox:latest
    resources:
      requests:
        cpu: "100m"
        memory: "64Mi"

  # === 分區設定 ===
  partition:
    enabled: true
    config: |
      State=UP
      MaxTime=UNLIMITED
      Default=YES

  # === 額外 Slurm 配置 ===
  extraConf: |
    RealMemory=8000
    Sockets=2
    CoresPerSocket=4
    ThreadsPerCore=2

  # === Pod 範本 ===
  template:
    metadata:
      labels:
        app.kubernetes.io/component: worker
      annotations:
        prometheus.io/scrape: "true"
    spec:
      nodeSelector:
        node-type: compute
      tolerations:
        - key: "slurm-worker"
          operator: "Exists"
          effect: "NoSchedule"
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    slinky.slurm.net/component: slurmd
                topologyKey: kubernetes.io/hostname

  # === 儲存設定 ===
  volumeClaimTemplates:
    - metadata:
        name: scratch
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: fast-ssd
        resources:
          requests:
            storage: 100Gi

  # === 更新策略 ===
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1

  # === 進階設定 ===
  revisionHistoryLimit: 3
  minReadySeconds: 10
  taintKubeNodes: false
  workloadDisruptionProtection: true

  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Retain
    whenScaled: Delete
```

---

## Spec 欄位詳解

### 基本設定

| 欄位 | 類型 | 必填 | 預設值 | 說明 |
|------|------|------|--------|------|
| `controllerRef` | ObjectReference | ✅ | - | 參考的 Controller CR |
| `controllerRef.name` | string | ✅ | - | Controller 名稱 |
| `controllerRef.namespace` | string | ❌ | 同 NodeSet | Controller 的 namespace |
| `replicas` | *int32 | ❌ | 1 | Worker Pod 數量 |

```yaml
spec:
  controllerRef:
    name: slinky
    namespace: slurm
  replicas: 5
```

### 容器設定 (slurmd)

`slurmd` 欄位包裝了標準的 Kubernetes `corev1.Container`，支援所有容器配置選項。

| 欄位 | 類型 | 說明 |
|------|------|------|
| `image` | string | 容器映像 |
| `resources` | ResourceRequirements | CPU/Memory 請求與限制 |
| `env` | []EnvVar | 環境變數 |
| `envFrom` | []EnvFromSource | 從 ConfigMap/Secret 載入環境變數 |
| `volumeMounts` | []VolumeMount | 掛載點 |
| `ports` | []ContainerPort | 額外暴露的端口 |
| `securityContext` | SecurityContext | 安全上下文 |

```yaml
spec:
  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.05-rockylinux9
    resources:
      requests:
        cpu: "2"
        memory: "4Gi"
        nvidia.com/gpu: "1"    # GPU 請求
      limits:
        cpu: "4"
        memory: "8Gi"
        nvidia.com/gpu: "1"
    env:
      - name: SLURMD_OPTIONS
        value: "-D"
    volumeMounts:
      - name: shared-data
        mountPath: /data
```

### SSH 設定

控制是否允許 SSH 連入 Worker Pod（需搭配 `pam_slurm_adopt`）。

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `ssh.enabled` | bool | false | 啟用 SSH |
| `ssh.extraSshdConfig` | string | - | 額外的 sshd_config 設定 |
| `ssh.sssdConfRef` | SecretKeySelector | - | SSSD 配置檔參考 |

```yaml
spec:
  ssh:
    enabled: true
    extraSshdConfig: |
      PermitRootLogin no
      PasswordAuthentication yes
    sssdConfRef:
      name: slurm-sssd-conf
      key: sssd.conf
```

**注意**：啟用 SSH 時，只有在該節點上有運行作業的使用者才能 SSH 進入（由 `pam_slurm_adopt` 控制）。

### 分區設定 (partition)

定義此 NodeSet 對應的 Slurm 分區配置。

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `partition.enabled` | bool | true | 是否建立分區 |
| `partition.config` | string | - | 分區配置參數 |

```yaml
spec:
  partition:
    enabled: true
    config: |
      State=UP
      MaxTime=7-00:00:00
      Default=NO
      PriorityTier=10
      OverSubscribe=FORCE:1
```

**常用分區參數**：

| 參數 | 說明 | 範例 |
|------|------|------|
| `State` | 分區狀態 | `UP`, `DOWN`, `DRAIN`, `INACTIVE` |
| `MaxTime` | 最大執行時間 | `1-00:00:00` (1天), `UNLIMITED` |
| `Default` | 是否為預設分區 | `YES`, `NO` |
| `PriorityTier` | 優先順序層級 | 數字越大優先 |
| `OverSubscribe` | 超額訂閱模式 | `NO`, `YES`, `FORCE` |

### Pod 範本 (template)

自訂 Worker Pod 的 metadata 和 spec。

```yaml
spec:
  template:
    metadata:
      labels:
        environment: production
      annotations:
        backup.velero.io/backup-volumes: scratch
    spec:
      # 節點選擇器
      nodeSelector:
        node-type: gpu-compute

      # 容忍度
      tolerations:
        - key: "nvidia.com/gpu"
          operator: "Exists"
          effect: "NoSchedule"

      # 親和性
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: gpu-type
                    operator: In
                    values: ["a100", "h100"]

      # 額外 Volume
      volumes:
        - name: shared-nfs
          nfs:
            server: nfs.example.com
            path: /exports/slurm

      # 優先級
      priorityClassName: high-priority

      # 服務帳戶
      serviceAccountName: slurm-worker
```

### 儲存設定 (volumeClaimTemplates)

為每個 Worker Pod 建立獨立的 PVC。

```yaml
spec:
  volumeClaimTemplates:
    - metadata:
        name: scratch
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: fast-ssd
        resources:
          requests:
            storage: 500Gi
    - metadata:
        name: logs
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: standard
        resources:
          requests:
            storage: 10Gi
```

**注意**：PVC 名稱會自動加上 Pod 名稱後綴，例如：`scratch-slurm-worker-compute-0`

### 更新策略 (updateStrategy)

控制 NodeSet 更新時的行為。

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `updateStrategy.type` | string | RollingUpdate | 更新類型 |
| `updateStrategy.rollingUpdate.maxUnavailable` | IntOrString | 1 | 最大不可用數量 |

**更新類型**：

| 類型 | 說明 |
|------|------|
| `RollingUpdate` | 滾動更新，逐一替換 Pod |
| `OnDelete` | 手動刪除 Pod 後才會用新版本重建 |

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 2      # 最多同時 2 個 Pod 不可用
      # 或使用百分比
      # maxUnavailable: "25%"
```

### 進階設定

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `extraConf` | string | - | 額外的 slurmd 配置參數 |
| `revisionHistoryLimit` | *int32 | 0 | 保留的歷史版本數量 |
| `minReadySeconds` | int32 | 0 | Pod 就緒後等待秒數 |
| `taintKubeNodes` | bool | false | 是否污染 K8s 節點 |
| `workloadDisruptionProtection` | bool | true | 工作負載保護（PDB） |

#### extraConf

傳遞給 slurmd 的額外節點配置：

```yaml
spec:
  extraConf: |
    RealMemory=64000
    Sockets=2
    CoresPerSocket=16
    ThreadsPerCore=2
    Gres=gpu:nvidia:4
```

#### taintKubeNodes

當設為 `true` 時，運行 NodeSet Pod 的 K8s 節點會被加上 NoExecute 污點，防止其他 Pod 調度到該節點。

```yaml
spec:
  taintKubeNodes: true
```

#### workloadDisruptionProtection

當設為 `true`（預設）時，正在執行 Slurm 作業的 Pod 會受到 PodDisruptionBudget 保護，避免被意外中斷。

```yaml
spec:
  workloadDisruptionProtection: true
```

#### PVC 保留策略

控制 PVC 在 NodeSet 刪除或縮容時的行為：

```yaml
spec:
  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Retain    # NodeSet 刪除時：Retain 或 Delete
    whenScaled: Delete     # 縮容時：Retain 或 Delete
```

---

## Status 欄位說明

NodeSet 的 status 欄位由 Controller 自動更新：

```yaml
status:
  # Pod 統計
  replicas: 5              # 總 Pod 數
  updatedReplicas: 5       # 已更新的 Pod 數
  readyReplicas: 5         # 就緒的 Pod 數
  availableReplicas: 5     # 可用的 Pod 數
  unavailableReplicas: 0   # 不可用的 Pod 數

  # Slurm 節點狀態
  slurmIdle: 3             # IDLE 節點（閒置）
  slurmAllocated: 2        # ALLOCATED/MIXED 節點（執行作業中）
  slurmDown: 0             # DOWN 節點（不可用）
  slurmDrain: 0            # DRAIN 節點（排空中）

  # 版本控制
  observedGeneration: 3    # 觀察到的世代
  nodeSetHash: "abc123"    # 當前配置雜湊
  collisionCount: 0        # 雜湊碰撞計數

  # 條件
  conditions:
    - type: Available
      status: "True"
      reason: MinimumReplicasAvailable
      message: "NodeSet has minimum availability"

  # HPA 支援
  selector: "slinky.slurm.net/nodeset=compute"
```

**查看狀態**：

```bash
# 簡要狀態
kubectl get nodeset -n slurm

# 詳細狀態
kubectl get nodeset compute -n slurm -o wide

# 完整 YAML
kubectl get nodeset compute -n slurm -o yaml
```

---

## 常用設定範例

### 基本 CPU 叢集

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: cpu-workers
  namespace: slurm
spec:
  controllerRef:
    name: slinky
  replicas: 10
  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.05-rockylinux9
    resources:
      requests:
        cpu: "4"
        memory: "16Gi"
  partition:
    enabled: true
    config: |
      State=UP
      Default=YES
```

### GPU 計算節點

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: gpu-workers
  namespace: slurm
spec:
  controllerRef:
    name: slinky
  replicas: 4
  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.05-rockylinux9
    resources:
      requests:
        cpu: "8"
        memory: "64Gi"
        nvidia.com/gpu: "4"
      limits:
        nvidia.com/gpu: "4"
  extraConf: |
    Gres=gpu:nvidia:4
  template:
    spec:
      tolerations:
        - key: "nvidia.com/gpu"
          operator: "Exists"
          effect: "NoSchedule"
  partition:
    enabled: true
    config: |
      State=UP
      MaxTime=7-00:00:00
```

### 高記憶體節點

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: highmem-workers
  namespace: slurm
spec:
  controllerRef:
    name: slinky
  replicas: 2
  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.05-rockylinux9
    resources:
      requests:
        cpu: "16"
        memory: "256Gi"
  extraConf: |
    RealMemory=256000
  partition:
    enabled: true
    config: |
      State=UP
      MaxMemPerNode=256000
```

---

> 相關文件：
> - [Slurm 官方文件 - slurmd](https://slurm.schedmd.com/slurmd.html)
> - [Slurm 官方文件 - slurm.conf](https://slurm.schedmd.com/slurm.conf.html)
> - [Kubernetes PodSpec](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#PodSpec)
