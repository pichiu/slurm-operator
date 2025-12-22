# Slurm Job 與 Storage 深入分析：Pyxis vs 動態 NodeSet

> 由 BMAD Party Mode 討論產生
> 日期: 2025-12-22

---

## 1. 議題背景

### 原始問題
> 如果 Job 需要指定的 storage（特定的訓練資料或特定的模型），應該使用 Pyxis 還是動態建立 NodeSet Pod？

### 環境條件
- **多租戶環境**：每個 namespace 代表不同的 project/客戶
- **資料隔離需求**：不同租戶的資料不能被其他人看見
- **共享計算資源**：整座 K8s 共用 NodeSet
- **客戶存取限制**：客戶無法直接操作 K8s，透過 Backend API 提交 Job

---

## 2. 最終決定

### ✅ 採用方案：Pyxis + 共享 NodeSet + Backend API 控制

| 決策項目 | 結論 |
|---------|------|
| Job 執行方式 | Pyxis 容器化 Job |
| NodeSet 模式 | 共享（單一 namespace，所有租戶共用）|
| 資料隔離機制 | `--container-mounts` 限制掛載路徑 |
| 安全邊界 | Backend API（客戶無法直接操作 K8s/Slurm）|
| Storage 結構 | NFS `/data/{tenant_id}/`, `/models/{tenant_id}/` |

---

## 3. 架構設計

### 3.1 整體架構圖

```
┌──────────────────────────────────────────────────────────────────┐
│                        多租戶 Slurm 平台架構                      │
│                                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                          │
│  │ 客戶 A  │  │ 客戶 B  │  │ 客戶 C  │                          │
│  │ (Web UI)│  │ (API)   │  │ (SDK)   │                          │
│  └────┬────┘  └────┬────┘  └────┬────┘                          │
│       │            │            │                                │
│       └────────────┼────────────┘                                │
│                    ▼                                             │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Backend API Server                        ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  ││
│  │  │ 租戶認證    │  │ 權限驗證    │  │ Job 生成器          │  ││
│  │  │ (JWT/OAuth) │  │ (RBAC)      │  │ (安全 mount 路徑)   │  ││
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘  ││
│  └─────────────────────────────────────────────────────────────┘│
│                    │                                             │
│                    │ sbatch (Backend 生成，客戶無法修改)         │
│                    ▼                                             │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    ns: slurm                                 ││
│  │  ┌───────────────────────────────────────────────────────┐  ││
│  │  │              共享 NodeSet (所有租戶共用)               │  ││
│  │  │  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐             │  ││
│  │  │  │ Pod-0 │ │ Pod-1 │ │ Pod-2 │ │ Pod-3 │  ...        │  ││
│  │  │  │slurmd │ │slurmd │ │slurmd │ │slurmd │             │  ││
│  │  │  │+pyxis │ │+pyxis │ │+pyxis │ │+pyxis │             │  ││
│  │  │  └───────┘ └───────┘ └───────┘ └───────┘             │  ││
│  │  └───────────────────────────────────────────────────────┘  ││
│  │                              │                               ││
│  │                              ▼                               ││
│  │  ┌───────────────────────────────────────────────────────┐  ││
│  │  │              共享 NFS Storage                          │  ││
│  │  │  /data/tenant-a/     ← Pyxis Job A 只能看這裡         │  ││
│  │  │  /data/tenant-b/     ← Pyxis Job B 只能看這裡         │  ││
│  │  │  /data/tenant-c/     ← Pyxis Job C 只能看這裡         │  ││
│  │  │  /models/tenant-a/   ← 模型同樣隔離                    │  ││
│  │  │  /models/tenant-b/                                     │  ││
│  │  │  /models/tenant-c/                                     │  ││
│  │  └───────────────────────────────────────────────────────┘  ││
│  └─────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────┘
```

### 3.2 Job 執行流程

