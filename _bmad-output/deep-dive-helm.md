# Helm Charts - 深入分析文檔

**生成時間:** 2025-12-22
**範圍:** `helm/`
**分析檔案數:** 70
**程式碼行數:** 6,126
**工作流程模式:** 詳盡深入分析

---

## 概覽

本資料夾包含 Slurm Operator 專案的三個 Helm Charts，用於在 Kubernetes 上部署和管理 Slurm 工作負載管理系統。

**目的:** 提供在 Kubernetes 環境中部署完整 Slurm 叢集的基礎設施即程式碼 (IaC) 解決方案。

**核心職責:**
- 定義 Slurm 叢集各元件的 Kubernetes 資源
- 管理 Slurm Operator 及其 Webhook 的部署
- 提供 CRD (Custom Resource Definitions) 安裝機制
- 支援 NVIDIA GPU 整合 (DCGM)

**整合點:** Kubernetes API、cert-manager、Prometheus ServiceMonitor、MariaDB

---

## 架構總覽

```
helm/
├── slurm/                    # Slurm 叢集部署 Chart
│   ├── Chart.yaml            # Chart 元資料
│   ├── values.yaml           # 預設配置值
│   ├── templates/            # Kubernetes 資源模板
│   │   ├── accounting/       # slurmdbd 會計元件
│   │   ├── cluster/          # 叢集級配置 (SSSD)
│   │   ├── controller/       # slurmctld 控制器
│   │   ├── loginset/         # 登入節點集
│   │   ├── nodeset/          # 計算節點集
│   │   ├── priorityclass/    # Pod 優先級類別
│   │   ├── restapi/          # slurmrestd REST API
│   │   ├── secrets/          # 認證金鑰
│   │   └── vendor/nvidia/    # NVIDIA 廠商整合
│   └── _vendor/nvidia/       # NVIDIA 腳本原始碼
│
├── slurm-operator/           # Operator 部署 Chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── cert-manager/     # PKI 憑證配置
│       ├── operator/         # Operator Deployment/RBAC
│       ├── tests/            # Helm 測試
│       └── webhook/          # Admission Webhook
│
└── slurm-operator-crds/      # CRD 定義 Chart
    ├── Chart.yaml
    └── templates/            # 6 個 CRD 定義
```

---

## 完整檔案清單

### Chart 1: slurm (Slurm 叢集)

#### helm/slurm/Chart.yaml
**用途:** 定義 slurm Chart 的元資料，包含版本、依賴關係和維護者資訊。
**程式碼行數:** 26
**檔案類型:** Helm Chart 定義

**未來貢獻者須知:** 此 Chart 需要 Kubernetes 1.29+，且使用 Slurm 25.11 版本。

**關鍵配置:**
- `apiVersion`: v2 (Helm 3)
- `kubeVersion`: ">= 1.29.0-0"
- `appVersion`: "25.11"
- `version`: 1.0.0

---

#### helm/slurm/values.yaml
**用途:** 定義 Slurm 叢集的所有可配置參數，是用戶客製化部署的主要介面。
**程式碼行數:** 777
**檔案類型:** YAML 配置

**未來貢獻者須知:** 這是最重要的配置檔案，包含所有 Slurm 元件的預設值。修改時需確保向後相容性。

**主要配置區塊:**
```yaml
# 認證金鑰配置
slurmKeyRef: {}           # Slurm auth/slurm 金鑰
jwtHs256KeyRef: {}        # JWT HS256 金鑰

# 元件配置
controller:               # slurmctld 控制器
  - slurmctld, reconfigure, logfile 容器
  - persistence 持久化儲存
  - metrics 指標收集

restapi:                  # slurmrestd REST API
  - replicas: 1

accounting:               # slurmdbd 會計
  - enabled: false (預設停用)
  - storageConfig 資料庫連線

nodesets:                 # 計算節點集 (slurmd)
  slinky:                 # 預設節點集
    - replicas, partition, ssh, updateStrategy

loginsets:                # 登入節點集
  slinky:
    - enabled: false (預設停用)

partitions:               # Slurm 分區
  all:
    - nodesets: [ALL]

vendor:                   # 廠商整合
  nvidia:
    dcgm:
      enabled: false
```

**匯出項目:**
- 全域配置: `nameOverride`, `fullnameOverride`, `imagePullSecrets`
- 優先級類別: `priorityClass.name`, `priorityClass.value`
- Prolog/Epilog 腳本掛載

---

