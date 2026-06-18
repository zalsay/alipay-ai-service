#!/usr/bin/env bash
set -euo pipefail

# Install an Nginx reverse proxy location that exposes this service under /alipay.
# Public path:  /alipay/v1/paid-resource/prepare
# Upstream path: /v1/paid-resource/prepare
#
# Usage:
#   sudo bash scripts/install-nginx-alipay-prefix.sh your-domain.example.com
#   sudo bash scripts/install-nginx-alipay-prefix.sh your-domain.example.com 127.0.0.1:18080
#
# Default upstream resolution:
#   1. Use explicit second argument if provided.
#   2. Otherwise read SERVER_ADDR from .env or ENV_FILE.
#   3. Otherwise fallback to 127.0.0.1:8080.
#
# ENV_FILE can be used when .env is not in the current directory:
#   sudo ENV_FILE=/opt/alipay-ai-service/.env bash scripts/install-nginx-alipay-prefix.sh pay.example.com

DOMAIN="${1:-}"
EXPLICIT_UPSTREAM="${2:-}"
ENV_FILE="${ENV_FILE:-.env}"
CONF_DIR="/etc/nginx/conf.d"
CONF_FILE="${CONF_DIR}/alipay-ai-service.conf"

if [[ -z "${DOMAIN}" ]]; then
  echo "Usage: sudo bash $0 <domain> [upstream_host:port]" >&2
  echo "Example: sudo bash $0 pay.example.com" >&2
  echo "Example: sudo bash $0 pay.example.com 127.0.0.1:18080" >&2
  echo "Example: sudo ENV_FILE=/opt/alipay-ai-service/.env bash $0 pay.example.com" >&2
  exit 1
fi

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root, for example: sudo bash $0 ${DOMAIN}" >&2
  exit 1
fi

if ! command -v nginx >/dev/null 2>&1; then
  echo "nginx command not found. Please install nginx first." >&2
  exit 1
fi

read_env_value() {
  local key="$1"
  local file="$2"
  if [[ ! -f "${file}" ]]; then
    return 0
  fi
  grep -E "^[[:space:]]*${key}=" "${file}" \
    | tail -n 1 \
    | sed -E "s/^[[:space:]]*${key}=//" \
    | sed -E 's/[[:space:]]+#.*$//' \
    | sed -E 's/^"(.*)"$/\1/' \
    | sed -E "s/^'(.*)'$/\1/"
}

server_addr_to_upstream() {
  local server_addr="$1"
  server_addr="${server_addr%\"}"
  server_addr="${server_addr#\"}"
  server_addr="${server_addr%\'}"
  server_addr="${server_addr#\'}"

  if [[ -z "${server_addr}" ]]; then
    echo "127.0.0.1:8080"
    return
  fi

  # Common Go listen address: :8080
  if [[ "${server_addr}" =~ ^:([0-9]+)$ ]]; then
    echo "127.0.0.1:${BASH_REMATCH[1]}"
    return
  fi

  # IPv4 or hostname with port, e.g. 0.0.0.0:8080 or localhost:8080.
  if [[ "${server_addr}" =~ ^0\.0\.0\.0:([0-9]+)$ ]]; then
    echo "127.0.0.1:${BASH_REMATCH[1]}"
    return
  fi

  if [[ "${server_addr}" =~ ^\[::\]:([0-9]+)$ ]]; then
    echo "127.0.0.1:${BASH_REMATCH[1]}"
    return
  fi

  if [[ "${server_addr}" =~ ^([a-zA-Z0-9._-]+):([0-9]+)$ ]]; then
    echo "${BASH_REMATCH[1]}:${BASH_REMATCH[2]}"
    return
  fi

  echo "Unsupported SERVER_ADDR format: ${server_addr}" >&2
  exit 1
}

if [[ -n "${EXPLICIT_UPSTREAM}" ]]; then
  UPSTREAM="${EXPLICIT_UPSTREAM}"
else
  SERVER_ADDR_VALUE="$(read_env_value SERVER_ADDR "${ENV_FILE}")"
  UPSTREAM="$(server_addr_to_upstream "${SERVER_ADDR_VALUE}")"
fi

mkdir -p "${CONF_DIR}"

cat > "${CONF_FILE}" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    client_max_body_size 10m;

    # Optional convenience health check with the /alipay prefix.
    location = /alipay/healthz {
        proxy_pass http://${UPSTREAM}/healthz;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # Main prefix rewrite:
    # /alipay/v1/... -> /v1/...
    location /alipay/ {
        rewrite ^/alipay/(.*) /\$1 break;
        proxy_pass http://${UPSTREAM};
        proxy_http_version 1.1;

        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # Preserve AI 收 protocol headers.
        proxy_set_header Payment-Proof \$http_payment_proof;
        proxy_pass_header Payment-Needed;

        proxy_connect_timeout 10s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
EOF

nginx -t
systemctl reload nginx 2>/dev/null || nginx -s reload

echo "Installed ${CONF_FILE}"
echo "Public prefix: http://${DOMAIN}/alipay"
echo "ENV_FILE: ${ENV_FILE}"
echo "Upstream: http://${UPSTREAM}"
