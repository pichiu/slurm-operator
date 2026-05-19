# Extension Points — slurm-operator

## 擴充點概覽

slurm-operator 是一個 Kubernetes operator，其主要擴充機制是 Kubernetes 原生的 CRD + controller 模式，而非傳統 plugin 架構。以下列出主要可延伸的機制。

---

## 1. 事件驅動（Event Handlers）

每個 controller 都有自己的 `eventhandler/` 子套件，負責把非目標資源的變化轉換成對目標 CR 的 reconcile 請求。

### NodeSet Controller 的 Event Handlers

| 事件來源 | Handler | 行為 |
|---------|---------|------|
| `corev1.Pod` | `PodEventHandler` | Pod create/update/delete → 更新 expectations，enqueue NodeSet |
| `corev1.Node` | `NodeEventHandler` | Node 狀態變化（cordon 等）→ 影響所在 NodeSet |
| `slinkyv1beta1.Controller` | `ControllerEventHandler` | Controller CR 變化 → 通知相關 NodeSet |
| `corev1.Secret` | `SecretEventHandler` | Slurm key/JWT key secret 變化 → 觸發 reconcile |

**新增觀察資源**：在 `SetupWithManager()` 加入 `.Watches()`：
```go
// nodeset_controller.go:SetupWithManager()
ctrl.NewControllerManagedBy(mgr).
    For(&slinkyv1beta1.NodeSet{}).
    Watches(&corev1.Node{}, eventhandler.NewNodeEventHandler(r.Client)).
    Watches(&slinkyv1beta1.Controller{}, eventhandler.NewControllerEventHandler(r.Client)).
    // ... 可加入更多
```

---

## 2. SyncSteps Pipeline（可組合的同步步驟）

`internal/syncsteps/syncsteps.go` 提供泛型 pipeline，各 controller 自行定義步驟清單：

```go
steps := []syncsteps.Step[*slinkyv1beta1.NodeSet]{
    {Name: "Service", SyncFn: ...},
    {Name: "Config", SyncFn: ...},
    // 新增步驟只需加入此清單
}
syncsteps.Sync(ctx, recorder, obj, steps)
```

如需擴充某個 controller 的行為，可在 `sync()` 函式的步驟清單中插入新步驟。

---

## 3. Builder 系統（Kubernetes 物件建構器）

`internal/builder/` 中每個 builder 套件負責從 CRD spec 產生 Kubernetes 物件：

| Builder | 輸入 CR | 產生的物件 |
|---------|---------|----------|
| `controllerbuilder` | `Controller` | StatefulSet, ConfigMap, Service, ServiceMonitor |
| `workerbuilder` | `NodeSet` + `Controller` | Pod, PVC, Service |
| `loginbuilder` | `LoginSet` | Deployment, Service, Secret, ConfigMap |
| `accountingbuilder` | `Accounting` | StatefulSet, ConfigMap, Service |
| `restapibuilder` | `RestApi` | Deployment, Service |
| `common/` | 共用 | Container, Secret, PodDisruptionBudget |

**擴充新欄位**：
1. 在 `api/v1beta1/*_types.go` 新增欄位
2. 在對應的 builder 套件中更新建構邏輯
3. 在 `internal/defaults/` 設定預設值
4. 在 `internal/webhook/` 更新驗證邏輯

---

## 4. SlurmControlInterface（Slurm 操作抽象）

`slurmcontrol.SlurmControlInterface` 是 NodeSet controller 與 Slurm REST API 之間的抽象層：

```go
type SlurmControlInterface interface {
    MakeNodeDrain(ctx, nodeset, pod, reason) error
    // ... 14 個方法
}
```

**測試擴充**：可實作 `SlurmControlInterface` 替換 `realSlurmControl`，注入 mock 進行單元測試：
```go
// nodeset_controller_test.go 中使用
type fakeSlurmControl struct { ... }
```

**生產擴充**：如需對接不同版本的 Slurm API，可實作新的 `SlurmControlInterface`。

---

## 5. CRD Webhook 驗證（Admission Webhooks）

