# Trace Metadata

## 分支資訊

- **Base Branch**: `claude/generate-project-docs-LmQlv`
- **Trace Branch**: `trace/docs`

## 最後 Trace 資訊

- **Base Commit Hash**: `d5c49dfd32a12b5215acb5dab52a1fda126e91f5`
- **日期**: 2026-05-19
- **Trace 類型**: full
- **涵蓋範圍**: 全部（609 個路徑，Infrastructure/Operator 型專案）

## 文件清單

| 文件 | 對應 Base Commit | 最後更新日期 |
|------|-----------------|-------------|
| INDEX.md | d5c49df | 2026-05-19 |
| ARCHITECTURE.md | d5c49df | 2026-05-19 |
| DATA_MODEL.md | d5c49df | 2026-05-19 |
| API_SURFACE.md | d5c49df | 2026-05-19 |
| DEV_GUIDE.md | d5c49df | 2026-05-19 |
| CODEBASE_MAP.md | d5c49df | 2026-05-19 |
| DISCOVERY_LOG.md | d5c49df | 2026-05-19 |

## 中繼 Context 檔案（`_context/`）

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

## 變更歷程

| 日期 | 類型 | Base Commit 範圍 | 更新的文件 | 摘要 |
|------|------|-----------------|-----------|------|
| 2026-05-19 | full | initial..d5c49df | 全部 | 初次 trace，含 6 個 _context 中繼檔案 |

## 增量更新說明

下次更新時，執行以下指令找出 source code 的變更範圍：

```bash
# 找出自上次 trace 後新增/修改的檔案
git diff d5c49dfd32a12b5215acb5dab52a1fda126e91f5..HEAD --name-only

# 根據變更檔案，決定需要刷新哪些文件：
# - api/v1beta1/ 變更 → 刷新 DATA_MODEL.md, API_SURFACE.md
# - internal/controller/ 變更 → 刷新 ARCHITECTURE.md, CODEBASE_MAP.md
# - internal/builder/ 變更 → 刷新 DATA_MODEL.md
# - helm/ 變更 → 刷新 DEV_GUIDE.md
# - docs/ 變更 → 更新 DISCOVERY_LOG.md 的落差分析
```