#### helm/slurm/templates/_helpers.tpl
**用途:** 提供可重用的 Helm 模板輔助函數，用於名稱生成和格式化。
**程式碼行數:** 163
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.name` - Chart 名稱
- `slurm.fullname` - 完整資源名稱
- `slurm.namespace` - 目標命名空間
- `slurm.labels` - 通用標籤
- `slurm.chart` - Chart 標識
- `format-image` - 容器映像格式化
- `format-container` - 容器規格格式化
- `format-podTemplate` - Pod 模板格式化
- `resource-quantity` - 資源單位轉換
- `toYaml-set-storageClassName` - StorageClass 處理
- `slurm.priorityClassName` - 優先級類別名稱

**依賴項目:** 無

**使用者:** 所有模板檔案

---

#### helm/slurm/templates/_slurm.tpl
**用途:** 定義 Slurm 認證相關的 Secret 參考名稱。
**程式碼行數:** 33
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.authSlurmRef.name` - Slurm auth key Secret 名稱
- `slurm.authSlurmRef.key` - Secret 鍵名 ("slurm.key")
- `slurm.authJwtHs256Ref.name` - JWT HS256 Secret 名稱
- `slurm.authJwtHs256Ref.key` - Secret 鍵名 ("jwt_hs256.key")

---

#### helm/slurm/templates/NOTES.txt
**用途:** Helm 安裝後顯示的使用說明，包含 ASCII Art 標誌和連線指南。
**程式碼行數:** 109
**檔案類型:** 文字模板

**關鍵實現細節:**
- 顯示 Slurm ASCII 藝術標誌
- 動態生成 SSH 連線指令 (LoginSet)
- 動態生成 REST API curl 指令
- 提供官方文檔連結

---

#### helm/slurm/templates/controller/_helpers.tpl
**用途:** Controller 元件的輔助函數，處理名稱生成和配置格式化。
**程式碼行數:** 108
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.controller.name` - Controller 資源名稱
- `slurm.controller.service` - Service 名稱
- `slurm.controller.port` - 預設端口 (6817)
- `slurm.controller.extraConf` - 動態生成 slurm.conf 額外配置
- `slurm.controller.configName` - ConfigMap 名稱
- `slurm.controller.prologName` - Prolog 腳本 ConfigMap
- `slurm.controller.epilogName` - Epilog 腳本 ConfigMap
- `slurm.controller.prologSlurmctldName` - PrologSlurmctld ConfigMap
- `slurm.controller.epilogSlurmctldName` - EpilogSlurmctld ConfigMap

**關鍵實現細節:**
```go
// 動態生成 Partition 配置
{{- range $partName, $part := .Values.partitions -}}
  // 驗證 NodeSet 存在性
  // 組合 PartitionName=xxx Nodes=xxx State=xxx
{{- end }}
```

---

#### helm/slurm/templates/controller/controller-cr.yaml
**用途:** 生成 Controller CRD 實例，是 Slurm 叢集的核心控制器。
**程式碼行數:** 105
**檔案類型:** Kubernetes CRD 模板

**關鍵實現細節:**
- apiVersion: `slinky.slurm.net/v1beta1`
- kind: `Controller`
- 支援外部 (external) 模式
- 整合 Accounting 參考
- 掛載 prolog/epilog 腳本
- 配置容器規格 (slurmctld, reconfigure, logfile)
- 設定持久化儲存 (PVC)
- 整合 Prometheus ServiceMonitor

---

#### helm/slurm/templates/controller/config-configmap.yaml
**用途:** 建立包含額外 Slurm 配置檔案的 ConfigMap。
**程式碼行數:** 20
**檔案類型:** Kubernetes ConfigMap 模板

**使用條件:** 僅當 `.Values.configFiles` 非空時建立

---

#### helm/slurm/templates/controller/prolog-configmap.yaml
**用途:** 建立 Prolog 腳本 ConfigMap。
**程式碼行數:** ~15
**檔案類型:** Kubernetes ConfigMap 模板

---

#### helm/slurm/templates/controller/epilog-configmap.yaml
**用途:** 建立 Epilog 腳本 ConfigMap。
**程式碼行數:** ~15
**檔案類型:** Kubernetes ConfigMap 模板

---

#### helm/slurm/templates/controller/prolog-slurmctld-configmap.yaml
**用途:** 建立 PrologSlurmctld 腳本 ConfigMap (在 slurmctld 執行)。
**程式碼行數:** ~15
**檔案類型:** Kubernetes ConfigMap 模板

---

#### helm/slurm/templates/controller/epilog-slurmctld-configmap.yaml
**用途:** 建立 EpilogSlurmctld 腳本 ConfigMap (在 slurmctld 執行)。
**程式碼行數:** ~15
**檔案類型:** Kubernetes ConfigMap 模板

---

#### helm/slurm/templates/accounting/_helpers.tpl
**用途:** Accounting 元件的輔助函數。
**程式碼行數:** 18
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.accounting.extraConf` - slurmdbd.conf 額外配置

---

#### helm/slurm/templates/accounting/accounting-cr.yaml
**用途:** 生成 Accounting CRD 實例 (slurmdbd)。
**程式碼行數:** 62
**檔案類型:** Kubernetes CRD 模板

**關鍵實現細節:**
- apiVersion: `slinky.slurm.net/v1beta1`
- kind: `Accounting`
- 條件: 僅當 `accounting.enabled: true` 時建立
- 配置資料庫連線 (storageConfig)
- 支援外部 slurmdbd

---

