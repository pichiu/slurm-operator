# slurm-operator API 通訊完整指南

本文件深入探討 slurm-operator 的通訊機制、API 使用方式、認證流程，以及相關的最佳實踐。

## 目錄

- [通訊架構概覽](#通訊架構概覽)
- [通訊方式](#通訊方式)
- [API 詳解](#api-詳解)
- [認證機制](#認證機制)
- [關鍵檔案對照表](#關鍵檔案對照表)
- [完整流程圖](#完整流程圖)
- [程式碼範例](#程式碼範例)
- [Best Practices](#best-practices)
- [常見問題 (FAQ)](#常見問題-faq)
- [Lessons Learned](#lessons-learned)

---

## 通訊架構概覽

slurm-operator 採用多層通訊架構：

```
┌─────────────────────────────────────────────────────────────────┐
│                        使用者 / 應用程式                           │
└────────────────────────────┬────────────────────────────────────┘
                             │
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│    kubectl      │ │  Slurm REST API │ │    SSH 存取     │
│  (CRD 管理)     │ │   (Port 6820)   │ │  (LoginSet)     │
└────────┬────────┘ └────────┬────────┘ └────────┬────────┘
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes API Server                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                       slurm-operator                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐  │
│  │ Controller  │ │  NodeSet    │ │   RestApi   │ │   Token   │  │
│  │ Reconciler  │ │ Reconciler  │ │ Reconciler  │ │ Reconciler│  │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └─────┬─────┘  │
│         │               │               │              │        │
│         └───────────────┴───────┬───────┴──────────────┘        │
│                                 │                                │
│                    ┌────────────▼────────────┐                  │
│                    │      ClientMap          │                  │
│                    │  (Slurm Client 連接池)  │                  │
│                    └────────────┬────────────┘                  │
└─────────────────────────────────┼───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Slurm REST API (slurmrestd)                │
│                           Port 6820                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Slurm Cluster                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ slurmctld│  │ slurmdbd │  │  slurmd  │  │  login   │        │
│  │ (6817)   │  │ (6819)   │  │ (workers)│  │  (SSH)   │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 通訊方式

### 1. Kubernetes CRD API (主要管理方式)

透過 kubectl 管理 Custom Resources，這是管理 Slurm 叢集的主要方式。

**可用的 CRD:**

| CRD | Short Name | 用途 |
|-----|------------|------|
| `Controller` | - | Slurm 控制平面 (slurmctld) |
| `NodeSet` | - | Slurm 工作節點 (slurmd) |
| `LoginSet` | - | 登入節點 (SSH 存取) |
| `Accounting` | - | 會計服務 (slurmdbd) |
| `RestApi` | `slurmrestd` | REST API 服務 |
| `Token` | `jwt`, `tokens` | JWT 認證令牌 |

**基本操作:**

```bash
# 查看所有 Slurm 相關資源
kubectl get controller,nodeset,loginset,accounting,restapi,token

# 查看特定資源詳情
kubectl describe controller my-cluster
kubectl describe restapi my-cluster

# 套用配置
kubectl apply -f slurm-cluster.yaml

# 查看 CRD 定義
kubectl explain restapi.spec
kubectl explain token.spec
```

### 2. Slurm REST API (應用程式整合)

slurmrestd 提供 HTTP REST API，適用於應用程式整合。

**端點資訊:**

| 項目 | 值 |
|------|---|
| 預設端口 | 6820 |
| 協議 | HTTP |
| 認證 | JWT Token (X-SLURM-USER-TOKEN header) |
| API 版本 | v0.0.41+ |

**可用 API 端點:**

```
# 節點管理
GET  /slurm/v0.0.41/nodes           # 列出所有節點
GET  /slurm/v0.0.41/node/{name}     # 取得特定節點
POST /slurm/v0.0.41/node/{name}     # 更新節點狀態

# 作業管理
GET  /slurm/v0.0.41/jobs            # 列出所有作業
GET  /slurm/v0.0.41/job/{job_id}    # 取得特定作業
POST /slurm/v0.0.41/job/submit      # 提交作業
DELETE /slurm/v0.0.41/job/{job_id}  # 取消作業

# 分區管理
GET  /slurm/v0.0.41/partitions      # 列出所有分區
GET  /slurm/v0.0.41/partition/{name}# 取得特定分區

# 會計 (需啟用 slurmdbd)
GET  /slurmdb/v0.0.41/accounts      # 列出帳戶
GET  /slurmdb/v0.0.41/users         # 列出使用者

# 系統狀態
GET  /slurm/v0.0.41/ping            # 健康檢查
GET  /slurm/v0.0.41/diag            # 診斷資訊
```

### 3. SSH 存取 (互動式操作)

LoginSet 提供 SSH 登入節點：

```bash
# 取得 LoginSet Service IP
kubectl get svc -l app.kubernetes.io/component=login

# SSH 連線
ssh user@<login-service-ip>
```

---

## API 詳解

### Slurm Client 內部架構

Operator 使用 `slurm-client` 函式庫與 Slurm REST API 通訊。

**檔案位置:** `internal/clientmap/clientmap.go`

```go
// ClientMap 管理多個 Slurm 客戶端連接
type ClientMap struct {
    lock    sync.RWMutex
    clients map[string]client.Client
}

// 主要方法
func (m *ClientMap) Get(name types.NamespacedName) client.Client   // 取得客戶端
func (m *ClientMap) Has(names ...types.NamespacedName) bool        // 檢查是否存在
func (m *ClientMap) Add(name types.NamespacedName, client client.Client) bool // 新增客戶端
func (m *ClientMap) Remove(name types.NamespacedName) bool         // 移除客戶端
```

**客戶端配置:**

```go
// 檔案: internal/controller/slurmclient/slurmclient_sync.go
config := &slurmclient.Config{
    Server:    "http://restapi-service:6820",  // REST API 位址
    AuthToken: authToken,                      // JWT Token
}

options := &slurmclient.ClientOptions{
    DisableFor: []slurmobject.Object{
        &slurmtypes.V0041ControllerPing{},     // 可停用特定操作
    },
}

slurmClient, err := slurmclient.NewClient(config, options)
```

### API 操作範例

**檔案位置:** `internal/controller/nodeset/slurmcontrol/slurmcontrol.go`

#### 節點操作

```go
// 取得單一節點
slurmNode := &slurmtypes.V0044Node{}
key := slurmobject.ObjectKey(nodeName)
if err := slurmClient.Get(ctx, key, slurmNode); err != nil {
    return err
}

// 列出所有節點
nodeList := &slurmtypes.V0044NodeList{}
if err := slurmClient.List(ctx, nodeList); err != nil {
    return err
}

// 帶快取刷新的列表
opts := &slurmclient.ListOptions{RefreshCache: true}
if err := slurmClient.List(ctx, nodeList, opts); err != nil {
    return err
}

// 更新節點狀態 (設定為 DRAIN)
req := slurmapi.V0044UpdateNodeMsg{
    State: ptr.To([]slurmapi.V0044UpdateNodeMsgState{
        slurmapi.V0044UpdateNodeMsgStateDRAIN,
    }),
    Reason: ptr.To("slurm-operator: maintenance"),
}
if err := slurmClient.Update(ctx, slurmNode, req); err != nil {
    return err
}

// 查詢節點狀態
stateSet := slurmNode.GetStateAsSet()
isDrain := stateSet.Has(slurmapi.V0044NodeStateDRAIN)
isAllocated := stateSet.Has(slurmapi.V0044NodeStateALLOCATED)
```

#### 作業操作

```go
// 列出所有作業
jobList := &slurmtypes.V0044JobInfoList{}
if err := slurmClient.List(ctx, jobList); err != nil {
    return err
}

// 篩選運行中的作業
for _, job := range jobList.Items {
    if !job.GetStateAsSet().Has(slurmapi.V0044JobInfoJobStateRUNNING) {
        continue
    }

    // 取得作業分配的節點
    nodes, err := hostlist.Expand(ptr.Deref(job.Nodes, ""))
    if err != nil {
        return err
    }

    // 計算作業截止時間
    startTime := time.Unix(ptr.Deref(job.StartTime.Number, 0), 0)
    timeLimit := time.Duration(ptr.Deref(job.TimeLimit.Number, 0)) * time.Minute
    deadline := startTime.Add(timeLimit)
}
```

### SlurmControlInterface

完整的 Slurm 控制介面：

```go
type SlurmControlInterface interface {
    // 快取管理
    RefreshNodeCache(ctx context.Context, nodeset *NodeSet) error

    // 節點資訊更新
    UpdateNodeWithPodInfo(ctx context.Context, nodeset *NodeSet, pod *Pod) error
    UpdateNodeTopology(ctx context.Context, nodeset *NodeSet, pod *Pod, topology string) error

    // 節點狀態管理
    MakeNodeDrain(ctx context.Context, nodeset *NodeSet, pod *Pod, reason string) error
    MakeNodeUndrain(ctx context.Context, nodeset *NodeSet, pod *Pod, reason string) error

    // 狀態查詢
    IsNodeDrain(ctx context.Context, nodeset *NodeSet, pod *Pod) (bool, error)
    IsNodeDrained(ctx context.Context, nodeset *NodeSet, pod *Pod) (bool, error)
    IsNodeDownForUnresponsive(ctx context.Context, nodeset *NodeSet, pod *Pod) (bool, error)

    // 統計
    CalculateNodeStatus(ctx context.Context, nodeset *NodeSet, pods []*Pod) (SlurmNodeStatus, error)
    GetNodeDeadlines(ctx context.Context, nodeset *NodeSet, pods []*Pod) (*TimeStore, error)
}
```

---

## 認證機制

### JWT 認證流程

slurm-operator 使用 JWT (JSON Web Token) 進行認證。

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Token CR  │     │   Operator  │     │  K8s Secret │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │ 1. 建立            │                   │
       ├──────────────────►│                   │
       │                   │ 2. 讀取簽名密鑰    │
       │                   ├──────────────────►│
       │                   │◄─────────────────┤
       │                   │                   │
       │                   │ 3. 產生 JWT       │
       │                   │ (HS256 簽名)      │
       │                   │                   │
       │                   │ 4. 儲存 JWT       │
       │                   ├──────────────────►│
       │                   │                   │
       │                   │ 5. 更新 Status    │
       │◄─────────────────┤                   │
       │                   │                   │
```

### Token CRD 規格

**檔案位置:** `api/v1beta1/token_types.go`

```go
type TokenSpec struct {
    // JWT HS256 簽名密鑰參考
    JwtHs256KeyRef JwtSecretKeySelector `json:"jwtHs256KeyRef,omitzero"`

    // Token 對應的使用者名稱
    Username string `json:"username,omitzero"`

    // Token 有效期 (預設 15 分鐘)
    Lifetime *metav1.Duration `json:"lifetime,omitempty"`

    // 是否自動刷新 (預設 true)
    Refresh bool `json:"refresh,omitzero"`

    // 儲存 JWT 的 Secret 參考
    SecretRef *corev1.SecretKeySelector `json:"secretRef,omitempty"`
}
```

### JWT Token 結構

**檔案位置:** `internal/controller/token/slurmjwt/token.go`

```go
type TokenClaims struct {
    jwt.RegisteredClaims `json:",inline"`
    SlurmUsername string `json:"sun"`  // Slurm 特有欄位
}

// Token 產生
func (t *Token) NewSignedToken() (string, error) {
    now := time.Now()
    claims := TokenClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            ID:        string(uuid.NewUUID()),       // 唯一 ID
            Issuer:    "slurm-operator",             // 簽發者
            IssuedAt:  jwt.NewNumericDate(now),      // 簽發時間
            ExpiresAt: jwt.NewNumericDate(now.Add(t.lifetime)), // 過期時間
            NotBefore: jwt.NewNumericDate(now),      // 有效起始時間
        },
        SlurmUsername: t.username,
    }

    token := jwt.NewWithClaims(t.method, claims)  // HS256
    return token.SignedString(t.signingKey)
}
```

### Token 刷新機制

```go
// 刷新條件: Token 距離過期時間少於 1/5 壽命時觸發
// 例如: 15 分鐘壽命 → 12 分鐘時開始刷新 (過期前 3 分鐘)

refreshTime := expirationTime.Add(-token.Lifetime() * 1 / 5)

if now.Before(refreshTime) {
    return nil  // 還未到刷新時間
}

// 重新產生 Token
object, err := r.builder.BuildTokenSecret(token)
if err := objectutils.SyncObject(r.Client, ctx, object, true); err != nil {
    return err
}
```

### 使用 Token 範例

```yaml
# 1. 建立 JWT 簽名密鑰 Secret
apiVersion: v1
kind: Secret
metadata:
  name: slurm-jwt-key
type: Opaque
data:
  jwt_hs256.key: <base64-encoded-key>  # 至少 32 bytes

---
# 2. 建立 Token CR
apiVersion: slinky.slurm.net/v1beta1
kind: Token
metadata:
  name: admin-token
spec:
  username: admin
  jwtHs256KeyRef:
    name: slurm-jwt-key
    key: jwt_hs256.key
  lifetime: 1h
  refresh: true
  secretRef:
    name: admin-jwt-secret
    key: SLURM_JWT
```

```bash
# 3. 取得 JWT Token
JWT=$(kubectl get secret admin-jwt-secret -o jsonpath='{.data.SLURM_JWT}' | base64 -d)

# 4. 呼叫 REST API
curl -H "X-SLURM-USER-TOKEN: $JWT" \
     http://restapi-service:6820/slurm/v0.0.41/nodes
```

---

## 關鍵檔案對照表

### API 定義

| 檔案路徑 | 說明 |
|---------|------|
| `api/v1beta1/controller_types.go` | Controller CRD 定義 |
| `api/v1beta1/nodeset_types.go` | NodeSet CRD 定義 |
| `api/v1beta1/loginset_types.go` | LoginSet CRD 定義 |
| `api/v1beta1/accounting_types.go` | Accounting CRD 定義 |
| `api/v1beta1/restapi_types.go` | RestApi CRD 定義 |
| `api/v1beta1/token_types.go` | Token CRD 定義 |

### 控制器

| 檔案路徑 | 說明 |
|---------|------|
| `internal/controller/controller/` | Controller 協調器 |
| `internal/controller/nodeset/` | NodeSet 協調器 |
| `internal/controller/loginset/` | LoginSet 協調器 |
| `internal/controller/accounting/` | Accounting 協調器 |
| `internal/controller/restapi/` | RestApi 協調器 |
| `internal/controller/token/` | Token 協調器 |
| `internal/controller/slurmclient/` | Slurm 客戶端管理 |

### Slurm 客戶端

| 檔案路徑 | 說明 |
|---------|------|
| `internal/clientmap/clientmap.go` | 客戶端連接池 |
| `internal/controller/slurmclient/slurmclient_sync.go` | 客戶端同步邏輯 |
| `internal/controller/nodeset/slurmcontrol/slurmcontrol.go` | Slurm 控制操作 |
| `internal/controller/nodeset/eventhandler/eventhandler_pod.go` | 事件處理 |

### 認證

| 檔案路徑 | 說明 |
|---------|------|
| `internal/controller/token/slurmjwt/token.go` | JWT 產生/驗證 |
| `internal/builder/token_secret.go` | Token Secret 建構 |
| `internal/utils/crypto/signing_key.go` | 簽名密鑰產生 |

### 建構器

| 檔案路徑 | 說明 |
|---------|------|
| `internal/builder/restapi_app.go` | RestApi Deployment 建構 |
| `internal/builder/restapi_service.go` | RestApi Service 建構 |
| `internal/builder/controller_app.go` | Controller Deployment 建構 |
| `internal/builder/nodeset_app.go` | NodeSet StatefulSet 建構 |

---

## 完整流程圖

### Slurm Client 生命週期

```
使用者建立 Controller CR
         │
         ▼
┌─────────────────────────────────────────────┐
│      SlurmClient Controller 偵測變更         │
└──────────────────────┬──────────────────────┘
                       │
         ┌─────────────┴─────────────┐
         ▼                           ▼
┌─────────────────┐         ┌─────────────────┐
│ 取得 RestApi    │         │ 取得 JWT 密鑰   │
│ Service 位址    │         │ 產生認證 Token   │
└────────┬────────┘         └────────┬────────┘
         │                           │
         └─────────────┬─────────────┘
                       │
                       ▼
         ┌─────────────────────────────┐
         │  建立 Slurm Client          │
         │  - 設定 Server URL          │
         │  - 設定 AuthToken           │
         └──────────────┬──────────────┘
                        │
                        ▼
         ┌─────────────────────────────┐
         │  啟動事件處理器              │
         │  - 監聽節點變更              │
         │  - 推送事件到 EventChannel  │
         └──────────────┬──────────────┘
                        │
                        ▼
         ┌─────────────────────────────┐
         │  加入 ClientMap             │
         │  - 鍵: Controller NamespacedName │
         │  - 值: Slurm Client         │
         └──────────────┬──────────────┘
                        │
                        ▼
         ┌─────────────────────────────┐
         │  定期刷新 Token             │
         │  - 每 12 分鐘檢查           │
         │  - 過期前 3 分鐘刷新        │
         └─────────────────────────────┘
```

### REST API 請求流程

```
應用程式
    │
    │ HTTP Request + JWT Header
    ▼
┌─────────────────────────────────────────┐
│          RestApi Service                │
│    (ClusterIP/LoadBalancer:6820)        │
└───────────────────┬─────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│         slurmrestd Container            │
│  ┌────────────────────────────────┐     │
│  │ 1. 驗證 JWT Token              │     │
│  │ 2. 解析請求                    │     │
│  │ 3. 透過 slurm.key 與 slurmctld │     │
│  │    通訊                        │     │
│  └────────────────────────────────┘     │
└───────────────────┬─────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│            slurmctld                    │
│         (Port 6817)                     │
│  ┌────────────────────────────────┐     │
│  │ - 執行請求的操作               │     │
│  │ - 回傳結果                     │     │
│  └────────────────────────────────┘     │
└───────────────────┬─────────────────────┘
                    │
                    ▼
              JSON Response
```

---

## 程式碼範例

### 使用 curl 呼叫 REST API

```bash
#!/bin/bash

# 設定變數
NAMESPACE="default"
RESTAPI_SVC="my-cluster-restapi"
TOKEN_SECRET="admin-jwt-secret"

# 取得 REST API 端點
RESTAPI_URL=$(kubectl get svc $RESTAPI_SVC -n $NAMESPACE \
  -o jsonpath='{.spec.clusterIP}'):6820

# 取得 JWT Token
JWT=$(kubectl get secret $TOKEN_SECRET -n $NAMESPACE \
  -o jsonpath='{.data.SLURM_JWT}' | base64 -d)

# 健康檢查
curl -s -H "X-SLURM-USER-TOKEN: $JWT" \
  "http://$RESTAPI_URL/slurm/v0.0.41/ping" | jq .

# 列出所有節點
curl -s -H "X-SLURM-USER-TOKEN: $JWT" \
  "http://$RESTAPI_URL/slurm/v0.0.41/nodes" | jq .

# 列出所有作業
curl -s -H "X-SLURM-USER-TOKEN: $JWT" \
  "http://$RESTAPI_URL/slurm/v0.0.41/jobs" | jq .

# 列出所有分區
curl -s -H "X-SLURM-USER-TOKEN: $JWT" \
  "http://$RESTAPI_URL/slurm/v0.0.41/partitions" | jq .

# 提交作業
curl -s -X POST -H "X-SLURM-USER-TOKEN: $JWT" \
  -H "Content-Type: application/json" \
  "http://$RESTAPI_URL/slurm/v0.0.41/job/submit" \
  -d '{
    "job": {
      "name": "test-job",
      "nodes": "1",
      "tasks": 1,
      "cpus_per_task": 1,
      "time_limit": "00:10:00",
      "current_working_directory": "/tmp",
      "environment": ["PATH=/usr/bin:/bin"],
      "script": "#!/bin/bash\necho Hello World\nsleep 60"
    }
  }' | jq .
```

### Python 客戶端範例

```python
#!/usr/bin/env python3
"""
Slurm REST API Python 客戶端範例
"""

import requests
import subprocess
import base64
import json
from typing import Optional, Dict, Any

class SlurmClient:
    """Slurm REST API 客戶端"""

    def __init__(
        self,
        base_url: str,
        token: Optional[str] = None,
        namespace: str = "default",
        token_secret: str = "admin-jwt-secret"
    ):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()

        if token:
            self.token = token
        else:
            # 從 Kubernetes Secret 取得 Token
            self.token = self._get_token_from_k8s(namespace, token_secret)

        self.session.headers.update({
            "X-SLURM-USER-TOKEN": self.token,
            "Content-Type": "application/json"
        })

    def _get_token_from_k8s(self, namespace: str, secret_name: str) -> str:
        """從 Kubernetes Secret 取得 JWT Token"""
        cmd = [
            "kubectl", "get", "secret", secret_name,
            "-n", namespace,
            "-o", "jsonpath={.data.SLURM_JWT}"
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        return base64.b64decode(result.stdout).decode()

    def _request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """發送 API 請求"""
        url = f"{self.base_url}{endpoint}"
        response = self.session.request(method, url, json=data)
        response.raise_for_status()
        return response.json()

    # ===== 節點操作 =====

    def list_nodes(self) -> Dict:
        """列出所有節點"""
        return self._request("GET", "/slurm/v0.0.41/nodes")

    def get_node(self, name: str) -> Dict:
        """取得特定節點"""
        return self._request("GET", f"/slurm/v0.0.41/node/{name}")

    def update_node(self, name: str, state: str, reason: str = "") -> Dict:
        """更新節點狀態"""
        data = {"state": [state]}
        if reason:
            data["reason"] = reason
        return self._request("POST", f"/slurm/v0.0.41/node/{name}", data)

    # ===== 作業操作 =====

    def list_jobs(self) -> Dict:
        """列出所有作業"""
        return self._request("GET", "/slurm/v0.0.41/jobs")

    def get_job(self, job_id: int) -> Dict:
        """取得特定作業"""
        return self._request("GET", f"/slurm/v0.0.41/job/{job_id}")

    def submit_job(
        self,
        name: str,
        script: str,
        nodes: int = 1,
        tasks: int = 1,
        cpus_per_task: int = 1,
        time_limit: str = "01:00:00",
        partition: Optional[str] = None
    ) -> Dict:
        """提交作業"""
        job = {
            "name": name,
            "nodes": str(nodes),
            "tasks": tasks,
            "cpus_per_task": cpus_per_task,
            "time_limit": time_limit,
            "current_working_directory": "/tmp",
            "environment": ["PATH=/usr/bin:/bin"],
            "script": script
        }
        if partition:
            job["partition"] = partition
        return self._request("POST", "/slurm/v0.0.41/job/submit", {"job": job})

    def cancel_job(self, job_id: int) -> Dict:
        """取消作業"""
        return self._request("DELETE", f"/slurm/v0.0.41/job/{job_id}")

    # ===== 分區操作 =====

    def list_partitions(self) -> Dict:
        """列出所有分區"""
        return self._request("GET", "/slurm/v0.0.41/partitions")

    def get_partition(self, name: str) -> Dict:
        """取得特定分區"""
        return self._request("GET", f"/slurm/v0.0.41/partition/{name}")

    # ===== 系統操作 =====

    def ping(self) -> Dict:
        """健康檢查"""
        return self._request("GET", "/slurm/v0.0.41/ping")

    def diag(self) -> Dict:
        """診斷資訊"""
        return self._request("GET", "/slurm/v0.0.41/diag")


# 使用範例
if __name__ == "__main__":
    # 建立客戶端
    client = SlurmClient(
        base_url="http://my-cluster-restapi:6820",
        namespace="default",
        token_secret="admin-jwt-secret"
    )

    # 健康檢查
    print("=== Ping ===")
    print(json.dumps(client.ping(), indent=2))

    # 列出節點
    print("\n=== Nodes ===")
    nodes = client.list_nodes()
    for node in nodes.get("nodes", []):
        print(f"  - {node.get('name')}: {node.get('state')}")

    # 列出分區
    print("\n=== Partitions ===")
    partitions = client.list_partitions()
    for part in partitions.get("partitions", []):
        print(f"  - {part.get('name')}: {part.get('state')}")

    # 提交作業
    print("\n=== Submit Job ===")
    result = client.submit_job(
        name="test-job",
        script="#!/bin/bash\necho Hello from Slurm!\nhostname\nsleep 30",
        nodes=1,
        tasks=1,
        time_limit="00:05:00"
    )
    print(json.dumps(result, indent=2))
```

### Go 客戶端範例

```go
package main

import (
    "context"
    "fmt"
    "log"

    slurmclient "github.com/SlinkyProject/slurm-client"
    slurmtypes "github.com/SlinkyProject/slurm-client/pkg/types"
    "k8s.io/utils/ptr"
)

func main() {
    ctx := context.Background()

    // 建立客戶端配置
    config := &slurmclient.Config{
        Server:    "http://my-cluster-restapi:6820",
        AuthToken: "your-jwt-token-here",
    }

    // 建立客戶端
    client, err := slurmclient.NewClient(config, nil)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Stop()

    // 啟動客戶端
    if err := client.Start(ctx); err != nil {
        log.Fatalf("Failed to start client: %v", err)
    }

    // 列出所有節點
    nodeList := &slurmtypes.V0044NodeList{}
    if err := client.List(ctx, nodeList); err != nil {
        log.Fatalf("Failed to list nodes: %v", err)
    }

    fmt.Println("=== Nodes ===")
    for _, node := range nodeList.Items {
        name := ptr.Deref(node.Name, "unknown")
        state := node.GetStateAsSet()
        fmt.Printf("  - %s: %v\n", name, state)
    }

    // 列出所有作業
    jobList := &slurmtypes.V0044JobInfoList{}
    if err := client.List(ctx, jobList); err != nil {
        log.Fatalf("Failed to list jobs: %v", err)
    }

    fmt.Println("\n=== Jobs ===")
    for _, job := range jobList.Items {
        id := ptr.Deref(job.JobId, 0)
        name := ptr.Deref(job.Name, "unknown")
        state := job.GetStateAsSet()
        fmt.Printf("  - %d (%s): %v\n", id, name, state)
    }
}
```

### Helm 部署完整範例

```yaml
# values.yaml - 完整的 Slurm 叢集配置
nameOverride: my-cluster

# 認證密鑰 (如不指定會自動產生)
slurmKeyRef: {}
jwtHs256KeyRef: {}

# 叢集名稱
clusterName: my-slurm-cluster

# Controller (slurmctld)
controller:
  slurmctld:
    image:
      repository: ghcr.io/slinkyproject/slurmctld
      tag: 25.11-ubuntu24.04
    resources:
      requests:
        cpu: 500m
        memory: 512Mi
  persistence:
    enabled: true
    storageClassName: standard
    resources:
      requests:
        storage: 10Gi

# REST API (slurmrestd)
restapi:
  replicas: 2
  slurmrestd:
    image:
      repository: ghcr.io/slinkyproject/slurmrestd
      tag: 25.11-ubuntu24.04
    resources:
      requests:
        cpu: 250m
        memory: 256Mi
  service:
    spec:
      type: LoadBalancer  # 外部存取

# Accounting (slurmdbd) - 可選
accounting:
  enabled: true
  slurmdbd:
    image:
      repository: ghcr.io/slinkyproject/slurmdbd
      tag: 25.11-ubuntu24.04
  storageConfig:
    host: mariadb
    port: 3306
    database: slurm_acct_db
    username: slurm
    passwordKeyRef:
      name: mariadb-password
      key: password

# NodeSet (slurmd)
nodesets:
  compute:
    enabled: true
    replicas: 4
    slurmd:
      image:
        repository: ghcr.io/slinkyproject/slurmd
        tag: 25.11-ubuntu24.04
      resources:
        requests:
          cpu: 2
          memory: 4Gi
        limits:
          cpu: 4
          memory: 8Gi
    partition:
      enabled: true
      configMap:
        State: UP
        MaxTime: UNLIMITED
    useResourceLimits: true

# Partitions
partitions:
  all:
    enabled: true
    nodesets:
      - ALL
    configMap:
      State: UP
      Default: "YES"
      MaxTime: UNLIMITED

# LoginSet
loginsets:
  login:
    enabled: true
    replicas: 1
    service:
      spec:
        type: LoadBalancer
```

```bash
# 部署
helm install my-cluster ./helm/slurm -f values.yaml -n slurm --create-namespace

# 驗證
kubectl get pods -n slurm
kubectl get svc -n slurm

# 取得 REST API 端點
kubectl get svc my-cluster-restapi -n slurm -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

---

## Best Practices

### 1. 認證管理

```yaml
# ✅ 建議: 使用外部密鑰管理系統
apiVersion: v1
kind: Secret
metadata:
  name: slurm-jwt-key
  annotations:
    # 使用 External Secrets Operator 或 Vault
    external-secrets.io/refresh-interval: "1h"
type: Opaque

# ✅ 建議: 設定合理的 Token 有效期
apiVersion: slinky.slurm.net/v1beta1
kind: Token
spec:
  lifetime: 1h      # 不要設太長
  refresh: true     # 保持自動刷新

# ❌ 避免: 在 ConfigMap 或程式碼中硬編碼 Token
```

### 2. REST API 存取

```yaml
# ✅ 建議: 使用 NetworkPolicy 限制存取
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: restapi-policy
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: slurmrestd
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              slurm-access: "true"
      ports:
        - port: 6820

# ✅ 建議: 使用 Ingress 加上 TLS
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: restapi-ingress
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
spec:
  tls:
    - hosts:
        - slurm-api.example.com
      secretName: slurm-api-tls
  rules:
    - host: slurm-api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-cluster-restapi
                port:
                  number: 6820
```

### 3. 高可用性

```yaml
# ✅ 建議: 部署多個 RestApi 副本
restapi:
  replicas: 3
  podSpec:
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app.kubernetes.io/name: slurmrestd
            topologyKey: kubernetes.io/hostname
```

### 4. 監控

```yaml
# ✅ 建議: 啟用 Prometheus 監控
controller:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
      interval: 30s
```

### 5. 錯誤處理

```go
// ✅ 建議: 實作重試邏輯
func callAPIWithRetry(client *SlurmClient, maxRetries int) error {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        _, err := client.ListNodes()
        if err == nil {
            return nil
        }
        lastErr = err
        time.Sleep(time.Second * time.Duration(i+1))  // 指數退避
    }
    return lastErr
}

// ✅ 建議: 檢查 API 回應中的錯誤
response := client.ListNodes()
if len(response.Errors) > 0 {
    for _, err := range response.Errors {
        log.Printf("API Error: %s", err.Description)
    }
}
```

---

## 常見問題 (FAQ)

### Q1: 如何確認 REST API 是否正常運作？

```bash
# 檢查 Pod 狀態
kubectl get pods -l app.kubernetes.io/name=slurmrestd

# 檢查 Service
kubectl get svc -l app.kubernetes.io/component=restapi

# 測試連線 (從叢集內)
kubectl run -it --rm debug --image=curlimages/curl -- \
  curl -s http://my-cluster-restapi:6820/slurm/v0.0.41/ping

# 查看 Pod 日誌
kubectl logs -l app.kubernetes.io/name=slurmrestd
```

### Q2: JWT Token 認證失敗怎麼辦？

```bash
# 1. 確認 Token Secret 存在
kubectl get secret admin-jwt-secret

# 2. 確認 Token 未過期
JWT=$(kubectl get secret admin-jwt-secret -o jsonpath='{.data.SLURM_JWT}' | base64 -d)
echo $JWT | cut -d. -f2 | base64 -d 2>/dev/null | jq .exp

# 3. 確認 JWT HS256 密鑰一致
kubectl get secret slurm-jwt-key -o yaml

# 4. 強制刷新 Token
kubectl delete secret admin-jwt-secret
kubectl delete pod -l app.kubernetes.io/name=slurmrestd
```

### Q3: 如何從外部存取 REST API？

```bash
# 方法 1: Port Forward
kubectl port-forward svc/my-cluster-restapi 6820:6820

# 方法 2: LoadBalancer (雲端環境)
kubectl patch svc my-cluster-restapi -p '{"spec":{"type":"LoadBalancer"}}'

# 方法 3: Ingress
# 見上方 Best Practices 範例
```

### Q4: API 回傳 404 或空結果？

```bash
# 檢查 slurmctld 狀態
kubectl get pods -l app.kubernetes.io/name=slurmctld
kubectl logs -l app.kubernetes.io/name=slurmctld

# 確認 slurmrestd 能連到 slurmctld
kubectl exec -it deploy/my-cluster-restapi -- \
  cat /etc/slurm/slurm.conf | grep ControlMachine

# 檢查 Slurm 內部狀態
kubectl exec -it deploy/my-cluster-controller -- scontrol show nodes
```

### Q5: 如何除錯 Slurm Client 問題？

```bash
# 啟用詳細日誌
kubectl set env deploy/slurm-operator LOG_LEVEL=debug

# 查看 operator 日誌
kubectl logs -l app.kubernetes.io/name=slurm-operator -f

# 檢查 ClientMap 狀態
kubectl exec -it deploy/slurm-operator -- curl localhost:8080/debug/vars
```

---

## Lessons Learned

### 1. Token 生命週期管理

**問題:** Token 過期導致 API 呼叫失敗。

**解決方案:**
- 使用 Token CR 的 `refresh: true` 啟用自動刷新
- 設定合理的 `lifetime`（建議 15-60 分鐘）
- 應用程式端實作 Token 刷新邏輯

### 2. 事件驅動架構

**觀察:** Operator 使用事件驅動架構處理 Slurm 節點狀態變更。

**實作重點:**
- `EventChannel` 傳播 Slurm 節點狀態變化
- `Informer` 快取 Slurm API 回應以減少 API 呼叫
- 僅在狀態實際變更時觸發事件

### 3. 錯誤容忍

**實作模式:**

```go
func tolerateError(err error) bool {
    if err == nil {
        return true
    }
    errText := err.Error()
    // 404 和 204 被視為可接受的結果
    if errText == http.StatusText(http.StatusNotFound) ||
        errText == http.StatusText(http.StatusNoContent) {
        return true
    }
    return false
}
```

### 4. 資源命名慣例

**模式:**
- Deployment: `{cr-name}-{component}` (例: `my-cluster-restapi`)
- Service: `{cr-name}-{component}` (例: `my-cluster-restapi`)
- Secret: `{cr-name}-{type}-{user}` (例: `my-cluster-jwt-admin`)

### 5. 配置即程式碼

**建議:**
- 使用 Helm 管理所有配置
- 將 `values.yaml` 納入版本控制
- 使用 GitOps 工作流程 (ArgoCD/Flux)

---

## 附錄: API 版本對照

| Slurm 版本 | API 版本 | 說明 |
|-----------|---------|------|
| 25.11 | v0.0.41 | 目前支援版本 |
| 24.11 | v0.0.40 | 舊版支援 |
| 24.05 | v0.0.39 | 舊版支援 |

**注意:** API 版本會隨 Slurm 更新而變化，請確認使用正確的版本。

---

*文件版本: 1.0.0*
*最後更新: 2025-12-18*
*適用於: slurm-operator v1.0+*
