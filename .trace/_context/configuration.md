# Configuration — slurm-operator

## 設定載入機制

slurm-operator 採用多層設定，優先順序（高→低）：

```
CLI Flags > Kubernetes CRD Spec > internal/defaults/* > 程式碼預設值
```

沒有傳統的 env var 設定檔，主要透過：
1. **CLI flags**（二進位啟動參數）
2. **Helm values.yaml**（Kubernetes 部署設定）
3. **CRD 資源 Spec**（使用者設定 Slurm 叢集）
4. **Kubernetes Secrets**（敏感資訊，如 JWT key、DB 密碼）

---

## 1. Controller Manager CLI Flags

`cmd/manager/main.go:parseFlags()`

| Flag | 型別 | 預設 | 說明 |
|------|------|------|------|
| `--health-addr` | string | `:8081` | Healthz/Readyz 端點 |
| `--metrics-addr` | string | `:8080` | Prometheus metrics |
| `--leader-elect` | bool | `false` | Leader election（多副本時啟用） |
| `--leader-elect-namespace` | string | `""` | Leader election 用的 namespace |
| `--metrics-secure` | bool | `false` | Metrics 使用 TLS |
| `--enable-http2` | bool | `false` | 啟用 HTTP/2（預設關閉） |
| `--namespaces` | string | `""` | 限制 watch namespace（逗號分隔，空=全部） |
| `--propagated-node-conditions` | string | `""` | 傳播到 Slurm drain 的 Node condition types |
| `--nodeset-workers` | int | `1` | NodeSet controller 並行 worker 數 |
| zap flags | — | — | log level, format 等（來自 `zap.Options`） |

### zap 日誌設定

透過 `--zap-log-level`、`--zap-encoder` 等 flag 控制：
```bash
# 開啟 verbose 日誌
./slurm-operator --zap-log-level=debug
```

---

## 2. Webhook Server CLI Flags

`cmd/webhook/main.go:parseFlags()`

| Flag | 型別 | 預設 | 說明 |
|------|------|------|------|
| `--server-addr` | string | `:9443` | Webhook TLS 監聽位址 |
| `--health-addr` | string | `:8081` | Healthz 端點 |
| `--metrics-addr` | string | `0` | Metrics（預設停用） |
| `--leader-elect` | bool | `false` | Leader election |
| `--namespaces` | string | `""` | 限制 watch namespace |

---

## 3. Helm Values（helm/slurm-operator/values.yaml）

### Operator 部分

```yaml
operator:
  enabled: true
  replicas: 1
  image:
    repository: ghcr.io/slinkyproject/slurm-operator
    tag: ""                 # 預設使用 Chart appVersion
  namespaces: ""            # 空=全部 namespace
  propagatedNodeConditions: ""
  # RBAC, ServiceAccount, Affinity, Resources 等標準 k8s 設定

webhook:
  enabled: true
  replicas: 1
  image:
    repository: ghcr.io/slinkyproject/slurm-operator-webhook
    tag: ""

crds:
  enabled: false            # 是否讓 Helm 管理 CRDs

certManager:
  enabled: true             # 是否使用 cert-manager 簽發 TLS 憑證
```

### Namespace Scoping

```bash
# 限制 operator 只 watch 特定 namespace
helm install slurm-operator ... \
  --set 'operator.namespaces=slurm-system,production' \
  --set 'webhook.namespaces=slurm-system,production'
```

---

## 4. CRD 欄位設定（使用者層）

### Controller CR（slurmctld）

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: Controller
spec:
  clusterName: "mycluster"     # Slurm ClusterName

  # 認證 Key References（引用 Kubernetes Secret）
  slurmKeyRef:                 # auth/slurm 共用 key
    name: slurm-key
    key: slurm.key
  jwtKeyRef:                   # auth/jwt signing key
    name: slurm-jwt
    key: jwt.key
  jwksKeyRef:                  # auth/jwt JWKS（用於 key rotation）
    name: slurm-jwks
    key: slurm.jwks

  # 可選：關聯 Accounting（slurmdbd）
  accountingRef:
    name: slurm-accounting
    namespace: slurm

  # 設定覆蓋（附加到 slurm.conf 末尾）
  extraConf: |
    GresTypes=gpu
    DebugLevel=debug3

  # 額外 ConfigMap 掛載到 /etc/slurm/
  configFileRefs:
    - name: gres-conf
    - name: topology-conf

  # In-place reconfigure（不重啟 pod）
  inplaceReconfigure: false

  # 持久化（slurmctld StateSaveLocation）
  persistence:
    enabled: true
    storageClassName: fast-ssd
    accessModes: [ReadWriteOnce]
    resources:
      requests:
        storage: 10Gi
