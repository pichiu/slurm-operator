# Trace Metadata

## 分支資訊

- **Base Branch**: `main`
- **Trace Branch**: `claude/trace-incremental-updates-it6xi8`

## 最後 Trace 資訊

- **Base Commit Hash**: `cfb5029f69d5cc0547a03960e05716138fc66a54`
- **日期**: 2026-06-30
- **Trace 類型**: incremental
- **涵蓋範圍**: 全部（~488 個路徑，Infrastructure/Operator 型專案）

## 文件清單

| 文件 | 對應 Base Commit | 最後更新日期 |
|------|-----------------|-------------|
| INDEX.md | cfb5029 | 2026-06-30 |
| ARCHITECTURE.md | cfb5029 | 2026-06-30 |
| DATA_MODEL.md | cfb5029 | 2026-06-30 |
| API_SURFACE.md | cfb5029 | 2026-06-30 |
| API_SURFACE_part1.md | cfb5029 | 2026-06-30 |
| API_SURFACE_part2.md | cfb5029 | 2026-06-30 |
| DEV_GUIDE.md | cfb5029 | 2026-06-30 |
| CODEBASE_MAP.md | cfb5029 | 2026-06-30 |
| DISCOVERY_LOG.md | cfb5029 | 2026-06-30 |

## 中繼 Context 檔案（`_context/`）

> **注意**：這些是 trace pipeline 的 staging artifacts，用於生成主文件的中間產物。
> 內容**未經逐一驗證**，可能含有錯誤（已知案例：`core_logic.md` 的 `sl_ung` 欄位不存在）。
> 保留目的是讓下次增量更新有足夠的 context，**請勿將其視為 ground truth**，主文件才是驗證過的資訊來源。

| 檔案 | 用途 |
|------|------|
| `_context/recon.md` | Stage 1 偵察結果：技術棧、目錄結構、既有文件摘要 |
| `_context/web_findings.md` | Stage 1 Web search 結果摘要 |
| `_context/entry_points.md` | Stage 2：二進位進入點、初始化流程 |
| `_context/data_flow.md` | Stage 2：代表性 use case 的完整 data flow |
| `_context/core_logic.md` | Stage 2：核心領域邏輯、設計模式 |
| `_context/extensions.md` | Stage 2：擴充點、plugin 機制 |
| `_context/integrations.md` | Stage 2：外部整合（Slurm API、cert-manager、Prometheus 等） |
| `_context/configuration.md` | Stage 2：設定載入機制、CRD 欄位說明 |
| `_context/changelog.md` | 增量更新：d5c49df..cfb5029 的變更摘要 |
| `_context/update_plan.md` | 增量更新：受影響文件清單與更新策略 |

## 變更歷程

| 日期 | 類型 | Base Commit 範圍 | 更新的文件 | 摘要 |
|------|------|-----------------|-----------|------|
| 2026-05-19 | full | initial..d5c49df | 全部 | 初次 trace，含 6 個 _context 中繼檔案 |
| 2026-06-30 | incremental | d5c49df..cfb5029 | INDEX, ARCHITECTURE, DATA_MODEL, API_SURFACE（+part1/2）, DEV_GUIDE, CODEBASE_MAP, DISCOVERY_LOG | 85 commits：ScheduledUpdate 策略、跨 namespace 禁止、TaintKubeNodes 移除、OversubscribeNode 新增、SlurmClient eventhandler、pprof、Helm PDB/namespace-scope/image.digest |

## 增量更新說明

下次更新時，執行以下指令找出 source code 的變更範圍：

```bash
# 找出自上次 trace 後新增/修改的檔案
git diff cfb5029f69d5cc0547a03960e05716138fc66a54..origin/main --name-only -- ':!.trace/'

# 根據變更檔案，決定需要刷新哪些文件：
# - api/v1beta1/ 變更 → 刷新 DATA_MODEL.md, API_SURFACE.md
# - internal/controller/ 變更 → 刷新 ARCHITECTURE.md, CODEBASE_MAP.md
# - internal/builder/ 變更 → 刷新 DATA_MODEL.md
# - helm/ 變更 → 刷新 DEV_GUIDE.md
# - docs/ 變更 → 更新 DISCOVERY_LOG.md 的落差分析
```
