# 混合式部署 (Hybrid)

## TL;DR

混合式叢集結合了多種基礎設施類型——裸機、虛擬機器、容器和雲端。透過 slurm-operator，可以將部分 Slurm 元件部署在 Kubernetes 中，其他元件部署在外部。本指南說明網路配置和 Slurm 配置選項。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [混合式部署](#混合式部署-hybrid)
  - [目錄](#目錄)
  - [概述](#概述)
  - [Slurm](#slurm)
  - [網路](#網路)
    - [主機網路](#主機網路)
    - [網路對等互連](#網路對等互連)
  - [Slurm 配置](#slurm-配置)
    - [外部 Slurmdbd](#外部-slurmdbd)
    - [外部 Slurmctld](#外部-slurmctld)
    - [外部 Slurmd](#外部-slurmd)
    - [外部登入](#外部登入)
    - [外部 Slurmrestd](#外部-slurmrestd)

<!-- mdformat-toc end -->

## 概述

混合式叢集 (Hybrid Cluster) 是結合多種基礎設施協調類型的叢集——裸機 (Bare-metal)、虛擬機器 (VMs)、容器（例如 Kubernetes、Docker）和雲端基礎設施（例如 AWS、GCP、Azure、OpenStack）。

透過 slurm-operator 及其 CRDs，可以表達混合式 Slurm 叢集，使部分 Slurm 叢集元件存在於 Kubernetes 中，而其他元件存在於外部。

## Slurm

Slinky 目前要求 Slurm 使用 [configless]、[auth/slurm]、[auth/jwt] 和 [use_client_ids]。這決定了 Slurm 叢集可以如何定義。

將 `slurm.key` 儲存為 Kubernetes 中的 Secret。

```sh
kubectl create secret generic external-auth-slurm \
  --namespace=slurm --from-file="slurm.key=/etc/slurm/slurm.key"
```

將 `jwt_hs256.key` 儲存為 Kubernetes 中的 Secret。

```sh
kubectl create secret generic external-auth-jwths256 \
  --namespace=slurm --from-file="jwt_hs256.key=/etc/slurm/jwt_hs256.key"
```

## 網路

在混合式配置的情境中，有兩種流量路由需要考慮：內部-內部通訊；以及外部-內部通訊。Kubernetes 內部-內部通訊通常是 Pod 到 Pod 的流量，這是一個具有 DNS 的扁平網路。外部-內部通訊通常涉及透過 NAT 將外部流量代理到 Pod。

Slurm 期望一個完全連接的網路，所有 Slurm 守護程式和用戶端之間可以雙向通訊。這意味著 NAT 類型的網路通常會阻礙通訊。

因此，網路配置需要設定為允許 Slurm 元件直接透過網路通訊。有兩種設定可供選擇，各有其優缺點。

### 主機網路

這種方法是透過讓 Slurm Pod 直接使用 Kubernetes 節點主機網路來避開 Kubernetes NAT。雖然這是最簡單的方法，但它確實有[安全性][pod-security-standards]和 Slurm 配置方面的考量。

每個 Slurm Pod 會配置如下。

```yaml
hostNetwork: true
dNSPolicy: ClusterFirstWithHostNet
```

> [!NOTE]
> 由於 Slurm 配置與 Kubernetes 的競爭條件，Controller 和 Accounting 不支援此選項。

> [!WARNING]
> 一次只能在一個 Kubernetes 節點上運行一個啟用主機網路的 Pod。它將繼承節點的主機名稱並在主機的命名空間中運行，使 Pod 可以存取整個網路和所有埠。

### 網路對等互連

這種方法配置網路對等互連 (Network Peering)，使內部和外部服務可以直接通訊。雖然這是最複雜的方法，但它不會降低安全性，且只需要最少的 Slurm 配置。

這通常涉及配置進階 [CNI]，如 [Calico]，並使用網路[對等互連][bgp]來實現跨 Kubernetes 邊界的雙向通訊。

一般而言，Slurm Helm chart 不需要特殊配置。

## Slurm 配置

Slinky 目前要求 Slurm 使用 [configless]、[auth/slurm]、[auth/jwt] 和 [use_client_ids]。這決定了 Slurm 叢集可以如何定義。

將 `slurm.key` 複製為 Kubernetes 中的 Secret。

```sh
kubectl create secret generic external-auth-slurm \
  --namespace=slurm --from-file="slurm.key=/etc/slurm/slurm.key"
```

將 `jwt_hs256.key` 複製為 Kubernetes 中的 Secret。

```sh
kubectl create secret generic external-auth-jwths256 \
  --namespace=slurm --from-file="jwt_hs256.key=/etc/slurm/jwt_hs256.key"
```

配置 Slurm Helm chart 時，將 Slurm 金鑰和 JWT 金鑰設定為複製到 Kubernetes 中的 Secret，否則 Slurm 元件將無法與叢集其餘部分進行認證。

```yaml
slurmKeyRef:
  name: external-auth-slurm
  key: slurm.key
jwtHs256KeyRef:
  name: external-auth-jwths256
  key: jwt_hs256.key
```

> [!WARNING]
> 將容器化 slurmd 與非容器化 slurmd 混合使用可能會有問題，因為 Slurm 假設所有節點具有同質性配置。特別是，`cgroup.conf` 中的 `IgnoreSystemd=yes` 可能無法在兩種類型的節點上都正常運作。

### 外部 Slurmdbd

當 slurmdbd 在 Kubernetes 外部時，Slurm Helm chart 需要配置 accounting CR，使其知道如何與之通訊。

```yaml
accounting:
  external: true
  externalConfig:
    host: $SLURMDBD_HOST
    port: $SLURMDBD_PORT # 預設：6819
```

### 外部 Slurmctld

當 slurmctld 在 Kubernetes 外部時，Slurm Helm chart 需要配置 controller CR，使其知道如何與之通訊。

```yaml
controller:
  external: true
  externalConfig:
    host: $SLURMCTLD_HOST
    port: $SLURMCTLD_PORT # 預設：6817
```

### 外部 Slurmd

當 slurmd 在 Kubernetes 外部時，Slurm Helm chart 只提供額外的工作節點。外部 slurmd 必須使用以下選項啟動。

```sh
slurmd --conf-server "${SLURMCTLD_HOST}:${SLURMCTLD_PORT}"
```

### 外部登入

當登入主機在 Kubernetes 外部時，Slurm Helm chart 只提供額外的登入 Pod。外部 sackd 必須使用以下選項啟動。

```sh
sackd --conf-server "${SLURMCTLD_HOST}:${SLURMCTLD_PORT}"
```

### 外部 Slurmrestd

Slurm Helm chart 始終提供一個 slurmrestd Pod，使 slurm-operator 可以使用它來正確地對 Kubernetes 中的 Slurm 資源採取行動。

您仍然可以有一個可在 Kubernetes 外部存取的 slurmrestd 來處理 Kubernetes 外部的請求。

<!-- Links -->

[auth/jwt]: https://slurm.schedmd.com/authentication.html#jwt
[auth/slurm]: https://slurm.schedmd.com/authentication.html#slurm
[bgp]: https://docs.tigera.io/calico/latest/networking/configuring/bgp
[calico]: https://docs.tigera.io/calico/latest/about/
[cni]: https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/
[configless]: https://slurm.schedmd.com/configless_slurm.html
[pod-security-standards]: https://kubernetes.io/docs/concepts/security/pod-security-standards/
[use_client_ids]: https://slurm.schedmd.com/slurm.conf.html#OPT_use_client_ids

---

## Explanation

### 什麼是混合式 Slurm 叢集？

混合式叢集允許組織：
- 利用現有的裸機 HPC 資源
- 在需要時擴展到雲端
- 逐步將工作負載遷移到容器

### 常見混合式部署模式

| 模式 | 描述 |
|-----|------|
| **控制平面在 K8s** | slurmctld、slurmdbd 在 K8s，slurmd 在裸機 |
| **工作節點在 K8s** | slurmctld 在裸機，slurmd 在 K8s（彈性擴展） |
| **完全混合** | 部分元件在 K8s，部分在裸機，根據需要分配 |

### 認證需求

Slinky 要求特定的認證配置：
- **auth/slurm**：Slurm 內建認證
- **auth/jwt**：JSON Web Token 認證
- **configless**：無需在每個節點上維護設定檔

---

## Practical Example

### 將現有 Slurm 叢集與 Kubernetes 整合

```bash
# 1. 在現有 Slurm 控制器上取得金鑰
# 複製 slurm.key 和 jwt_hs256.key 到本地

# 2. 在 Kubernetes 中建立 Secrets
kubectl create namespace slurm

kubectl create secret generic external-auth-slurm \
  --namespace=slurm \
  --from-file="slurm.key=/path/to/slurm.key"

kubectl create secret generic external-auth-jwths256 \
  --namespace=slurm \
  --from-file="jwt_hs256.key=/path/to/jwt_hs256.key"

# 3. 驗證 Secrets 已建立
kubectl get secrets -n slurm
```

### 配置外部控制器的 values.yaml

```yaml
# values-hybrid.yaml
# 連接到外部 Slurm 控制器

# 使用外部的認證金鑰
slurmKeyRef:
  name: external-auth-slurm
  key: slurm.key
jwtHs256KeyRef:
  name: external-auth-jwths256
  key: jwt_hs256.key

# 控制器在外部
controller:
  external: true
  externalConfig:
    host: slurm-controller.example.com  # 外部 slurmctld 主機名稱
    port: 6817                           # slurmctld 埠

# 帳務在外部
accounting:
  external: true
  externalConfig:
    host: slurm-accounting.example.com  # 外部 slurmdbd 主機名稱
    port: 6819                           # slurmdbd 埠

# 在 K8s 中部署額外的工作節點
nodesets:
  slinky:
    enabled: true
    replicas: 5
```

### 安裝混合式配置

```bash
# 使用自訂 values 安裝
helm install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values-hybrid.yaml \
  --namespace=slurm --create-namespace

# 驗證 Pod 狀態
kubectl get pods -n slurm

# 檢查 slurmd 是否已加入外部控制器
kubectl exec -n slurm deployment/slurm-login-slinky -- sinfo
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| 認證失敗 | 確保金鑰檔案與外部 Slurm 叢集一致 |
| 網路連線失敗 | 檢查防火牆規則，確保必要的埠開放 |
| 節點無法註冊 | 驗證 DNS 解析和網路路由正確 |
| cgroup 配置不相容 | 檢查容器化和裸機節點的 cgroup 設定 |

### 必要的網路埠

| 埠 | 服務 | 說明 |
|---|------|------|
| 6817 | slurmctld | Slurm 控制器 |
| 6818 | slurmd | Slurm 守護程式 |
| 6819 | slurmdbd | Slurm 帳務 |
| 6820 | slurmrestd | REST API |

### 小技巧

1. **測試網路連通性**：先確保所有元件可以互相通訊
2. **統一認證金鑰**：確保所有節點使用相同的 slurm.key 和 jwt 金鑰
3. **考慮延遲**：跨雲端的混合部署可能有較高延遲
4. **監控日誌**：同時監控 K8s 和外部 Slurm 的日誌

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `kubectl create secret generic <name> --from-file=<file>` | 建立 Secret |
| `slurmd --conf-server "${HOST}:${PORT}"` | 啟動外部 slurmd |
| `sackd --conf-server "${HOST}:${PORT}"` | 啟動外部 sackd |
| `kubectl get pods -n slurm` | 檢查 Slurm Pod 狀態 |
| `sinfo` | 檢查 Slurm 節點狀態 |
