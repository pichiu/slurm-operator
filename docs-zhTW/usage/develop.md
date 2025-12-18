# 開發 (Development)

## TL;DR

本文件提供 slurm-operator 專案開發所需的資訊。包括安裝開發相依套件（pre-commit、Docker、Helm、Skaffold、kubectl）、在叢集上執行、安裝/解除安裝 CRDs，以及在本地執行 Operator 進行偵錯。

---

## Translation

本文件旨在提供足夠的資訊，讓您可以開始在此專案上進行開發。

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [開發](#開發-development)
  - [目錄](#目錄)
  - [開始之前](#開始之前)
    - [相依套件](#相依套件)
      - [Pre-Commit](#pre-commit)
      - [Docker](#docker)
      - [Helm](#helm)
      - [Skaffold](#skaffold)
      - [Kubernetes Client](#kubernetes-client)
    - [在叢集上執行](#在叢集上執行)
      - [自動化](#自動化)
  - [Operator](#operator)
    - [安裝 CRDs](#安裝-crds)
    - [解除安裝 CRDs](#解除安裝-crds)
    - [修改 API 定義](#修改-api-定義)
    - [Slurm 版本變更](#slurm-版本變更)
    - [在本地執行 Operator](#在本地執行-operator)
    - [Slurm 叢集](#slurm-叢集)

<!-- mdformat-toc end -->

## 開始之前

您需要一個 Kubernetes 叢集來執行。您可以使用 [KIND] 取得用於測試的本地叢集，或執行您選擇的遠端叢集。

**注意**：您的控制器會自動使用 kubeconfig 檔案中的當前 context（即 `kubectl cluster-info` 顯示的任何叢集）。

### 相依套件

安裝 [KIND] 和 [Golang] 二進位檔以供 [pre-commit] hooks 使用。

```sh
sudo apt-get install golang
make install
```

#### Pre-Commit

安裝 [pre-commit] 並安裝 git hooks。

```sh
sudo apt-get install pre-commit
pre-commit install
```

#### Docker

安裝 [Docker] 並配置[無 root Docker][rootless-docker]。

之後，測試您的使用者帳戶可以與 docker 通訊。

```sh
docker run hello-world
```

#### Helm

安裝 [Helm]。

```sh
sudo snap install helm --classic
```

#### Skaffold

安裝 [Skaffold]。

```sh
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
```

如果已安裝 [google-cloud-sdk]，[skaffold] 可作為額外元件使用。

```sh
sudo apt-get install -y google-cloud-cli-skaffold
```

#### Kubernetes Client

安裝 [kubectl]。

```sh
sudo snap install kubectl --classic
```

如果已安裝 [google-cloud-sdk]，[kubectl] 可作為額外元件使用。

```sh
sudo apt-get install -y kubectl
```

### 在叢集上執行

對於開發，所有 [Helm] 部署使用 `values-dev.yaml`。如果它們在您的環境中尚不存在或您不確定，請執行以下指令安全地複製 `values.yaml` 作為基礎：

```sh
make values-dev
```

#### 自動化

您可以使用 [Skaffold] 建置和推送映像，並使用以下指令部署元件：

```sh
cd helm/slurm-operator/
skaffold run
```

**注意**：`skaffold.yaml` 配置為將映像和標籤注入到 `values-dev.yaml` 中，以便正確參考它們。

## Operator

slurm operator 旨在遵循 Kubernetes [Operator 模式][operator-pattern]。

它使用[控制器][operator-controller]，提供負責同步資源直到在叢集上達到所需狀態的 reconcile 函數。

### 安裝 CRDs

使用 [skaffold] 或 [helm] 部署 [helm] chart 時，如果叢集中尚不存在 CRDs，則會安裝其 `crds/` 目錄中定義的 CRDs。

### 解除安裝 CRDs

要從叢集中刪除 Operator CRDs：

```sh
make uninstall
```

> [!WARNING]
> CRDs 不會升級！必須先解除安裝舊的 CRDs，才能安裝新的。這應該只在開發中進行。

### 修改 API 定義

如果您正在編輯 API 定義，請使用以下指令生成資源清單，如 CRs 或 CRDs：

```sh
make manifests
```

### Slurm 版本變更

如果 Slurm 版本已變更，請使用以下指令生成新的 OpenAPI 規格及其 golang 用戶端程式碼：

```sh
make generate
```

> [!NOTE]
> 根據 [slurmrestd 外掛程式生命週期][plugin-lifecycle]更新與 API 互動的程式碼。

### 在本地執行 Operator

使用 `make install` 安裝 Operator 的 CRDs。

透過 VSCode 偵錯器使用「Launch Operator」啟動任務啟動 Operator。

因為 Operator 將在 Kubernetes 外部執行並需要與 Slurm 叢集通訊，請在您的 Slurm helm chart 的 `values.yaml` 中設定以下選項：

- `debug.enable=true`
- `debug.localOperator=true`

如果在 [Kind] 叢集上執行，還需設定：

- `debug.disableCgroups=true`

如果 Slurm [helm] chart 正在使用 [skaffold] 部署，請執行 `skaffold run --port-forward --tail`。它配置為自動[埠轉發][skaffold-port-forwarding] restapi，以便本地 Operator 與 Slurm 叢集通訊。

如果未使用 [skaffold]，請手動執行 `kubectl port-forward --namespace slurm services/slurm-restapi 6820:6820`，以便本地 Operator 與 Slurm 叢集通訊。

啟動 Operator 後，透過檢查 Cluster CR 是否已標記為 ready 來驗證它能夠聯繫 Slurm 叢集：

```sh
$ kubectl get --namespace slurm clusters.slinky.slurm.net
NAME     READY   AGE
slurm    true    110s
```

請參閱 [skaffold 埠轉發][skaffold-port-forwarding]以了解 [skaffold] 如何自動偵測要轉發的服務。

### Slurm 叢集

進入可以提交工作負載的 Slurm Pod。

```bash
kubectl --namespace=slurm exec -it deployments/slurm-login -- bash -l
kubectl --namespace=slurm exec -it statefulsets/slurm-controller -- bash -l
```

```bash
cloud-provider-kind -enable-lb-port-mapping &
SLURM_LOGIN_PORT="$(kubectl --namespace=slurm get services -l app.kubernetes.io/name=login,app.kubernetes.io/instance=slurm -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ports[0].port}")"
SLURM_LOGIN_IP="$(kubectl --namespace=slurm get services -l app.kubernetes.io/name=login,app.kubernetes.io/instance=slurm -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ip}")"
ssh -p "$SLURM_LOGIN_PORT" "${USER}@${SLURM_LOGIN_IP}"
```

<!-- Links -->

[docker]: https://docs.docker.com/engine/install/
[golang]: https://go.dev/
[google-cloud-sdk]: https://cloud.google.com/sdk/docs/install
[helm]: https://helm.sh/
[kind]: https://kind.sigs.k8s.io/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/
[operator-controller]: https://kubernetes.io/docs/concepts/architecture/controller/
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[plugin-lifecycle]: https://slurm.schedmd.com/rest.html#lifecycle
[pre-commit]: https://pre-commit.com/
[rootless-docker]: https://docs.docker.com/engine/security/rootless/
[skaffold]: https://skaffold.dev/docs/install/
[skaffold-port-forwarding]: https://skaffold.dev/docs/port-forwarding/

---

## Explanation

### 開發工具鏈概述

| 工具 | 用途 |
|-----|------|
| **KIND** | 本地 Kubernetes 叢集，用於開發測試 |
| **Docker** | 建置和執行容器映像 |
| **Helm** | Kubernetes 套件管理器 |
| **Skaffold** | 自動化建置、推送和部署流程 |
| **kubectl** | Kubernetes 命令列工具 |
| **pre-commit** | Git hooks 管理，確保程式碼品質 |

### 開發流程

1. **設定環境**：安裝所有相依套件
2. **建立叢集**：使用 KIND 建立本地 Kubernetes 叢集
3. **安裝 CRDs**：將自訂資源定義安裝到叢集
4. **開發迭代**：修改程式碼、建置、部署、測試
5. **提交變更**：pre-commit hooks 確保程式碼品質

---

## Practical Example

### 設定開發環境

```bash
# 1. 複製專案
git clone https://github.com/SlinkyProject/slurm-operator.git
cd slurm-operator

# 2. 安裝 Go 相依套件
go mod download

# 3. 安裝 pre-commit hooks
pre-commit install

# 4. 建立開發用的 values 檔案
make values-dev

# 5. 建立 KIND 叢集
./hack/kind.sh
```

### 使用 Skaffold 部署

```bash
# 進入 Operator 目錄
cd helm/slurm-operator/

# 建置並部署（持續監控變更）
skaffold dev

# 或一次性部署
skaffold run

# 查看部署狀態
kubectl get pods -n slinky
```

### 本地偵錯 Operator

```bash
# 1. 安裝 CRDs
make install

# 2. 部署 Slurm 叢集（啟用偵錯模式）
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set 'debug.enable=true' \
  --set 'debug.localOperator=true' \
  --set 'debug.disableCgroups=true' \
  --namespace=slurm --create-namespace

# 3. 設定埠轉發
kubectl port-forward --namespace slurm services/slurm-restapi 6820:6820 &

# 4. 在本地執行 Operator（或使用 VSCode 偵錯器）
go run ./cmd/manager/main.go

# 5. 驗證連線
kubectl get --namespace slurm clusters.slinky.slurm.net
```

### 修改 API 並重新生成

```bash
# 修改 api/ 目錄下的程式碼後
# 重新生成 CRD manifests
make manifests

# 如果 Slurm 版本變更，重新生成用戶端程式碼
make generate

# 重新安裝 CRDs
make install
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| KIND 叢集連線失敗 | 確認 kubeconfig context 正確 |
| Docker 權限問題 | 配置 rootless Docker 或將使用者加入 docker 群組 |
| CRD 未更新 | 先執行 `make uninstall` 再重新安裝 |
| Operator 無法連線 Slurm | 確認埠轉發正在執行 |

### 小技巧

1. **使用 Skaffold dev 模式**：自動偵測變更並重新部署
2. **保持 CRDs 同步**：修改 API 後記得執行 `make manifests`
3. **使用偵錯日誌**：增加日誌級別有助於排錯
4. **定期清理**：定期清理舊的 Docker 映像和 KIND 叢集

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `make install` | 安裝 CRDs |
| `make uninstall` | 解除安裝 CRDs |
| `make manifests` | 生成 CRD manifests |
| `make generate` | 生成用戶端程式碼 |
| `make values-dev` | 建立開發用 values 檔案 |
| `skaffold run` | 建置並部署 |
| `skaffold dev` | 開發模式（持續監控） |
| `./hack/kind.sh` | 建立 KIND 叢集 |
