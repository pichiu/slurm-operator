# 使用教學 (Usage Tutorial)

## TL;DR

本教學提供在使用 Slurm Operator 的 Slurm 上執行 PyTorch 的工作流程範例。主要步驟包括：修改容器映像安裝 PyTorch、配置 Helm values 檔案、建立 sbatch 腳本執行 PyTorch 範例（圖卷積網路 GCN）。

---

## Translation

## 概述

本指南提供在使用 Slurm Operator 於 Slurm 上簡單使用 Pytorch 的範例工作流程。在本教學結束時，[圖卷積網路 (Graph Convolutional Network)][Graph Convolutional Network] Pytorch 範例應該可以在 Kubernetes 內的 Slurm 叢集上執行。

> [!WARNING]
> 這些說明不適用於正式環境。

## 先決條件

- 已部署啟用 LoginSet 的 `slurm-operator`。
- 在本地複製以下儲存庫的最新官方發行版：
  - Slurm Operator：[slurm-operator]
  - Slinky 容器：[containers]

## 建置映像

詳細說明可在[建置 Slinky 容器][Building Slinky Containers]中找到。

Slinky 專案使用 Docker Buildx Bake 為每個 Slurm 元件建置模組化映像。為了擴展或更改功能，必須修改包含映像建置配置的 Dockerfile。例如，使用 Slurm 25.11 和 Rocky Linux 9 建置的映像配置可以在 [containers] 儲存庫的以下位置找到：`containers/schedmd/slurm/25.11/rockylinux9/Dockerfile`。

要為此範例安裝 Pytorch，請更改 `base-extra` 層中 `dnf` 安裝的套件。這可以透過將所需的 Python 和 Pytorch 套件附加到清單中、複製 Pytorch 儲存庫，並從其目錄中透過 pip 安裝 GCN 範例的相依套件來完成。以下是如何執行此操作的範例：

```dockerfile
################################################################################
FROM base AS base-extra

SHELL ["bash", "-c"]

RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked <<EOR
# 安裝額外套件
set -xeuo pipefail
dnf -q -y install --setopt='install_weak_deps=False' \
  openmpi python3-pip python-setuptools python3-scipy
EOR

# pytorch
RUN git clone --depth=2 https://github.com/pytorch/examples.git
WORKDIR /tmp/examples/gcn
RUN pip install -r requirements.txt

################################################################################
```

進行修改後，可以透過設定 `SUFFIX` 環境變數來標記映像，這會在您選擇建置的版本標籤上附加 `-<SUFFIX>`。然後，從容器儲存庫的根目錄，可以執行以下指令建置映像，其中 `<version>` 必須替換為 Slurm 版本，`<distribution>` 替換為所需的發行版：

```bash
docker bake --file containers/schedmd/slurm/docker-bake.hcl --file containers/schedmd/slurm/<version>/<distribution>/slurm.hcl
```

建置映像後，必須透過容器登錄檔在您的叢集上提供它們。

## 載入

Slurm Helm chart 的 [values 檔案][values file]可以在[此處][slurm-values]下載，或可以在 [slurm-operator] 儲存庫的 `helm/slurm/values.yaml` 中找到。`nodesets.slinky.slurmd.image` 的值（其中 `slinky` 替換為您的 NodeSet 名稱）可用於指定 Slurm Operator 用於部署該 NodeSet 的 slurmd Pod 的映像儲存庫和標籤。這些值必須修改為參考託管您在上一步建置的映像的容器登錄檔和套用於它們的標籤。

> [!NOTE]
> 注意 [Kubernetes 預設設定的 ImagePullPolicy][set by default by Kubernetes]。

## 執行

可以透過在 Slurm 中執行 Pytorch 來測試功能。建立以下 sbatch 腳本，檔名為 `pytorch-sbatch.sh`：

