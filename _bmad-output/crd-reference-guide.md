# slurm-operator CRD 完整參考指南

本文件詳細說明 slurm-operator 的所有 Custom Resource Definitions (CRDs)，包含完整的欄位定義、驗證規則、範例與最佳實踐。

## 目錄

- [CRD 概覽](#crd-概覽)
- [Controller CRD](#controller-crd)
- [NodeSet CRD](#nodeset-crd)
- [LoginSet CRD](#loginset-crd)
- [Accounting CRD](#accounting-crd)
- [RestApi CRD](#restapi-crd)
- [Token CRD](#token-crd)
- [CRD 關聯圖](#crd-關聯圖)
- [完整部署範例](#完整部署範例)

---

## CRD 概覽

| CRD | API Group | Short Names | 用途 |
|-----|-----------|-------------|------|
| Controller | `slinky.slurm.net/v1beta1` | `slurmctld` | Slurm 控制平面 |
| NodeSet | `slinky.slurm.net/v1beta1` | - | Slurm 計算節點 |
| LoginSet | `slinky.slurm.net/v1beta1` | - | SSH 登入節點 |
| Accounting | `slinky.slurm.net/v1beta1` | - | 會計服務 (slurmdbd) |
| RestApi | `slinky.slurm.net/v1beta1` | `slurmrestd` | REST API 服務 |
| Token | `slinky.slurm.net/v1beta1` | `jwt`, `tokens` | JWT 認證令牌 |

---

## Controller CRD

Controller 是 Slurm 叢集的核心，管理 slurmctld 控制器守護程序。

### Spec 欄位

#### 認證欄位 (必填，不可更新)

| 欄位 | 類型 | 說明 |
|------|------|------|
| `slurmKeyRef` | `SecretKeySelector` | Slurm `auth/slurm` 共享認證密鑰 |
| `jwtHs256KeyRef` | `SecretKeySelector` | JWT HS256 簽名密鑰 |

```yaml
spec:
  slurmKeyRef:
    name: slurm-auth-slurm
    key: slurm.key
  jwtHs256KeyRef:
    name: slurm-auth-jwt
    key: jwt_hs256.key
```

#### 叢集識別欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `clusterName` | `string` | `{namespace}_{name}` | Slurm ClusterName (最大 40 字元) |
| `accountingRef` | `ObjectReference` | - | 關聯的 Accounting CR |

**驗證規則:**
- `clusterName` 必須匹配 `[0-9a-zA-Z$_]+`
- 不建議使用大寫字母

#### 外部模式欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `external` | `bool` | `false` | 是否為外部 slurmctld |
| `externalConfig.host` | `string` | - | 外部主機位址 |
| `externalConfig.port` | `int` | `6817` | 外部埠號 |

```yaml
spec:
  external: true
  externalConfig:
    host: slurmctld.external.com
    port: 6817
```

#### 容器配置欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `slurmctld` | `ContainerWrapper` | slurmctld 主容器 |
| `reconfigure` | `ContainerWrapper` | 重新配置 init 容器 |
| `logfile` | `ContainerWrapper` | 日誌 sidecar 容器 |

```yaml
spec:
  slurmctld:
    image: ghcr.io/slinkyproject/slurmctld:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 1
        memory: 1Gi
    args:
      - -vvv  # 詳細日誌
```

#### Slurm 配置欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `extraConf` | `string` | 附加到 slurm.conf 的額外配置 |
| `configFileRefs` | `[]ObjectReference` | 額外配置檔案 ConfigMap |
| `prologScriptRefs` | `[]ObjectReference` | Prolog 腳本 |
| `epilogScriptRefs` | `[]ObjectReference` | Epilog 腳本 |
| `prologSlurmctldScriptRefs` | `[]ObjectReference` | PrologSlurmctld 腳本 |
| `epilogSlurmctldScriptRefs` | `[]ObjectReference` | EpilogSlurmctld 腳本 |

**禁止的配置檔案:** `slurm.conf`, `slurmdbd.conf` (由 operator 管理)

```yaml
spec:
  extraConf: |
    SlurmctldDebug=debug2
    SchedulerParameters=bf_continue,bf_interval=60
  configFileRefs:
    - name: slurm-gres-config
      namespace: default
```

#### 持久化欄位 (不可更新)

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `persistence.enabled` | `bool` | `true` | 啟用持久化 |
| `persistence.existingClaim` | `string` | - | 使用現有 PVC |
| `persistence.storageClassName` | `string` | default | 儲存類別 |
| `persistence.accessModes` | `[]string` | `[ReadWriteOnce]` | 存取模式 |
| `persistence.resources.requests.storage` | `string` | `4Gi` | 儲存大小 |

```yaml
spec:
  persistence:
    enabled: true
    storageClassName: fast-ssd
    resources:
      requests:
        storage: 10Gi
```

#### 服務和監控欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `template` | `PodTemplate` | Pod 模板 (nodeSelector, affinity, tolerations) |
| `service` | `ServiceSpec` | Service 配置 |
| `metrics.enabled` | `bool` | 啟用 Prometheus 指標 |
| `metrics.serviceMonitor` | `ServiceMonitor` | Prometheus ServiceMonitor 配置 |

```yaml
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
        - key: dedicated
          value: slurm
          effect: NoSchedule
  service:
    spec:
      type: ClusterIP
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
      interval: 30s
      endpoints:
        - path: /metrics/jobs
        - path: /metrics/nodes
```

### Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `conditions` | `[]Condition` | 狀態條件列表 |

### 產生的資源

| 資源類型 | 名稱模式 |
|---------|---------|
| StatefulSet | `{name}-controller` |
| Service | `{name}-controller` |
| ConfigMap | `{name}-config` |
| PVC | `statesave-{name}-controller-0` |
| ServiceMonitor | `{name}-controller` (如啟用) |

---

## NodeSet CRD

NodeSet 管理 Slurm 計算節點 (slurmd)，類似於 StatefulSet。

### Spec 欄位

#### 核心欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `controllerRef` | `ObjectReference` | (必填) | 關聯的 Controller CR |
| `replicas` | `*int32` | `1` | Pod 副本數 |

```yaml
spec:
  controllerRef:
    namespace: default
    name: my-cluster
  replicas: 10
```

#### 容器配置欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `slurmd` | `ContainerWrapper` | slurmd 容器 |
| `logfile` | `ContainerWrapper` | 日誌 sidecar 容器 |

```yaml
spec:
  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 2
        memory: 4Gi
      limits:
        cpu: 4
        memory: 8Gi
    env:
      - name: PAM_SLURM_ADOPT_OPTIONS
        value: action_adopt_failure=deny
```

#### 分區配置欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `partition.enabled` | `bool` | `true` | 自動建立分區 |
| `partition.config` | `string` | - | 分區配置參數 |

```yaml
spec:
  partition:
    enabled: true
    config: "State=UP MaxTime=UNLIMITED DefaultTime=1:00:00"
```

#### SSH 配置欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `ssh.enabled` | `bool` | `false` | 啟用 SSH 存取 |
| `ssh.extraSshdConfig` | `string` | - | 額外 sshd 配置 |
| `ssh.sssdConfRef` | `SecretKeySelector` | - | SSSD 配置 Secret |

```yaml
spec:
  ssh:
    enabled: true
    extraSshdConfig: |
      PermitRootLogin no
      PasswordAuthentication yes
    sssdConfRef:
      name: sssd-config
      key: sssd.conf
```

#### 更新策略欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `updateStrategy.type` | `string` | `RollingUpdate` | 更新策略類型 |
| `updateStrategy.rollingUpdate.maxUnavailable` | `IntOrString` | `1` | 最大不可用數 |

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 25%  # 或絕對值如 2
```

#### 節點保護欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `taintKubeNodes` | `bool` | `false` | 污染 K8s 節點 |
| `workloadDisruptionProtection` | `bool` | `true` | 啟用 PDB 保護 |
| `minReadySeconds` | `int32` | `0` | 最小就緒秒數 |

#### 持久化欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `volumeClaimTemplates` | `[]PVC` | PVC 模板 |
| `persistentVolumeClaimRetentionPolicy.whenDeleted` | `string` | 刪除時的 PVC 行為 |
| `persistentVolumeClaimRetentionPolicy.whenScaled` | `string` | 縮容時的 PVC 行為 |

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
            storage: 100Gi
  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Delete
    whenScaled: Retain
```

#### 額外配置

| 欄位 | 類型 | 說明 |
|------|------|------|
| `extraConf` | `string` | 傳給 slurmd 的 --conf 參數 |
| `template` | `PodTemplate` | Pod 模板 |
| `revisionHistoryLimit` | `*int32` | 修訂歷史限制 |

```yaml
spec:
  extraConf: "RealMemory=7680 Features=gpu,fast-ssd Gres=gpu:nvidia:2"
```

### Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `replicas` | `int32` | 總 Pod 數 |
| `updatedReplicas` | `int32` | 已更新 Pod 數 |
| `readyReplicas` | `int32` | 就緒 Pod 數 |
| `availableReplicas` | `int32` | 可用 Pod 數 |
| `unavailableReplicas` | `int32` | 不可用 Pod 數 |
| `slurmIdle` | `int32` | IDLE 狀態節點數 |
| `slurmAllocated` | `int32` | ALLOCATED/MIXED 狀態節點數 |
| `slurmDown` | `int32` | DOWN 狀態節點數 |
| `slurmDrain` | `int32` | DRAIN 狀態節點數 |
| `nodeSetHash` | `string` | 控制器修訂哈希 |
| `selector` | `string` | 標籤選擇器 (HPA 用) |
| `conditions` | `[]Condition` | 狀態條件列表 |

---

## LoginSet CRD

LoginSet 管理 Slurm SSH 登入節點。

### Spec 欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `controllerRef` | `ObjectReference` | (必填) | 關聯的 Controller CR |
| `replicas` | `*int32` | `1` | Pod 副本數 |
| `login` | `ContainerWrapper` | - | 登入容器配置 |
| `initconf` | `ContainerWrapper` | - | 初始化容器配置 |
| `template` | `PodTemplate` | - | Pod 模板 |
| `rootSshAuthorizedKeys` | `string` | - | root SSH 公鑰 |
| `extraSshdConfig` | `string` | - | 額外 sshd 配置 |
| `sssdConfRef` | `SecretKeySelector` | (必填) | SSSD 配置 Secret |
| `service` | `ServiceSpec` | - | Service 配置 |
| `workloadDisruptionProtection` | `bool` | `true` | 啟用 PDB 保護 |

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: LoginSet
metadata:
  name: login
spec:
  controllerRef:
    namespace: default
    name: my-cluster
  replicas: 2

  login:
    image: ghcr.io/slinkyproject/login:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 200m
        memory: 256Mi

  rootSshAuthorizedKeys: |
    ssh-rsa AAAAB3... admin@example.com

  extraSshdConfig: |
    PermitRootLogin yes
    PasswordAuthentication no
    X11Forwarding yes

  sssdConfRef:
    name: sssd-config
    key: sssd.conf

  service:
    spec:
      type: LoadBalancer
    port: 22
```

### 自動產生的 SSH 主機金鑰

控制器自動產生三種類型的 SSH 主機金鑰:
- RSA (`ssh_host_rsa_key`)
- Ed25519 (`ssh_host_ed25519_key`)
- ECDSA (`ssh_host_ecdsa_key`)

### Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `replicas` | `int32` | 非終止 Pod 數 |
| `selector` | `string` | 標籤選擇器 |
| `conditions` | `[]Condition` | 狀態條件列表 |

---

## Accounting CRD

Accounting 管理 Slurm 資料庫守護程序 (slurmdbd)。

### Spec 欄位

#### 認證欄位

| 欄位 | 類型 | 條件 | 說明 |
|------|------|------|------|
| `slurmKeyRef` | `SecretKeySelector` | `external=false` 時必填 | Slurm 認證密鑰 |
| `jwtHs256KeyRef` | `SecretKeySelector` | `external=false` 時必填 | JWT 密鑰 |

#### 外部模式欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `external` | `bool` | `false` | 是否為外部 slurmdbd |
| `externalConfig.host` | `string` | - | 外部主機位址 |
| `externalConfig.port` | `int` | `6819` | 外部埠號 |

#### 資料庫配置欄位 (storageConfig)

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `storageConfig.host` | `string` | (必填) | 資料庫主機 |
| `storageConfig.port` | `int` | `3306` | 資料庫埠號 |
| `storageConfig.database` | `string` | `slurm_acct_db` | 資料庫名稱 |
| `storageConfig.username` | `string` | - | 資料庫使用者 |
| `storageConfig.passwordKeyRef` | `SecretKeySelector` | (必填) | 密碼 Secret |

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: Accounting
metadata:
  name: accounting
spec:
  slurmKeyRef:
    name: slurm-auth
    key: slurm.key
  jwtHs256KeyRef:
    name: jwt-auth
    key: jwt_hs256.key

  slurmdbd:
    image: ghcr.io/slinkyproject/slurmdbd:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 500m
        memory: 512Mi

  storageConfig:
    host: mariadb.default.svc.cluster.local
    port: 3306
    database: slurm_acct_db
    username: slurm
    passwordKeyRef:
      name: mariadb-password
      key: password

  extraConf: |
    CommitDelay=1
    DebugLevel=info

  service:
    spec:
      type: ClusterIP
```

### Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `conditions` | `[]Condition` | 狀態條件列表 |

---

## RestApi CRD

RestApi 管理 Slurm REST API 服務 (slurmrestd)。

### Spec 欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `controllerRef` | `ObjectReference` | (必填) | 關聯的 Controller CR |
| `replicas` | `*int32` | `1` | Pod 副本數 |
| `slurmrestd` | `ContainerWrapper` | - | slurmrestd 容器配置 |
| `template` | `PodTemplate` | - | Pod 模板 |
| `service` | `ServiceSpec` | - | Service 配置 (預設埠 6820) |

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: RestApi
metadata:
  name: restapi
spec:
  controllerRef:
    namespace: default
    name: my-cluster
  replicas: 2

  slurmrestd:
    image: ghcr.io/slinkyproject/slurmrestd:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 250m
        memory: 256Mi
    env:
      - name: SLURMRESTD_DEBUG_STDERR
        value: "9"

  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app.kubernetes.io/name: slurmrestd
              topologyKey: kubernetes.io/hostname

  service:
    spec:
      type: LoadBalancer
    port: 6820
```

### 環境變數

slurmrestd 容器自動設定:
- `SLURM_JWT=daemon`
- `SLURMRESTD_SECURITY=disable_unshare_files,disable_unshare_sysv`

### 健康檢查

- **StartupProbe**: TCP 6820, 失敗閾值 6, 週期 10s
- **LivenessProbe**: TCP 6820, 失敗閾值 6, 週期 10s
- **ReadinessProbe**: TCP 6820

### Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `conditions` | `[]Condition` | 狀態條件列表 |

---

## Token CRD

Token 管理 JWT 認證令牌的自動產生和刷新。

### Spec 欄位

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `jwtHs256KeyRef` | `JwtSecretKeySelector` | (必填) | JWT HS256 簽名密鑰 |
| `username` | `string` | - | Token 對應的使用者名稱 |
| `lifetime` | `Duration` | `15m` | Token 有效期 |
| `refresh` | `bool` | `true` | 是否自動刷新 |
| `secretRef` | `SecretKeySelector` | - | 儲存 JWT 的 Secret |

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: Token
metadata:
  name: admin-token
spec:
  jwtHs256KeyRef:
    name: slurm-jwt-key
    key: jwt_hs256.key

  username: admin
  lifetime: 1h
  refresh: true

  secretRef:
    name: admin-jwt-secret
    key: SLURM_JWT
```

### Token 刷新機制

- **刷新時機**: Token 距離過期時間少於 1/5 壽命時觸發
- **範例**: 15 分鐘壽命 → 12 分鐘時開始刷新 (過期前 3 分鐘)
- **Immutable**: 如果 `refresh=false`，Secret 會被標記為 immutable

### JWT Token 結構

```json
{
  "jti": "unique-uuid",
  "iss": "slurm-operator",
  "iat": 1234567890,
  "exp": 1234571490,
  "nbf": 1234567890,
  "sun": "admin"        // Slurm Username (自訂 claim)
}
```

### Status 欄位

| 欄位 | 類型 | 說明 |
|------|------|------|
| `issuedAt` | `Time` | Token 簽發時間 |
| `conditions` | `[]Condition` | 狀態條件列表 |

### 產生的 Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: admin-jwt-secret
  ownerReferences:
    - apiVersion: slinky.slurm.net/v1beta1
      kind: Token
      name: admin-token
type: Opaque
data:
  SLURM_JWT: <base64-encoded-jwt>
```

---

## CRD 關聯圖

```
┌─────────────────────────────────────────────────────────────────┐
│                         Controller                               │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  - slurmctld 控制平面                                    │    │
│  │  - 核心配置 (slurm.conf)                                 │    │
│  │  - 認證密鑰管理                                          │    │
│  └─────────────────────────────────────────────────────────┘    │
└───────────────────────────┬─────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│     NodeSet     │ │    LoginSet     │ │     RestApi     │
│  ┌───────────┐  │ │  ┌───────────┐  │ │  ┌───────────┐  │
│  │  slurmd   │  │ │  │ SSH login │  │ │  │slurmrestd │  │
│  │ (計算節點) │  │ │  │ (登入節點) │  │ │  │ (REST API)│  │
│  └───────────┘  │ │  └───────────┘  │ │  └───────────┘  │
│  controllerRef  │ │  controllerRef  │ │  controllerRef  │
│       ↑         │ │       ↑         │ │       ↑         │
└───────┼─────────┘ └───────┼─────────┘ └───────┼─────────┘
        │                   │                   │
        └───────────────────┴───────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Accounting                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  - slurmdbd 會計服務                                     │    │
│  │  - 作業歷史記錄                                          │    │
│  │  - 資料庫連接                                            │    │
│  └─────────────────────────────────────────────────────────┘    │
│  Controller.accountingRef ──────────────────────────────────────│
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Token                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  - JWT 令牌產生                                          │    │
│  │  - 自動刷新                                              │    │
│  │  - Secret 管理                                           │    │
│  └─────────────────────────────────────────────────────────┘    │
│  使用 Controller 的 jwtHs256KeyRef                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## 完整部署範例

### 基本 Slurm 叢集

```yaml
# 1. 認證密鑰 (通常由 Helm 自動產生)
---
apiVersion: v1
kind: Secret
metadata:
  name: slurm-auth
type: Opaque
stringData:
  slurm.key: <random-key>
  jwt_hs256.key: <random-key>

# 2. Controller
---
apiVersion: slinky.slurm.net/v1beta1
kind: Controller
metadata:
  name: my-cluster
spec:
  slurmKeyRef:
    name: slurm-auth
    key: slurm.key
  jwtHs256KeyRef:
    name: slurm-auth
    key: jwt_hs256.key

  slurmctld:
    image: ghcr.io/slinkyproject/slurmctld:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 500m
        memory: 512Mi

  persistence:
    enabled: true
    resources:
      requests:
        storage: 4Gi

# 3. NodeSet (計算節點)
---
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: compute
spec:
  controllerRef:
    name: my-cluster
  replicas: 4

  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 2
        memory: 4Gi
      limits:
        cpu: 4
        memory: 8Gi

  partition:
    enabled: true
    config: "State=UP Default=YES MaxTime=UNLIMITED"

  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1

  workloadDisruptionProtection: true

# 4. RestApi
---
apiVersion: slinky.slurm.net/v1beta1
kind: RestApi
metadata:
  name: restapi
spec:
  controllerRef:
    name: my-cluster
  replicas: 2

  slurmrestd:
    image: ghcr.io/slinkyproject/slurmrestd:25.11-ubuntu24.04
    resources:
      requests:
        cpu: 250m
        memory: 256Mi

  service:
    spec:
      type: ClusterIP

# 5. Token (用於 REST API 認證)
---
apiVersion: slinky.slurm.net/v1beta1
kind: Token
metadata:
  name: admin-token
spec:
  jwtHs256KeyRef:
    name: slurm-auth
    key: jwt_hs256.key
  username: admin
  lifetime: 1h
  refresh: true
  secretRef:
    name: admin-jwt-secret
    key: SLURM_JWT
```

### 使用 REST API

```bash
# 取得 JWT Token
JWT=$(kubectl get secret admin-jwt-secret -o jsonpath='{.data.SLURM_JWT}' | base64 -d)

# 列出節點
curl -H "X-SLURM-USER-TOKEN: $JWT" \
  http://restapi:6820/slurm/v0.0.41/nodes

# 列出作業
curl -H "X-SLURM-USER-TOKEN: $JWT" \
  http://restapi:6820/slurm/v0.0.41/jobs

# 提交作業
curl -X POST -H "X-SLURM-USER-TOKEN: $JWT" \
  -H "Content-Type: application/json" \
  http://restapi:6820/slurm/v0.0.41/job/submit \
  -d '{
    "job": {
      "name": "test",
      "nodes": "1",
      "tasks": 1,
      "script": "#!/bin/bash\necho Hello"
    }
  }'
```

---

## 欄位類型參考

### ObjectReference

```yaml
namespace: default  # 命名空間
name: my-resource   # 資源名稱
```

### SecretKeySelector

```yaml
name: my-secret    # Secret 名稱
key: my-key        # Secret 中的 key
```

### ContainerWrapper

包裝 `corev1.Container`，支援所有標準容器欄位:
- `image`, `imagePullPolicy`
- `command`, `args`
- `env`, `envFrom`
- `resources` (requests/limits)
- `volumeMounts`
- `securityContext`
- `livenessProbe`, `readinessProbe`, `startupProbe`

### PodTemplate

```yaml
metadata:
  labels: {}
  annotations: {}
spec:           # PodSpec
  nodeSelector: {}
  affinity: {}
  tolerations: []
  volumes: []
  initContainers: []
```

### ServiceSpec

```yaml
metadata:
  labels: {}
  annotations: {}
spec:           # Kubernetes ServiceSpec
  type: ClusterIP
  ports: []
port: 6817      # 簡化的端口設定
nodePort: 30817
```

---

*文件版本: 1.0.0*
*最後更新: 2025-12-18*
*適用於: slurm-operator v1.0+ / API v1beta1*
