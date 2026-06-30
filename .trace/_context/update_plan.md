# 更新計畫
<!-- 產出於 2026-06-30 -->

## 變更摘要
- **Base Commit 範圍**: d5c49df..cfb5029
- **變更檔案數**: 200（不含 .trace/）
- **總檔案數**: ~488
- **變更幅度**: ~41%（中度更新）
- **判定**: 繼續增量更新

## 受影響文件與更新策略

### CODEBASE_MAP.md — 需要更新（Main Agent）
- **原因**: 新增 `internal/controller/slurmclient/eventhandler/` 子目錄及 `internal/controller/slurmclient/utils/` 子目錄；新增 `hack/profile.sh`、`hack/fix-vulns.sh`；新增 `NOTICE`、`THIRD_PARTY_LICENSES`；Helm 新增 vendor template 層
- **影響段落**: 目錄樹、速查表（slurmclient 相關）
- **更新策略**: Main Agent（增補數個子目錄條目）

### DATA_MODEL.md — 需要更新（Sub-Agent）
- **原因**: 多項 API 型別變更：
  - 移除 `ObjectReference`、`JwtSecretKeySelector` 自訂型別
  - `NodeSetSpec.ControllerRef` 從 `ObjectReference` → `corev1.LocalObjectReference`（跨 namespace 禁止）
  - `TokenSpec.JwtKeyRef` / `JwtHs256KeyRef` 改用 `corev1.SecretKeySelector`
  - 移除 `NodeSetSpec.TaintKubeNodes`
  - 新增 `NodeSetSpec.OversubscribeNode`
  - 新增 `ScheduledUpdate` UpdateStrategy 及 `ScheduledUpdateNodeSetStrategy` struct（含 StartTime、Duration、Flags）
  - `NodeSetPartition.Config` 新增 kubebuilder validation pattern
- **影響段落**: base_types 表格、NodeSetSpec 表格、TokenSpec 表格、UpdateStrategy 段落、ER Diagram
- **更新策略**: Sub-Agent（多個段落改寫）

### API_SURFACE.md — 需要更新（Sub-Agent）
- **原因**: 同 DATA_MODEL 的 API 型別變更反映在 CRD 欄位說明；Webhook 新增 namespace scope；Token webhook 跨 namespace 禁止；Partition Config 新增 validation
- **影響段落**: NodeSet CRD spec、Token CRD spec、Webhook 說明段落
- **更新策略**: Sub-Agent

### ARCHITECTURE.md — 需要更新（Sub-Agent）
- **原因**:
  - SlurmClient Controller 新增 RestAPI watch（`eventhandler_restapi.go`）和排序工具（`utils/sort.go`）
  - NodeSet Controller: DaemonSet 同時 scale in/out、重啟未註冊 pods、drain reason 重構
  - 新增 `makePodCordonAndDrain` / `makePodUncordonAndUndrain` pattern
  - `ScheduledUpdate` 新策略影響 NodeSet 更新流程
  - 新增 pprof 端點（可觀察性）
  - 安全性：跨 namespace CR 參照禁止
- **影響段落**: 元件說明（SlurmClient）、NodeSet Controller 流程、架構 Mermaid 圖（SlurmClient→RestAPI 邊）
- **更新策略**: Sub-Agent

### DEV_GUIDE.md — 需要更新（Main Agent）
- **原因**:
  - Helm: PDB for restapi/operator/webhook、namespace-scoped webhook、image.digest pinning、securityContext、nodeSelector、failurePolicy 改 Ignore
  - 新 hack/ 腳本：`profile.sh`（pprof）、`fix-vulns.sh`
  - Go toolchain 升至 1.26.4
  - 預設 image tag 改 `26.05-ubuntu26.04`
  - Google A3 Mega vendor values 支援
  - 移除 TaintKubeNodes 選項
- **影響段落**: 部署設定段落、Helm values 說明、開發工具段落
- **更新策略**: Main Agent（多處小幅修改）

### INDEX.md — 需要更新（Main Agent）
- **原因**: Go toolchain 1.26.4；slurm-client 依賴版本更新；新增 NOTICE/THIRD_PARTY_LICENSES 法律文件；預設映像版本改 26.05-ubuntu26.04
- **影響段落**: 技術棧表格、版本資訊
- **更新策略**: Main Agent（數行更新）

### DISCOVERY_LOG.md — 需要更新（Main Agent）
- **原因**: 追加本次增量更新的發現與觀察
- **更新策略**: Main Agent（追加段落）

## 執行順序
1. CODEBASE_MAP.md
2. INDEX.md
3. DATA_MODEL.md（Sub-Agent）
4. API_SURFACE.md（Sub-Agent）
5. ARCHITECTURE.md（Sub-Agent）
6. DEV_GUIDE.md
7. DISCOVERY_LOG.md