```bash
#!/bin/bash
# Slurm 參數
#SBATCH -n 3                      # 執行 n 個任務
#SBATCH -N 3                      # 跨 N 個節點執行
#SBATCH -t 0-00:10                # 時間限制，格式為 D-HH:MM
#SBATCH -p slinky                 # 提交到的分割區
#SBATCH --mem=100                 # 所有核心的記憶體池
#SBATCH -o pytorch-output_%j.out  # STDOUT 將寫入的檔案，%j 插入 jobid
#SBATCH -e pytorch-errors_%j.err  # STDERR 將寫入的檔案，%j 插入 jobid

echo "這在 $SLURM_JOB_NODELIST 上執行。"
srun -D /tmp/examples/gcn python3 main.py --epochs 200 --lr 0.01 --l2 5e-4 --dropout-p 0.5 --hidden-dim 16 --val-every 20 --include-bias
```

此腳本在三個節點上執行 GCN Pytorch 範例，請根據需要修改或調整規模。它必須複製到您的 `slurm-login-slinky` Pod，可以使用以下指令完成，其中 `<hash>` 替換為當前 slurm-login-slinky Pod 的雜湊值：

```bash
kubectl -n slurm cp ~/location/of/script/pytorch-sbatch.sh slurm-login-slinky-<hash>:/root/
```

然後可以從 slurm-login-slinky 節點執行腳本：

```bash
sbatch pytorch-sbatch.sh
```

輸出應該在名為 `pytorch-output_<JOB>.out` 的檔案中，建立在執行的工作節點上。任何錯誤都會在 `pytorch-errors_<JOB>.err` 中。要取回這些檔案，可以使用以下指令從 `slurm-worker` 節點複製它們，其中 `<N>` 替換為作業執行的 Pod 的序號，`<JOB>` 替換為 jobid：

```
kubectl -n slurm cp slurm-worker-slinky-<N>:/root/pytorch-output_<JOB>.out .
```

如需更多閱讀，請參閱 [Slinky] 和 [Slurm] 文件。

<!-- Links -->

[building slinky containers]: https://slinky.schedmd.com/projects/containers/en/latest/build.html
[containers]: https://github.com/SlinkyProject/containers
[graph convolutional network]: https://github.com/pytorch/examples/tree/main/gcn
[set by default by kubernetes]: https://kubernetes.io/docs/concepts/containers/images/#updating-images
[slinky]: https://slinky.schedmd.com/en/latest/
[slurm]: https://slurm.schedmd.com/documentation.html
[slurm-operator]: https://github.com/SlinkyProject/slurm-operator
[slurm-values]: https://raw.githubusercontent.com/SlinkyProject/slurm-operator/refs/tags/v1.0.0/helm/slurm/values.yaml
[values file]: https://helm.sh/docs/chart_template_guide/values_files/

---

## Explanation

### 為何在 Slurm 上執行 PyTorch？

在 HPC 環境中執行深度學習任務的優勢：
- **資源管理**：Slurm 有效分配 GPU 和其他資源
- **作業排程**：自動處理作業排隊和優先權
- **可擴展性**：輕鬆擴展到多節點訓練
- **隔離性**：每個作業有獨立的資源配額

### 工作流程概述

1. **準備映像**：在容器中安裝 PyTorch 和相依套件
2. **配置 Helm**：設定 NodeSet 使用自訂映像
3. **撰寫腳本**：建立 sbatch 腳本指定作業參數
4. **提交作業**：使用 sbatch 提交並監控執行

---

## Practical Example

### 完整的 PyTorch 範例流程

```bash
# 1. 準備 values.yaml，指定 PyTorch 映像
cat > values-pytorch.yaml <<EOF
nodesets:
  slinky:
    enabled: true
    replicas: 3
    slurmd:
      image:
        repository: your-registry.io/slurmd
        tag: pytorch
EOF

# 2. 安裝/升級 Slurm 叢集
helm upgrade --install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values-pytorch.yaml \
  --namespace=slurm --create-namespace

# 3. 等待 Pod 就緒
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=slurmd -n slurm --timeout=300s
```