```
客戶透過 Web/API 提交訓練請求
         │
         ▼
┌─────────────────────────────────────────┐
│ Backend 驗證層                          │
│ 1. 驗證 JWT/API Key                     │
│ 2. 確認租戶身份 (tenant_id)              │
│ 3. 驗證請求的 dataset 屬於該租戶         │
│ 4. 驗證請求的 model 屬於該租戶           │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Backend 生成安全的 sbatch 命令          │
│                                         │
│ sbatch --partition=shared               │
│   srun --container-image=<safe_image>   │
│        --container-mounts=              │
│          /data/tenant-123:/data         │  ◄── Backend 硬編碼
│          /models/tenant-123:/models     │  ◄── 客戶無法修改
│        python train.py                  │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Slurm 排程 + Pyxis 執行                 │
│                                         │
│ Job 容器內可見：                         │
│   /data   → 實際是 /data/tenant-123     │
│   /models → 實際是 /models/tenant-123   │
│                                         │
│ Job 無法看到其他租戶的任何資料 ✓         │
└─────────────────────────────────────────┘
```

---

## 4. 方案比較

### 4.1 Pyxis vs 動態 NodeSet

| 比較項目 | Pyxis + 共享 NodeSet | 動態建立 NodeSet |
|---------|---------------------|-----------------|
| **資源利用率** | ✅ 高（共享計算資源） | ⚠️ 中（可能閒置）|
| **啟動延遲** | ✅ 低（Pod 已運行） | ❌ 高（30s-2min）|
| **架構複雜度** | ✅ 低 | ❌ 高 |
| **維護成本** | ✅ 低 | ❌ 高 |
| **資料隔離** | ✅ 透過 mount 控制 | ✅ 原生 namespace 隔離 |
| **安全性** | ✅ Backend 控制下安全 | ✅ 更強隔離 |

### 4.2 為什麼選擇 Pyxis？

1. **資源效率**：共享 NodeSet 避免資源閒置
2. **快速啟動**：Job 直接在已運行的 Pod 中執行
3. **簡化管理**：無需為每個租戶管理獨立 NodeSet
4. **Backend 控制**：客戶無法繞過，安全有保障

---

## 5. 安全機制

### 5.1 安全保證

| 威脅 | 防護機制 | 風險等級 |
|------|---------|---------|
| 惡意掛載路徑 | Backend 硬編碼 mount 路徑 | ✅ 無風險 |
| 客戶直接操作 K8s | 無 kubectl 存取權限 | ✅ 無風險 |
| 客戶直接操作 Slurm | 無 sbatch/srun 存取權限 | ✅ 無風險 |
| 容器逃逸 | Pyxis/Enroot 安全配置 | ⚠️ 需配置 |
| 資料洩漏 | container-mounts 隔離 | ✅ 低風險 |

### 5.2 Pyxis/Enroot 安全配置

```bash
# /etc/enroot/enroot.conf
ENROOT_ROOTFS_WRITABLE=no        # 唯讀 rootfs
ENROOT_MOUNT_HOME=no             # 不掛載 host home
ENROOT_ALLOW_SUPERUSER=no        # 禁止容器內 root
```

### 5.3 Backend 驗證邏輯

```python
# Backend API 範例
@app.post("/api/v1/jobs")
async def submit_job(
    request: JobRequest,
    tenant: Tenant = Depends(get_current_tenant)
):
    # ✅ 強制使用租戶自己的資料路徑
    allowed_data_mount = f"/data/{tenant.id}"
    allowed_model_mount = f"/models/{tenant.id}"

    # ✅ 驗證請求的 dataset 路徑
    if not request.dataset_path.startswith(allowed_data_mount):
        raise HTTPException(403, "無權存取此資料集")

    # ✅ 驗證請求的 model 路徑
    if request.model_path and not request.model_path.startswith(allowed_model_mount):
        raise HTTPException(403, "無權存取此模型")

    # ✅ 生成安全的 sbatch 命令
    sbatch_script = f"""#!/bin/bash
#SBATCH --job-name={tenant.id}-{request.job_name}
#SBATCH --partition=shared

srun --container-image={request.image} \\
     --container-mounts={allowed_data_mount}:/data,{allowed_model_mount}:/models \\
     python {request.script}
"""

    job_id = await slurm_client.submit(sbatch_script)
    return {"job_id": job_id}
```

