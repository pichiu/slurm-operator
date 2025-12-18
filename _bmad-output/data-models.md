# Data Models - slurm-operator CRD 定義

> 生成日期：2025-12-18

## API 概述

- **API 群組**: `slinky.slurm.net`
- **API 版本**: `v1beta1`
- **CRD 數量**: 6

---

## 1. Controller

**用途**: 管理 Slurm 控制器守護程式 (slurmctld)

**短名稱**: `slurmctld`

### 1.1 Spec 欄位

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `clusterName` | string | 是 | Slurm 叢集名稱 (最多 40 字元) |
| `slurmKeyRef` | SecretKeySelector | 是 | Slurm 認證金鑰 |
| `jwtHs256KeyRef` | SecretKeySelector | 是 | JWT HS256 金鑰 |
| `accountingRef` | ObjectReference | 否 | Accounting CR 參考 |
| `external` | bool | 否 | 外部部署標誌 (預設: false) |
| `externalConfig` | ExternalConfig | 否 | 外部配置 (host, port) |
| `slurmctld` | ContainerWrapper | 否 | slurmctld 容器配置 |
| `reconfigure` | ContainerWrapper | 否 | 重配置容器 |
| `logfile` | ContainerWrapper | 否 | 日誌邊車 |
| `template` | PodTemplate | 否 | Pod 範本 |
| `extraConf` | string | 否 | slurm.conf 額外設定 |
| `configFileRefs` | []ObjectReference | 否 | ConfigMap 參考 |
| `prologScriptRefs` | []ObjectReference | 否 | Prolog 腳本 |
| `epilogScriptRefs` | []ObjectReference | 否 | Epilog 腳本 |
| `persistence` | ControllerPersistence | 否 | 持久化設定 |
| `service` | ServiceSpec | 否 | Service 配置 |
| `metrics` | Metrics | 否 | Prometheus 指標 |

### 1.2 Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `conditions` | []metav1.Condition | 狀態條件 |

### 1.3 驗證規則

- `clusterName` 不可超過 40 字元
- `clusterName` 必須符合 `[0-9a-zA-Z$_]+`
- `clusterName`, `slurmKeyRef`, `jwtHs256KeyRef`, `persistence.enabled` 為不可變欄位

---

## 2. NodeSet

**用途**: 管理 Slurm 計算節點集合 (slurmd)

**短名稱**: `nodesets`, `nss`, `slurmd`

### 2.1 Spec 欄位

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `controllerRef` | ObjectReference | 是 | Controller CR 參考 |
| `replicas` | *int32 | 否 | Pod 副本數 (預設: 1) |
| `slurmd` | ContainerWrapper | 否 | slurmd 容器配置 |
| `ssh` | NodeSetSsh | 否 | SSH 配置 |
| `logfile` | ContainerWrapper | 否 | 日誌邊車 |
| `template` | PodTemplate | 否 | Pod 範本 |
| `extraConf` | string | 否 | slurmd 額外設定 |
| `partition` | NodeSetPartition | 否 | 分區配置 |
| `volumeClaimTemplates` | []PVC | 否 | PVC 範本 |
| `updateStrategy` | NodeSetUpdateStrategy | 否 | 更新策略 |
| `revisionHistoryLimit` | *int32 | 否 | 版本歷史限制 |
| `pvcRetentionPolicy` | PVCRetentionPolicy | 否 | PVC 保留策略 |
| `minReadySeconds` | int32 | 否 | 最小就緒秒數 |
| `taintKubeNodes` | bool | 否 | 污染 K8s 節點 |
| `workloadDisruptionProtection` | bool | 否 | 工作負載保護 (預設: true) |

### 2.2 Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `replicas` | int32 | 總副本數 |
| `updatedReplicas` | int32 | 已更新副本數 |
| `readyReplicas` | int32 | 就緒副本數 |
| `availableReplicas` | int32 | 可用副本數 |
| `unavailableReplicas` | int32 | 不可用副本數 |
| `slurmIdle` | int32 | Slurm Idle 節點數 |
| `slurmAllocated` | int32 | Slurm Allocated 節點數 |
| `slurmDown` | int32 | Slurm Down 節點數 |
| `slurmDrain` | int32 | Slurm Drain 節點數 |
| `observedGeneration` | int64 | 觀察到的世代 |
| `nodeSetHash` | string | 配置雜湊 |
| `collisionCount` | *int32 | 碰撞計數 |
| `conditions` | []metav1.Condition | 狀態條件 |

### 2.3 NodeSetSsh

| 欄位 | 類型 | 說明 |
|------|------|------|
| `enabled` | bool | 啟用 SSH (預設: false) |
| `extraSshdConfig` | string | sshd_config 額外設定 |
| `sssdConfRef` | SecretKeySelector | sssd.conf 參考 |

### 2.4 NodeSetPartition

| 欄位 | 類型 | 說明 |
|------|------|------|
| `enabled` | bool | 建立分區 (預設: true) |
| `config` | string | 分區配置 |

---

## 3. LoginSet

**用途**: 管理 Slurm 登入節點集合

**短名稱**: `loginsets`, `lss`, `sackd`

