# Helm NodeSet 管理指南

> 最後更新：2025-12-23
> 相關文件：[FAQ](./slurm-faq.md) | [使用指南](./slurm-usage-guide.md) | [NodeSet API 參考](./nodeset-api-reference.md)

---

## 目錄

- [基本結構](#基本結構)
- [常見操作](#常見操作)
- [手動 CR 轉換為 Helm 管理](#手動-cr-轉換為-helm-管理)
- [多環境管理](#多環境管理)
- [版本控制與回滾](#版本控制與回滾)
- [最佳實踐](#最佳實踐)

---

## 基本結構

使用 Helm 管理 NodeSet 時，所有設定都在 `values.yaml` 中定義：

```yaml
# values.yaml
nodesets:
  <nodeset-name>:        # NodeSet 名稱，可以有多個
    enabled: true
    replicas: 3
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.05-rockylinux9
      resources:
        requests:
          cpu: "1"
          memory: "2Gi"
    partition:
      enabled: true
```

---

## 常見操作

### 新增 NodeSet

在 `values.yaml` 中新增區塊：

```yaml
nodesets:
  compute:             # 原有的
    enabled: true
    replicas: 3
  gpu:                 # 新增 GPU 節點池
    enabled: true
    replicas: 2
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.05-rockylinux9
      resources:
        limits:
          nvidia.com/gpu: 1
    partition:
      enabled: true
      name: gpu
```

執行更新：

```bash
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm
```

### 擴縮容

**方法 1：修改 values.yaml**

```yaml
nodesets:
  compute:
    replicas: 5    # 從 3 改成 5
```

```bash
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm
```

**方法 2：使用 --set 臨時覆蓋**

```bash
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml \
  --set nodesets.compute.replicas=5 \
  -n slurm
```

> **注意**：用 `--set` 只是臨時覆蓋，下次 upgrade 如果沒帶這個參數會回到 values.yaml 的值。

### 更新 Image

```yaml
nodesets:
  compute:
    slurmd:
      image:
        tag: 25.05.1-rockylinux9   # 更新版本
```

```bash
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm
```

### 刪除 NodeSet

**方法 1：停用（推薦，保留設定）**

```yaml
nodesets:
  compute:
    enabled: false    # 停用
```

**方法 2：移除整個區塊**

直接從 values.yaml 刪除該 NodeSet 的設定。

執行更新：

```bash
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm
```

### 查看目前設定

```bash
# 查看 Helm release 的 values
helm get values slurm -n slurm

# 查看完整 values（包含預設值）
helm get values slurm -n slurm --all

# 查看實際產生的 manifest
helm get manifest slurm -n slurm | grep -A 50 "kind: NodeSet"
```

---

## 手動 CR 轉換為 Helm 管理

如果你已經手動建立了 NodeSet CR，可以透過以下步驟讓 Helm 接管管理。

### 方法 1：零停機接管（推薦）

```bash
# 1. 備份現有 CR
kubectl get nodeset compute -n slurm -o yaml > nodeset-backup.yaml

# 2. 加上 Helm 管理標籤
kubectl label nodeset compute -n slurm \
  app.kubernetes.io/managed-by=Helm

kubectl annotate nodeset compute -n slurm \
  meta.helm.sh/release-name=slurm \
  meta.helm.sh/release-namespace=slurm

# 3. 準備對應的 values.yaml
# ⚠️ 設定必須和現有 CR 完全一致！
cat > values.yaml <<EOF
nodesets:
  compute:
    enabled: true
    replicas: 3  # 與現有 CR 一致
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.05-rockylinux9
    partition:
      enabled: true
EOF

# 4. 執行 helm upgrade
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm --install
```

### 方法 2：刪除重建（有停機時間）

```bash
# 1. 備份現有 CR
kubectl get nodeset compute -n slurm -o yaml > nodeset-backup.yaml

# 2. 等待所有作業完成
squeue  # 確認沒有執行中的作業

# 3. 刪除現有 CR
kubectl delete nodeset compute -n slurm

# 4. 用 Helm 重新部署
helm upgrade --install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm
```

### 驗證轉換成功

```bash
# 確認 Helm 已接管
kubectl get nodeset compute -n slurm \
  -o jsonpath='{.metadata.labels.app\.kubernetes\.io/managed-by}'
# 應該輸出：Helm

# 確認在 Helm release 中
helm get manifest slurm -n slurm | grep "kind: NodeSet" -A 5
```

---

## 多環境管理

使用多個 values 檔案管理不同環境：

```
values/
├── base.yaml        # 共用設定
├── dev.yaml         # 開發環境
├── staging.yaml     # 測試環境
└── prod.yaml        # 生產環境
```

**範例：base.yaml**

```yaml
# 共用設定
controller:
  enabled: true
restapi:
  enabled: true
nodesets:
  compute:
    enabled: true
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.05-rockylinux9
```

**範例：prod.yaml**

```yaml
# 生產環境覆蓋
nodesets:
  compute:
    replicas: 10
    slurmd:
      resources:
        requests:
          cpu: "4"
          memory: "8Gi"
```

**部署時合併檔案：**

```bash
# 後面的檔案會覆蓋前面的相同設定
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values/base.yaml \
  -f values/prod.yaml \
  -n slurm
```

---

## 版本控制與回滾

### 查看部署歷史

```bash
helm history slurm -n slurm
```

輸出範例：

```
REVISION  UPDATED                   STATUS      DESCRIPTION
1         Mon Dec 23 10:00:00 2025  superseded  Install complete
2         Mon Dec 23 11:00:00 2025  superseded  Upgrade complete
3         Mon Dec 23 12:00:00 2025  deployed    Upgrade complete
```

### 回滾操作

```bash
# 回滾到上一版
helm rollback slurm -n slurm

# 回滾到指定版本
helm rollback slurm 2 -n slurm
```

### 升級前預覽差異

安裝 helm-diff plugin：

```bash
helm plugin install https://github.com/databus23/helm-diff
```

使用：

```bash
helm diff upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm
```

---

## 最佳實踐

| 實踐 | 說明 |
|------|------|
| values.yaml 納入 Git | 版本控制所有設定變更 |
| 避免用 `--set` 做永久變更 | 容易遺忘，造成設定不一致 |
| 升級前先 diff | 使用 `helm diff upgrade` 預覽變更 |
| 保留 backup | 升級前 `helm get values slurm -n slurm > backup.yaml` |
| 生產環境使用固定版本 | 避免使用 `latest` 標籤 |
| 分離環境設定 | 使用多檔案結構管理不同環境 |

---

## 常見問題

### Helm 說資源已存在

確認已加上正確的 label 和 annotation：

```bash
kubectl label nodeset <name> -n slurm app.kubernetes.io/managed-by=Helm
kubectl annotate nodeset <name> -n slurm \
  meta.helm.sh/release-name=slurm \
  meta.helm.sh/release-namespace=slurm
```

### 升級後設定被覆蓋

values.yaml 設定要與現有 CR 完全對齊。建議升級前先備份：

```bash
kubectl get nodeset <name> -n slurm -o yaml > backup.yaml
```

### 多個 NodeSet 要轉換

每個都要加標籤，values.yaml 要包含全部 NodeSet。

---

> 更多問題請參考 [FAQ](./slurm-faq.md) 或到 [GitHub Issues](https://github.com/SlinkyProject/slurm-operator/issues) 提問。
