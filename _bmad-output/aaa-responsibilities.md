# Authentication 與 Accounting 的職責邊界說明

> 最後更新：2025-12-30
> 適用版本：slurm-operator v1.x
> 目標讀者：Slurm 叢集管理員、平台工程師
> 相關文件：[CRD 參考](./crd-reference-guide.md) | [使用指南](./slurm-usage-guide.md) | [FAQ](./slurm-faq.md)

---

## 目錄

- [1. 文件目的](#1-文件目的)
- [2. 設計結論](#2-設計結論)
- [3. 職責邊界架構圖](#3-職責邊界架構圖)
- [4. slurm-operator 的責任範圍](#4-slurm-operator-的責任範圍)
  - [4.1 負責的事項](#41-負責的事項)
  - [4.2 不負責的事項](#42-不負責的事項)
- [5. 認證機制詳解](#5-認證機制詳解)
  - [5.1 auth/slurm vs auth/jwt](#51-authslurm-vs-authjwt)
  - [5.2 認證流程](#52-認證流程)
  - [5.3 SSSD 與 LDAP 整合](#53-sssd-與-ldap-整合)
- [6. 授權角色說明](#6-授權角色說明)
- [7. 為何 Slurm Accounting 必須人工 Bootstrap](#7-為何-slurm-accounting-必須人工-bootstrap)
  - [7.1 Slurm 原生授權模型](#71-slurm-原生授權模型)
  - [7.2 Chicken-Egg 問題](#72-chicken-egg-問題)
- [8. Bootstrap 操作指南](#8-bootstrap-操作指南)
  - [8.1 前置條件檢查](#81-前置條件檢查)
  - [8.2 初始化步驟](#82-初始化步驟)
  - [8.3 完整範例腳本](#83-完整範例腳本)
- [9. 使用者名稱一致性要求](#9-使用者名稱一致性要求)
- [10. Hybrid Cluster 認證考量](#10-hybrid-cluster-認證考量)
- [11. 常見問題排除](#11-常見問題排除)
- [12. 設計澄清](#12-設計澄清)
- [13. 總結](#13-總結)

---

## 1. 文件目的

本文件說明在 **Slinky（slurm-operator）** 架構下，**Authentication、Authorization 與 Accounting（AAA）相關行為的設計邊界與預期操作方式**。

此說明用於：

- 釐清哪些行為屬於 **Slurm 原生設計**
- 說明哪些行為 **不在 slurm-operator 的管理責任內**
- 解釋為何 **Slurm Accounting 必須進行人工 Bootstrap**
- 避免在導入時誤判為功能缺失或部署錯誤

---

## 2. 設計結論

> **slurm-operator 不負責建立或管理 Slurm Accounting 內的資料內容。**

具體而言：

| 層面 | slurm-operator 負責 | 管理者負責 |
|------|---------------------|-----------|
| **基礎設施** | 元件部署、生命週期、連線 | - |
| **認證配置** | auth/slurm、auth/jwt 金鑰管理 | LDAP/AD 整合策略 |
| **會計服務** | slurmdbd 部署、資料庫連線 | Account/User/QOS 建立 |
| **授權策略** | - | Admin/Operator/Coordinator 指派 |

**這是刻意的設計選擇，而非功能缺漏。**

---

## 3. 職責邊界架構圖

```mermaid
flowchart TB
    subgraph Operator["slurm-operator 責任範圍"]
        direction TB
        subgraph Infra["基礎設施層"]
            CRD["CRD 管理<br/>Controller, NodeSet, LoginSet,<br/>Accounting, RestApi, Token"]
            K8s["Kubernetes 資源<br/>Pod, Service, ConfigMap, Secret"]
            Sync["狀態同步<br/>節點狀態, Topology, Drain"]
        end

        subgraph Auth["認證基礎設施"]
            SlurmKey["auth/slurm 金鑰<br/>(SlurmKeyRef)"]
            JwtKey["auth/jwt 金鑰<br/>(JwtHs256KeyRef)"]
            Token["JWT 令牌自動生成<br/>(Token CRD)"]
            SSSD["SSSD 配置整合<br/>(SssdConfRef)"]
        end

        subgraph AcctInfra["會計基礎設施"]
            Slurmdbd["slurmdbd 服務部署"]
            DBConn["MySQL/MariaDB 連線"]
            ConfGen["slurmdbd.conf 生成"]
        end
    end

    subgraph Admin["管理者責任範圍"]
        direction TB
        subgraph AcctData["會計資料層"]
            Cluster["sacctmgr add cluster"]
            Account["sacctmgr add account"]
            User["sacctmgr add user"]
            Assoc["User-Account Association"]
        end

        subgraph Policy["授權策略層"]
            Roles["Admin/Operator/Coordinator"]
            QOS["QOS 定義"]
            Limits["資源限制<br/>MaxJobs, GrpTRES"]
            Fairshare["Fairshare 配置"]
        end
    end

    Operator -->|部署並連線| AcctInfra
    AcctInfra -.->|管道已建立<br/>內容由管理者填充| AcctData
    AcctData --> Policy

    style Operator fill:#e1f5fe,stroke:#01579b
    style Admin fill:#fff3e0,stroke:#e65100
```

### 層級對照表

| 層級 | 比喻 | slurm-operator | 管理者 |
|------|------|----------------|--------|
| 基礎設施 | 蓋好辦公大樓 | ✅ | - |
| 認證配置 | 安裝門禁系統 | ✅ | - |
| 會計服務 | 鋪設水電管線 | ✅ | - |
| 帳戶管理 | 發放門禁卡給員工 | - | ✅ |
| 授權策略 | 決定誰能進哪個樓層 | - | ✅ |
| 資源限制 | 安裝電錶、分配用電配額 | - | ✅ |

---

## 4. slurm-operator 的責任範圍

### 4.1 負責的事項

slurm-operator 採用 Kubernetes Operator Pattern，管理範圍限於 **基礎設施與控制面層級**：

#### 元件生命週期管理

| 元件 | CRD | 對應 Slurm 守護程序 |
|------|-----|-------------------|
| Controller | `Controller` | slurmctld |
| 計算節點 | `NodeSet` | slurmd |
| 登入節點 | `LoginSet` | sackd + sshd |
| 會計服務 | `Accounting` | slurmdbd |
| REST API | `RestApi` | slurmrestd |
| JWT 令牌 | `Token` | - |

> 詳細 CRD 欄位請參考 [CRD 參考指南](./crd-reference-guide.md)

#### 認證金鑰管理

```yaml
# Controller CRD 中的認證配置
spec:
  slurmKeyRef:           # auth/slurm 共享金鑰
    name: slurm-auth-slurm
    key: slurm.key
  jwtHs256KeyRef:        # auth/jwt 簽名金鑰
    name: slurm-auth-jwt
    key: jwt_hs256.key
```

#### 會計服務連線配置

```yaml
# Accounting CRD 中的資料庫配置
spec:
  storageConfig:
    host: mariadb.database.svc
    port: 3306
    database: slurm_acct_db
    username: slurm
    passwordKeyRef:
      name: slurm-db-password
      key: password
```

#### 自動生成的配置

slurm-operator 會自動生成以下配置檔：

| 配置檔 | 來源 CRD | 包含內容 |
|--------|---------|---------|
| `slurm.conf` | Controller | ClusterName, AuthType, AccountingStorageType 等 |
| `slurmdbd.conf` | Accounting | DbdHost, StorageType, 資料庫連線參數 |

> 詳細配置請參考 [Helm Chart 深入解析](./deep-dive-helm.md)

### 4.2 不負責的事項

以下項目明確不屬於 slurm-operator 的管理範圍：

```mermaid
flowchart LR
    subgraph NotManaged["❌ slurm-operator 不管理"]
        A["Slurm Account<br/>組織/專案帳戶"]
        B["Slurm User<br/>使用者與帳戶關聯"]
        C["Association<br/>使用者-帳戶-分區"]
        D["QOS<br/>服務品質策略"]
        E["資源限制<br/>MaxJobs, GrpTRES"]
        F["Fairshare<br/>公平分享配置"]
        G["授權角色<br/>Admin/Operator/Coordinator"]
    end

    style NotManaged fill:#ffebee,stroke:#c62828
```

**原因**：這些屬於 **Slurm policy 與組織治理層面**，高度依賴各組織的具體需求，無法通用化。

---

## 5. 認證機制詳解

### 5.1 auth/slurm vs auth/jwt

Slurm 支援兩種主要認證機制，slurm-operator 同時管理這兩種金鑰：

```mermaid
flowchart TB
    subgraph AuthMethods["認證機制"]
        direction TB
        subgraph AuthSlurm["auth/slurm"]
            SlurmKey["slurm.key<br/>對稱金鑰"]
            SlurmUse["用於內部元件通訊<br/>slurmctld ↔ slurmd<br/>slurmctld ↔ slurmdbd"]
        end

        subgraph AuthJWT["auth/jwt"]
            JwtKey["jwt_hs256.key<br/>HS256 簽名金鑰"]
            JwtUse["用於外部 API 認證<br/>REST API (slurmrestd)<br/>Operator 連線"]
        end
    end

    SlurmKey --> SlurmUse
    JwtKey --> JwtUse

    style AuthSlurm fill:#e3f2fd,stroke:#1565c0
    style AuthJWT fill:#fff3e0,stroke:#ef6c00
```

| 認證類型 | 金鑰檔案 | 用途 | CRD 欄位 |
|----------|----------|------|----------|
| `auth/slurm` | `slurm.key` | Slurm daemon 間通訊 | `slurmKeyRef` |
| `auth/jwt` | `jwt_hs256.key` | REST API、Token 生成 | `jwtHs256KeyRef` |

> **重要**: 部署後這兩個金鑰參考 **不可變更**。如需更換金鑰，必須重新部署整個叢集。

### 5.2 認證流程

以下是使用者從登入到提交作業的完整認證流程：

```mermaid
sequenceDiagram
    autonumber
    participant User as 使用者
    participant LDAP as LDAP/AD
    participant Login as LoginSet Pod
    participant SSSD as SSSD
    participant Ctrl as slurmctld
    participant Acct as slurmdbd

    rect rgb(230, 245, 255)
        note right of User: 階段 1: 系統認證
        User->>Login: SSH 連線
        Login->>SSSD: PAM 認證請求
        SSSD->>LDAP: LDAP bind
        LDAP-->>SSSD: 認證結果
        SSSD-->>Login: PAM 結果
        Login-->>User: Shell 存取
    end

    rect rgb(255, 243, 224)
        note right of User: 階段 2: Slurm 認證
        User->>Login: sbatch job.sh
        Login->>Ctrl: 作業提交 (auth/slurm)
        Ctrl->>Ctrl: 驗證 slurm.key
    end

    rect rgb(232, 245, 233)
        note right of User: 階段 3: Accounting 授權
        Ctrl->>Acct: 查詢使用者關聯
        Acct-->>Ctrl: Account/QOS/Limits
        alt 使用者存在於 Accounting
            Ctrl-->>User: 作業已排程
        else 使用者不存在
            Ctrl-->>User: Invalid account
        end
    end
```

### 5.3 SSSD 與 LDAP 整合

LoginSet 和 NodeSet（啟用 SSH 時）需要配置 SSSD 以整合 LDAP/AD 認證。

#### CRD 配置

```yaml
# LoginSet CRD
apiVersion: slinky.slurm.net/v1beta1
kind: LoginSet
spec:
  sssdConfRef:
    name: slurm-sssd-config
    key: sssd.conf
```

#### SSSD 配置範例

```yaml
# Secret 定義
apiVersion: v1
kind: Secret
metadata:
  name: slurm-sssd-config
  namespace: slurm
stringData:
  sssd.conf: |
    [sssd]
    services = nss, pam
    config_file_version = 2
    domains = EXAMPLE.COM

    [nss]
    filter_groups = root
    filter_users = root

    [pam]
    offline_credentials_expiration = 7

    [domain/EXAMPLE.COM]
    id_provider = ldap
    auth_provider = ldap
    ldap_uri = ldaps://ldap.example.com:636
    ldap_search_base = dc=example,dc=com
    ldap_user_search_base = ou=People,dc=example,dc=com
    ldap_group_search_base = ou=Groups,dc=example,dc=com
    ldap_id_use_start_tls = false
    ldap_tls_reqcert = demand
    ldap_tls_cacert = /etc/ssl/certs/ca-certificates.crt
    cache_credentials = true
    enumerate = false
```

#### Helm 配置

```yaml
# values.yaml
loginset:
  sssd:
    secretRef:
      name: slurm-sssd-config
      key: sssd.conf
```

> 詳細 LDAP 配置請參考 [使用指南 - LDAP 認證配置](./slurm-usage-guide.md#5-ldap-認證配置)

---

## 6. 授權角色說明

Slurm Accounting 支援三種授權角色層級：

```mermaid
flowchart TD
    subgraph Roles["授權角色層級"]
        Admin["AdminLevel=Admin<br/>最高權限"]
        Operator["AdminLevel=Operator<br/>操作權限"]
        Coordinator["Account Coordinator<br/>帳戶協調者"]
        None["AdminLevel=None<br/>一般使用者"]
    end

    Admin -->|可管理| Operator
    Admin -->|可管理| Coordinator
    Admin -->|可管理| None
    Operator -->|可管理| None
    Coordinator -->|限定帳戶內| None

    style Admin fill:#ffcdd2,stroke:#c62828
    style Operator fill:#fff9c4,stroke:#f9a825
    style Coordinator fill:#c8e6c9,stroke:#2e7d32
    style None fill:#e3f2fd,stroke:#1565c0
```

### 角色權限對照表

| 權限 | Admin | Operator | Coordinator | None |
|------|:-----:|:--------:|:-----------:|:----:|
| 管理 Cluster | ✅ | ❌ | ❌ | ❌ |
| 建立/刪除 Account | ✅ | ❌ | ❌ | ❌ |
| 建立/刪除 User | ✅ | ✅ | 限帳戶內 | ❌ |
| 修改 QOS | ✅ | ❌ | ❌ | ❌ |
| 設定資源限制 | ✅ | ✅ | 限帳戶內 | ❌ |
| 查看所有作業 | ✅ | ✅ | ❌ | ❌ |
| 取消他人作業 | ✅ | ✅ | 限帳戶內 | ❌ |
| 提交作業 | ✅ | ✅ | ✅ | ✅ |

### 設定範例

```bash
# 設定 Admin
sacctmgr modify user alice set adminlevel=admin

# 設定 Operator
sacctmgr modify user bob set adminlevel=operator

# 設定 Account Coordinator
sacctmgr add coordinator account=research names=charlie

# 查看使用者角色
sacctmgr show user format=user,adminlevel,coordinator
```

---

## 7. 為何 Slurm Accounting 必須人工 Bootstrap

### 7.1 Slurm 原生授權模型

Slurm Accounting 採用階梯式授權模型：

```mermaid
flowchart TD
    A["1. Cluster 必須存在"] --> B["2. Account 必須存在"]
    B --> C["3. User 必須屬於 Account"]
    C --> D["4. User 必須具備 Admin 角色"]
    D --> E["5. 才能建立或修改其他 Account/User"]

    style A fill:#e3f2fd
    style B fill:#e3f2fd
    style C fill:#e3f2fd
    style D fill:#e3f2fd
    style E fill:#c8e6c9
```

### 7.2 Chicken-Egg 問題

在一個全新的 slurmdbd 中：

| 狀態 | 說明 |
|------|------|
| ❌ 不存在任何 Cluster | 資料庫是空的 |
| ❌ 不存在任何 Account | 沒有組織結構 |
| ❌ 不存在任何 Admin | 沒有人有權限建立 |
| ❌ 無法透過正常權限路徑建立 | 死結 |

👉 **第一位管理者必須透過 out-of-band 方式建立**

這是 **Slurm 的設計前提**，而非 Slinky 的限制。

---

## 8. Bootstrap 操作指南

### 8.1 前置條件檢查

在開始 Bootstrap 之前，確認以下條件：

```bash
# 1. 確認 Controller Pod 運行中
kubectl get pods -l app.kubernetes.io/component=slurmctld -n slurm

# 2. 確認 Accounting (slurmdbd) 運行中
kubectl get pods -l app.kubernetes.io/component=slurmdbd -n slurm

# 3. 確認資料庫連線正常
kubectl logs -l app.kubernetes.io/component=slurmdbd -n slurm | grep -i "database"
```

### 8.2 初始化步驟

#### 步驟 1：進入 Controller Pod

```bash
# 取得 Controller Pod 名稱
CONTROLLER_POD=$(kubectl get pods -l app.kubernetes.io/component=slurmctld \
  -n slurm -o jsonpath='{.items[0].metadata.name}')

# 進入 Pod
kubectl exec -it $CONTROLLER_POD -n slurm -- bash
```

#### 步驟 2：註冊 Cluster 到 Accounting

```bash
# 查看目前的 ClusterName（從 slurm.conf）
grep ClusterName /etc/slurm/slurm.conf

# 註冊 Cluster（將 <cluster-name> 替換為實際名稱）
sacctmgr add cluster <cluster-name>

# 驗證
sacctmgr show cluster
```

#### 步驟 3：建立根帳戶

```bash
# 建立組織根帳戶
sacctmgr add account root description="Root account"

# 建立部門/專案帳戶（範例）
sacctmgr add account research parent=root description="Research group"
sacctmgr add account engineering parent=root description="Engineering team"

# 驗證帳戶結構
sacctmgr show account tree
```

#### 步驟 4：建立第一位 Admin 使用者

```bash
# 建立使用者並指派 Admin 權限
# 注意：使用者名稱必須與 LDAP/SSSD 解析的名稱完全一致
sacctmgr add user admin account=root adminlevel=admin

# 驗證
sacctmgr show user admin withassoc
```

#### 步驟 5：建立一般使用者

```bash
# 建立使用者並關聯帳戶
sacctmgr add user alice account=research
sacctmgr add user bob account=engineering

# 批次建立（使用檔案）
cat << 'EOF' > /tmp/users.txt
User|Account|AdminLevel
alice|research|None
bob|engineering|None
charlie|research|None
EOF

sacctmgr load /tmp/users.txt
```

#### 步驟 6：設定 QOS（可選）

```bash
# 建立 QOS
sacctmgr add qos normal priority=50
sacctmgr add qos high priority=100 maxjobs=10
sacctmgr add qos low priority=10 maxjobs=100

# 指派 QOS 給帳戶
sacctmgr modify account research set qos=normal,high
sacctmgr modify account engineering set qos=normal

# 驗證
sacctmgr show qos format=name,priority,maxjobs
```

#### 步驟 7：設定資源限制（可選）

```bash
# 設定帳戶層級的資源限制
sacctmgr modify account research set GrpTRES=cpu=1000,mem=500G
sacctmgr modify account engineering set GrpTRES=cpu=500,mem=250G

# 設定使用者層級的資源限制
sacctmgr modify user alice set MaxJobs=50 MaxSubmitJobs=100

# 驗證
sacctmgr show assoc format=user,account,grptres,maxjobs
```

### 8.3 完整範例腳本

以下是一個完整的 Bootstrap 腳本範例：

```bash
#!/bin/bash
# Slurm Accounting Bootstrap Script
# 請在 Controller Pod 內執行

set -e

# 配置變數（請根據實際情況修改）
CLUSTER_NAME="my_cluster"
ADMIN_USER="admin"           # 必須與 LDAP 使用者名稱一致
ADMIN_ACCOUNT="root"

echo "=== Slurm Accounting Bootstrap ==="

# 1. 註冊 Cluster
echo "[1/5] 註冊 Cluster..."
sacctmgr -i add cluster $CLUSTER_NAME

# 2. 建立根帳戶
echo "[2/5] 建立帳戶結構..."
sacctmgr -i add account root description="Root account"
sacctmgr -i add account research parent=root description="Research"
sacctmgr -i add account engineering parent=root description="Engineering"

# 3. 建立 Admin 使用者
echo "[3/5] 建立 Admin 使用者..."
sacctmgr -i add user $ADMIN_USER account=$ADMIN_ACCOUNT adminlevel=admin

# 4. 建立 QOS
echo "[4/5] 建立 QOS..."
sacctmgr -i add qos normal priority=50
sacctmgr -i add qos high priority=100 maxjobs=10
sacctmgr -i add qos low priority=10 maxjobs=100

# 5. 驗證
echo "[5/5] 驗證設定..."
echo ""
echo "=== Cluster ==="
sacctmgr show cluster
echo ""
echo "=== Account Tree ==="
sacctmgr show account tree
echo ""
echo "=== Admin User ==="
sacctmgr show user $ADMIN_USER withassoc
echo ""
echo "=== QOS ==="
sacctmgr show qos

echo ""
echo "=== Bootstrap 完成 ==="
echo "現在 $ADMIN_USER 可以透過正常權限路徑管理其他使用者了。"
```

---

## 9. 使用者名稱一致性要求

### 關鍵設計原則

Slurm Accounting 中的使用者名稱，**必須與系統層實際辨識到的使用者名稱完全一致**。

此名稱來源於：

- SSSD 設定
- LDAP user mapping

### 驗證步驟

在 **LoginSet Pod** 中執行：

```bash
# 取得 LoginSet Pod 名稱
LOGIN_POD=$(kubectl get pods -l app.kubernetes.io/component=loginset \
  -n slurm -o jsonpath='{.items[0].metadata.name}')

# 進入 Pod 並驗證使用者名稱
kubectl exec -it $LOGIN_POD -n slurm -- id <username>
```

Accounting 中所使用的 `User` 欄位，必須與輸出結果完全相同：

| 系統顯示 (`id` 輸出) | Accounting 應使用 |
|---------------------|-------------------|
| `admin@domain` | `admin@domain` |
| `admin` | `admin` |
| `uid=1000(alice)` → `alice` | `alice` |

### 常見錯誤

```bash
# ❌ 錯誤：名稱不一致
# 系統顯示: alice@EXAMPLE.COM
# Accounting 使用: alice
sacctmgr add user alice account=research  # 權限將無法生效

# ✅ 正確：名稱一致
sacctmgr add user alice@EXAMPLE.COM account=research
```

> 詳細 LDAP 配置請參考 [使用指南 - LDAP 認證配置](./slurm-usage-guide.md#5-ldap-認證配置)

---

## 10. Hybrid Cluster 認證考量

在 Hybrid 模式（K8s 內 + 外部節點混合）下，認證配置需要額外考量：

```mermaid
flowchart TB
    subgraph K8s["Kubernetes 叢集"]
        Ctrl[slurmctld]
        K8sNodes[K8s Worker Pods]
        Keys["金鑰 Secrets<br/>slurm.key, jwt_hs256.key"]
    end

    subgraph External["外部節點"]
        ExtNodes[外部 slurmd]
        ExtKeys["金鑰檔案<br/>/etc/slurm/*.key"]
    end

    Keys -->|自動掛載| Ctrl
    Keys -->|自動掛載| K8sNodes
    Keys -.->|需手動複製| ExtKeys
    ExtKeys --> ExtNodes

    style K8s fill:#e3f2fd
    style External fill:#fff3e0
```

### 金鑰同步要求

外部節點必須手動同步以下金鑰：

| 金鑰 | 路徑 | 用途 |
|------|------|------|
| `slurm.key` | `/etc/slurm/slurm.key` | 節點認證 |
| `jwt_hs256.key` | `/etc/slurm/jwt_hs256.key` | REST API (如需) |

### 匯出金鑰

```bash
# 從 Kubernetes Secret 匯出金鑰
kubectl get secret slurm-auth-slurm -n slurm \
  -o jsonpath='{.data.slurm\.key}' | base64 -d > slurm.key

kubectl get secret slurm-auth-jwths256 -n slurm \
  -o jsonpath='{.data.jwt_hs256\.key}' | base64 -d > jwt_hs256.key

# 複製到外部節點
scp slurm.key jwt_hs256.key external-node:/etc/slurm/

# 設定權限
ssh external-node 'chmod 600 /etc/slurm/*.key && chown slurm:slurm /etc/slurm/*.key'
```

### 外部節點 SSSD 配置

外部節點的 SSSD 配置必須與 K8s 內 LoginSet 使用相同的設定，確保使用者名稱解析一致。

```bash
# 驗證外部節點使用者名稱解析
ssh external-node id alice

# 應與 LoginSet 中的結果一致
kubectl exec -n slurm loginset-pod -- id alice
```

> 詳細 Hybrid 模式設定請參考 [Hybrid 部署指南](./hybrid.md)

---

## 11. 常見問題排除

### Q1: 在 Controller Pod 中出現 "user not found on this system"

**現象**：
```bash
$ sacctmgr add user alice account=research
 Warning: user 'alice' not found on this system
```

**說明**：這是 **預期行為**，原因如下：

- Controller Pod 可能未配置 SSSD
- `sacctmgr` 操作的是 slurmdbd 資料庫
- 不依賴該容器本身是否能解析 LDAP 使用者

**該警告不影響 Accounting 資料寫入。**

### Q2: sacctmgr add account 失敗，顯示權限不足

**現象**：
```bash
$ sacctmgr add account newproject
 Problem adding accounts: Access/permission denied
```

**原因**：當前使用者沒有 Admin 權限。

**解決方案**：
```bash
# 在 Controller Pod 中以 root 身份執行 Bootstrap
kubectl exec -it $CONTROLLER_POD -n slurm -- bash
sacctmgr add user <your-ldap-user> account=root adminlevel=admin
```

### Q3: 提交作業時顯示 "Invalid account"

**現象**：
```bash
$ sbatch --account=research job.sh
sbatch: error: Invalid account or account/partition combination specified
```

**原因**：使用者未與該帳戶關聯。

**解決方案**：
```bash
# 檢查使用者關聯
sacctmgr show user <username> withassoc

# 新增關聯
sacctmgr add user <username> account=research
```

### Q4: 如何查看目前的帳戶/使用者/QOS 設定？

```bash
# 查看所有帳戶（樹狀結構）
sacctmgr show account tree

# 查看所有使用者及其關聯
sacctmgr show assoc format=user,account,partition,qos,grptres

# 查看所有 QOS
sacctmgr show qos

# 查看特定使用者的詳細資訊
sacctmgr show user alice withassoc
```

---

## 12. 設計澄清

### Non-Goals（非設計目標）

以下並非 slurm-operator 的設計目標：

| 項目 | 原因 |
|------|------|
| 將 Slurm Accounting 宣告化（CRD 化） | 帳戶結構高度依賴組織政策 |
| 強制統一組織的 project/account 模型 | 無法通用化 |
| 自動推導使用者與資源歸屬關係 | 需要組織層級的決策 |
| 自動執行 `sacctmgr add cluster` | 應由管理者明確執行 |

### 設計定位

Bootstrap 操作被視為：

- ✅ 一次性初始化（one-time bootstrap）
- ✅ 緊急維運或 break-glass 操作
- ❌ **不屬於 Operator Reconciliation 範圍**

slurm-operator 不會、也不應該嘗試重播或管理此狀態。

---

## 13. 總結

```mermaid
flowchart LR
    subgraph Conclusion["結論"]
        A["觀察到的行為<br/>符合 Slurm 原生設計"]
        B["slurm-operator 行為<br/>符合設計責任邊界"]
        C["人工 Bootstrap<br/>是必要且預期的操作"]
    end

    A --> D["✅ 非異常"]
    B --> D
    C --> D

    style Conclusion fill:#e8f5e9
    style D fill:#c8e6c9,stroke:#2e7d32
```

### 核心要點

1. **所觀察到的行為符合 Slurm 原生設計**
2. **slurm-operator 的行為符合其設計責任邊界**
3. **Slurm Accounting 的人工 Bootstrap 是必要且預期的操作**

### 設計保證

此設計確保：

- Operator 的職責清晰
- Slurm 的治理彈性不被限制
- 組織可以自由定義自己的帳戶結構和授權策略

> slurm-operator 的目標是讓 Slurm 在 Kubernetes 上穩定運作，
> 而不是定義組織如何分配與治理計算資源。

---

## 相關文件

| 文件 | 說明 |
|------|------|
| [REST API 與 JWT Token 認證](./rest-api.md) | REST API 使用與 Token CRD |
| [slurm.conf 配置指南](./slurm-conf-guide.md) | 配置生成與自訂 |
| [CRD 參考指南](./crd-reference-guide.md) | 完整 CRD 欄位定義 |
| [使用指南](./slurm-usage-guide.md) | 入門操作與 LDAP 配置 |
| [Hybrid 部署指南](./hybrid.md) | K8s + 外部節點混合部署 |
| [FAQ](./slurm-faq.md) | 常見問題 |
| [Helm Chart 深入解析](./deep-dive-helm.md) | Helm 部署配置 |
| [架構文件](./architecture.md) | 系統架構設計 |

---

## 附錄：sacctmgr 常用命令速查

| 操作 | 命令 |
|------|------|
| 註冊 Cluster | `sacctmgr add cluster <name>` |
| 新增帳戶 | `sacctmgr add account <name> [parent=<parent>]` |
| 新增使用者 | `sacctmgr add user <name> account=<account>` |
| 設定 Admin | `sacctmgr modify user <name> set adminlevel=admin` |
| 新增 QOS | `sacctmgr add qos <name> [priority=N] [maxjobs=N]` |
| 設定資源限制 | `sacctmgr modify account <name> set GrpTRES=cpu=N,mem=NG` |
| 查看帳戶樹 | `sacctmgr show account tree` |
| 查看使用者關聯 | `sacctmgr show user <name> withassoc` |
| 查看所有關聯 | `sacctmgr show assoc` |
| 刪除使用者 | `sacctmgr delete user <name>` |
| 刪除帳戶 | `sacctmgr delete account <name>` |

> 完整 sacctmgr 文件：[https://slurm.schedmd.com/sacctmgr.html](https://slurm.schedmd.com/sacctmgr.html)
