# Pyxis 指南 (Pyxis Guide)

## TL;DR

本指南說明如何配置 Slurm 叢集使用 Pyxis（和 enroot），這是一個支援 Nvidia GPU 的容器化作業 SPANK 外掛。主要步驟包括：配置 `plugstack.conf`、使用 pyxis 映像建立 NodeSet 和登入 Pod，並設定適當的安全上下文。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [Pyxis 指南](#pyxis-指南-pyxis-guide)
  - [目錄](#目錄)
  - [概述](#概述)
  - [配置](#配置)
  - [測試](#測試)

<!-- mdformat-toc end -->

## 概述

本指南說明如何配置您的 Slurm 叢集使用 [pyxis]（和 [enroot]），這是一個支援 Nvidia GPU 的容器化作業 Slurm [SPANK] 外掛。

## 配置

配置 `plugstack.conf` 以包含 pyxis 配置。

> [!WARNING]
> 在 `plugstack.conf` 中，您必須使用 glob 語法以避免 slurmctld 在嘗試解析 includes 中的路徑時失敗。只有登入和 slurmd Pod 應該實際安裝 pyxis 函式庫。

```yaml
configFiles:
  plugstack.conf: |
    include /usr/share/pyxis/*
  ...
```

配置一個或多個 NodeSet 和登入 Pod 使用 pyxis OCI 映像。

```yaml
loginsets:
  - name: pyxis
    image:
      repository: ghcr.io/slinkyproject/login-pyxis
    ...
nodesets:
  - name: pyxis
    image:
      repository: ghcr.io/slinkyproject/slurmd-pyxis
    ...
```

為了讓登入容器中的 enroot 活動可被允許，它需要 `securityContext.privileged=true`。

```yaml
loginsets:
  - name: pyxis
    image:
      repository: ghcr.io/slinkyproject/login-pyxis
    securityContext:
      privileged: true
    ...
```

## 測試

提交作業到 Slurm 節點。

```bash
$ srun --partition=pyxis grep PRETTY /etc/os-release
PRETTY_NAME="Ubuntu 24.04.2 LTS"
```

使用 pyxis 提交作業到 Slurm 節點，它將在請求的容器中啟動。

```bash
$ srun --partition=pyxis --container-image=alpine:latest grep PRETTY /etc/os-release
pyxis: importing docker image: alpine:latest
pyxis: imported docker image: alpine:latest
PRETTY_NAME="Alpine Linux v3.21"
```

> [!WARNING]
> SPANK 外掛只能在安裝並配置使用它們的特定 Slurm 節點上運作。最好使用 `--partition=<partition>`、`--batch=<features>` 和/或 `--constraint=<features>` 來限制作業執行的位置，以確保相容的運算環境。

如果登入容器有 `securityContext.privileged=true`，enroot 活動是可被允許的。您可以使用以下指令測試功能：

```bash
enroot import docker://alpine:latest
```

<!-- Links -->

[enroot]: https://github.com/NVIDIA/enroot
[pyxis]: https://github.com/NVIDIA/pyxis
[spank]: https://slurm.schedmd.com/spank.html

---

## Explanation

### 什麼是 Pyxis？

Pyxis 是 NVIDIA 開發的 Slurm SPANK 外掛，它允許：
- 在 Slurm 作業中使用容器
- 無縫整合 Docker/enroot 容器
- 支援 GPU 直通

### 什麼是 SPANK？

SPANK (Slurm Plug-in Architecture for Node and Job Control) 是 Slurm 的外掛系統，允許：
- 擴展 Slurm 功能
- 在作業執行的不同階段插入自訂邏輯
- 新增自訂命令列選項（如 `--container-image`）

### Pyxis 工作流程

1. 使用者提交帶有 `--container-image` 的作業
2. Pyxis 攔截作業提交
3. enroot 拉取並轉換容器映像
4. 作業在容器環境中執行
5. 容器結束後清理

---

## Practical Example

### 完整的 Pyxis 配置

```yaml
# values-pyxis.yaml
# 配置 plugstack.conf
configFiles:
  plugstack.conf: |
    include /usr/share/pyxis/*

# 配置 pyxis 登入 Pod
loginsets:
  pyxis:
    enabled: true
    image:
      repository: ghcr.io/slinkyproject/login-pyxis
      tag: latest
    securityContext:
      privileged: true    # 必要：允許 enroot 操作

# 配置 pyxis 工作節點
nodesets:
  pyxis:
    enabled: true
    replicas: 3
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd-pyxis
        tag: latest
```

### 安裝並測試

```bash
# 1. 安裝 Slurm 叢集（使用 pyxis 配置）
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values-pyxis.yaml \
  --namespace=slurm --create-namespace

# 2. 等待 Pod 就緒
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=slurmd -n slurm --timeout=300s

# 3. 進入登入 Pod
kubectl exec -it -n slurm deployment/slurm-login-pyxis -- bash
```

### 執行容器化作業

```bash
# 基本測試：執行 Alpine 容器
srun --partition=pyxis --container-image=alpine:latest cat /etc/os-release

# 使用 NVIDIA 容器（需要 GPU 節點）
srun --partition=pyxis \
  --container-image=nvcr.io/nvidia/pytorch:24.01-py3 \
  --gres=gpu:1 \
  python -c "import torch; print(torch.cuda.is_available())"

# 批次作業
sbatch << 'EOF'
#!/bin/bash
#SBATCH --partition=pyxis
#SBATCH --container-image=ubuntu:22.04
#SBATCH --nodes=1

echo "在容器中執行"
cat /etc/os-release
EOF

# 預先拉取映像（在登入節點）
enroot import docker://pytorch/pytorch:latest
```

### 進階用法

```bash
# 掛載主機目錄到容器
srun --partition=pyxis \
  --container-image=python:3.11 \
  --container-mounts=/data:/data:ro \
  python /data/script.py

# 使用自訂 enroot 配置
srun --partition=pyxis \
  --container-image=custom-image:latest \
  --container-env=MY_VAR=value \
  my_program
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| `--container-image` 無法識別 | 確認作業提交到配置了 pyxis 的分割區 |
| 映像拉取失敗 | 檢查網路連線和登錄檔認證 |
| 權限錯誤（enroot） | 在登入 Pod 啟用 `privileged: true` |
| SPANK 外掛未載入 | 驗證 `plugstack.conf` 配置正確 |

### Pyxis 指令列選項

| 選項 | 說明 |
|-----|------|
| `--container-image=<image>` | 指定容器映像 |
| `--container-mounts=<src>:<dst>` | 掛載目錄 |
| `--container-env=<VAR>=<value>` | 設定環境變數 |
| `--container-workdir=<path>` | 設定工作目錄 |

### 小技巧

1. **預先拉取映像**：在登入節點使用 `enroot import` 可加速後續作業
2. **使用分割區限制**：確保只在支援 pyxis 的節點上執行容器作業
3. **配置登錄檔認證**：設定 `$ENROOT_SQUASH_OPTIONS` 以使用私有登錄檔
4. **監控儲存空間**：容器映像會佔用大量空間，定期清理

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `srun --container-image=<image> <cmd>` | 在容器中執行指令 |
| `sbatch --container-image=<image> script.sh` | 提交容器化批次作業 |
| `enroot import docker://<image>` | 預先拉取 Docker 映像 |
| `enroot list` | 列出已匯入的映像 |
| `enroot remove <image>` | 移除映像 |
| `sinfo -p pyxis` | 查看 pyxis 分割區狀態 |
