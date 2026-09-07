# 环境变量

Kite 默认支持一些环境变量，来改变一些配置项的默认值。

- **KITE_USERNAME**：设置初始管理员用户名。在没有任何用户时随启动创建。
- **KITE_PASSWORD**：设置初始管理员密码。与 `KITE_USERNAME` 配合使用。
- **KUBECONFIG**：Kubernetes 配置文件路径, 默认值为 `~/.kube/config`，当 kite 没有配置集群时默认从此路径发现并导入集群到 Kite。
- **KITE_SEALOS_STALE_TTL_DAYS**：Sealos 自动创建的用户及其自动创建的集群/角色，在多少天未登录后被每小时执行的残留清理删除。默认 `30`，设为 `0` 关闭。

- **JWT_SECRET**：用于签名和验证 JWT 的密钥
- **KITE_ENCRYPT_KEY**：用于加密敏感数据的密钥, 例如用户密码，OAuth 的 clientSecret ，kubeconfig 等。

- **AUTH_COOKIE_SAMESITE**：认证 Cookie 的 SameSite 策略。可选值：`lax`、`strict`、`none`，默认 `lax`。当 Kite 以跨站 iframe 方式嵌入（例如 Sealos）时，建议设置为 `none`。
- **AUTH_COOKIE_SECURE**：认证 Cookie 的 Secure 策略。可选值：`auto`、`true`、`false`，默认 `auto`。当 `AUTH_COOKIE_SAMESITE=none` 时，必须设置为 `true`，并且使用 HTTPS。

