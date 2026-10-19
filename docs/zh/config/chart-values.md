# Chart Values

本文档描述了 Kite Helm Chart 的所有可用配置选项。

## 基础配置

| 参数               | 描述                                               | 默认值                |
| ------------------ | -------------------------------------------------- | --------------------- |
| `replicaCount`     | 副本数量                                           | `1`                   |
| `image.repository` | 容器镜像仓库                                       | `crpi-7jr40k6elhldekqp.cn-hangzhou.personal.cr.aliyuncs.com/mlhiter/kite` |
| `image.pullPolicy` | 镜像拉取策略                                       | `IfNotPresent`        |
| `image.tag`        | 镜像标签。如果设置，将覆盖 chart 的 `appVersion`。 | `"1.0.8"`             |
| `imagePullSecrets` | 私有镜像仓库的拉取密钥                             | `[]`                  |
| `nameOverride`     | 覆盖 chart 名称                                    | `""`                  |
| `fullnameOverride` | 覆盖完整名称                                       | `""`                  |
| `debug`            | 启用调试模式                                       | `false`               |
| `basePath`         | 应用的基础路径，详见安装文档中的说明。     | `""`                 |

## 认证与安全

| 参数                   | 描述                                                       | 默认值                                               |
| ---------------------- | ---------------------------------------------------------- | ---------------------------------------------------- |
| `anonymousUserEnabled` | 启用匿名用户访问，拥有完全管理员权限。生产环境请谨慎使用。 | `false`                                              |
| `jwtSecret`            | 用于签名 JWT 令牌的密钥。生产环境请修改此值。              | `"kite-default-jwt-secret-key-change-in-production"` |
| `encryptKey`           | 用于加密敏感数据的密钥。生产环境请修改此值。               | `"kite-default-encryption-key-change-in-production"` |
| `host`                 | 应用程序的主机名                                           | `""`                                                 |
| `cloudDomain`          | Sealos 域名，用于渲染 ingress/app 主机名                   | `"127.0.0.1.nip.io"`                                |
| `cloudPort`            | HTTPS 模式下渲染 Sealos App 外部 URL 使用的端口             | `"443"`                                             |
| `httpPort`             | `disableHttps=true` 时渲染 HTTP 外部 URL 使用的端口          | `"80"`                                              |
| `disableHttps`         | 以 HTTP/无 TLS 模式渲染 Sealos App URL 和 Ingress            | `false`                                             |
| `certSecretName`       | HTTPS 模式下 Ingress TLS 使用的 Secret 名称                 | `"wildcard-cert"`                                   |
| `platform.tlsRejectUnauthorized` | 注入环境变量 `NODE_TLS_REJECT_UNAUTHORIZED` 的值    | `"1"`                                               |
| `sealos.jwtSecret`     | 注入环境变量 `SEALOS_JWT_SECRET` 的值                      | `""`                                                 |

## 数据库配置

| 参数            | 描述                                                                                                 | 默认值   |
| --------------- | ---------------------------------------------------------------------------------------------------- | -------- |
| `db.type`       | 数据库类型：`sqlite`、`postgres`、`mysql`                                                            | `sqlite` |
| `db.dsn`        | 外部 MySQL/Postgres 数据库的完整 DSN 字符串。`db.type` 为 `mysql`/`postgres` 时必需（sqlite 下忽略） | `""`     |
| `db.autoCreate` | Kite 是否在运行表结构迁移前自动创建目标 MySQL/Postgres database（sqlite 下无效果）                    | `true`   |

开启 `db.autoCreate` 时，配置的数据库账号需要有创建 database 的权限。Kite 只负责创建目标 database，表结构仍由应用正常迁移流程创建。

Chart 不再内置 PostgreSQL（KubeBlocks）子 chart：默认部署通过 PersistentVolumeClaim 持久化存储 sqlite 数据库文件。`postgres`/`mysql` 仍可通过 `db.dsn` 对接外部数据库使用。

### SQLite 配置