#### helm/slurm/templates/nodeset/_helpers.tpl
**用途:** NodeSet 元件的輔助函數，包含資源限制計算。
**程式碼行數:** 67
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.worker.name` - Worker 資源名稱
- `slurm.worker.port` - 預設端口 (6818)
- `slurm.worker.extraConf` - 節點額外配置
- `slurm.worker.partitionConfig` - 分區配置
- `slurm.worker.podCpus` - 解析 CPU 限制為整數
- `slurm.worker.podMemory` - 解析記憶體限制為 MiB

**關鍵實現細節:**
```go
// 資源單位轉換 (支援 Ki, Mi, Gi, m, k, M, G 等)
{{- define "resource-quantity" -}}
  {{- $base2 := dict "Ki" 0x1p10 "Mi" 0x1p20 ... -}}
  {{- $base10 := dict "m" 1e-3 "k" 1e3 ... -}}
{{- end -}}
```

---

#### helm/slurm/templates/nodeset/nodeset-cr.yaml
**用途:** 生成 NodeSet CRD 實例 (slurmd 計算節點)。
**程式碼行數:** 78
**檔案類型:** Kubernetes CRD 模板

**關鍵實現細節:**
- apiVersion: `slinky.slurm.net/v1beta1`
- kind: `NodeSet`
- 迴圈處理多個 NodeSet 定義
- 支援 GPU 資源限制自動偵測
- DCGM 整合 (自動掛載 volume)
- 配置 updateStrategy (RollingUpdate/OnDelete)
- 支援 SSH 存取 (pam_slurm_adopt)

---

#### helm/slurm/templates/loginset/_helpers.tpl
**用途:** LoginSet 元件的輔助函數。
**程式碼行數:** 12
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.login.name` - Login 資源名稱

---

#### helm/slurm/templates/loginset/loginset-cr.yaml
**用途:** 生成 LoginSet CRD 實例 (登入節點)。
**程式碼行數:** 54
**檔案類型:** Kubernetes CRD 模板

**關鍵實現細節:**
- apiVersion: `slinky.slurm.net/v1beta1`
- kind: `LoginSet`
- 迴圈處理多個 LoginSet 定義
- 配置 SSSD (LDAP 整合)
- 支援 SSH 公鑰注入
- 配置 Service (LoadBalancer)

---

#### helm/slurm/templates/restapi/_helpers.tpl
**用途:** REST API 元件的輔助函數。
**程式碼行數:** 19
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.restapi.name` - RestAPI 資源名稱
- `slurm.restapi.port` - 預設端口 (6820)

---

#### helm/slurm/templates/restapi/restapi-cr.yaml
**用途:** 生成 RestApi CRD 實例 (slurmrestd)。
**程式碼行數:** 36
**檔案類型:** Kubernetes CRD 模板

**關鍵實現細節:**
- apiVersion: `slinky.slurm.net/v1beta1`
- kind: `RestApi`
- 配置 slurmrestd 容器
- 支援多副本部署

---

#### helm/slurm/templates/cluster/_helpers.tpl
**用途:** 叢集級配置的輔助函數 (SSSD)。
**程式碼行數:** 27
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm.sssdConf.name` - SSSD 配置 Secret 名稱
- `slurm.sssdConf.key` - Secret 鍵名 ("sssd.conf")

---

#### helm/slurm/templates/cluster/sssd-conf-secret.yaml
**用途:** 建立 SSSD 配置 Secret (用於 LDAP 認證)。
**程式碼行數:** ~20
**檔案類型:** Kubernetes Secret 模板

---

#### helm/slurm/templates/priorityclass/priorityclass.yaml
**用途:** 建立 PriorityClass 以確保 Slurm Pod 的調度優先級。
**程式碼行數:** ~15
**檔案類型:** Kubernetes PriorityClass 模板

**配置:**
- value: 1000000000 (預設)
- preemptionPolicy: PreemptLowerPriority

---

#### helm/slurm/templates/secrets/slurmkey.yaml
**用途:** 自動生成 Slurm auth/slurm 認證金鑰。
**程式碼行數:** 26
**檔案類型:** Kubernetes Secret 模板

**關鍵實現細節:**
```go
// 使用 lookup 函數檢查 Secret 是否已存在
{{- $secret := lookup "v1" "Secret" namespace $secretName -}}
{{- if not $secret }}
  // 生成隨機 1024 字元 ASCII 金鑰
  {{ randAscii 1024 | b64enc }}
{{- end }}
```

**安全考量:**
- 使用 `helm.sh/resource-policy: keep` 防止升級時刪除
- 設定 `immutable: true` 防止意外修改

---

#### helm/slurm/templates/secrets/jwths256key.yaml
**用途:** 自動生成 JWT HS256 認證金鑰。
**程式碼行數:** ~25
**檔案類型:** Kubernetes Secret 模板

---

#### helm/slurm/templates/vendor/nvidia/dcgm/_helpers.tpl
**用途:** NVIDIA DCGM 整合的輔助函數。
**程式碼行數:** 75
**檔案類型:** Helm 模板

