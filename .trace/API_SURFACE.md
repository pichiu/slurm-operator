# API_SURFACE.md — slurm-operator CRD API 與 Webhook 介面參考

> 本文件因超過 500 行已拆分為兩個部分。

| 部分 | 內容 |
|------|------|
| [API_SURFACE_part1.md](./API_SURFACE_part1.md) | API Group 資訊、CRD 清單、各 CRD 完整 YAML 範例、Webhook 驗證規則一覽 |
| [API_SURFACE_part2.md](./API_SURFACE_part2.md) | Status 欄位說明、HPA / scale subresource、kubectl 操作範例、Annotation 操作、欄位預設值、已棄用欄位 |

---

## 快速摘要

**API Group**: `slinky.slurm.net` / **Version**: `v1beta1`

| Kind | shortName | Slurm 元件 |
|------|-----------|-----------|
| Controller | `slurmctld` | slurmctld |
| NodeSet | `nodesets`, `nss`, `slurmd` | slurmd |
| LoginSet | `loginsets`, `lss`, `sackd` | sackd |
| Accounting | `slurmdbd` | slurmdbd |
| RestApi | `slurmrestd` | slurmrestd |
| Token | `tokens`, `jwt` | JWT 管理 |