| 参数                                      | 描述                                  | 默认值              |
| ----------------------------------------- | ------------------------------------- | ------------------- |
| `db.sqlite.persistence.pvc.enabled`       | 是否创建 PVC 来存储 sqlite 数据库文件 | `true`              |
| `db.sqlite.persistence.pvc.existingClaim` | 使用现有的 PVC（设置后不再新建 PVC）  | `""`                |
| `db.sqlite.persistence.pvc.storageClass`  | PVC 的 StorageClass（可选）           | `""`                |
| `db.sqlite.persistence.pvc.accessModes`   | PVC 的访问模式                        | `["ReadWriteOnce"]` |
| `db.sqlite.persistence.pvc.size`          | PVC 请求的存储大小                    | `1Gi`               |
| `db.sqlite.persistence.hostPath.enabled`  | 是否使用 hostPath 存储                | `false`             |
| `db.sqlite.persistence.hostPath.path`     | hostPath 路径                         | `/path/to/host/dir` |
| `db.sqlite.persistence.hostPath.type`     | hostPath 类型                         | `DirectoryOrCreate` |
| `db.sqlite.persistence.mountPath`         | 容器内的挂载路径                      | `/data`             |
| `db.sqlite.persistence.filename`          | 挂载路径内的 sqlite 文件名            | `kite.db`           |

## 环境变量

| 参数        | 描述               | 默认值 |
| ----------- | ------------------ | ------ |
| `extraEnvs` | 额外的环境变量列表 | `[]`   |

## Helm Chart Catalog 配置

| 参数                                           | 描述                                           | 默认值    |
| ---------------------------------------------- | ---------------------------------------------- | --------- |
| `helmCatalog.artifactHub.enabled`              | 是否启用 Artifact Hub Chart 搜索/详情代理 API | `true`    |
| `helmCatalog.oci.base`                         | 用于发现离线 Helm OCI Chart 的 `oci://` registry 前缀 | `""` |
| `helmCatalog.oci.repositoryName`               | Kite 页面中展示的 OCI Chart 仓库名称          | `offline` |
| `helmCatalog.oci.discoveryPageSize`            | 列出 registry repositories/tags 时使用的分页大小 | `100` |
| `helmCatalog.oci.discoveryMaxRepositories`     | 查找配置前缀时最多检查的 registry repository 数量 | `1000` |
| `helmCatalog.oci.discoveryMaxTagsPerRepository` | 每个 repository 最多扫描的 tag 数量           | `200`     |
| `helmCatalog.oci.uploadMaxBytes`               | 管理员上传 Helm chart 包 API 接受的最大包大小 | `512MiB` |
| `helmCatalog.oci.plainHTTP`                    | 是否使用 HTTP 访问 OCI registry API 和 Chart 包 | `false` |
| `helmCatalog.oci.insecureSkipTLSVerify`        | 是否跳过私有 registry/token endpoint 的 TLS 校验 | `false` |
| `helmCatalog.oci.caFile`                       | 挂载到 Kite 容器内的私有 registry CA bundle 路径 | `""` |
| `helmCatalog.oci.username`                     | Kite 访问 OCI registry 列表和 Chart 包时使用的用户名 | `""` |
| `helmCatalog.oci.password`                     | 写入 Kite Secret 的 OCI registry 密码；生产安装优先使用 `passwordSecretName` | `""` |
| `helmCatalog.oci.passwordSecretName`           | 保存 OCI registry 密码的已有 Secret           | `""`      |
| `helmCatalog.oci.passwordSecretKey`            | `passwordSecretName` 中作为 `KITE_HELM_OCI_REGISTRY_PASSWORD` 使用的 key | `KITE_HELM_OCI_REGISTRY_PASSWORD` |
| `helmCatalog.imageUploads.registry`            | 上传容器镜像归档使用的 registry host；留空时复用已配置的 `helmCatalog.offlineImages.registry` | `""` |
| `helmCatalog.imageUploads.repositoryPrefix`    | 上传容器镜像归档时追加的 repository 前缀      | `kite-images` |
| `helmCatalog.imageUploads.maxBytes`            | 管理员上传容器镜像归档或导入离线应用包 API 接受的最大归档大小 | `4GiB` |
| `helmCatalog.imageUploads.plainHTTP`           | 容器镜像归档上传是否使用 HTTP                 | `false` |
| `helmCatalog.imageUploads.insecureSkipTLSVerify` | 是否跳过容器镜像上传 registry/token endpoint 的 TLS 校验 | `false` |
| `helmCatalog.imageUploads.caFile`              | 挂载到 Kite 容器内的容器镜像上传 registry CA bundle 路径 | `""` |
| `helmCatalog.imageUploads.username`            | Kite 推送上传的容器镜像归档时使用的用户名     | `""` |
| `helmCatalog.imageUploads.password`            | 写入 Kite Secret 的容器镜像上传 registry 密码；生产安装优先使用 `passwordSecretName` | `""` |
| `helmCatalog.imageUploads.passwordSecretName`  | 保存容器镜像上传 registry 密码的已有 Secret   | `""` |
| `helmCatalog.imageUploads.passwordSecretKey`   | `passwordSecretName` 中作为 `KITE_IMAGE_UPLOAD_REGISTRY_PASSWORD` 使用的 key | `KITE_IMAGE_UPLOAD_REGISTRY_PASSWORD` |
| `helmCatalog.offlineImages.enabled`            | 为 OCI catalog 安装/升级注入离线容器镜像默认值 | `false` |
| `helmCatalog.offlineImages.registry`           | 离线 OCI Chart 渲染 workload 镜像时应使用的 registry host | `""` |
| `helmCatalog.offlineImages.enforce`            | 当渲染出的 workload 镜像仍指向离线 registry 之外时阻止安装/升级 | `true` |

