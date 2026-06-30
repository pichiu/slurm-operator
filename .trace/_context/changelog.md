# Changelog — Incremental Update
<!-- 更新於 2026-06-30，commit 範圍: d5c49df..cfb5029 -->

## Base Commit 範圍
- **From**: `d5c49dfd32a12b5215acb5dab52a1fda126e91f5`（上次 trace）
- **To**: `cfb5029f69d5cc0547a03960e05716138fc66a54`（origin/main HEAD）
- **Base Branch**: `main`（原 trace 所用 `claude/generate-project-docs-LmQlv` 已合入 main）

## 統計摘要
- 變更檔案數：200（不含 .trace/）
- 總檔案數：~488
- 變更幅度：~41%（中度更新，繼續增量）
- Commit 數量（non-merge）：85

## Commit 摘要（依主題分組）

### API / Data Model 變更
- `feat: remove unused JwtSecretKeySelector` — 移除自訂 `JwtSecretKeySelector`，改用 `corev1.SecretKeySelector`
- `fix: disallow cross namespace Slinky CR references` — 禁止跨 namespace 參照，`ObjectReference` → `corev1.LocalObjectReference`
- `fix(nodeset): remove TaintKubeNodes feature` — 移除已棄用的 `TaintKubeNodes` 欄位
- `feat(nodeset): adds Scheduled UpdateStrategy` — 新增 `ScheduledUpdate` 策略，含 `ScheduledUpdateNodeSetStrategy` struct
- `feat: add NodeSet oversubscribeNode option` — 新增 `OversubscribeNode` 欄位
- `fix: prevent slurm.conf injection via nodeset partition config` — 新增 kubebuilder validation pattern

### Controller / 核心邏輯
- `feat(nodeset): handle DaemonSet scale in and out at once` — DaemonSet 同時 scale in/out
- `feat(nodeset): restart healthy but unregistered NodeSet pods` — 重啟健康但未註冊的 pods
- `fix(slurmclient): deterministic restapi selection for slurmclient` — 排序穩定化
- `fix(restapi): add RestAPI watch to SlurmClient Controller` — 新增 RestAPI event handler
- `fix(restapi): delete slurmclient when restapi not found` — 新增刪除邏輯
- `feat: update doPodScale drain reason` — Drain reason 改寫
- `refactor: condense logic into makePodCordonAndDrain` / `makePodUncordonAndUndrain` — 重構 Pod 管理
- `fix: preserve previous reason if already drained` — Bug fix
- `feat: add StatusPatchObject() utility func` — 新增 patch 工具函式
- `fix: use StatusPatchObject() func to stop empty patches` — 避免空 patch 請求
- `perf: constrain list to namespace` — 效能：list 限縮到 namespace

### Builder 變更
- `feat(builder): remove non-required gres.conf` — 移除非必要的 gres.conf
- `feat(builder): prune all non-required Slurm config options` — 精簡 Slurm config
- `fix(builder): honor service.nodePort for restapi, controller, accounting` — nodePort 修正
- `fix(login): raise RSA SSH host key default to 4096 bits` — 安全性提升

### Helm / 部署
- `feat(helm): gate operator/webhook PDBs on HA and expose maxUnavailable` — PDB 設定
- `feat(charts): add PodDisruptionBudget for restapi` — RestAPI PDB
- `feat(helm): add namespaced-scope watching for webhook` — Webhook namespace scope
- `feat(charts): expose nodeSelector for operator` — nodeSelector 支援
- `feat(charts): add image.digest option` — image digest pinning
- `feat(charts): add {node,login}setDefaults object variable` — 預設值抽象化
- `feat(charts): add basic support for a3mega` — Google A3 Mega vendor 支援
- `feat(charts): remove default Partition / NodeSet` — 移除 chart 預設值
- `fix(helm): parametrize webhook failurePolicy/matchPolicy` — Webhook 預設改 Ignore
- `bfb8d16 feat(helm): add securityContext and podSecurityContext for operator`
- `chore: update default image version to 26.05-ubuntu26.04`

### 可觀察性
- `feat: enable live Go profiling of Slurm-operator using pprof` — pprof 端點
- `feat: enable flame graphs for key functions`

### 安全性 / 依賴
- `fix(deps): bump Go toolchain to 1.26.4`
- `chore(deps): bump slurm-client to latest`
- `chore(legal): add NOTICE and THIRD_PARTY_LICENSES generation`
- `a730aaa chore: fix vulns identified by govulncheck`

## 新增檔案（有架構意義）
- `internal/controller/slurmclient/eventhandler/eventhandler_restapi.go` — RestAPI 事件監聽
- `internal/controller/slurmclient/utils/sort.go` — 排序工具
- `hack/profile.sh` — pprof profiling 腳本
- `hack/fix-vulns.sh` — 漏洞修復腳本
- `helm/slurm/templates/_vendor.tpl` — vendor 抽象層
- `helm/slurm/templates/vendor/google/a3mega/` — Google A3 Mega 支援
- `helm/slurm/templates/restapi/restapi-pdb.yaml` — RestAPI PDB
- `NOTICE`, `THIRD_PARTY_LICENSES` — 法律文件