```

### NodeSet CR（slurmd workers）

```yaml
spec:
  controllerRef:              # 必填：指向哪個 Controller CR
    name: slurm-controller
    namespace: slurm

  replicas: 4                 # StatefulSet 模式下的副本數
  scalingMode: StatefulSet    # StatefulSet | DaemonSet

  # Slurm partition 設定
  partition:
    enabled: true
    config: "MaxTime=INFINITE"

  # 額外 slurmd 啟動參數
  extraConf: "Gres=gpu:h100:8"

  # 更新策略
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: "25%"

  # Pod 固定 Kube 節點
  pinToNode: false

  # 工作負載保護（PDB）
  workloadDisruptionProtection: true

  # 孤兒 Slurm node 記錄清理
  pruneSlurmNodeRecords: Never  # Never | NodeNotFound

  # PVC 保留策略
  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Retain
    whenScaled: Retain
```

---

## 5. Secret 管理

所有敏感資訊透過 Kubernetes Secret 引用，不直接在 CRD spec 中存放：

| 敏感資訊 | 引用方式 | CRD 欄位 |
|---------|---------|---------|
| Slurm auth/slurm key | `SecretKeySelector` | `Controller.spec.slurmKeyRef` |
| Slurm auth/jwt signing key | `SecretKeySelector` | `Controller.spec.jwtKeyRef` |
| Slurm auth/jwt JWKS | `ConfigMapKeySelector` | `Controller.spec.jwksKeyRef` |
| MariaDB 密碼 | `SecretKeySelector` | `Accounting.spec.storageConfig.passwordKeyRef` |
| SSSD 設定 | `SecretKeySelector` | `LoginSet.spec.sssdConfRef` |
| SSH Authorized Keys | inline string | `LoginSet.spec.rootSshAuthorizedKeys` |
| JWT Token 輸出 | `SecretKeySelector` | `Token.spec.secretRef` |

---

## 6. Feature Flags

以 CRD 欄位實作，非環境變數：

| Feature | CRD | 欄位 | 預設 |
|---------|-----|------|------|
| In-place reconfigure | Controller | `inplaceReconfigure` | `false` |
| Controller persistence | Controller | `persistence.enabled` | `true` |
| Accounting | Accounting CR | 是否建立 Accounting CR | N/A |
| Partition 建立 | NodeSet | `partition.enabled` | `false` |
| SSH 存取 | NodeSet | `ssh.enabled` | `false` |
| Kube Node Taint | NodeSet | `taintKubeNodes` | `false`（棄用） |
| Workload PDB | NodeSet | `workloadDisruptionProtection` | `true` |
| Pin to Node | NodeSet | `pinToNode` | `false` |
| Token 輪換 | Token | `refresh` | `true` |
| Metrics | Controller | `metrics.enabled` | `false` |
| ServiceMonitor | Controller | `metrics.serviceMonitor.enabled` | `false` |

---

## 7. DEBUG 環境變數

僅有一個非 CRD 設定：

```go
// internal/controller/slurmclient/slurmclient_sync.go
if val := os.Getenv("DEBUG"); val == "1" {
    server = fmt.Sprintf("http://localhost:%d", builder.SlurmrestdPort)
}
```

在開發環境中，設定 `DEBUG=1` 讓 operator 連線到本機的 slurmrestd（用於 `.vscode/scripts/patch-webhook-local-debug.sh`）。

---

## 8. 設定優先順序與覆蓋機制

### slurm.conf 組裝

Controller builder 動態組裝 slurm.conf：

```
基礎設定（operator 產生）
+ NodeSet partition 設定（從每個 NodeSet 收集）
+ ExtraConf（Controller.spec.extraConf 末尾追加）
+ ConfigFileRefs 中的額外 ConfigMap 掛載
```

### 預設值系統（internal/defaults/）

```go
// internal/defaults/nodeset.go
func SetNodeSetDefaults(nodeset *slinkyv1beta1.NodeSet) {
    if s.Replicas == nil { s.Replicas = ptr.To(int32(1)) }
    if s.ScalingMode == "" { s.ScalingMode = ScalingModeStatefulset }
    if s.WorkloadDisruptionProtection == nil { s.WorkloadDisruptionProtection = ptr.To(true) }
    // ...
}
```

預設值在 `Reconcile()` 開始前套用（`DeepCopy()` 後立即呼叫），不修改 etcd 中的原始物件。