离线部署可以关闭 Artifact Hub，并让 Kite 只扫描一个受控 OCI registry
前缀：

```yaml
helmCatalog:
  artifactHub:
    enabled: false
  oci:
    base: oci://registry.internal/kite-helm
    repositoryName: offline
    plainHTTP: true
    insecureSkipTLSVerify: true
    username: admin
    passwordSecretName: registry-credentials
    passwordSecretKey: KITE_HELM_OCI_REGISTRY_PASSWORD
    uploadMaxBytes: 512MiB
  imageUploads:
    registry: registry.internal
    repositoryPrefix: kite-images
    maxBytes: 4GiB
    plainHTTP: true
    insecureSkipTLSVerify: true
    username: admin
    passwordSecretName: registry-credentials
    passwordSecretKey: KITE_IMAGE_UPLOAD_REGISTRY_PASSWORD
  offlineImages:
    enabled: true
    registry: registry.internal
    enforce: true
```

如果 registry 中存在 `kite-helm/demo-chart` 的 `0.1.0` 和 `0.2.0`
tag，Kite 会解析为
`oci://registry.internal/kite-helm/demo-chart:0.2.0`，用于浏览、安装、
升级和自动升级。`base` 必须是带 repository 前缀的 `oci://` URL，不能包含
凭据、查询参数、fragment、tag 或 digest。私有 registry 在生产安装中推荐使用
`helmCatalog.oci.username` 加 `helmCatalog.oci.passwordSecretName`；简易安装
仍可使用 `helmCatalog.oci.password`，它会写入 chart 管理的 Secret。

Kite 管理员 UI 以 `.kiteapp.tar.gz` 离线应用包作为传递单位。一个包会把 Helm
chart 归档和按离线镜像 values 渲染后需要的 workload 镜像归档绑定在一起，
可以包含多个应用，也可以从一个已配置好的 Kite 集群导出后导入到另一个集群。
导入会先校验包内容，先推送必需镜像，最后才推送 Chart，避免 OCI catalog
里出现缺镜像的 Chart。UI 会在服务端接受离线包上传后启动后台导入任务；
任务状态保存在内存里，Kite Pod 重启后可能丢失。

Kite 底层后端流程仍然分开。Helm chart 包（`.tgz`）通过 Chart 上传 API 推送到
`helmCatalog.oci.base` 配置的前缀下，并进入 OCI Chart catalog；容器镜像归档
通过独立的镜像上传 API 推送到 `helmCatalog.imageUploads.registry` 加
`helmCatalog.imageUploads.repositoryPrefix`，不会变成 Chart catalog 条目。
如果 `helmCatalog.imageUploads.registry` 为空，Kite 会在已配置
`helmCatalog.offlineImages.registry` 时复用该离线镜像 registry。Chart registry
和镜像 registry 凭据不同时，请使用单独的 Secret key 注入
`KITE_IMAGE_UPLOAD_REGISTRY_PASSWORD`。离线应用包可能包含容器镜像归档，因此同样
受镜像上传大小限制约束。

`helmCatalog.offlineImages` 与 OCI Chart 发现是两件事。Helm OCI Chart
artifact 放在配置的 Chart 前缀下，例如
`oci://registry.internal/kite-helm/nginx:25.0.12`；容器镜像放在同一个
registry host 的原始仓库路径下，例如
`registry.internal/bitnami/nginx:latest`。对 OCI catalog Chart，Kite 会在
用户未显式配置时注入 `global.imageRegistry=<registry>` 和
`global.security.allowInsecureImages=true`，渲染 release，并在 workload 镜像
仍指向外部 registry 时阻止写入。Kite 会把 Chart 来源记录在已安装 release
中，所以从 OCI catalog 安装的 release 后续即使客户端沿用当前 Chart、没有
再次传 `source`，升级时也会继续应用离线镜像策略。Kite 安装时不会复制容器
镜像；需要提前通过离线应用包导入、`scripts/mirror-helm-chart-images.sh` 或
等价 registry 同步流程完成镜像同步。