**匯出項目:**
- `vendor.dcgm.enabled` - 檢查 DCGM 是否啟用
- `vendor.dcgm.jobMappingDir` - GPU-Job 映射目錄
- `vendor.dcgm.nodesetHasGPU` - 檢查 NodeSet 是否有 GPU
- `vendor.dcgm.prologName` - Prolog ConfigMap 名稱
- `vendor.dcgm.epilogName` - Epilog ConfigMap 名稱
- `vendor.dcgm.prologScripts` - 動態生成 Prolog 腳本
- `vendor.dcgm.epilogScripts` - 動態生成 Epilog 腳本

**關鍵實現細節:**
```go
// 偵測 nvidia.com/gpu 資源
{{- if index .resources.limits "nvidia.com/gpu" -}}
  {{- $hasGPU = "nvidia.com/gpu" -}}
{{- end -}}
```

---

#### helm/slurm/templates/vendor/nvidia/dcgm/prolog-configmap.yaml
**用途:** DCGM Prolog 腳本 ConfigMap。
**程式碼行數:** 19
**檔案類型:** Kubernetes ConfigMap 模板

---

#### helm/slurm/templates/vendor/nvidia/dcgm/epilog-configmap.yaml
**用途:** DCGM Epilog 腳本 ConfigMap。
**程式碼行數:** 19
**檔案類型:** Kubernetes ConfigMap 模板

---

#### helm/slurm/_vendor/nvidia/dcgm/scripts/prolog/dcgm.sh
**用途:** DCGM GPU-Job 映射 Prolog 腳本，在 Job 啟動時執行。
**程式碼行數:** 44
**檔案類型:** Bash 腳本

**關鍵實現細節:**
```bash
# 解析 CUDA_VISIBLE_DEVICES 環境變數
mapfile -t -d ',' cuda_devs <<<"${CUDA_VISIBLE_DEVICES:-}"

# 為每個 GPU 建立 Job ID 映射檔案
for gpu_id in "${cuda_devs[@]}"; do
  printf "%s" "${SLURM_JOB_ID:-0}" >"${METRICS_DIR:-}/${gpu_id:-99}"
done
```

---

#### helm/slurm/_vendor/nvidia/dcgm/scripts/epilog/dcgm.sh
**用途:** DCGM GPU-Job 映射 Epilog 腳本，在 Job 結束時清理。
**程式碼行數:** ~40
**檔案類型:** Bash 腳本

---

### Chart 2: slurm-operator (Operator 部署)

#### helm/slurm-operator/Chart.yaml
**用途:** 定義 slurm-operator Chart 元資料和依賴關係。
**程式碼行數:** 30
**檔案類型:** Helm Chart 定義

**依賴項目:**
```yaml
dependencies:
  - name: slurm-operator-crds
    version: 1.0.0
    condition: crds.enabled
    repository: file://../slurm-operator-crds
```

---

#### helm/slurm-operator/values.yaml
**用途:** 定義 Operator 和 Webhook 的可配置參數。
**程式碼行數:** 163
**檔案類型:** YAML 配置

**主要配置區塊:**
```yaml
operator:
  enabled: true
  replicas: 1
  image:
    repository: ghcr.io/slinkyproject/slurm-operator
  # 各控制器的工作執行緒數量
  accountingWorkers: 4
  controllerWorkers: 4
  nodesetWorkers: 4
  # ...

webhook:
  enabled: true
  replicas: 1
  image:
    repository: ghcr.io/slinkyproject/slurm-operator-webhook
  serverPort: 9443

certManager:
  enabled: true
  secretName: slurm-operator-webhook-ca
  duration: 43800h0m0s  # 5 年
  renewBefore: 8760h0m0s # 1 年
```

---

#### helm/slurm-operator/templates/_helpers.tpl
**用途:** Operator Chart 的通用輔助函數。
**程式碼行數:** 99
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm-operator.name` - Chart 名稱
- `slurm-operator.fullname` - 完整資源名稱
- `slurm-operator.namespace` - 目標命名空間
- `slurm-operator.operator.labels` - Operator 標籤
- `slurm-operator.operator.selectorLabels` - Pod 選擇器標籤
- `slurm-operator.operator.serviceAccountName` - SA 名稱
- `slurm-operator.imagePullPolicy` - 映像拉取策略
- `slurm-operator.imagePullSecrets` - 映像拉取密鑰
- `slurm-operator.apiGroup` - API Group ("slinky.slurm.net")

---

#### helm/slurm-operator/templates/_operator.tpl
**用途:** Operator 元件特定的輔助函數。
**程式碼行數:** 33
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm-operator.operator.image.repository`
- `slurm-operator.operator.image.tag`
- `slurm-operator.operator.imageRef`
- `slurm-operator.operator.imagePullPolicy`

---