每個 CRD 都有對應的 Validator webhook，可在不修改核心邏輯的情況下擴充驗證規則：

| Webhook | 檔案 | 驗證內容 |
|---------|------|----------|
| `ControllerWebhook` | `controller_webhook.go` | slurmKeyRef/jwtKeyRef 二選一驗證、external 欄位規則 |
| `NodeSetWebhook` | `nodeset_webhook.go` | controllerRef 不可修改、maxUnavailable > 0、SSH 設定完整性 |
| `LoginSetWebhook` | `loginset_webhook.go` | controllerRef 不可修改 |
| `AccountingWebhook` | `accounting_webhook.go` | slurmKeyRef 要求 |
| `RestapiWebhook` | `restapi_webhook.go` | 基本驗證 |
| `TokenWebhook` | `token_webhook.go` | jwtKeyRef 或 jwtHs256KeyRef 要求 |
| `PodBindingWebhook` | `pod_binding_webhook.go` | **Mutator**：注入 topology annotation |

**重要限制**（由 webhook 強制）：
- `NodeSet.spec.controllerRef` 建立後不可修改
- `NodeSet.spec.volumeClaimTemplates` 建立後不可修改
- `LoginSet.spec.controllerRef` 建立後不可修改

---

## 6. External 模式（Hybrid 擴充）

Controller 和 Accounting CR 支援 `external=true`，讓 operator 管理不在 Kubernetes 中的 Slurm 元件：

```yaml
apiVersion: slinky.slurm.net/v1beta1
kind: Controller
spec:
  external: true
  externalConfig:
    host: "192.168.1.100"
    port: 6817
  jwtKeyRef:
    name: slurm-jwt
    key: jwt.key
```

此模式讓 operator 管理 Kubernetes 中的 NodeSet pods，但 slurmctld 運行在 bare-metal 上。

---

## 7. 設定覆蓋機制

Controller CR 支援多種設定注入方式：

```go
// controller_types.go
ConfigFileRefs         []ObjectReference  // 額外的 ConfigMap 掛載到 /etc/slurm/
PrologScriptRefs       []ObjectReference  // Prolog 腳本
EpilogScriptRefs       []ObjectReference  // Epilog 腳本
PrologSlurmctldScriptRefs []ObjectReference
EpilogSlurmctldScriptRefs []ObjectReference
ExtraConf              string             // 附加到 slurm.conf 末尾
```

這讓 Slurm 設定可以由使用者透過 ConfigMap 完全自訂，而不修改 operator 程式碼。

---

## 8. Propagated Node Conditions（可設定的狀態傳播）

Controller Manager CLI flag `--propagated-node-conditions` 允許設定哪些 Kube Node conditions 會傳播成 Slurm drain reason：

```bash
# 例：當 GPU 故障或記憶體壓力時，Slurm node 會收到對應 drain reason
./slurm-operator --propagated-node-conditions="GPUDeviceDown,MemoryPressure"
```

這讓 operator 可以感知硬體狀態並自動保護工作負載。

---

## 如何新增一個 CRD

遵循 Kubebuilder 模式：

1. 在 `api/v1beta1/` 建立 `{name}_types.go`
2. 執行 `make generate` 產生 `zz_generated.deepcopy.go`
3. 執行 `make manifests` 產生 CRD YAML
4. 在 `internal/builder/{name}builder/` 實作 builder
5. 在 `internal/defaults/{name}.go` 實作預設值
6. 在 `internal/controller/{name}/` 實作 controller
7. 在 `internal/webhook/{name}_webhook.go` 實作 webhook
8. 在 `cmd/manager/main.go` 和 `cmd/webhook/main.go` 註冊

---

## 不存在的擴充點

以下功能在本 codebase 中**未實作**，需要外部解決：
- 自定義 Slurm Plugin（plugin 邏輯在 Slurm 本身）
- Event-driven autoscaling 觸發器（靠外部 HPA + metrics）
- 多叢集管理（每個 Controller CR 對應一個 Slurm 叢集，多叢集需要多個 Controller CR）
