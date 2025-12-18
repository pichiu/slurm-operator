# Development Guide - slurm-operator

> 生成日期：2025-12-18

## 1. 先決條件

### 1.1 必要軟體

| 軟體 | 版本 | 說明 |
|------|------|------|
| Go | 1.25.0+ | 主要開發語言 |
| Docker | 24.0+ | 容器構建 |
| kubectl | 1.29+ | Kubernetes CLI |
| Helm | 3.19+ | Chart 管理 |
| Kind | 0.20+ | 本地 K8s 叢集 |
| Make | 4.0+ | 構建工具 |

### 1.2 可選工具

```bash
# 安裝開發工具
make install-dev

# 這會安裝：
# - delve (調試器)
# - kind (本地 K8s)
# - cloud-provider-kind
```

## 2. 環境設定

### 2.1 克隆專案

```bash
git clone https://github.com/SlinkyProject/slurm-operator.git
cd slurm-operator
```

### 2.2 安裝依賴

```bash
# 下載 Go 依賴
go mod download

# 安裝構建工具
make controller-gen
make envtest
```

### 2.3 創建本地叢集

```bash
# 使用 Kind 創建叢集
kind create cluster --config hack/kind.yaml
```

## 3. 構建命令

### 3.1 基本構建

```bash
# 構建所有 (映像 + charts)
make all

# 只構建映像
make build-images

# 只構建 charts
make build-chart
```

### 3.2 生成代碼

```bash
# 生成 CRD 清單
make manifests

# 生成 DeepCopy 方法
make generate

# 生成文檔
make generate-docs
```

### 3.3 代碼品質

```bash
# 格式化代碼
make fmt

# 整理模組
make tidy

# 運行 linter
make golangci-lint

# 漏洞檢查
make govulncheck
```

## 4. 測試

### 4.1 單元測試

```bash
# 運行所有測試 (需要 66% 覆蓋率)
make test

# 測試結果會生成：
# - cover.out (覆蓋率報告)
# - cover.html (HTML 報告)
```

### 4.2 端對端測試

```bash
# 運行 E2E 測試 (15 分鐘超時)
make test-e2e
```

### 4.3 測試框架

專案使用 Ginkgo/Gomega BDD 測試框架：

```go
var _ = Describe("Controller", func() {
    Context("When creating a Controller", func() {
        It("Should create a StatefulSet", func() {
            // 測試邏輯
            Expect(err).NotTo(HaveOccurred())
        })
    })
})
```

## 5. 本地開發

### 5.1 運行 Operator

```bash
# 安裝 CRDs
kubectl apply -f config/crd/bases/

# 運行 operator (本地)
go run ./cmd/manager/main.go
```

### 5.2 運行 Webhook

```bash
# 運行 webhook (本地)
go run ./cmd/webhook/main.go
```

### 5.3 調試

```bash
# 使用 Delve 調試
dlv debug ./cmd/manager/main.go
```

## 6. Helm Chart 開發

### 6.1 驗證 Charts

```bash
# Lint 所有 charts
make helm-lint

# 驗證 charts
make helm-validate

# 更新依賴
make helm-dependency-update
```

### 6.2 生成 Chart 文檔

```bash
# 生成 README
make helm-docs
```

### 6.3 開發配置

```bash
# 創建開發配置 (values-dev.yaml)
make values-dev

# 這會複製 values.yaml 到 values-dev.yaml 供本地測試
```

## 7. 部署到本地叢集

### 7.1 安裝 cert-manager

```bash
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --set 'crds.enabled=true' \
  --namespace cert-manager --create-namespace
```

### 7.2 安裝 Operator

```bash
# 構建並推送映像
make push-images

# 安裝 CRDs
helm install slurm-operator-crds ./helm/slurm-operator-crds

# 安裝 Operator
helm install slurm-operator ./helm/slurm-operator \
  --namespace slinky --create-namespace
```

### 7.3 安裝 Slurm 叢集

```bash
helm install slurm ./helm/slurm \
  --namespace slurm --create-namespace \
  -f helm/slurm/values-dev.yaml
```

## 8. 專案結構

### 8.1 關鍵檔案位置

| 用途 | 路徑 |
|------|------|
| CRD 類型 | `api/v1beta1/*_types.go` |
| 控制器 | `internal/controller/*/` |
| Webhook | `internal/webhook/*.go` |
| Builder | `internal/builder/*.go` |
| 工具函數 | `internal/utils/*/` |
| Helm Charts | `helm/*/` |
| 測試 | `*_test.go`, `test/e2e/` |

### 8.2 新增 CRD

1. 定義類型 (`api/v1beta1/xxx_types.go`)
2. 生成代碼 (`make generate manifests`)
3. 實現控制器 (`internal/controller/xxx/`)
4. 實現 webhook (`internal/webhook/xxx_webhook.go`)
5. 註冊控制器 (`cmd/manager/main.go`)
6. 更新 Helm charts

## 9. 代碼規範

### 9.1 命名慣例

- **套件名**: 小寫單詞 (`nodeset`, `eventhandler`)
- **檔案名**: 小寫加底線 (`nodeset_controller.go`)
- **類型名**: 駝峰式 (`NodeSetReconciler`)
- **常量**: 全大寫或駝峰式

### 9.2 錯誤處理

```go
if err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to get NodeSet: %w", err)
}
```

### 9.3 日誌

```go
log := ctrl.LoggerFrom(ctx)
log.Info("Reconciling NodeSet", "name", req.Name)
log.Error(err, "Failed to sync pods")
```

## 10. CI/CD

### 10.1 Pre-commit Hooks

```bash
# 安裝 pre-commit
pre-commit install

# 手動運行
pre-commit run --all-files
```

### 10.2 版本管理

```bash
# 檢查版本
make version

# 更新所有版本號
make version-match
```

## 11. 常見任務

### 11.1 添加新依賴

```bash
go get github.com/example/package@v1.0.0
make tidy
```

### 11.2 更新 CRD

```bash
# 編輯 api/v1beta1/*_types.go
# 然後運行：
make manifests generate
```

### 11.3 清理

```bash
make clean
```

## 12. 疑難排解

### 12.1 構建錯誤

```bash
# 清理並重新構建
make clean
go mod download
make all
```

### 12.2 測試失敗

```bash
# 更新 envtest 二進制
make envtest
```

### 12.3 Helm 錯誤

```bash
# 更新依賴
make helm-dependency-update
```

## 13. 資源連結

- [Kubebuilder 文檔](https://book.kubebuilder.io/)
- [controller-runtime 文檔](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Operator SDK](https://sdk.operatorframework.io/)
- [Slurm 文檔](https://slurm.schedmd.com/)