#### helm/slurm-operator/templates/_webhook.tpl
**用途:** Webhook 元件特定的輔助函數。
**程式碼行數:** 71
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm-operator.webhook.name`
- `slurm-operator.webhook.labels`
- `slurm-operator.webhook.selectorLabels`
- `slurm-operator.webhook.serviceAccountName`
- `slurm-operator.webhook.image.repository`
- `slurm-operator.webhook.image.tag`
- `slurm-operator.webhook.imageRef`
- `slurm-operator.webhook.imagePullPolicy`

---

#### helm/slurm-operator/templates/_certManager.tpl
**用途:** cert-manager 整合的輔助函數。
**程式碼行數:** 33
**檔案類型:** Helm 模板

**匯出項目:**
- `slurm-operator.certManager.rootCA` - 根 CA 名稱
- `slurm-operator.certManager.rootIssuer` - 根 Issuer 名稱
- `slurm-operator.certManager.selfCert` - 自簽憑證名稱
- `slurm-operator.certManager.selfIssuer` - 自簽 Issuer 名稱

---

#### helm/slurm-operator/templates/operator/deployment.yaml
**用途:** 定義 Operator Deployment。
**程式碼行數:** 96
**檔案類型:** Kubernetes Deployment 模板

**關鍵實現細節:**
- 配置健康檢查 (liveness/readiness probes)
- 傳遞控制器工作執行緒參數
- 配置 leader election

**命令行參數:**
```yaml
args:
  - --accounting-workers
  - --controller-workers
  - --nodeset-workers
  - --zap-log-level
  - --health-addr
  - --metrics-addr
  - --leader-elect
```

---

#### helm/slurm-operator/templates/operator/rbac.yaml
**用途:** 定義 Operator 的 RBAC 權限。
**程式碼行數:** 161
**檔案類型:** Kubernetes RBAC 模板

**資源類型:**
- ServiceAccount
- ClusterRole
- ClusterRoleBinding

**權限範圍:**
- Core API: configmaps, events, pods, secrets, services, nodes, PVCs
- apps: deployments, statefulsets, controllerrevisions
- monitoring.coreos.com: servicemonitors
- policy: poddisruptionbudgets
- slinky.slurm.net: accountings, controllers, loginsets, nodesets, restapis, tokens
- coordination.k8s.io: leases (用於 leader election)

---

#### helm/slurm-operator/templates/operator/service.yaml
**用途:** 定義 Operator 的 Kubernetes Service。
**程式碼行數:** ~20
**檔案類型:** Kubernetes Service 模板

---

#### helm/slurm-operator/templates/operator/poddisruptionbudget.yaml
**用途:** 定義 Operator 的 PodDisruptionBudget。
**程式碼行數:** ~20
**檔案類型:** Kubernetes PDB 模板

---

#### helm/slurm-operator/templates/webhook/deployment.yaml
**用途:** 定義 Webhook Deployment。
**程式碼行數:** 83
**檔案類型:** Kubernetes Deployment 模板

**關鍵實現細節:**
- RollingUpdate 策略 (maxSurge: 1, maxUnavailable: 0)
- 掛載 TLS 憑證
- 配置健康檢查

---

#### helm/slurm-operator/templates/webhook/webhook.yaml
**用途:** 定義 ValidatingWebhookConfiguration 和 MutatingWebhookConfiguration。
**程式碼行數:** 292
**檔案類型:** Kubernetes Webhook 模板

**驗證 Webhooks (ValidatingWebhookConfiguration):**
- `accounting-v1beta1.kb.io` → `/validate-slinky-slurm-net-v1beta1-accounting`
- `controller-v1beta1.kb.io` → `/validate-slinky-slurm-net-v1beta1-controller`
- `loginset-v1beta1.kb.io` → `/validate-slinky-slurm-net-v1beta1-loginset`
- `nodeset-v1beta1.kb.io` → `/validate-slinky-slurm-net-v1beta1-nodeset`
- `restapi-v1beta1.kb.io` → `/validate-slinky-slurm-net-v1beta1-restapi`
- `token-v1beta1.kb.io` → `/validate-slinky-slurm-net-v1beta1-token`

**變更 Webhooks (MutatingWebhookConfiguration):**
- `podsbinding-v1.kb.io` → `/mutate--v1-binding` (Pod 調度攔截)

**關鍵實現細節:**
- 排除 kube-system 和 operator namespace
- 支援 cert-manager CA 注入
- 備援: 使用 genCA/genSignedCert 自動生成憑證

---

#### helm/slurm-operator/templates/webhook/service.yaml
**用途:** 定義 Webhook 的 Kubernetes Service。
**程式碼行數:** ~25
**檔案類型:** Kubernetes Service 模板

---

#### helm/slurm-operator/templates/webhook/rbac.yaml
**用途:** 定義 Webhook 的 RBAC 權限。
**程式碼行數:** ~50
**檔案類型:** Kubernetes RBAC 模板

---

#### helm/slurm-operator/templates/webhook/poddisruptionbudget.yaml
**用途:** 定義 Webhook 的 PodDisruptionBudget。
**程式碼行數:** ~20
**檔案類型:** Kubernetes PDB 模板

---

#### helm/slurm-operator/templates/cert-manager/pki.yaml
**用途:** 定義 cert-manager 資源以管理 TLS 憑證。
**程式碼行數:** 68
**檔案類型:** cert-manager CRD 模板

**資源類型:**
1. `Issuer` (selfSigned) - 自簽發行者
2. `Certificate` (rootCA) - 根 CA 憑證
3. `Issuer` (rootIssuer) - 使用根 CA 的發行者
4. `Certificate` (serving) - Webhook 服務憑證

---

#### helm/slurm-operator/templates/tests/test-operator.yaml
**用途:** Helm 測試 Pod 驗證 Operator 健康狀態。
**程式碼行數:** ~25
**檔案類型:** Kubernetes Pod 模板

---

#### helm/slurm-operator/templates/tests/test-webhook.yaml
**用途:** Helm 測試 Pod 驗證 Webhook 健康狀態。
**程式碼行數:** ~25
**檔案類型:** Kubernetes Pod 模板

---

### Chart 3: slurm-operator-crds (CRD 定義)

#### helm/slurm-operator-crds/Chart.yaml
**用途:** 定義 CRDs Chart 元資料。
**程式碼行數:** 22
**檔案類型:** Helm Chart 定義

---

#### helm/slurm-operator-crds/values.yaml
**用途:** CRDs Chart 的配置值 (目前為空)。
**程式碼行數:** 3
**檔案類型:** YAML 配置

---

#### helm/slurm-operator-crds/templates/slinky.slurm.net_controllers.yaml
**用途:** 定義 Controller CRD schema。
**程式碼行數:** 713
**檔案類型:** Kubernetes CRD 模板

**CRD 規格:**
- Group: `slinky.slurm.net`
- Version: `v1beta1`
- Kind: `Controller`
- ShortName: `slurmctld`
- Scope: Namespaced

**主要欄位:**
- `spec.accountingRef` - Accounting 參考
- `spec.clusterName` - Slurm 叢集名稱
- `spec.slurmKeyRef` - 認證金鑰參考
- `spec.jwtHs256KeyRef` - JWT 金鑰參考
- `spec.slurmctld` - 容器配置
- `spec.persistence` - 持久化儲存
- `spec.external` - 外部模式
- `spec.configFileRefs` - 配置檔案參考
- `spec.prologScriptRefs` / `spec.epilogScriptRefs` - Prolog/Epilog 腳本

**驗證規則:**
```yaml
x-kubernetes-validations:
- message: externalConfig must be set when external is true
  rule: 'self.external ? has(self.externalConfig) : true'
