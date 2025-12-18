# Slurm

## TL;DR

Slurm 是一套開源、容錯且高度可擴展的叢集管理和作業排程系統，適用於各種規模的 Linux 叢集。它負責資源分配、作業執行監控，以及透過佇列管理待處理工作的競爭。

---

## Translation

> [Slurm] 是一套開源、容錯且高度可擴展的叢集管理和作業排程系統 (Job Scheduling System)，適用於大型和小型 Linux 叢集。Slurm 的運作不需要修改核心 (Kernel)，而且相對獨立。作為叢集工作負載管理器 (Cluster Workload Manager)，Slurm 有三個關鍵功能。
>
> 首先，它為使用者分配資源（運算節點）的獨占和/或非獨占存取權限，讓他們可以在一段時間內執行工作。
>
> 其次，它提供一個框架來啟動、執行和監控工作（通常是平行作業）在分配的節點集合上。
>
> 最後，它透過管理待處理工作的佇列 (Queue) 來仲裁資源競爭。
>
> 可選的外掛程式 (Plugins) 可用於帳務 (Accounting)、進階預約 (Advanced Reservation)、群組排程（平行作業的時間共享）、回填排程 (Backfill Scheduling)、拓撲最佳化資源選擇、依使用者或銀行帳戶的資源限制，以及複雜的多因素作業優先權演算法。

## 架構

![Slurm 架構](https://slurm.schedmd.com/arch.gif)

請參閱 Slurm [架構文件][architecture]以獲取更多資訊。

<!-- Links -->

[architecture]: https://slurm.schedmd.com/quickstart.html#arch
[slurm]: https://slurm.schedmd.com/overview.html

---

## Explanation

### Slurm 是什麼？

Slurm（原名 Simple Linux Utility for Resource Management）是目前最廣泛使用的高效能運算 (HPC) 叢集排程系統之一。它被全球許多超級電腦和研究機構採用。

### 核心元件

| 元件 | 說明 |
|-----|------|
| **slurmctld** | 控制器守護程式 (Controller Daemon)，負責整體叢集管理和作業排程 |
| **slurmd** | 運算節點守護程式，在每個工作節點上執行，負責執行分配的作業 |
| **slurmdbd** | 資料庫守護程式 (Database Daemon)，負責帳務和作業歷史記錄 |
| **slurmrestd** | REST API 守護程式，提供 HTTP 介面存取 Slurm |
| **sackd** | 用戶端認證守護程式 (Client Authentication Daemon) |

### 關鍵概念

- **作業 (Job)**：使用者提交給 Slurm 執行的工作單元
- **分割區 (Partition)**：節點的邏輯群組，類似傳統系統中的「佇列」
- **節點 (Node)**：叢集中的運算資源單位
- **帳戶 (Account)**：用於追蹤和限制資源使用的組織單位

---

## Practical Example

### 常用 Slurm 指令

```bash
# 查看叢集狀態和所有節點資訊
# sinfo: Slurm 資訊指令，顯示分割區和節點狀態
sinfo

# 提交一個批次作業
# sbatch: 提交批次腳本到 Slurm 佇列
# --wrap: 將指令包裝成簡單的批次腳本
sbatch --wrap="hostname"

# 查看佇列中的作業
# squeue: 顯示排隊中和執行中的作業
squeue

# 互動式執行指令
# srun: 即時執行指令，會等待資源分配
srun hostname

# 查看作業帳務資訊
# sacct: 顯示已完成作業的帳務資訊
sacct

# 取消作業
# scancel: 取消指定的作業
scancel <job_id>

# 查看特定節點的詳細資訊
# scontrol: Slurm 控制指令，用於查看和修改 Slurm 配置
scontrol show node <node_name>
```

### 簡單的批次腳本範例

```bash
#!/bin/bash
#SBATCH --job-name=my_job       # 作業名稱
#SBATCH --nodes=2               # 請求 2 個節點
#SBATCH --ntasks=4              # 總共 4 個任務
#SBATCH --time=00:30:00         # 最大執行時間 30 分鐘
#SBATCH --partition=slinky      # 指定分割區

# 實際執行的指令
srun my_program
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| 作業一直處於 PENDING 狀態 | 使用 `squeue -j <job_id> --reason` 查看原因 |
| 節點顯示 DOWN 狀態 | 檢查節點上的 slurmd 服務是否正常運行 |
| 作業執行失敗 | 檢查輸出檔案中的錯誤訊息 |
| 資源請求過大 | 調整 `--nodes`、`--ntasks` 或 `--mem` 參數 |

### 小技巧

1. **合理估計資源**：避免請求過多資源導致等待時間過長
2. **設定時間限制**：使用 `--time` 避免作業無限期執行
3. **使用帳務功能**：追蹤資源使用情況有助於最佳化
4. **善用分割區**：不同分割區可能有不同的資源配額和優先權

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `sinfo` | 查看叢集和節點狀態 |
| `squeue` | 查看作業佇列 |
| `sbatch <script>` | 提交批次作業 |
| `srun <command>` | 互動式執行指令 |
| `scancel <job_id>` | 取消作業 |
| `sacct` | 查看作業帳務 |
| `scontrol show job <job_id>` | 查看作業詳細資訊 |
| `scontrol show node <node>` | 查看節點詳細資訊 |