## Sealos App 配置

| 参数          | 描述                                  | 默认值 |
| ------------- | ------------------------------------- | ------ |
| `app.enabled` | 是否创建 `app.sealos.io/v1 App`      | `false` |

App 元数据在模板中固定：
- namespace: `app-system`
- name: `kite`
- type/displayType: `iframe` / `normal`
- icon/url：根据 `disableHttps`、`cloudPort` 和 `httpPort` 渲染。

## 服务账户配置

| 参数                         | 描述                        | 默认值 |
| ---------------------------- | --------------------------- | ------ |
| `serviceAccount.create`      | 是否创建服务账户            | `true` |
| `serviceAccount.automount`   | 自动挂载服务账户的 API 凭据 | `true` |
| `serviceAccount.annotations` | 服务账户的注解              | `{}`   |
| `serviceAccount.name`        | 使用的服务账户名称          | `""`   |

## RBAC 配置

| 参数          | 描述               | 默认值     |
| ------------- | ------------------ | ---------- |
| `rbac.create` | 是否创建 RBAC 资源 | `true`     |
| `rbac.rules`  | RBAC 规则列表      | 见下方示例 |

### RBAC 规则示例

```yaml
rbac:
  rules:
    - apiGroups: ["*"]
      resources: ["*"]
      verbs: ["*"]
    - nonResourceURLs: ["*"]
      verbs: ["*"]
```

## Pod 配置

| 参数                 | 描述                   | 默认值 |
| -------------------- | ---------------------- | ------ |
| `podAnnotations`     | Pod 的 Kubernetes 注解 | `{}`   |
| `podLabels`          | Pod 的 Kubernetes 标签 | `{}`   |
| `podSecurityContext` | Pod 安全上下文         | `{}`   |
| `securityContext`    | 容器安全上下文         | `{}`   |

## 服务配置

| 参数           | 描述     | 默认值      |
| -------------- | -------- | ----------- |
| `service.type` | 服务类型 | `ClusterIP` |
| `service.port` | 服务端口 | `8080`      |

## Ingress 配置

| 参数                  | 描述             | 默认值     |
| --------------------- | ---------------- | ---------- |
| `ingress.enabled`     | 是否启用 Ingress | `true`     |
| `ingress.proxyBodySize` | NGINX 请求体大小限制，用于仓库上传；应不小于最大的上传限制 | `4g` |

Ingress 行为在模板中固定：
- host: `kite.<cloudDomain>`
- className: `nginx`
- path/pathType: `/` / `Prefix`
- TLS：仅在 `disableHttps=false` 时渲染，secret 名称来自 `certSecretName`
- annotations：
  - `nginx.ingress.kubernetes.io/proxy-read-timeout: '3600'`
  - `nginx.ingress.kubernetes.io/proxy-send-timeout: '3600'`
  - `nginx.ingress.kubernetes.io/proxy-body-size: <ingress.proxyBodySize>`

### Ingress 配置示例

```yaml
ingress:
  enabled: true
cloudDomain: 192.168.10.70.nip.io
```

## 资源限制

| 参数        | 描述               | 默认值 |
| ----------- | ------------------ | ------ |
| `resources` | 容器资源限制和请求 | `{}`   |

### 资源限制示例

```yaml
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

## 健康检查

| 参数             | 描述         | 默认值     |
| ---------------- | ------------ | ---------- |
| `livenessProbe`  | 存活探针配置 | 见下方示例 |
| `readinessProbe` | 就绪探针配置 | 见下方示例 |

### 健康检查示例

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 10
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 10
  periodSeconds: 10
```

## 存储配置

| 参数           | 描述             | 默认值 |
| -------------- | ---------------- | ------ |
| `volumes`      | 额外的卷配置     | `[]`   |
| `volumeMounts` | 额外的卷挂载配置 | `[]`   |

## 调度配置

| 参数           | 描述       | 默认值 |
| -------------- | ---------- | ------ |
| `nodeSelector` | 节点选择器 | `{}`   |
| `tolerations`  | 容忍度配置 | `[]`   |
| `affinity`     | 亲和性配置 | `{}`   |
