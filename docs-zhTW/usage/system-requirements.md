# 系統需求指南 (System Requirements Guide)

## TL;DR

本指南提供在 Kubernetes 上執行 Slurm Operator 和 Slurm 叢集的硬體建議。涵蓋 Kubernetes 叢集需求（多節點、儲存類別）、Operator 需求（支援 amd64/arm64）、以及 Slurm 各元件的硬體考量（slurmctld 需要快速單核、slurmd 需要快速儲存等）。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [系統需求指南](#系統需求指南-system-requirements-guide)
  - [目錄](#目錄)
  - [概述](#概述)
  - [Kubernetes](#kubernetes)
    - [硬體](#硬體)
    - [儲存類別](#儲存類別)
  - [Operator](#operator)
    - [作業系統和架構](#作業系統和架構)
    - [硬體](#硬體-1)
  - [Slurm](#slurm)
    - [作業系統和架構](#作業系統和架構-1)
    - [硬體](#硬體-2)

<!-- mdformat-toc end -->

## 概述

本指南提供在 Kubernetes 上執行 Slurm Operator 和 Slurm 叢集的建議硬體指引。

## Kubernetes

### 硬體

一般而言，您的 Kubernetes 叢集應該由多個節點組成，其中至少有一個控制平面 (Control-plane)。

> [!NOTE]
> 我們無法為您的工作負載提供最低系統需求。

### 儲存類別

建議至少有一個[儲存類別 (Storage Class)][storageclass]和一個[預設儲存類別][default-storageclass]。安裝在您的 Kubernetes 叢集上的 Slurm 和其他服務可能會使用[持久卷宣告 (Persistent Volume Claim, PVC)][persistent-volume]和[持久卷 (Persistent Volume, PV)][persistent-volume]。

## Operator

Operator 元件包含：

- `slurm-operator`
- `slurm-operator-webhook`

### 作業系統和架構

Slinky 容器映像建立在 [distroless] 映像之上。

支援以下機器架構：

- amd64 (x86_64)
- arm64 (aarch64)

請檢查 [OCI artifacts][oci-slurm-operator] 以獲取具體詳細資訊。

### 硬體

Operator 因處理網路請求和回應而受益於更多核心和記憶體。核心和記憶體的數量取決於配置了多少工作執行緒以及 Operator 有多繁忙。

> [!NOTE]
> 我們無法為您的工作負載提供最低系統需求。雖然 Operator 可以用 1 個核心和 1GB 記憶體執行，但正式環境可能會發現這些資源不足。

## Slurm

Slurm 元件包含：

- `slurmctld`
- `slurmd`
- `slurmdbd`
- `slurmrestd`
- `sackd`

如需更多資訊，請參閱 [Slurm 文件][slurm-docs]。

### 作業系統和架構

Slurm 對 Linux 發行版有廣泛的支援，對 FreeBSD 和 NetBSD 有有限的支援。

> Slurm 已在大多數流行的 Linux 發行版上使用 arm64 (aarch64)、ppc64 和 x86_64 架構進行了徹底測試。某些功能僅限於較新的版本和較新的 Linux 核心版本。

請參閱 Slurm [文件][platforms]以獲取詳細資訊。

為 Slinky 建置的 Slurm [容器][container]映像僅涵蓋 Slurm 作業系統和架構支援的子集。

支援以下機器架構：

- amd64 (x86_64)
- arm64 (aarch64)

請檢查 [OCI artifacts][oci-containers] 以獲取具體詳細資訊。

### 硬體

所有 Slurm 守護程式 (Daemons) 因處理網路請求和回應而受益於更多核心和記憶體。核心和記憶體的數量取決於您的叢集有多繁忙。由於內部資料鎖定，核心數量和單核心效能之間需要平衡。有些守護程式比其他守護程式更敏感。有些 Slurm 守護程式在檔案系統的特定區域受益於快速儲存。所有守護程式都偏好沒有嘈雜的鄰居，也就是說——機器上的其他程序會造成核心和記憶體的競爭。

以下是值得注意的事項：

- slurmctld
  - 排程大大受益於單核心效能
  - [StateSaveLocation] 大大受益於快速儲存
- slurmd
  - [SlurmdSpoolDir] 大大受益於快速儲存
  - 根據使用者的作業，應考慮硬體
- slurmdbd
  - 受益於與資料庫機器共置，透過 socket 而非網路通訊。
- slurmrestd
  - 受益於與 slurmctld 和/或 slurmdbd 機器共置，透過 socket 而非網路通訊。
- sackd
  - 在系統需求方面視同 [munged]
- 資料庫
  - 受益於快速儲存

請參閱[現場筆記][slug22-field-notes]第 17 頁，以獲取系統需求的筆記。

> [!NOTE]
> 我們無法為您的工作負載提供最低系統需求。雖然 Slurm 守護程式可以用 1 個核心和 1GB 記憶體執行，但正式環境可能會發現這些資源不足。

<!-- Links -->

[container]: https://github.com/SlinkyProject/containers
[default-storageclass]: https://kubernetes.io/docs/concepts/storage/storage-classes/#default-storageclass
[distroless]: https://github.com/GoogleContainerTools/distroless
[munged]: https://dun.github.io/munge/
[oci-containers]: https://github.com/orgs/SlinkyProject/packages?repo_name=containers
[oci-slurm-operator]: https://github.com/orgs/SlinkyProject/packages?repo_name=slurm-operator
[persistent-volume]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
[platforms]: https://slurm.schedmd.com/platforms.html#os
[slug22-field-notes]: https://slurm.schedmd.com/SLUG22/Field_Notes_6.pdf
[slurm-docs]: ../concepts/slurm.md
[slurmdspooldir]: https://slurm.schedmd.com/slurm.conf.html#OPT_SlurmdSpoolDir
[statesavelocation]: https://slurm.schedmd.com/slurm.conf.html#OPT_StateSaveLocation
[storageclass]: https://kubernetes.io/docs/concepts/storage/storage-classes/

---

## Explanation

### 各元件的資源需求

| 元件 | CPU 需求 | 記憶體需求 | 儲存需求 |
|-----|---------|----------|---------|
| **slurm-operator** | 低-中 | 低 | 無特別需求 |
| **slurmctld** | 高單核效能 | 中-高 | 高速 I/O（StateSaveLocation） |
| **slurmd** | 依工作負載 | 依工作負載 | 高速 I/O（SlurmdSpoolDir） |
| **slurmdbd** | 中 | 中 | 低延遲網路到資料庫 |
| **slurmrestd** | 中 | 低-中 | 低延遲網路 |
| **Database** | 中-高 | 高 | 高速 I/O |

### 為何單核效能重要？

Slurm 排程器（slurmctld）內部有許多資料鎖定機制。這意味著：
- 增加更多核心不一定能線性提升效能
- 高時脈頻率的 CPU 對排程效能有顯著影響
- 對於大型叢集，CPU 選擇可能比核心數更重要

### 儲存考量

| 路徑 | 建議 |
|-----|------|
| StateSaveLocation | NVMe SSD 或同等快速儲存 |
| SlurmdSpoolDir | 本地 SSD |
| 資料庫儲存 | 低延遲高 IOPS 儲存 |

---

## Practical Example

### 檢查 Kubernetes 儲存類別

```bash
# 列出所有儲存類別
kubectl get storageclasses

# 查看預設儲存類別
kubectl get sc -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}'

# 查看儲存類別詳細資訊
kubectl describe storageclass <storage-class-name>
```

### 檢查節點資源

```bash
# 查看所有節點的資源
kubectl get nodes -o wide

# 查看特定節點的詳細資源
kubectl describe node <node-name>

# 查看節點的可分配資源
kubectl get nodes -o custom-columns=\
"NAME:.metadata.name,\
CPU:.status.allocatable.cpu,\
MEMORY:.status.allocatable.memory"
```

### 監控 Slurm 資源使用

```bash
# 查看 Slurm Pod 的資源使用
kubectl top pods -n slurm

# 查看 Operator Pod 的資源使用
kubectl top pods -n slinky

# 設定資源限制（在 values.yaml 中）
cat > values-resources.yaml <<EOF
controller:
  resources:
    requests:
      cpu: "2"
      memory: "4Gi"
    limits:
      cpu: "4"
      memory: "8Gi"

nodesets:
  slinky:
    slurmd:
      resources:
        requests:
          cpu: "1"
          memory: "2Gi"
        limits:
          cpu: "2"
          memory: "4Gi"
EOF
```

### 配置持久卷

```bash
# 查看持久卷宣告
kubectl get pvc -n slurm

# 查看持久卷
kubectl get pv

# 建立使用特定儲存類別的 Slurm 叢集
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --set "controller.persistence.storageClassName=fast-storage" \
  --namespace=slurm --create-namespace
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| 未設定預設儲存類別 | 建立或指定一個儲存類別 |
| 控制器效能不佳 | 增加 CPU 資源，優先考慮單核效能 |
| 資料庫延遲高 | 考慮將 slurmdbd 與資料庫共置 |
| 節點資源耗盡 | 設定適當的資源 requests 和 limits |

### 規模建議

| 叢集規模 | slurmctld 建議 | 資料庫建議 |
|---------|---------------|-----------|
| < 100 節點 | 2 核心, 4GB RAM | 2 核心, 8GB RAM |
| 100-500 節點 | 4 核心, 8GB RAM | 4 核心, 16GB RAM |
| 500-1000 節點 | 8 核心, 16GB RAM | 8 核心, 32GB RAM |
| > 1000 節點 | 16+ 核心, 32GB+ RAM | 考慮專用叢集 |

### 小技巧

1. **監控資源使用**：定期檢查 `kubectl top` 輸出
2. **使用 PodDisruptionBudget**：確保關鍵元件的高可用性
3. **考慮節點親和性**：將 slurmctld 排程到效能較好的節點
4. **規劃儲存容量**：StateSaveLocation 會隨叢集規模增長

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `kubectl get sc` | 列出儲存類別 |
| `kubectl get pv` | 列出持久卷 |
| `kubectl get pvc -n slurm` | 列出 PVC |
| `kubectl top pods -n slurm` | 查看 Pod 資源使用 |
| `kubectl describe node <name>` | 查看節點詳細資訊 |
| `kubectl get nodes -o wide` | 查看所有節點 |
