#!/usr/bin/env bash
set -euo pipefail

timestamp() {
  date +"%Y-%m-%d %T"
}

info() {
  local flag
  flag="$(timestamp)"
  echo -e "\033[36m INFO [$flag] >> $* \033[0m"
}

warn() {
  local flag
  flag="$(timestamp)"
  echo -e "\033[33m WARN [$flag] >> $* \033[0m"
}

error() {
  local flag
  flag="$(timestamp)"
  echo -e "\033[31m ERROR [$flag] >> $* \033[0m"
  exit 1
}

RELEASE_NAME="${RELEASE_NAME:-kite}"
NAMESPACE="${NAMESPACE:-kite-system}"
HELM_OPTS="${HELM_OPTS:-}"
ENABLE_APP="${ENABLE_APP:-true}"
STRICT_SECRET_REUSE="${STRICT_SECRET_REUSE:-true}"
APP_NAME="${APP_NAME:-kite}"
CHART_DIR="${CHART_DIR:-charts/kite}"
VALUES_DIR_FILE="/root/.sealos/cloud/values/apps/${APP_NAME}/${APP_NAME}-values.yaml"
DEFAULT_VALUES_FILE="${CHART_DIR}/${APP_NAME}-values.yaml"
GLOBAL_VALUES_FILE="/root/.sealos/cloud/values/global.yaml"

die() {
  error "$@"
  exit 1
}

load_cloud_tools() {
  local required_functions=(
    bool_is_true
    global_http_disable_https
    global_http_external_url
    read_yaml_file_path
    read_jwt_internal
    fetch_configmap_data_key
    read_cert_tls_reject_unauthorized
  )
  local missing_functions=()
  local function_name

  if [ ! -f /root/.sealos/cloud/scripts/tools.sh ]; then
    die "tools.sh not found in /root/.sealos/cloud/scripts"
  fi

  # shellcheck source=/dev/null
  source /root/.sealos/cloud/scripts/tools.sh
  for function_name in "${required_functions[@]}"; do
    if ! declare -f "$function_name" >/dev/null 2>&1; then
      missing_functions+=("$function_name")
    fi
  done
  [ "${#missing_functions[@]}" -eq 0 ] || die "tools.sh is missing required functions: ${missing_functions[*]}"
}

load_cloud_tools
ensure_cloud_values_file() {
  local source_file="$1"
  local target_file="$2"
  local label="$3"
  local target_dir="${target_file%/*}"

  [ -f "$source_file" ] || die "default ${label} values file not found: ${source_file}"
  mkdir -p "$target_dir"
  if [ ! -f "$target_file" ]; then
    warn "/root/.sealos/cloud/values/apps/kite ${target_file} not found, copying default ${label} values"
    cp "$source_file" "$target_file"
    info "Copied default ${label} Helm values to ${target_file}"
  else
    info "Using existing ${label} Helm values from ${target_file}"
  fi
}

decode_base64() {
  local raw=$1
  local decoded=""

  if decoded="$(printf '%s' "${raw}" | base64 --decode 2>/dev/null)"; then
    printf '%s' "${decoded}"
    return 0
  fi

  if decoded="$(printf '%s' "${raw}" | base64 -d 2>/dev/null)"; then
    printf '%s' "${decoded}"
    return 0
  fi

  return 1
}

get_secret_data() {
  local secret_name=$1
  local key=$2
  local encoded=""

  encoded="$(kubectl get secret "${secret_name}" -n "${NAMESPACE}" -o "jsonpath={.data.${key}}" 2>/dev/null || true)"
  [ -n "${encoded}" ] || return 1

  decode_base64 "${encoded}"
}

find_existing_kite_secret() {
  local name=""

  for name in "${RELEASE_NAME}-secret" "${RELEASE_NAME}-kite-secret"; do
    if kubectl get secret "${name}" -n "${NAMESPACE}" >/dev/null 2>&1; then
      echo "${name}"
      return 0
    fi
  done

  while IFS= read -r name; do
    [ -n "${name}" ] || continue
    if kubectl get secret "${name}" -n "${NAMESPACE}" >/dev/null 2>&1; then
      echo "${name}"
      return 0
    fi
  done < <(kubectl get secret -n "${NAMESPACE}" \
    -l "app.kubernetes.io/instance=${RELEASE_NAME},app.kubernetes.io/name=kite" \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)

  return 1
}

