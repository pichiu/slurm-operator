# Web Findings — slurm-operator

## 搜尋摘要

執行了 2 次 web search，以下為關鍵發現。

---

## 搜尋 1：官方文件與 GitHub

**查詢**: `Slinky slurm-operator SchedMD Kubernetes operator site:github.com OR site:slinky.schedmd.com`

### 關鍵連結

| 連結 | 說明 |
|------|------|
| [官方文件首頁](https://slinky.schedmd.com/projects/slurm-operator) | Slinky slurm-operator 文件 |
| [GitHub Repo](https://github.com/SlinkyProject/slurm-operator) | 原始碼倉庫 |
| [架構文件 (release-1.0)](https://slinky.schedmd.com/projects/slurm-operator/en/release-1.0/concepts/architecture.html) | 官方架構說明 |
| [Slinky 總覽](https://slinky.schedmd.com/docs/) | Slinky 平台總覽 |
| [Workload Isolation (release-0.4)](https://slinky.schedmd.com/projects/slurm-operator/en/release-0.4/workload-isolation.html) | 工作負載隔離文件 |

### 關鍵 Takeaway
- Slinky 由 **SchedMD**（Slurm 的主要開發商）開發，並由 **NVIDIA** 支援
- 官方文件網站：https://slinky.schedmd.com/
- 核心理念：讓 Slurm 和 Kubernetes 並存，充分發揮各自優勢

---

## 搜尋 2：架構設計決策

**查詢**: `Slinky slurm-operator architecture design decisions CRD controllers 2025`

### 關鍵連結

| 連結 | 說明 |
|------|------|
| [架構文件 (release-1.0)](https://slinky.schedmd.com/projects/slurm-operator/en/release-1.0/concepts/architecture.html) | 最新架構說明 |
| [Crusoe AI 案例研究](https://www.crusoe.ai/resources/blog/slurm-on-crusoe-managed-kubernetes-how-we-built-managed-gpu-training-infrastructure) | Slurm on Kubernetes GPU 訓練基礎設施 |
| [Red Hat OpenShift 整合](https://developers.redhat.com/articles/2026/03/10/how-run-slurm-workloads-openshift-slinky-operator) | OpenShift 上運行 Slurm 工作負載 |
| [AWS EKS 整合](https://aws.amazon.com/blogs/containers/running-slurm-on-amazon-eks-with-slinky/) | Amazon EKS 上的 Slurm |
| [Slurm 官方 Slinky 頁面](https://slurm.schedmd.com/slinky.html) | SchedMD 官方 Slinky 說明 |

### 關鍵 Takeaway

#### v1.0 API 重大變更（2025年底）
- v1alpha1 CRD 替換為 v1beta1，支援 **CRD conversion**
- v1.Y 升級流程：升級 CRD chart → 升級 operator chart，**不需要 uninstall 或刪除 CRD**
- 現有 Slurm 叢集在升級過程中**不中斷服務**

#### 架構設計重點
- Operator 不只看 Kubernetes API，也要看 **Slurm API**（slurmrestd），這是與標準 Kubernetes operator 的關鍵差異
- Slurm HA 透過 Kubernetes pod 重啟實現（比傳統 Slurm HA 更快）
- 採用 **auth/slurm** 取代 MUNGE，避免每個 pod 都需要 MUNGE sidecar
- **Dynamic Nodes** 讓 slurmd 容器無需在 `slurm.conf` 中預先定義

#### 主要使用場景
- GPU 訓練叢集（AI/ML workload）— Crusoe AI 案例
- 混合 HPC/Kubernetes 環境
- 需要 HPA 自動縮放的 HPC workload
- OpenShift / AWS EKS / 各大雲端平台

---

## 社群資源

- **GitHub Issues/PR**: https://github.com/SlinkyProject/slurm-operator/issues
- **SchedMD 官方 Issue Tracker**: https://support.schedmd.com/
- **聯絡 SchedMD**: https://www.schedmd.com/slurm-resources/contact-schedmd/
- **文件網站**: https://slinky.schedmd.com/

---

## 相關比較

Slinky slurm-operator 的定位與以下項目不同：
- **Volcano / Kueue** — Kubernetes 原生的 batch 排程器，不提供 Slurm 相容性
- **Open HPC** — 傳統 HPC 軟體棧，非 Kubernetes 原生
- **Slurm-on-EKS (社群版本)** — 非官方解法，Slinky 是 SchedMD 官方版本

SchedMD 在 Slurm 官網 (slurm.schedmd.com/slinky.html) 直接推薦 Slinky 作為 Slurm on Kubernetes 的官方解法。