### 3.1 Spec 欄位

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `controllerRef` | ObjectReference | 是 | Controller CR 參考 |
| `replicas` | *int32 | 否 | Pod 副本數 (預設: 1) |
| `login` | ContainerWrapper | 否 | 登入容器配置 |
| `initconf` | ContainerWrapper | 否 | init 容器 |
| `template` | PodTemplate | 否 | Pod 範本 |
| `rootSshAuthorizedKeys` | string | 否 | root SSH 授權金鑰 |
| `extraSshdConfig` | string | 否 | sshd_config 額外設定 |
| `sssdConfRef` | SecretKeySelector | 是 | sssd.conf 參考 |
| `service` | ServiceSpec | 否 | Service 配置 |

### 3.2 Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `replicas` | int32 | 副本數 |
| `selector` | string | 標籤選擇器 (HPA 支援) |
| `conditions` | []metav1.Condition | 狀態條件 |

---

## 4. Accounting

**用途**: 管理 Slurm 會計資料庫守護程式 (slurmdbd)

**短名稱**: `slurmdbd`

### 4.1 Spec 欄位

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `slurmKeyRef` | SecretKeySelector | 條件 | Slurm 認證金鑰 |
| `jwtHs256KeyRef` | SecretKeySelector | 條件 | JWT HS256 金鑰 |
| `external` | bool | 否 | 外部部署標誌 |
| `externalConfig` | ExternalConfig | 否 | 外部配置 |
| `slurmdbd` | ContainerWrapper | 否 | slurmdbd 容器 |
| `template` | PodTemplate | 否 | Pod 範本 |
| `storageConfig` | StorageConfig | 是 | 資料庫配置 |
| `extraConf` | string | 否 | slurmdbd.conf 額外設定 |
| `service` | ServiceSpec | 否 | Service 配置 |

### 4.2 StorageConfig

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `host` | string | 是 | 資料庫主機 |
| `port` | int | 否 | 連接埠 (預設: 3306) |
| `database` | string | 否 | 資料庫名稱 (預設: slurm_acct_db) |
| `username` | string | 否 | 使用者名稱 |
| `passwordKeyRef` | SecretKeySelector | 是 | 密碼參考 |

---

## 5. RestApi

**用途**: 管理 Slurm REST API 服務 (slurmrestd)

**短名稱**: `slurmrestd`

### 5.1 Spec 欄位

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `controllerRef` | ObjectReference | 是 | Controller CR 參考 |
| `replicas` | *int32 | 否 | Pod 副本數 (預設: 1) |
| `slurmrestd` | ContainerWrapper | 否 | slurmrestd 容器 |
| `template` | PodTemplate | 否 | Pod 範本 |
| `service` | ServiceSpec | 否 | Service 配置 |

### 5.2 Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `conditions` | []metav1.Condition | 狀態條件 |

---

## 6. Token

**用途**: 管理 JWT 令牌生命週期

**短名稱**: `tokens`, `jwt`

### 6.1 Spec 欄位

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `jwtHs256KeyRef` | JwtSecretKeySelector | 是 | JWT 金鑰參考 |
| `username` | string | 是 | 使用者名稱 |
| `lifetime` | *metav1.Duration | 否 | JWT 有效期 |
| `refresh` | bool | 否 | 自動輪換 (預設: true) |
| `secretRef` | *SecretKeySelector | 否 | 密鑰參考 |

### 6.2 Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `issuedAt` | metav1.Time | 發行時間 |
| `conditions` | []metav1.Condition | 狀態條件 |

---

## 共享類型

### ObjectReference

```go
type ObjectReference struct {
    Namespace string `json:"namespace,omitempty"`
    Name      string `json:"name,omitempty"`
}
```

### ExternalConfig

```go
type ExternalConfig struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}
```

### ContainerWrapper

包裝 `corev1.Container`，支援自訂映像、環境變數、資源限制等。

### PodTemplate

```go
type PodTemplate struct {
    Metadata Metadata       `json:"metadata,omitempty"`
    Spec     PodSpecWrapper `json:"spec,omitempty"`
}
```

### ServiceSpec

```go
type ServiceSpec struct {
    Metadata Metadata           `json:"metadata,omitempty"`
    Spec     ServiceSpecWrapper `json:"spec,omitempty"`
    Port     int                `json:"port,omitempty"`
    NodePort int                `json:"nodePort,omitempty"`
}
```

---

## 已知標籤和註釋

### 標籤前綴

`slinky.slurm.net/`

### Pod 註釋

| 註釋 | 用途 |
|------|------|
| `slinky.slurm.net/pod-cordon` | Pod 應被 DRAIN |
| `slinky.slurm.net/pod-deletion-cost` | 刪除優先順序 |
| `slinky.slurm.net/pod-deadline` | 工作負載截止時間 |

### Node 註釋

| 註釋 | 用途 |
|------|------|
| `slinky.slurm.net/node-cordon-reason` | DRAIN 原因 |
| `topology.slinky.slurm.net/line` | 拓撲線 |

### Pod 標籤

| 標籤 | 用途 |
|------|------|
| `slinky.slurm.net/pod-name` | Pod 名稱 |
| `slinky.slurm.net/pod-index` | Pod 序數 |
| `slinky.slurm.net/pod-hostname` | Pod 主機名 |
| `slinky.slurm.net/pod-protect` | PDB 保護 |