```

---

#### helm/slurm-operator-crds/templates/slinky.slurm.net_accountings.yaml
**用途:** 定義 Accounting CRD schema (slurmdbd)。
**程式碼行數:** ~400
**檔案類型:** Kubernetes CRD 模板

---

#### helm/slurm-operator-crds/templates/slinky.slurm.net_nodesets.yaml
**用途:** 定義 NodeSet CRD schema (slurmd)。
**程式碼行數:** ~500
**檔案類型:** Kubernetes CRD 模板

---

#### helm/slurm-operator-crds/templates/slinky.slurm.net_loginsets.yaml
**用途:** 定義 LoginSet CRD schema (登入節點)。
**程式碼行數:** ~400
**檔案類型:** Kubernetes CRD 模板

---

#### helm/slurm-operator-crds/templates/slinky.slurm.net_restapis.yaml
**用途:** 定義 RestApi CRD schema (slurmrestd)。
**程式碼行數:** ~300
**檔案類型:** Kubernetes CRD 模板

---

#### helm/slurm-operator-crds/templates/slinky.slurm.net_tokens.yaml
**用途:** 定義 Token CRD schema (JWT Token 管理)。
**程式碼行數:** ~200
**檔案類型:** Kubernetes CRD 模板

---

## 貢獻者檢查清單

### 風險與注意事項

1. **Secret 處理:** `slurmkey.yaml` 和 `jwths256key.yaml` 使用 `lookup` 函數，需注意 Helm 3 的行為差異
2. **CRD 升級:** CRDs 獨立 Chart 設計，需確保版本相容性
3. **cert-manager 依賴:** Webhook 需要 cert-manager 或自動生成憑證
4. **資源單位轉換:** `resource-quantity` 函數處理各種 K8s 資源單位格式

### 變更前驗證步驟

1. 執行 `helm lint helm/slurm`
2. 執行 `helm lint helm/slurm-operator`
3. 執行 `helm template` 驗證輸出
4. 檢查 values.yaml 的文檔註解完整性

### PR 前建議測試

- [ ] `helm template --debug helm/slurm > /dev/null`
- [ ] `helm template --debug helm/slurm-operator > /dev/null`
- [ ] `helm install --dry-run slurm ./helm/slurm`
- [ ] 驗證 CRD schema 變更的向後相容性
- [ ] 檢查 NOTES.txt 輸出正確性

---

## 架構與設計模式

### 程式碼組織

三層 Chart 架構：
1. **slurm-operator-crds:** 基礎層，定義 API 結構
2. **slurm-operator:** 控制層，部署 Operator 和 Webhook
3. **slurm:** 應用層，宣告式定義 Slurm 叢集

### 設計模式

- **Template 繼承:** 使用 `_helpers.tpl` 集中管理命名邏輯
- **條件渲染:** 廣泛使用 `{{- if }}` 控制資源生成
- **迴圈處理:** 使用 `range` 處理 nodesets/loginsets/partitions
- **資源策略:** 使用 `helm.sh/resource-policy: keep` 保護關鍵 Secrets

### 狀態管理策略

- CRD 狀態由 Operator 控制器更新
- 使用 Kubernetes Status subresource
- 支援 Conditions 標準化狀態報告

### 錯誤處理哲學

- Helm 模板使用 `required` 函數驗證必要參數
- 使用 `fail` 函數在配置錯誤時終止渲染
- CRD 使用 `x-kubernetes-validations` 進行 CEL 驗證

---

## 資料流

```
                                 ┌─────────────────────────────────┐
                                 │         Helm Values             │
                                 │  (values.yaml / --set / -f)     │
                                 └─────────────┬───────────────────┘
                                               │
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Helm Template Engine                             │
│                                                                              │
│   ┌─────────────┐     ┌─────────────┐     ┌──────────────┐                  │
│   │ _helpers.tpl│◄────┤ _slurm.tpl  │◄────┤ controller/  │                  │
│   │             │     │             │     │ _helpers.tpl │                  │
│   └──────┬──────┘     └──────┬──────┘     └──────┬───────┘                  │
│          │                   │                   │                           │
│          └───────────────────┴───────────────────┘                           │
│                              │                                               │
│                              ▼                                               │
│                    ┌─────────────────┐                                       │
│                    │ Rendered YAML    │                                       │
│                    │ Resources        │                                       │
│                    └────────┬─────────┘                                       │
└─────────────────────────────┼────────────────────────────────────────────────┘
                              │
                              ▼
              ┌───────────────────────────────────┐
              │         Kubernetes API            │
              └───────────────┬───────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
   │   Secrets    │    │  ConfigMaps  │    │     CRDs     │
   │ (auth keys)  │    │ (scripts)    │    │(Controller,  │
   │              │    │              │    │ NodeSet...)  │
   └──────────────┘    └──────────────┘    └──────┬───────┘
                                                  │
                                                  ▼
                                    ┌─────────────────────────┐
                                    │    Slurm Operator       │
                                    │  (reconciliation loop)  │
                                    └───────────┬─────────────┘
                                                │
                    ┌───────────────────────────┼───────────────────────────┐
                    │                           │                           │
                    ▼                           ▼                           ▼
          ┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐
          │    StatefulSet   │      │    Deployment    │      │     Service      │
          │ (slurmctld, etc) │      │  (slurmrestd)    │      │  (networking)    │
          └──────────────────┘      └──────────────────┘      └──────────────────┘
