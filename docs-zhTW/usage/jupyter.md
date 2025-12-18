# JupyterLab 部署指南 (JupyterLab Deployment Guide)

## TL;DR

本指南說明如何在使用 Slurm-operator 的 Kubernetes 上部署 JupyterLab。主要步驟包括：修改 slurmd 映像以安裝 JupyterLab、配置 Helm chart 開放埠、使用 sbatch 啟動 JupyterLab，以及透過 port-forward 存取。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [JupyterLab 部署指南](#jupyterlab-部署指南-jupyterlab-deployment-guide)
  - [目錄](#目錄)
  - [概述](#概述)
  - [在 `slurmd` 映像中安裝 JupyterLab](#在-slurmd-映像中安裝-jupyterlab)
  - [修改 Slurm Helm chart 的 values 以支援 JupyterLab](#修改-slurm-helm-chart-的-values-以支援-jupyterlab)
  - [使用 `sbatch` 啟動 JupyterLab](#使用-sbatch-啟動-jupyterlab)
  - [存取 Slinky 叢集中執行的 JupyterLab 實例](#存取-slinky-叢集中執行的-jupyterlab-實例)

<!-- mdformat-toc end -->

## 概述

[JupyterLab] 是一個高度可擴展、功能豐富的筆記本編寫應用程式和編輯環境。對於想要在 Slurm-operator 上部署互動式運算環境的使用者來說，JupyterLab 已被證明是一個可靠且易於配置的解決方案。

本文件提供在使用 Slurm-operator 於 Kubernetes 中執行的 Slurm 叢集上安裝和部署 JupyterLab 的指引。

## 在 `slurmd` 映像中安裝 JupyterLab

Slurm Helm Chart 依賴 SchedMD 發布的多個[容器映像][container images]。這些映像可以由執行 Slurm-operator 的站點按需修改，以為使用者提供自訂的程式設計環境。關於這些容器映像的更多資訊可以在[容器儲存庫][container images]中找到。

要在 Slinky 叢集上執行 JupyterLab，請修改工作節點 NodeSet 的 slurmd Pod 使用的映像，以安裝 Python 和 JupyterLab 軟體。根據 Slinky 容器的[映像建置文件][image build documentation]，這些修改應該在 `base-extra` 容器層中進行。

1. 複製 [Slinky 容器儲存庫][Slinky Container Repository]：

```bash
git clone git@github.com:SlinkyProject/containers.git
cd containers
```

2. 編輯您偏好版本和發行版 Dockerfile 中的 `base-extra` 層，以安裝 JupyterLab pip 套件。每個版本和風格的目錄可以在容器儲存庫的 `schedmd/slurm/` 目錄中找到。

要在 `rockylinux9` 映像中安裝 JupyterLab，可以如下修改 `base-extra` 區段：

```dockerfile
################################################################################
FROM base AS base-extra

SHELL ["bash", "-c"]

RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked <<EOR
# 安裝額外套件
set -xeuo pipefail
dnf -q -y install \
  python3
dnf -q -y install \
  python3-pip
pip install jupyterlab
EOR

################################################################################
```

3. 對 `base-extra` 容器層進行修改後，可以建置包含這些變更的 `slurmd` 映像：

```bash
export VERSION=25.11
export FLAVOR="rockylinux9"
export BAKE_IMPORTS="--file ./docker-bake.hcl --file ./$VERSION/$FLAVOR/slurm.hcl"
docker bake $BAKE_IMPORTS slurmd
```

映像建置完成後，應套用標籤以區分此映像與 SchedMD 發布的映像：

```bash
docker image ls | head -n 2
REPOSITORY                                                                       TAG                                                                IMAGE ID       CREATED          SIZE
ghcr.io/slinkyproject/slurmd                                                     25.11-rockylinux9                                                  9434c2d4fae4   43 seconds ago   712MB

docker tag 9434c2d4fae4 ghcr.io/slinkyproject/slurmd:jupyterlab
```

4. 接下來，修改後的映像需要上傳到您環境中的容器登錄檔 (Container Registry)。方法因雲端供應商和軟體堆疊而異。

## 修改 Slurm Helm chart 的 values 以支援 JupyterLab

Slurm Helm chart 的 `values.yaml` 檔案必須修改以部署 JupyterHub。具體而言，需要將 `nodesets.slinky.slurmd.image.repository`、`nodesets.slinky.slurmd.image.tag` 的值更改為參考您映像所在的登錄檔，並且必須修改 `nodesets.slinky.slurmd.ports` 以開放 JupyterLab 提供服務的埠。

在 Slurm-operator 環境中執行 JupyterLab 的最小 `values.yaml`：

```yaml
nodesets:
  slinky:
    enabled: true
    replicas: 1
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: jupyterlab
      ports:
        - containerPort: 9999
```

## 使用 `sbatch` 啟動 JupyterLab

在 Slurm 中，[`sbatch`] 指令用於使用腳本提交批次作業。以下是一個基本的 [`sbatch`] 腳本，可用於在 Slinky 叢集上啟動 JupyterLab：

```bash
#!/bin/bash
#SBATCH --job-name=jupyterlab-singleuser
#SBATCH --account=vivian
#SBATCH --nodes=1
#SBATCH --time=01:00:00
#
## 啟動 JupyterLab 伺服器：
jupyter lab --port=9999 --no-browser
```

## 存取 Slinky 叢集中執行的 JupyterLab 實例

提交上述 sbatch 腳本後，當產生的作業被分配資源後，JupyterHub 實例將在排程到的 Slurm 工作節點 Pod 的埠 9999 上提供服務。

[`kubectl port-forward`] 提供將本地埠轉發到 Pod 埠的方法。要存取在 `slurm-worker-slinky-0` Pod 的埠 9999 上執行的 JupyterLab 實例，可以使用以下指令：

```bash
kubectl port-forward -n slurm slurm-worker-slinky-0 8081:9999
```

在該 Pod 中執行的 JupyterLab 實例現在可以在您的瀏覽器中透過 http://localhost:8081 存取。

<!-- Links -->

[container images]: https://github.com/orgs/SlinkyProject/packages
[image build documentation]: https://slinky.schedmd.com/projects/containers/en/latest/build.html#extending-the-images-with-additional-software
[jupyterlab]: https://jupyterlab.readthedocs.io/en/stable/
[slinky container repository]: https://github.com/SlinkyProject/containers
[`kubectl port-forward`]: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_port-forward/
[`sbatch`]: https://slurm.schedmd.com/sbatch.html

---

## Explanation

### 為何要在 Slurm 上執行 JupyterLab？

JupyterLab 結合 Slurm 的優勢：
- **互動式開發**：在 Jupyter 中進行探索性資料分析
- **資源排程**：透過 Slurm 管理運算資源
- **可擴展性**：需要更多資源時可以提交大型作業
- **隔離性**：每個使用者的 Jupyter 環境獨立執行

### 工作流程說明

1. **建置自訂映像**：將 JupyterLab 包含在 slurmd 映像中
2. **配置埠開放**：在 values.yaml 中設定容器埠
3. **提交作業**：使用 sbatch 在工作節點上啟動 JupyterLab
4. **存取服務**：透過 port-forward 或 LoadBalancer 存取

---

## Practical Example

### 完整的 JupyterLab 部署流程

```bash
# 1. 準備自訂 values.yaml
cat > values-jupyter.yaml <<EOF
nodesets:
  slinky:
    enabled: true
    replicas: 2
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: jupyterlab
      ports:
        - containerPort: 9999   # JupyterLab 埠
        - containerPort: 8888   # 備用埠
EOF

# 2. 安裝/升級 Slurm 叢集
helm upgrade --install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values-jupyter.yaml \
  --namespace=slurm --create-namespace

# 3. 等待 Pod 就緒
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=slurmd -n slurm --timeout=300s
```

### 建立並提交 JupyterLab 作業

```bash
# 1. 將 sbatch 腳本複製到登入 Pod
kubectl exec -n slurm deployment/slurm-login-slinky -- bash -c 'cat > /tmp/jupyter.sh << EOF
#!/bin/bash
#SBATCH --job-name=jupyter
#SBATCH --nodes=1
#SBATCH --time=02:00:00
#SBATCH --partition=slinky

# 啟動 JupyterLab（不開啟瀏覽器）
jupyter lab --port=9999 --no-browser --ip=0.0.0.0
EOF'

# 2. 提交作業
kubectl exec -n slurm deployment/slurm-login-slinky -- sbatch /tmp/jupyter.sh

# 3. 查看作業狀態
kubectl exec -n slurm deployment/slurm-login-slinky -- squeue

# 4. 確認作業在哪個節點上執行
kubectl exec -n slurm deployment/slurm-login-slinky -- squeue -o "%i %N"
```

### 存取 JupyterLab

```bash
# 假設作業在 slurm-worker-slinky-0 上執行
# 設定 port-forward
kubectl port-forward -n slurm slurm-worker-slinky-0 8888:9999 &

# 現在可以在瀏覽器中開啟 http://localhost:8888
echo "JupyterLab 可透過 http://localhost:8888 存取"

# 查看 JupyterLab 日誌以取得 token
kubectl exec -n slurm slurm-worker-slinky-0 -- cat /root/slurm-*.out
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| 無法連接到 JupyterLab | 確認 port-forward 正在執行且埠正確 |
| 作業立即結束 | 檢查 JupyterLab 是否正確安裝在映像中 |
| Token 找不到 | 查看 sbatch 輸出檔案中的 token |
| 連接被拒絕 | 確認 `--ip=0.0.0.0` 參數已設定 |

### 小技巧

1. **設定固定 Token**：使用 `--NotebookApp.token='your-token'` 設定固定 token
2. **使用 Service**：考慮建立 Kubernetes Service 而非每次 port-forward
3. **配置 SSL**：生產環境應啟用 HTTPS
4. **資源限制**：在 sbatch 中設定適當的資源限制

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `sbatch jupyter.sh` | 提交 JupyterLab 作業 |
| `squeue` | 查看作業狀態 |
| `kubectl port-forward -n slurm <pod> 8888:9999` | 轉發 JupyterLab 埠 |
| `scancel <job_id>` | 取消 JupyterLab 作業 |
| `docker bake $BAKE_IMPORTS slurmd` | 建置自訂 slurmd 映像 |