### 撰寫並執行 PyTorch 作業

```bash
# 1. 建立 sbatch 腳本
cat > pytorch-job.sh << 'EOF'
#!/bin/bash
#SBATCH --job-name=pytorch-gcn      # 作業名稱
#SBATCH --nodes=3                    # 使用 3 個節點
#SBATCH --ntasks=3                   # 3 個任務
#SBATCH --time=00:30:00              # 最長執行 30 分鐘
#SBATCH --partition=slinky           # 分割區名稱
#SBATCH --output=pytorch_%j.out      # 輸出檔案
#SBATCH --error=pytorch_%j.err       # 錯誤檔案

# 顯示節點資訊
echo "執行節點: $SLURM_JOB_NODELIST"
echo "作業 ID: $SLURM_JOB_ID"

# 執行 PyTorch GCN 範例
srun -D /tmp/examples/gcn python3 main.py \
  --epochs 200 \
  --lr 0.01 \
  --l2 5e-4 \
  --dropout-p 0.5 \
  --hidden-dim 16 \
  --val-every 20 \
  --include-bias
EOF

# 2. 複製腳本到登入 Pod
LOGIN_POD=$(kubectl get pods -n slurm -l app.kubernetes.io/name=login -o jsonpath='{.items[0].metadata.name}')
kubectl cp pytorch-job.sh -n slurm ${LOGIN_POD}:/tmp/

# 3. 提交作業
kubectl exec -n slurm ${LOGIN_POD} -- sbatch /tmp/pytorch-job.sh

# 4. 監控作業狀態
kubectl exec -n slurm ${LOGIN_POD} -- squeue

# 5. 查看作業詳細資訊
kubectl exec -n slurm ${LOGIN_POD} -- sacct
```

### 取得輸出結果

```bash
# 找出作業執行的節點
JOB_ID=123  # 替換為實際的作業 ID
kubectl exec -n slurm ${LOGIN_POD} -- sacct -j ${JOB_ID} --format=JobID,NodeList

# 從工作節點複製輸出檔案
WORKER_POD=slurm-worker-slinky-0  # 替換為實際的 Pod 名稱
kubectl cp -n slurm ${WORKER_POD}:/root/pytorch_${JOB_ID}.out ./pytorch_output.txt

# 查看輸出
cat pytorch_output.txt
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| 映像中找不到 PyTorch | 確認 Dockerfile 正確安裝 PyTorch |
| 記憶體不足 | 增加 `--mem` 參數或減少批次大小 |
| 作業逾時 | 增加 `--time` 參數 |
| 找不到輸出檔案 | 檢查作業執行的節點並從該節點複製 |

### sbatch 參數說明

| 參數 | 說明 |
|-----|------|
| `-n, --ntasks` | 總任務數 |
| `-N, --nodes` | 節點數 |
| `-t, --time` | 時間限制（D-HH:MM:SS） |
| `-p, --partition` | 分割區名稱 |
| `--mem` | 記憶體需求 |
| `-o, --output` | 標準輸出檔案 |
| `-e, --error` | 錯誤輸出檔案 |

### 小技巧

1. **使用 GPU**：如果需要 GPU，使用 `--gres=gpu:N` 參數
2. **監控進度**：使用 `squeue -u $USER` 持續監控
3. **檢查資源**：先用 `sinfo` 確認可用資源
4. **測試小規模**：先用小資料集測試再擴展

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `sbatch script.sh` | 提交批次作業 |
| `squeue` | 查看作業佇列 |
| `scancel <job_id>` | 取消作業 |
| `sacct -j <job_id>` | 查看作業帳務 |
| `sinfo` | 查看叢集狀態 |
| `kubectl cp <src> <dest>` | 複製檔案到/從 Pod |
