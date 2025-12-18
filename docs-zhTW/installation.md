# 安裝指南 (Installation Guide)

## TL;DR

本指南說明如何在 Kubernetes 上安裝 Slurm Operator。主要步驟包括：安裝 cert-manager、安裝 slurm-operator 及其 CRDs，最後部署 Slurm 叢集。可選配置包括持久化儲存、帳務系統、指標監控和登入功能。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [安裝指南](#安裝指南-installation-guide)
  - [目錄](#目錄)
  - [概述](#概述)
  - [Slurm Operator 和 CRDs](#slurm-operator-和-crds)
    - [CRDs 作為子 Chart](#crds-作為子-chart)
    - [不使用 cert-manager](#不使用-cert-manager)
  - [Slurm 叢集](#slurm-叢集)
    - [控制器持久化](#控制器持久化)
    - [啟用帳務功能](#啟用帳務功能)
      - [MariaDB（社群版）](#mariadb社群版)
    - [啟用指標監控](#啟用指標監控)
    - [啟用登入功能](#啟用登入功能)
      - [設定 root 授權金鑰](#設定-root-授權金鑰)
      - [測試 Slurm](#測試-slurm)

<!-- mdformat-toc end -->

## 概述

本文件提供在 Kubernetes 上安裝 Slurm Operator 的說明。

## Slurm Operator 和 CRDs

若尚未安裝 [cert-manager]，請先安裝它及其 CRDs：

```sh
# 新增 jetstack Helm 儲存庫
helm repo add jetstack https://charts.jetstack.io
# 更新儲存庫
helm repo update
# 安裝 cert-manager，並啟用 CRDs
helm install cert-manager jetstack/cert-manager \
  --set 'crds.enabled=true' \
  --namespace cert-manager --create-namespace
```

安裝 slurm-operator 及其 CRDs：

```sh
# 安裝 slurm-operator 的 CRDs
helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds
# 安裝 slurm-operator
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace
```

檢查 slurm-operator 是否成功部署：

```sh
kubectl --namespace=slinky get pods --selector='app.kubernetes.io/instance=slurm-operator'
```

輸出應類似於：

```sh
NAME                                      READY   STATUS    RESTARTS   AGE
slurm-operator-5d86d75979-6wflf           1/1     Running   0          1m
slurm-operator-webhook-567c84547b-kr7zq   1/1     Running   0          1m
```

### CRDs 作為子 Chart

如果您想在同一個 Helm release 中管理 slurm-operator 和 CRDs，請使用 `--set 'crds.enabled=true'` 參數安裝。

```sh
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --set 'crds.enabled=true' \
  --namespace=slinky --create-namespace
```

### 不使用 cert-manager

如果未安裝 [cert-manager]，請使用 `--set 'certManager.enabled=false'` 參數安裝，以避免透過 cert-manager 簽署憑證。

```sh
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --set 'certManager.enabled=false' \
  --namespace=slinky --create-namespace
```

## Slurm 叢集

透過 Helm chart 安裝 Slurm 叢集：

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace
```

檢查 Slurm 叢集是否成功部署：

```sh
kubectl --namespace=slurm get pods
```

輸出應類似於：

```sh
NAME                                  READY   STATUS    RESTARTS   AGE
slurm-accounting-0                    1/1     Running   0          2m
slurm-controller-0                    3/3     Running   0          2m
slurm-login-slinky-7ff66445b5-wdjkn   1/1     Running   0          2m
slurm-restapi-77b9f969f7-kh4r8        1/1     Running   0          2m
slurm-worker-slinky-0                 2/2     Running   0          2m
```

> [!NOTE]
> 上述輸出是在所有 Slurm 元件都已啟用並正確配置的情況下。

### 控制器持久化

預設情況下，Slurm 控制器 (slurmctld) Pod 會將其[狀態儲存][statesavelocation]資料保存到[持久卷 (Persistent Volume, PV)][persistent-volume]。其[持久卷宣告 (Persistent Volume Claim, PVC)][persistent-volume] 會請求 Kubernetes [預設儲存類別 (Storage Class)][default-storageclass]。

如果未定義預設儲存類別或需要特定的儲存類別，您可以使用 `--set "controller.persistence.storageClassName=$STORAGE_CLASS"` 參數安裝 Slurm，其中 `$STORAGE_CLASS` 對應到現有的儲存類別。

```sh
kubectl get storageclasses.storage.k8s.io
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set "controller.persistence.storageClassName=$STORAGE_CLASS" \
  --namespace=slurm --create-namespace
```

> [!NOTE]
> 通常 PV 在 PVC 刪除後不會被刪除。因此，不再需要時可能需要手動刪除 PV。

如果不需要 Slurm 控制器 (slurmctld) 持久化（通常用於測試），可以使用 `--set 'controller.persistence.enabled=false'` 參數停用。

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set 'controller.persistence.enabled=false' \
  --namespace=slurm --create-namespace
```

> [!WARNING]
> 沒有 Slurm 控制器持久化，Slurm 叢集的狀態會在控制器 Pod 重啟之間遺失。此外，這些重啟可能會影響叢集的運作和執行中的工作負載。因此，**不建議**在正式環境中停用持久化。

### 啟用帳務功能

您需要配置 Slurm 帳務 (Accounting) 指向資料庫。有多種方法可以為 Slurm 提供資料庫。

可以使用：

- [mariadb-operator]
- [mysql-operator]
- 任何 Slurm 相容的資料庫
  - mysql/mariadb 相容的替代方案
  - 託管雲端資料庫服務

#### MariaDB（社群版）

如果您打算啟用帳務功能，請先安裝 [mariadb-operator] 及其 CRDs（若尚未安裝）：

```sh
helm repo add mariadb-operator https://helm.mariadb.com/mariadb-operator
helm repo update
helm install mariadb-operator-crds mariadb-operator/mariadb-operator-crds
helm install mariadb-operator mariadb-operator/mariadb-operator \
  --namespace mariadb --create-namespace
```

建立 slurm 命名空間。

```sh
kubectl create namespace slurm
```

透過 CR 建立 MariaDB 資料庫。

```sh
kubectl apply -f - <<EOF
apiVersion: k8s.mariadb.com/v1alpha1
kind: MariaDB
metadata:
  name: mariadb
  namespace: slurm
spec:
  rootPasswordSecretKeyRef:
    name: mariadb-root
    key: password
    generate: true
  username: slurm
  database: slurm_acct_db
  passwordSecretKeyRef:
    name: mariadb-password
    key: password
    generate: true
  storage:
    size: 16Gi
  myCnf: |
    [mariadb]
    bind-address=*
    default_storage_engine=InnoDB
    binlog_format=row
    innodb_autoinc_lock_mode=2
    innodb_buffer_pool_size=4096M
    innodb_lock_wait_timeout=900
    innodb_log_file_size=1024M
    max_allowed_packet=256M
EOF
```

> [!NOTE]
> 上述 MariaDB 資料庫範例與 Slurm chart 的預設 `accounting.storageConfig` 一致。如果您的實際資料庫配置不同，則需要更新 `accounting.storageConfig` 以配合您的配置。

然後使用 `--set 'accounting.enabled=true'` 參數透過 Helm chart 安裝 Slurm 叢集。

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set 'accounting.enabled=true' \
  --namespace=slurm --create-namespace
```

### 啟用指標監控

如果您打算收集指標 (Metrics)，請先安裝 Prometheus 及其 CRDs（若尚未安裝）：

```sh
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack \
  --set 'installCRDs=true' \
  --namespace prometheus --create-namespace
```

然後啟用 Slurm 指標和 Prometheus 服務監控器 (Service Monitor)，以進行指標發現。

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set 'controller.metrics.enabled=true' \
  --set 'controller.metrics.serviceMonitor.enabled=true' \
  --namespace=slurm --create-namespace
```

### 啟用登入功能

您需要配置 Slurm chart，讓登入 Pod 可以透過 [sssd] 與身分識別服務通訊。

> [!WARNING]
> 在此範例中，您需要提供一個為您環境配置的 `sssd.conf`（位於 `${HOME}/sssd.conf`）。

使用 `--set 'loginsets.slinky.enabled=true'` 和 `--set-file "loginsets.slinky.sssdConf=${HOME}/sssd.conf"` 參數透過 Helm chart 安裝 Slurm 叢集。

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set 'loginsets.slinky.enabled=true' \
  --set-file "loginsets.slinky.sssdConf=${HOME}/sssd.conf" \
  --namespace=slurm --create-namespace
```

#### 設定 root 授權金鑰

> [!NOTE]
> 即使 [sssd] 配置錯誤，此方法仍可用於 SSH 進入 Pod。

使用 `--set 'loginsets.slinky.enabled=true'` 和 `--set-file "loginsets.slinky.rootSshAuthorizedKeys=${HOME}/.ssh/id_ed25519.pub"` 參數透過 Helm chart 安裝 Slurm 叢集。

```sh
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set 'loginsets.slinky.enabled=true' \
  --set-file "loginsets.slinky.rootSshAuthorizedKeys=${HOME}/.ssh/id_ed25519.pub" \
  --namespace=slurm --create-namespace
```

#### 測試 Slurm

透過登入服務 SSH：

```sh
SLURM_LOGIN_IP="$(kubectl get services -n slurm slurm-login-slinky -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
SLURM_LOGIN_PORT="$(kubectl get services -n slurm slurm-login-slinky -o jsonpath='{.status.loadBalancer.ingress[0].ports[0].port}')"
## 假設您的公開 SSH 金鑰已在 `loginsets.slinky.rootSshAuthorizedKeys` 中配置。
ssh -p ${SLURM_LOGIN_PORT:-22} root@${SLURM_LOGIN_IP}
## 假設 SSSD 已正確配置。
ssh -p ${SLURM_LOGIN_PORT:-22} ${USER}@${SLURM_LOGIN_IP}
```

然後，從登入 Pod 執行 Slurm 指令，快速測試 Slurm 是否正常運作：

```sh
sinfo
srun hostname
sbatch --wrap="sleep 60"
squeue
sacct
```

請參閱 [Slurm 指令][slurm-commands]以了解更多與 Slurm 互動的詳細資訊。

<!-- Links -->

[cert-manager]: https://cert-manager.io/docs/installation/helm/
[default-storageclass]: https://kubernetes.io/docs/concepts/storage/storage-classes/#default-storageclass
[mariadb-operator]: https://github.com/mariadb-operator/mariadb-operator/blob/main/docs/helm.md
[mysql-operator]: https://dev.mysql.com/doc/mysql-operator/en/mysql-operator-installation-helm.html
[persistent-volume]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
[slurm-commands]: https://slurm.schedmd.com/quickstart.html#commands
[sssd]: https://sssd.io/
[statesavelocation]: https://slurm.schedmd.com/slurm.conf.html#OPT_StateSaveLocation

---

## Explanation

### 安裝流程概述

1. **cert-manager**：提供 TLS 憑證管理，用於 Webhook 安全通訊
2. **slurm-operator-crds**：安裝自訂資源定義（NodeSet、Cluster 等）
3. **slurm-operator**：部署 Operator 控制器
4. **slurm**：部署實際的 Slurm 叢集

### 各元件說明

| 元件 | 說明 |
|-----|------|
| **slurm-operator** | 主要控制器，監控和管理 Slurm 資源 |
| **slurm-operator-webhook** | 驗證 Webhook，確保 CR 規格正確 |
| **slurm-controller** | Slurm 控制器 (slurmctld) |
| **slurm-accounting** | Slurm 帳務守護程式 (slurmdbd) |
| **slurm-restapi** | Slurm REST API (slurmrestd) |
| **slurm-worker** | Slurm 工作節點 (slurmd) |
| **slurm-login** | 登入節點 |

---

## Practical Example

### 完整安裝腳本

```bash
#!/bin/bash
# 完整的 Slurm Operator 安裝腳本

# 1. 安裝 cert-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --set 'crds.enabled=true' \
  --namespace cert-manager --create-namespace

# 等待 cert-manager 就緒
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=120s

# 2. 安裝 slurm-operator
helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds
helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --namespace=slinky --create-namespace

# 等待 Operator 就緒
kubectl wait --for=condition=Available deployment/slurm-operator -n slinky --timeout=120s

# 3. 安裝 Slurm 叢集
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --namespace=slurm --create-namespace

# 4. 驗證安裝
kubectl get pods -n slinky
kubectl get pods -n slurm
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| cert-manager 未就緒就安裝 Operator | 使用 `kubectl wait` 確保 cert-manager 就緒 |
| 忘記建立命名空間 | 使用 `--create-namespace` 自動建立 |
| StorageClass 不存在 | 先執行 `kubectl get sc` 確認可用的儲存類別 |
| Webhook 憑證錯誤 | 確認 cert-manager 正常運行 |

### 小技巧

1. **使用 values.yaml**：複雜配置建議使用 values 檔案而非命令列參數
2. **檢查 Events**：安裝失敗時用 `kubectl describe` 查看事件
3. **保留安裝指令**：記錄使用的 Helm 參數以便日後維護
4. **測試環境先行**：在測試環境驗證配置後再套用到正式環境

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `helm repo add jetstack https://charts.jetstack.io` | 新增 cert-manager 儲存庫 |
| `helm install cert-manager jetstack/cert-manager --set 'crds.enabled=true' -n cert-manager --create-namespace` | 安裝 cert-manager |
| `helm install slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds` | 安裝 Operator CRDs |
| `helm install slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator -n slinky --create-namespace` | 安裝 Operator |
| `helm install slurm oci://ghcr.io/slinkyproject/charts/slurm -n slurm --create-namespace` | 安裝 Slurm 叢集 |
| `kubectl get pods -n slinky` | 檢查 Operator Pod |
| `kubectl get pods -n slurm` | 檢查 Slurm Pod |
