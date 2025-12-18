# 覆寫映像設定檔 (Overriding Image Configuration Files)

## TL;DR

Slinky Helm charts 的配置透過 `values.yaml` 完成，但有時需要覆寫映像中的設定檔。本指南說明如何使用 Kubernetes 的 Volume、VolumeMount 和 ConfigMap 來覆寫容器中的設定檔，以 `enroot.conf` 為例。

---

## Translation

## 目錄

<!-- mdformat-toc start --slug=github --no-anchors --maxlevel=6 --minlevel=1 -->

- [覆寫映像設定檔](#覆寫映像設定檔-overriding-image-configuration-files)
  - [目錄](#目錄)
  - [概述](#概述)
  - [先決條件](#先決條件)
  - [使用 Volume 和 ConfigMap 覆寫設定檔](#使用-volume-和-configmap-覆寫設定檔)

<!-- mdformat-toc end -->

## 概述

Slinky Helm charts 的配置透過 `values.yaml` 檔案完成，位於 `/helm/chart-name/` 中。然而，在某些環境中，可能需要覆寫 Helm chart 未控制的檔案，這些檔案存在於 [Slinky 映像][Slinky images]中。這可以使用 [Volume]、VolumeMount 和 [ConfigMap] 來完成。

## 先決條件

本指南假設使用者可以存取執行 slurm-operator 的功能性 Kubernetes 叢集。請參閱[快速入門指南][quickstart guide]以了解如何在 Kubernetes 叢集上設定 slurm-operator。

## 使用 Volume 和 ConfigMap 覆寫設定檔

在 [Slinky 映像][Slinky images]中提供的少數設定檔之一，預設無法透過 Helm charts 配置的是 `enroot.conf`。因此，本指南將使用 `enroot.conf` 作為如何使用 [Volume] 和 [ConfigMap] 覆寫檔案的範例。

首先，需要建立一個 ConfigMap，其中包含將用於覆寫容器中現有檔案的 `enroot.conf` 內容。以下是這種 ConfigMap 的範例，在本演示中命名為 `enroot-config.yaml`：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: enroot-config
  namespace: slurm
data:
  enroot: |
    ENROOT_RUNTIME_PATH         /run/enroot/${UID}/run
    ENROOT_CONFIG_PATH          /run/enroot/${UID}/config
    ENROOT_CACHE_PATH           /run/enroot/${UID}/cache
    ENROOT_DATA_PATH            /run/enroot/${UID}/data
    ENROOT_TEMP_PATH            /run/${UID}/tmp
```

將此 ConfigMap 套用到您的叢集：

```bash
kubectl apply -f enroot-config.yaml
```

建立 ConfigMap 後，必須使用 [Volume] 來覆寫 `/etc/enroot/enroot.conf` 的預設內容。這需要在每個 NodeSet 的基礎上完成，使用 `helm/slurm/values.yaml` 中的 volumes 和 volume mount 變數：

```yaml
nodesets:
  slinky:
    ...
    # -- 要使用的 Volume 清單。
    # 參考：https://kubernetes.io/docs/concepts/storage/volumes/
    volumes: []
      # - name: nfs-home
      #   nfs:
      #     server: nfs-server.example.com
    #     path: /exports/home
    # -- 要使用的 Volume Mount 清單。
    # 參考：https://kubernetes.io/docs/concepts/storage/volumes/
    volumeMounts: []
      # - name: nfs-home
      #   mountPath: /home
```

修改 NodeSet 規格的這個區段，以參考上面建立的 ConfigMap：

```yaml
nodesets:
  slinky:
    ...
    # -- 要使用的 Volume 清單。
    # 參考：https://kubernetes.io/docs/concepts/storage/volumes/
    volumes:
    - name: enroot-config
      configMap:
        name: enroot-config
        items:
          - key: enroot
            path: "enroot.conf"

    # -- 要使用的 Volume Mount 清單。
    # 參考：https://kubernetes.io/docs/concepts/storage/volumes/
    volumeMounts:
    - name: enroot-config
      mountPath: "/etc/enroot/enroot.conf"
      subPath: "enroot.conf"
```

此時，可以安裝 Helm chart。修改過的 NodeSet 應該會顯示 NodeSet Spec 包含上面指定的 Volumes 和 VolumeMounts：

```yaml
kubectl describe NodeSet -n slurm
Name:         slurm-compute-slinky
Namespace:    slurm
...
Spec:
  ...
  Template:
    Container:
      ...
      Volume Mounts:
        Mount Path:  /etc/enroot/enroot.conf
        Name:        enroot-config
        Sub Path:    enroot.conf
    ...
    Volumes:
      Config Map:
        Items:
          Key:   enroot
          Path:  enroot.conf
        Name:    enroot-config
      Name:      enroot-config

```

在 `slurm-compute-` Pod 的 `slurmd` 容器中，指定 Mount Path 的檔案應該被上述 ConfigMap 的內容取代：

```bash
kubectl exec -it -n slurm slurm-compute-slinky-0 -- cat /etc/enroot/enroot.conf
ENROOT_RUNTIME_PATH         /run/enroot/${UID}/run
ENROOT_CONFIG_PATH          /run/enroot/${UID}/config
ENROOT_CACHE_PATH           /run/enroot/${UID}/cache
ENROOT_DATA_PATH            /run/enroot/${UID}/data
ENROOT_TEMP_PATH            /run/${UID}/tmp
```

`/etc/enroot` 目錄中的其他檔案應保持原樣：

```bash
kubectl exec -it -n slurm slurm-compute-slinky-0 -- ls /etc/enroot
enroot.conf  enroot.conf.d  environ.d  hooks.d	mounts.d
```

如果目標是用自訂配置覆寫 `/etc/enroot` 的整個內容，也可以使用這種方法。需要從 Volume Mount 中移除 `SubPath` 指令，從 Volume Mount 的 mount path 中移除檔案名 `enroot.conf`，並且 Volume 需要為每個將從 ConfigMap 衍生的檔案新增一個 item。

<!-- Links -->

[configmap]: https://kubernetes.io/docs/concepts/configuration/configmap/
[quickstart guide]: ../installation.md
[slinky images]: https://github.com/SlinkyProject/containers/tree/main
[volume]: https://kubernetes.io/docs/concepts/storage/volumes/

---

## Explanation

### 什麼是 ConfigMap？

ConfigMap 是 Kubernetes 用於儲存非機密配置資料的物件。它可以：
- 儲存設定檔內容
- 儲存環境變數
- 儲存命令列參數

### Volume 和 VolumeMount 的關係

| 概念 | 說明 |
|-----|------|
| **Volume** | 定義可以掛載到 Pod 的儲存來源 |
| **VolumeMount** | 指定 Volume 在容器內的掛載路徑 |
| **subPath** | 只掛載 Volume 中的特定檔案或子目錄 |

### 為何使用 subPath？

使用 `subPath` 可以：
- 只覆寫單一檔案而非整個目錄
- 保留目錄中的其他檔案
- 避免意外刪除重要檔案

---

## Practical Example

### 覆寫 Slurm 設定檔範例

```yaml
# slurm-config.yaml
# 自訂 plugstack.conf
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-plugstack
  namespace: slurm
data:
  plugstack.conf: |
    # 自訂 SPANK 外掛配置
    include /usr/share/pyxis/*
    optional /etc/slurm/plugstack.conf.d/*
```

```bash
# 套用 ConfigMap
kubectl apply -f slurm-config.yaml
```

### 在 values.yaml 中配置

```yaml
# values-custom-config.yaml
nodesets:
  slinky:
    enabled: true
    replicas: 2

    # 定義 Volume，來源為 ConfigMap
    volumes:
    - name: custom-plugstack
      configMap:
        name: custom-plugstack
        items:
          - key: plugstack.conf
            path: "plugstack.conf"

    # 定義 VolumeMount，指定掛載路徑
    volumeMounts:
    - name: custom-plugstack
      mountPath: "/etc/slurm/plugstack.conf"
      subPath: "plugstack.conf"
```

### 驗證配置已套用

```bash
# 安裝/升級 Helm chart
helm upgrade --install slurm oci://ghcr.io/slinkyproject/charts/slurm \
  -f values-custom-config.yaml \
  --namespace=slurm --create-namespace

# 驗證檔案已被覆寫
kubectl exec -n slurm slurm-worker-slinky-0 -- cat /etc/slurm/plugstack.conf

# 查看 NodeSet 的 Volume 配置
kubectl describe nodeset slurm-worker-slinky -n slurm | grep -A 10 "Volumes"
```

### 覆寫整個目錄

```yaml
# 覆寫整個 /etc/enroot 目錄
volumes:
- name: enroot-config
  configMap:
    name: enroot-config
    items:
      - key: enroot.conf
        path: "enroot.conf"
      - key: environ
        path: "environ.d/00-custom.env"

volumeMounts:
- name: enroot-config
  mountPath: "/etc/enroot"
  # 注意：移除 subPath 會覆寫整個目錄
```

---

## Common Mistakes & Tips

| 常見錯誤 | 解決方法 |
|---------|---------|
| ConfigMap 不存在 | 先用 `kubectl apply` 建立 ConfigMap |
| 掛載後目錄為空 | 使用 `subPath` 只覆寫特定檔案 |
| 權限問題 | 檢查檔案權限和所有權 |
| 修改後未生效 | 可能需要重啟 Pod 或重新部署 |

### ConfigMap 更新注意事項

- ConfigMap 更新後，已掛載的檔案會自動更新（有延遲）
- 使用 `subPath` 掛載的檔案**不會**自動更新
- 更新 ConfigMap 後可能需要手動重啟 Pod

### 小技巧

1. **命名空間一致**：確保 ConfigMap 和 Pod 在同一命名空間
2. **驗證語法**：套用前先驗證 YAML 語法
3. **備份原檔**：覆寫前先查看原始檔案內容
4. **使用 Helm hooks**：可以用 Helm hooks 自動建立 ConfigMap

---

## Quick Reference

| 指令 | 說明 |
|-----|------|
| `kubectl create configmap <name> --from-file=<file>` | 從檔案建立 ConfigMap |
| `kubectl apply -f configmap.yaml` | 套用 ConfigMap |
| `kubectl get configmap -n slurm` | 列出 ConfigMap |
| `kubectl describe configmap <name> -n slurm` | 查看 ConfigMap 詳細資訊 |
| `kubectl exec <pod> -- cat <file>` | 驗證檔案內容 |
| `kubectl describe nodeset <name> -n slurm` | 查看 NodeSet Volume 配置 |