is_existing_release() {
  helm status "${RELEASE_NAME}" -n "${NAMESPACE}" >/dev/null 2>&1
}

ensure_namespace() {
  kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
}

ensure_registry_secret() {
  local username=$1
  local password=$2
  local secret_name="${RELEASE_NAME}-oci-registry"
  local password_file

  [ -n "${password}" ] || return 0
  ensure_namespace
  password_file="$(mktemp)"
  trap 'rm -f "${password_file}"' RETURN
  printf '%s' "${password}" > "${password_file}"
  kubectl create secret generic "${secret_name}" -n "${NAMESPACE}" \
    --from-file=KITE_HELM_OCI_REGISTRY_PASSWORD="${password_file}" \
    --dry-run=client -o yaml | kubectl apply -f -
  rm -f "${password_file}"
  trap - RETURN
  helm_set_args+=(
    --set-string "helmCatalog.oci.username=${username}"
    --set-string "helmCatalog.oci.passwordSecretName=${secret_name}"
    --set-string "helmCatalog.oci.passwordSecretKey=KITE_HELM_OCI_REGISTRY_PASSWORD"
    --set-string "helmCatalog.imageUploads.username=${username}"
    --set-string "helmCatalog.imageUploads.passwordSecretName=${secret_name}"
    --set-string "helmCatalog.imageUploads.passwordSecretKey=KITE_HELM_OCI_REGISTRY_PASSWORD"
  )
}

read_cert_tls_skip() {
    local cert_mode

    cert_mode="$(kubectl get configmap cert-config -n sealos-system -o jsonpath='{.data.CERT_MODE}' 2>/dev/null || true)"
    cert_mode="${CERT_MODE:-${cert_mode:-self-signed}}"
    cert_mode="$(printf '%s' "${cert_mode}" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')"

    case "${cert_mode}" in
        https|acme|acmedns) printf 'false' ;;
        *) printf 'true' ;;
    esac
}

sealos_jwt_secret="$(read_jwt_internal 2>/dev/null || true)"
sealos_cloud_domain="$(fetch_configmap_data_key sealos-config cloudDomain sealos-system 1 0 2>/dev/null || true)"
sealos_cloud_port="$(fetch_configmap_data_key sealos-config cloudPort sealos-system 1 0 2>/dev/null || true)"
sealos_http_port="$(fetch_configmap_data_key sealos-config httpPort sealos-system 1 0 2>/dev/null || true)"
sealos_disable_https="$(fetch_configmap_data_key sealos-config disableHttps sealos-system 1 0 2>/dev/null || true)"
sealos_cert_secret_name="$(fetch_configmap_data_key sealos-config certSecretName sealos-system 1 0 2>/dev/null || true)"
platform_tls_reject_unauthorized="$(read_cert_tls_reject_unauthorized)"

[ -n "${sealos_jwt_secret}" ] || error "Failed to read sealos-config.data.jwtInternal"
[ -n "${sealos_cloud_domain}" ] || error "Failed to read sealos-config.data.cloudDomain"

jwt_secret=""
encrypt_key=""
jwt_secret_source="generated"
encrypt_key_source="generated"
existing_secret_name="$(find_existing_kite_secret || true)"
release_exists="false"
if is_existing_release; then
  release_exists="true"
fi

strict_reuse_enabled="false"
if [ "${STRICT_SECRET_REUSE}" = "true" ]; then
  strict_reuse_enabled="true"
fi

if [ "${release_exists}" = "true" ] && [ "${strict_reuse_enabled}" = "true" ] && [ -z "${existing_secret_name}" ]; then
  error "Existing release ${RELEASE_NAME} detected, but secret not found in namespace ${NAMESPACE}. Refuse to generate new keys when STRICT_SECRET_REUSE=true"
fi