- **SEALOS_AUTH_ENABLED**：是否启用 Sealos 登录接口（`/api/auth/login/sealos`），默认 `false`。
- **SEALOS_JWT_SECRET**：用于校验 Sealos JWT 的密钥。Kite 对 Sealos token 固定使用 `HS256` 校验。
- **SEALOS_DEFAULT_PROMETHEUS_URL**：Sealos 管理集群的默认 Prometheus 地址。配置后，Kite 会在 Sealos 集群同步时为新集群写入该值，并在启动时自动补齐历史 Sealos 集群中为空的 `prometheus_url`。
- **KITE_NAMESPACE_SCOPE_EXEMPT_NAMESPACES**：逗号分隔的命名空间白名单（例如 `ns-admin,platform-admin`）。命中后会跳过 kubeconfig `current-context.namespace` 导致的 namespace-scope 锁定，Sealos SSO 自动生成的角色会被授予该集群下 `*` 命名空间权限，并且这些 Sealos 用户会被分配 Kite 内置 `admin` 角色，可访问 AI Agent 配置等管理员专属设置。仅应配置那些实际具备全局/集群管理员权限的命名空间。
- **KITE_HELM_ARTIFACT_HUB_ENABLED**：是否启用 Artifact Hub Chart 源 API 和前端 fallback，默认 `true`；离线部署应设为 `false`。
- **KITE_HELM_OCI_REGISTRY_BASE**：Kite 用于发现离线 Helm OCI Chart 的 `oci://` registry 前缀，例如 `oci://registry.internal/kite-helm`。
- **KITE_HELM_OCI_REPOSITORY_NAME**：Kite 页面中展示的 OCI Chart 仓库名称，默认 `offline`。
- **KITE_HELM_OCI_DISCOVERY_PAGE_SIZE**：列出 registry repositories/tags 时使用的分页大小，默认 `100`。
- **KITE_HELM_OCI_DISCOVERY_MAX_REPOSITORIES**：查找配置前缀时最多检查的 registry repository 数量，默认 `1000`。
- **KITE_HELM_OCI_DISCOVERY_MAX_TAGS_PER_REPOSITORY**：每个 repository 最多扫描的 tag 数量，默认 `200`。
- **KITE_HELM_OCI_UPLOAD_MAX_BYTES**：管理员上传 Helm chart 包 API 接受的最大包大小，默认 `512MiB`。
- **KITE_HELM_OCI_REGISTRY_PLAIN_HTTP**：是否使用 HTTP 访问 OCI registry API 和 Chart 包，默认 `false`。
- **KITE_HELM_OCI_REGISTRY_INSECURE_SKIP_TLS_VERIFY**：是否跳过私有 registry/token endpoint 的 TLS 校验，默认 `false`。
- **KITE_HELM_OCI_REGISTRY_CA_FILE**：挂载到 Kite 容器内的私有 registry CA bundle 路径。
- **KITE_HELM_OCI_REGISTRY_USERNAME**：Kite 访问 OCI registry 列表和 Chart 包时使用的用户名。
- **KITE_HELM_OCI_REGISTRY_PASSWORD**：Kite 访问 OCI registry 列表和 Chart 包时使用的密码。建议通过 Kubernetes Secret 注入，而不是写进 Helm values。
- **KITE_IMAGE_UPLOAD_REGISTRY**：管理员上传容器镜像归档 API 使用的 registry host，例如 `registry.internal` 或 `registry.internal:5000`。留空时会复用已配置的 `KITE_HELM_OFFLINE_IMAGE_REGISTRY`。
- **KITE_IMAGE_UPLOAD_REPOSITORY_PREFIX**：上传容器镜像归档时追加的 repository 前缀，默认 `kite-images`。
- **KITE_IMAGE_UPLOAD_MAX_BYTES**：管理员上传容器镜像归档 API 和离线应用包导入 API 接受的最大归档大小，默认 `4GiB`。
- **KITE_IMAGE_UPLOAD_REGISTRY_PLAIN_HTTP**：容器镜像归档上传是否使用 HTTP，默认 `false`。
- **KITE_IMAGE_UPLOAD_REGISTRY_INSECURE_SKIP_TLS_VERIFY**：是否跳过容器镜像上传 registry/token endpoint 的 TLS 校验，默认 `false`。
- **KITE_IMAGE_UPLOAD_REGISTRY_CA_FILE**：挂载到 Kite 容器内的容器镜像上传 registry CA bundle 路径。
- **KITE_IMAGE_UPLOAD_REGISTRY_USERNAME**：Kite 推送上传的容器镜像归档时使用的用户名。
- **KITE_IMAGE_UPLOAD_REGISTRY_PASSWORD**：Kite 推送上传的容器镜像归档时使用的密码。建议通过 Kubernetes Secret 注入，而不是写进 Helm values。
- **KITE_HELM_OFFLINE_IMAGES_ENABLED**：是否为 OCI catalog 安装的 Chart 启用离线容器镜像默认值，默认 `false`。
- **KITE_HELM_OFFLINE_IMAGE_REGISTRY**：离线 OCI Chart 渲染 workload 镜像时使用的容器镜像仓库 host，例如 `registry.internal` 或 `registry.internal:5000`。不要包含 `http://`、`https://` 或 `oci://` 前缀。
- **KITE_HELM_OFFLINE_IMAGES_ENFORCE**：启用离线镜像默认值后，如果渲染出的 workload 镜像仍指向 `KITE_HELM_OFFLINE_IMAGE_REGISTRY` 之外的仓库，则阻止安装和升级，默认 `true`。

- **HOST**: 用户 OAuth 2.0 授权回调地址生成，默认会从请求头获取，如果您发现结果不及预期可以手动配置此环境变量。

- **NODE_TERMINAL_IMAGE**: 用于生成 Node Terminal Agent 的 Docker 镜像。

- **ENABLE_ANALYTICS**：启用数据分析功能，默认值为 `false`。当启用后，Kite 将收集有限数据以帮助改进产品。

- **PORT**：Kite 运行的端口，默认值为 `8080`。
- **DISABLE_CACHE**：设置为 `true` 时使用直接 Kubernetes API client，
  不启动 controller-runtime informer cache。`make dev` 会在本地开发时默认设置
  该变量，避免集群 client cache 反复同步失败时造成高 CPU。

可选前端环境变量（构建时生效）：

- **VITE_SEALOS_AUTO_LOGIN**：`true` 或 `false`，控制前端是否自动尝试 Sealos SDK 会话登录，默认 `true`。Sealos 自动登录同时允许 iframe 部署和顶层独立/本地开发窗口，包括通过 `sealos-app-dev-bridge` 使用的场景；`/login` 上的 SDK 可用性提示仅用于诊断，不会阻断自动登录尝试。
