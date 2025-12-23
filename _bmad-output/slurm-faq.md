# Slurm Operator 常見問題 (FAQ)

> 最後更新：2025-12-23
> 相關文件：[使用指南](./slurm-usage-guide.md) | [NodeSet API 參考](./nodeset-api-reference.md) | [Helm NodeSet 管理指南](./helm-nodeset-guide.md)

---

## 目錄

- [基本概念](#基本概念)
- [NodeSet 與 RestApi](#nodeset-與-restapi)
- [Helm 管理](#helm-管理)
- [作業提交](#作業提交)
- [LDAP 認證](#ldap-認證)
- [故障排除](#故障排除)

---

## 基本概念

### Q: Slurm Operator 和直接安裝 Slurm 有什麼不同？

| 傳統安裝 | Slurm Operator |
|---------|----------------|
| 手動設定每台節點 | 自動化部署 |
| 固定節點數量 | 可動態擴縮容 |
| 需要自行管理升級 | Kubernetes 原生滾動更新 |
| 獨立於 K8s | 與 K8s 生態系整合 |

### Q: 我需要先學會 Slurm 還是 Kubernetes？

建議**兩者都有基礎認識**：

- **Kubernetes**：了解 Pod、Service、Deployment、kubectl 基本操作
- **Slurm**：了解 sinfo、squeue、sbatch 等基本指令

Slurm Operator 讓你用 Kubernetes 的方式管理 Slurm，但最終使用者還是用 Slurm 指令提交作業。

### Q: 最小的 Slurm 叢集需要什麼？

至少需要：

```yaml
# 必要元件
controller:     # 1 個（叢集大腦）
nodeset:        # 至少 1 個計算節點

# 選用元件
loginset:       # 如果需要 SSH 登入
restapi:        # 如果需要程式化存取
accounting:     # 如果需要記帳功能
```

---

## NodeSet 與 RestApi

### Q: 如何建立 NodeSet worker Pod？

Worker Pod 是透過建立 **NodeSet CR** 自動產生的，你不需要手動建立 Pod。

**方式 1：透過 Helm Chart（推薦）**

```yaml
# values.yaml
nodesets:
  compute:                   # NodeSet 名稱
    enabled: true
    replicas: 3              # 建立 3 個 worker Pod
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

```bash
helm upgrade --install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm
```

**方式 2：直接建立 NodeSet CR**

```yaml
# nodeset.yaml
apiVersion: slinky.slurm.net/v1beta1
kind: NodeSet
metadata:
  name: compute
  namespace: slurm
spec:
  controllerRef:
    name: slinky
    namespace: slurm
  replicas: 3
  slurmd:
    image: ghcr.io/slinkyproject/slurmd:25.05-rockylinux9
  partition:
    enabled: true
```

```bash
kubectl apply -f nodeset.yaml
```

**驗證**：

```bash
kubectl get nodeset -n slurm
kubectl get pods -n slurm -l slinky.slurm.net/component=slurmd
```

### Q: 如何擴縮容 NodeSet？

```bash
# 增加到 5 個 worker
kubectl scale nodeset/compute --replicas=5 -n slurm

# 或修改 CR
kubectl patch nodeset compute -n slurm --type=merge -p '{"spec":{"replicas":5}}'
```

### Q: NodeSet 可以透過 REST API 建立嗎？

**不行**。這是最常見的誤解。

- **NodeSet CR** 建立 Kubernetes Pod（基礎設施）
- **REST API** 只能操作已存在的 Slurm 節點（邏輯控制）

要增加節點，必須修改 NodeSet 的 `replicas`（參考上面的擴縮容方式）。

### Q: 為什麼縮容時節點不會立刻被刪除？

因為 **工作負載保護**。NodeSet Controller 會：

1. 先 Drain 節點（透過 REST API）
2. 等待該節點上的作業完成
3. 確認節點狀態為 DRAINED 後才刪除 Pod

這確保正在執行的作業不會被中斷。

### Q: 我可以手動 Drain 某個特定節點嗎？

可以，有兩種方式：

```bash
# 方式 1：透過 kubectl annotate
kubectl annotate pod slurm-worker-slinky-2 \
  slinky.slurm.net/pod-cordon=true -n slurm

# 方式 2：透過 REST API
curl -X POST "$SLURMRESTD/slurm/v0.0.44/node/slurm-worker-slinky-2" \
     -H "X-SLURM-USER-TOKEN: $TOKEN" \
     -d '{"state": ["DRAIN"], "reason": "Manual maintenance"}'
```

### Q: NodeSet 的 status 欄位有哪些 Slurm 狀態？

```yaml
status:
  replicas: 5           # 總 Pod 數
  readyReplicas: 5      # 就緒的 Pod
  slurmIdle: 3          # Slurm Idle 節點
  slurmAllocated: 2     # Slurm Allocated 節點（正在執行作業）
  slurmDown: 0          # Slurm Down 節點
  slurmDrain: 0         # Slurm Drain 節點
```

---

## Helm 管理

### Q: 如何確認 NodeSet 是由 Helm 還是手動建立的？

檢查資源的 labels 和 annotations：

```bash
kubectl get nodeset <name> -n slurm -o yaml | grep -A5 "labels:"
```

**Helm 建立的資源會有：**

```yaml
labels:
  app.kubernetes.io/managed-by: Helm
annotations:
  meta.helm.sh/release-name: slurm
  meta.helm.sh/release-namespace: slurm
```

如果沒有這些標籤，就是手動建立的 CR。

### Q: 使用 Helm 管理 NodeSet 有什麼好處？

| 優點 | 說明 |
|------|------|
| 統一管理 | 所有元件在同一個 values.yaml |
| 版本控制 | `helm history` 查看歷史，`helm rollback` 回滾 |
| 升級簡單 | `helm upgrade` 一個指令升級整個叢集 |
| 設定驗證 | Chart 內建 schema 減少設定錯誤 |
| 相依性處理 | 自動處理元件間的相依關係 |

### Q: 直接建立 CR 有什麼壞處？

| 缺點 | 說明 |
|------|------|
| 設定分散 | 每個 CR 一個 YAML，難以追蹤 |
| 無法整體回滾 | 沒有簡單方法回到之前的狀態 |
| Helm 衝突 | 混用時 `helm upgrade` 可能覆蓋手動變更 |
| 升級複雜 | 需要手動逐一更新每個 CR |

### Q: 如何將手動建立的 CR 轉成 Helm 管理？

**零停機方式**（推薦）：

```bash
# 1. 備份現有 CR
kubectl get nodeset compute -n slurm -o yaml > nodeset-backup.yaml

# 2. 加上 Helm 管理標籤
kubectl label nodeset compute -n slurm \
  app.kubernetes.io/managed-by=Helm

kubectl annotate nodeset compute -n slurm \
  meta.helm.sh/release-name=slurm \
  meta.helm.sh/release-namespace=slurm

# 3. 準備對應的 values.yaml（設定要和現有 CR 一致）
# 4. 執行 helm upgrade
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values.yaml -n slurm --install
```

**注意**：values.yaml 的設定必須和現有 CR 完全一致，否則會被覆蓋。

詳細操作請參考 [Helm NodeSet 管理指南](./helm-nodeset-guide.md)。

---

## 作業提交

### Q: 我該用 LoginSet 還是 REST API 提交作業？

| 情境 | 建議方式 |
|------|---------|
| 互動式開發、除錯 | LoginSet (SSH) |
| CI/CD 管線 | REST API |
| 批次自動化 | REST API |
| 傳統 HPC 使用者 | LoginSet (SSH) |

### Q: 作業輸出檔案存在哪裡？

作業輸出檔案存在**執行該作業的 Pod** 中。要取回：

```bash
# 查看作業在哪個節點執行
squeue -o "%i %N"

# 從該 Pod 複製檔案
kubectl cp slurm/slurm-worker-slinky-0:/root/output.txt ./output.txt
```

**建議**：使用共享儲存（如 NFS PVC）掛載到所有節點，避免檔案散落各處。

### Q: 如何設定作業的資源限制？

在 sbatch 腳本中指定：

```bash
#!/bin/bash
#SBATCH --nodes=2          # 節點數
#SBATCH --ntasks=4         # 任務數
#SBATCH --cpus-per-task=2  # 每個任務的 CPU
#SBATCH --mem=4G           # 記憶體
#SBATCH --time=01:00:00    # 時間限制
#SBATCH --gres=gpu:1       # GPU 數量（如果有）
```

---

## LDAP 認證

### Q: SSSD 設定套用到哪些元件？

**LoginSet** 和 **NodeSet** 都會使用 SSSD 設定，確保：

- 使用者可以 SSH 登入 Login 節點
- 作業以正確的 UID/GID 在 Worker 節點執行

### Q: 如何確認 LDAP 使用者可以登入？

```bash
# 進入 Login Pod
kubectl exec -it -n slurm deploy/slurm-login-slinky -- bash

# 測試使用者查詢
getent passwd your_ldap_user

# 測試登入
su - your_ldap_user
whoami
id
```

### Q: LDAP 連線失敗怎麼辦？

常見問題檢查清單：

1. **網路連通性**：Pod 能否連到 LDAP Server？
   ```bash
   kubectl exec -n slurm deploy/slurm-login-slinky -- \
     nc -zv ldap.mycompany.com 389
   ```

2. **Bind DN 權限**：確認 bind 帳號有讀取權限

3. **Search Base 正確性**：確認 DN 路徑與 LDAP 結構相符

4. **查看 SSSD 日誌**：
   ```bash
   kubectl exec -n slurm deploy/slurm-login-slinky -- \
     journalctl -u sssd --no-pager | tail -50
   ```

### Q: 可以同時支援 LDAP 和本地帳號嗎？

可以。SSSD 的 NSS 設定會先查本地 `/etc/passwd`，再查 LDAP。本地帳號（如 root、slurm）會被過濾掉不走 LDAP。

---

## 故障排除

### Q: 節點一直顯示 DOWN 狀態？

可能原因：

1. **slurmd 沒有啟動**：檢查 Pod logs
   ```bash
   kubectl logs -n slurm slurm-worker-slinky-0 -c slurmd
   ```

2. **網路問題**：slurmd 無法連到 slurmctld
   ```bash
   kubectl exec -n slurm slurm-worker-slinky-0 -- \
     nc -zv slurm-controller-slinky 6817
   ```

3. **Munge 認證失敗**：確認 slurm key 一致

### Q: 作業一直在 PENDING 狀態？

使用 `squeue` 查看原因：

```bash
squeue -o "%i %j %T %r"
```

常見原因：

| Reason | 說明 |
|--------|------|
| Priority | 等待排隊 |
| Resources | 資源不足（增加 NodeSet replicas） |
| PartitionDown | 分區未啟用 |
| QOSMaxJobsPerUser | 超過使用者作業上限 |

### Q: 如何查看 Slurm 配置？

```bash
# 在 Login Pod 中
scontrol show config | grep -i <keyword>

# 或直接看 ConfigMap
kubectl get configmap -n slurm slurm-config-slinky -o yaml
```

### Q: 如何重啟 slurmctld？

```bash
# 刪除 Pod，StatefulSet 會自動重建
kubectl delete pod -n slurm slurm-controller-slinky-0

# 或觸發滾動更新（修改任意 annotation）
kubectl annotate controller slinky -n slurm \
  restart-trigger="$(date +%s)" --overwrite
```

---

> 找不到你的問題？歡迎到 [GitHub Issues](https://github.com/SlinkyProject/slurm-operator/issues) 提問。