```

### 資料入口點

- **values.yaml:** 預設配置
- **用戶 values:** `--set` 或 `-f custom-values.yaml`
- **環境變數:** 透過 `env` 欄位傳遞

### 資料轉換

- **模板渲染:** Helm Go templates → YAML
- **資源單位轉換:** `resource-quantity` 輔助函數
- **名稱格式化:** `include "slurm.fullname"` 等

### 資料出口點

- Kubernetes API (apply rendered resources)
- Operator 控制器 (watch CRDs, create workloads)

---

## 整合點

### 消費的外部 API

| 端點 | 說明 | 認證方式 |
|------|------|----------|
| Kubernetes API | 建立/更新資源 | ServiceAccount |
| cert-manager API | 憑證管理 | Cluster-scoped |
| Prometheus API | ServiceMonitor CRD | 無 |
| MariaDB | slurmdbd 資料儲存 | 密碼 |
| LDAP | SSSD 用戶認證 | 配置於 sssd.conf |

### 暴露的 API

| 端點 | 說明 | 方法 |
|------|------|------|
| slurmrestd:6820 | Slurm REST API | HTTP |
| slurmctld:6817 | Slurm 控制器 | TCP |
| slurmdbd:6819 | 會計資料庫 | TCP |
| LoginSet SSH:22 | 登入節點 | SSH |

### 共享狀態

| 狀態名稱 | 說明 | 存取者 |
|----------|------|--------|
| slurm.key | Slurm 認證金鑰 | slurmctld, slurmd, slurmdbd, slurmrestd |
| jwt_hs256.key | JWT 簽名金鑰 | slurmctld, slurmrestd |
| sssd.conf | SSSD 配置 | LoginSet, NodeSet (SSH) |
| slurm.conf | Slurm 配置 | 所有 Slurm 元件 |

---

## 依賴關係圖

```
slurm-operator
    │
    └──depends on──► slurm-operator-crds (crds.enabled)
                          │
                          └──provides CRDs for──►
                                                  │
    ┌─────────────────────────────────────────────┘
    │
    ▼