if [ -n "${existing_secret_name}" ]; then
  info "Found existing secret ${existing_secret_name}, trying to reuse JWT_SECRET and KITE_ENCRYPT_KEY"
  jwt_secret="$(get_secret_data "${existing_secret_name}" "JWT_SECRET" || true)"
  encrypt_key="$(get_secret_data "${existing_secret_name}" "KITE_ENCRYPT_KEY" || true)"

  if [ -n "${jwt_secret}" ]; then
    jwt_secret_source="secret:${existing_secret_name}"
  fi
  if [ -n "${encrypt_key}" ]; then
    encrypt_key_source="secret:${existing_secret_name}"
  fi

  if [ "${strict_reuse_enabled}" = "true" ]; then
    [ -n "${jwt_secret}" ] || error "Missing JWT_SECRET in existing secret ${existing_secret_name} with STRICT_SECRET_REUSE=true"
    [ -n "${encrypt_key}" ] || error "Missing KITE_ENCRYPT_KEY in existing secret ${existing_secret_name} with STRICT_SECRET_REUSE=true"
  fi
fi

if [ -z "${jwt_secret}" ]; then
  warn "JWT_SECRET not found in existing secret, generating a new one"
  jwt_secret="$(openssl rand -hex 32)"
  jwt_secret_source="generated"
fi

if [ -z "${encrypt_key}" ]; then
  warn "KITE_ENCRYPT_KEY not found in existing secret, generating a new one"
  encrypt_key="$(openssl rand -hex 32)"
  encrypt_key_source="generated"
fi

info "Secret reuse summary: existing_secret=${existing_secret_name:-none}, jwt_source=${jwt_secret_source}, encrypt_source=${encrypt_key_source}, strict_reuse=${STRICT_SECRET_REUSE}"

helm_set_args=(
  --set-string "jwtSecret=${jwt_secret}"
  --set-string "encryptKey=${encrypt_key}"
  --set-string "sealos.jwtSecret=${sealos_jwt_secret}"
  --set-string "cloudDomain=${sealos_cloud_domain}"
  --set-string "cloudPort=${sealos_cloud_port:-443}"
  --set-string "httpPort=${sealos_http_port:-80}"
  --set-string "disableHttps=${sealos_disable_https:-false}"
  --set-string "certSecretName=${sealos_cert_secret_name:-wildcard-cert}"
  --set-string "platform.tlsRejectUnauthorized=${platform_tls_reject_unauthorized:-1}"
)

if declare -f global_http_external_url >/dev/null 2>&1; then
  kite_external_url="$(global_http_external_url "kite.${sealos_cloud_domain}" 2>/dev/null || true)"
  [ -n "${kite_external_url}" ] && info "Kite external URL: ${kite_external_url}"
fi

if [ "${ENABLE_APP}" = "true" ]; then
  helm_set_args+=(--set "app.enabled=true")
fi

registry_username="$(fetch_configmap_data_key registry-config ADMIN_USER sealos-system 1 0 2>/dev/null || true)"
registry_password="$(fetch_configmap_data_key registry-config ADMIN_PASSWORD sealos-system 1 0 2>/dev/null || true)"

helm_set_args+=(
  --set "helmCatalog.oci.base=oci://hub.${sealos_cloud_domain}/kite-helm"
  --set "helmCatalog.imageUploads.registry=hub.${sealos_cloud_domain}"
  --set "helmCatalog.offlineImages.enabled=true"
  --set "helmCatalog.offlineImages.registry=hub.${sealos_cloud_domain}"
  --set "helmCatalog.offlineImages.enforce=true"
)
ensure_registry_secret "${registry_username}" "${registry_password}"
if bool_is_true "${sealos_disable_https}"; then
  helm_set_args+=(--set "helmCatalog.oci.plainHTTP=true")
  helm_set_args+=(--set "helmCatalog.imageUploads.plainHTTP=true")
else
  helm_set_args+=(--set "helmCatalog.oci.plainHTTP=false")
  helm_set_args+=(--set "helmCatalog.imageUploads.plainHTTP=false")
fi

helm_set_args+=(--set "helmCatalog.oci.insecureSkipTLSVerify=$(read_cert_tls_skip)")
helm_set_args+=(--set "helmCatalog.imageUploads.insecureSkipTLSVerify=$(read_cert_tls_skip)")


helm_opts_arr=()
values_args=()
ensure_cloud_values_file ${DEFAULT_VALUES_FILE} ${VALUES_DIR_FILE} kite
helm_set_args+=(
    -f "${VALUES_DIR_FILE}"
  )

info "Installing chart ${CHART_DIR} into namespace ${NAMESPACE}"
helm upgrade -i "${RELEASE_NAME}" -n "${NAMESPACE}" --create-namespace "${CHART_DIR}" \
  "${values_args[@]}" \
  "${helm_set_args[@]}" \
  --wait