---

## 6. 配置範例

### 6.1 NodeSet Helm Values

```yaml
# values.yaml
nodesets:
  shared:
    enabled: true
    replicas: 8
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd-pyxis
        tag: 25.11-ubuntu24.04
      volumeMounts:
        - name: data
          mountPath: /data
        - name: models
          mountPath: /models
    podSpec:
      volumes:
        - name: data
          nfs:
            server: nfs.example.com
            path: /exports/data
        - name: models
          nfs:
            server: nfs.example.com
            path: /exports/models
```

### 6.2 NFS 目錄結構

```
/exports/
├── data/
│   ├── tenant-a/
│   │   ├── datasets/
│   │   │   ├── imagenet/
│   │   │   └── coco/
│   │   └── checkpoints/
│   ├── tenant-b/
│   │   └── ...
│   └── tenant-c/
│       └── ...
└── models/
    ├── tenant-a/
    │   ├── llama-7b/
    │   └── bert-base/
    ├── tenant-b/
    │   └── ...
    └── tenant-c/
        └── ...
```

### 6.3 Pyxis plugstack 配置

```yaml
# values.yaml - configFiles
configFiles:
  plugstack.conf: |
    include /usr/share/pyxis/*
```

---

## 7. 驗證測試

### 7.1 隔離性測試

```bash
# 測試 1: 確認 Job 只能看到自己的資料
# 以 tenant-a 身份提交
srun --container-image=alpine \
     --container-mounts=/data/tenant-a:/data \
     ls /data
# 預期結果: 只顯示 tenant-a 的資料

# 測試 2: 確認無法存取其他租戶
# 嘗試列出 parent directory
srun --container-image=alpine \
     --container-mounts=/data/tenant-a:/data \
     ls /data/../
# 預期結果: 只能看到 /data (mount point)，無法遍歷到其他租戶
```

### 7.2 Backend 驗證測試

```bash
# 測試 3: API 層面的權限驗證
curl -X POST https://api.example.com/v1/jobs \
  -H "Authorization: Bearer <tenant-a-token>" \
  -d '{
    "dataset_path": "/data/tenant-b/datasets/imagenet",
    "image": "pytorch:latest"
  }'
# 預期結果: 403 Forbidden - 無權存取此資料集
```

---

## 8. 相關檔案索引

| 檔案 | 用途 |
|------|------|
| `docs/usage/pyxis.md` | Pyxis 配置指南 |
| `helm/slurm/values.yaml` | NodeSet 和 Volume 配置 |
| `helm/slurm/templates/nodeset/nodeset-cr.yaml` | NodeSet CR 模板 |
| `_bmad-output/deep-dive-nodeset-storage.md` | NodeSet Storage 深入分析 |

---

## 9. 結論

對於多租戶環境下 Job 需要特定 Storage 的場景：

- **選擇 Pyxis + 共享 NodeSet**，而非動態建立 NodeSet
- **透過 Backend API 控制 `--container-mounts`** 實現資料隔離
- **客戶無法直接操作 K8s/Slurm**，安全邊界在 Backend

這個架構在保證資料隔離的同時，最大化了計算資源的利用率，並保持了系統的簡潔性和可維護性。

---

_由 BMAD Party Mode 討論產生_
_參與專家: Winston (架構師), Amelia (開發者), Barry (Quick Flow), Murat (測試架構師), Mary (業務分析師)_
_日期: 2025-12-22_