slurm
    │
    ├──creates──► Controller CR ◄──watches── slurm-operator
    │                  │
    │                  └──references──► Accounting CR (if enabled)
    │
    ├──creates──► NodeSet CRs ◄──watches── slurm-operator
    │                  │
    │                  └──references──► Controller CR
    │
    ├──creates──► LoginSet CRs ◄──watches── slurm-operator
    │                  │
    │                  └──references──► Controller CR
    │
    ├──creates──► RestApi CR ◄──watches── slurm-operator
    │                  │
    │                  └──references──► Controller CR
    │
    └──creates──► Secrets, ConfigMaps, PriorityClass
```

### 入口點 (不被其他元件 import)

- `helm/slurm/values.yaml`
- `helm/slurm-operator/values.yaml`
- `helm/slurm-operator-crds/values.yaml`

### 葉節點 (不 import 其他元件)

- `helm/slurm/_vendor/nvidia/dcgm/scripts/*.sh`
- `helm/slurm/templates/NOTES.txt`
- `helm/slurm-operator-crds/templates/*.yaml`

### 循環依賴

✓ 未偵測到循環依賴

---

## 測試分析

### 測試覆蓋率摘要

Helm Charts 主要透過以下方式測試：
- `helm lint` - 語法驗證
- `helm template --debug` - 渲染驗證
- `helm test` - 運行時測試 (test-operator.yaml, test-webhook.yaml)

### 測試檔案

| 測試檔案 | 測試數量 | 方法 |
|----------|----------|------|
| helm/slurm-operator/templates/tests/test-operator.yaml | 1 | HTTP health check |
| helm/slurm-operator/templates/tests/test-webhook.yaml | 1 | HTTP health check |

### 測試缺口

- 缺少 slurm Chart 的 Helm 測試
- 無自動化 E2E 測試
- 無 values.yaml schema 驗證測試

---

## 相關程式碼與重用機會

### 類似功能

| 功能 | 路徑 | 相似度 |
|------|------|--------|
| 資源命名 | `helm/*/templates/_helpers.tpl` | 相同模式 |
| 容器格式化 | `helm/slurm/templates/_helpers.tpl:format-container` | 可抽取為共享 |
| RBAC 定義 | `helm/slurm-operator/templates/*/rbac.yaml` | 重複模式 |

### 可重用工具

| 工具 | 路徑 | 用途 |
|------|------|------|
| `resource-quantity` | `helm/slurm/templates/_helpers.tpl:124` | Kubernetes 資源單位轉換 |
| `toYaml-set-storageClassName` | `helm/slurm/templates/_helpers.tpl:144` | StorageClass 處理 |

### 推薦遵循的模式

- 使用 `slurm.fullname` 作為資源名稱前綴
- 使用 `slurm.labels` 添加標準標籤
- 使用 `format-container` 處理容器規格
- 使用 `helm.sh/resource-policy: keep` 保護 Secrets

---

## 實現說明

### 程式碼品質觀察

- ✓ 良好的 SPDX 授權標頭
- ✓ 豐富的 values.yaml 註解
- ✓ 使用 helm-docs 生成 README
- ⚠ 部分模板缺少內聯註解

### TODOs 和未來工作

無明確的 TODO 註解發現。

### 已知問題

- `lookup` 函數在 `helm template` 時不可用
- CRDs Chart 需要單獨管理升級

### 優化機會

1. 將通用 helpers 抽取為 library chart
2. 添加 values.yaml JSON schema
3. 增加 Helm 測試覆蓋率

### 技術債務

1. slurm-operator Chart 對 cert-manager 的硬依賴
2. 缺少 CRD 版本遷移支援

---

## 修改指南

### 新增功能

1. 在 `values.yaml` 添加新配置欄位 (含註解)
2. 更新相關 `_helpers.tpl` 添加輔助函數
3. 在 templates 中使用新配置
4. 執行 `helm-docs` 更新 README
5. 添加測試驗證

### 修改現有功能

1. 查找使用該功能的所有模板
2. 考慮向後相容性
3. 更新 values.yaml 預設值和註解
4. 執行 `helm lint` 和 `helm template`

### 移除/棄用

1. 標記欄位為 deprecated (在註解中)
2. 添加向後相容的預設值
3. 在下個主版本中移除
4. 更新 CRD 驗證規則 (如適用)

### 變更測試檢查清單

- [ ] `helm lint helm/slurm` 無警告
- [ ] `helm lint helm/slurm-operator` 無警告
- [ ] `helm template` 輸出正確的 YAML
- [ ] CRD 變更不破壞現有 CR
- [ ] NOTES.txt 輸出正確
- [ ] helm-docs 生成的 README 正確

---

_由 `document-project` 工作流程生成 (deep-dive 模式)_
_掃描日期: 2025-12-22_
_分析模式: 詳盡_
