# Slinky 發行版本 (Slinky Release Versioning)

## TL;DR

Slinky 使用語意化版本 (Semantic Versioning) 管理版本號 X.Y.Z。所有 Slinky 元件（slurm-operator、slurm-bridge、slurm-client 及其 Helm charts）同步發行。容器映像則依據其包含的應用程式（如 Slurm）獨立版本管理。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [Slinky 發行版本](#slinky-發行版本-slinky-release-versioning)
  - [目錄](#目錄)
  - [發行版本](#發行版本)
    - [版本架構](#版本架構)
    - [主要版本](#主要版本)

<!-- mdformat-toc end -->

## 發行版本

**X.Y.Z** 指的是 Slinky 發行的版本（git tag）。（**X** 是主要版本 (Major)，**Y** 是次要版本 (Minor)，**Z** 是修補版本 (Patch)，遵循[語意化版本][semver]術語。）

所有 Slinky 元件（例如 [slurm-operator]、[slurm-bridge]、[slurm-client] 及其 helm charts 和映像）都同步進行版本管理。由於相依鏈和 CI 的關係，它們被標記和發行的實際日期可能會有偏差。

衍生自 Slinky [containers] 的映像是獨立版本管理和發行的。這些容器映像根據它們包含的應用程式進行版本管理。因此 Slurm 守護程式映像的版本與 Slurm 本身保持一致。

### 版本架構

- `X.Y.Z-rcW`（分支：`release-X.Y`）
  - 當 `main` 對 `X.Y` 功能完整時，我們可能會在預期的 `X.Y.0` 日期之前切出 `release-X.Y` 分支，並只 cherry-pick 對 `X.Y` 至關重要的 PR。
  - 此切割會被標記為 `X.Y.0-rc0`，而 `main` 會提升到 `X.(Y+1).0-rc0`。
  - 如果我們對 `X.Y.0-rc0` 不滿意，會根據需要發行其他 rc 版本（`X.Y.0-rcW`，其中 `W > 0`）。
- `X.Y.0`（分支：`release-X.Y`）
  - 最終發行版本，從 `release-X.Y` 分支切出。
  - `X.Y.0` 發生在 `X.(Y-1).0` 之後。
- `X.Y.Z`，`Z > 0`（分支：`release-X.Y`）
  - 修補版本會在我們根據需要將提交 cherry-pick 到 `release-X.Y` 分支時發行。
  - `X.Y.Z` 直接從 `release-X.Y` 分支切出。

### 主要版本

主要版本沒有強制的時間表，目前沒有發行 `v2.0.0` 的標準。到目前為止，我們沒有對任何類型的不相容變更（例如元件旗標變更）嚴格套用語意化版本解釋。

<!-- Links -->

[containers]: https://github.com/SlinkyProject/containers
[semver]: https://semver.org/
[slurm-bridge]: https://github.com/SlinkyProject/slurm-bridge
[slurm-client]: https://github.com/SlinkyProject/slurm-client
[slurm-operator]: https://github.com/SlinkyProject/slurm-operator

---

## Explanation

### 語意化版本 (Semantic Versioning)

語意化版本使用 `X.Y.Z` 格式：

| 部分 | 名稱 | 何時增加 |
|-----|------|---------|
| **X** | 主要版本 (Major) | 不相容的 API 變更 |
| **Y** | 次要版本 (Minor) | 新增向後相容的功能 |
| **Z** | 修補版本 (Patch) | 向後相容的錯誤修正 |

### 版本發行流程

```
main ────────────────────────────────────────────────►
        │
        │  切出 release 分支
        ▼
release-1.0 ─► rc0 ─► rc1 ─► 1.0.0 ─► 1.0.1 ─► 1.0.2
        │
        │  main 提升到下一個版本
        ▼
main (1.1.0-rc0) ───────────────────────────────────►
```

### 元件版本對照

| 元件 | 版本來源 |
|-----|---------|
| slurm-operator | Slinky 版本 (X.Y.Z) |
| slurm-operator-crds | Slinky 版本 (X.Y.Z) |
| slurm helm chart | Slinky 版本 (X.Y.Z) |
| slurmd 容器映像 | Slurm 版本 (如 25.11) |
| slurmctld 容器映像 | Slurm 版本 (如 25.11) |

---

## Practical Example

### 查看目前版本

```bash
# 查看 Helm chart 版本
helm list -n slinky
helm list -n slurm

# 查看 Operator 映像版本
kubectl get deployment -n slinky slurm-operator -o jsonpath='{.spec.template.spec.containers[0].image}'

# 查看 Slurm 版本
kubectl exec -n slurm statefulset/slurm-controller -- scontrol version
```

### 升級 Slinky

```bash
# 查看可用版本
helm search repo oci://ghcr.io/slinkyproject/charts/slurm-operator --versions

# 升級到特定版本
helm upgrade slurm-operator oci://ghcr.io/slinkyproject/charts/slurm-operator \
  --version 1.1.0 \
  --namespace=slinky

# 升級 CRDs（如果需要）
helm upgrade slurm-operator-crds oci://ghcr.io/slinkyproject/charts/slurm-operator-crds \
  --version 1.1.0

# 升級 Slurm 叢集
helm upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --version 1.1.0 \
  --namespace=slurm
```

### 查看變更日誌

```bash
# 從 GitHub 取得 release notes
# https://github.com/SlinkyProject/slurm-operator/releases

# 比較版本差異
helm diff upgrade slurm oci://ghcr.io/slinkyproject/charts/slurm \
  --version 1.1.0 \
  --namespace=slurm
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| 混淆 Slinky 和 Slurm 版本 | Slinky 是 Operator 版本，Slurm 是工作負載管理器版本 |
| 升級順序錯誤 | 先升級 CRDs，再升級 Operator，最後升級 Slurm 叢集 |
| 忽略 rc 版本 | rc 版本用於測試，正式環境應使用穩定版本 |
| 跨主要版本升級 | 主要版本可能有不相容變更，需查看升級指南 |

### 版本選擇建議

| 環境 | 建議版本 |
|-----|---------|
| 正式環境 | 最新穩定版 (X.Y.Z) |
| 測試環境 | 可使用 rc 版本測試新功能 |
| 開發環境 | 可使用 main 分支最新版 |

### 小技巧

1. **訂閱 Release**：在 GitHub 上 Watch Releases 以獲取通知
2. **測試升級**：先在測試環境驗證升級流程
3. **備份配置**：升級前備份 values.yaml 和重要配置
4. **查看相依性**：確認新版本對 Kubernetes 版本的要求

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `helm search repo <chart> --versions` | 列出所有可用版本 |
| `helm list -n <namespace>` | 查看已安裝的版本 |
| `helm upgrade <release> <chart> --version X.Y.Z` | 升級到指定版本 |
| `helm history <release> -n <namespace>` | 查看升級歷史 |
| `helm rollback <release> <revision> -n <namespace>` | 回滾到指定版本 |
