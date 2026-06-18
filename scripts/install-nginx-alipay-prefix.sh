#!/usr/bin/env bash
set -euo pipefail

# Install an Nginx reverse proxy location that exposes this service under /alipay over HTTPS.
# Public path:  https://your-domain.example.com/alipay/v1/paid-resource/prepare
# Upstream path: http://127.0.0.1:<SERVER_ADDR port>/v1/paid-resource/prepare
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
# Certificate resolution:
#   1. Use SSL_CERT_FILE and SSL_CERT_KEY if provided.
#   2. Otherwise use Let's Encrypt default paths:
#      /etc/letsencrypt/live/<domain>/fullchain.pem
#      /etc/letsencrypt/live/<domain>/privkey.pem
#
# Examples:
#   sudo bash scripts/install-nginx-alipay-prefix.sh pay.example.com
#   sudo ENV_FILE=/opt/alipay-ai-service/.env bash scripts/install-nginx-alipay-prefix.sh pay.example.com
#   sudo SSL_CERT_FILE=/path/fullchain.pem SSL_CERT_KEY=/path/privkey.pem bash scripts/install-nginx-alipay-prefix.sh pay.example.com

DOMAIN="${1:-}"
EXPLICIT_UPSTREAM="${2:-}"
ENV_FILE="${ENV_FILE:-.env}"
CONF_DIR="/etc/nginx/conf.d"
CONF_FILE="${CONF_DIR}/alipay-ai-service.conf"
SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/letsencrypt/live/${DOMAIN}/fullchain.pem}"
SSL_CERT_KEY="${SSL_CERT_KEY:-/etc/letsencrypt/live/${DOMAIN}/privkey.pem}"
ENABLE_HTTP_REDIRECT="${ENABLE_HTTP_REDIRECT:-1}"

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

if [[ ! -f "${SSL_CERT_FILE}" ]]; then
  echo "SSL certificate file not found: ${SSL_CERT_FILE}" >&2
  echo "Set SSL_CERT_FILE=/path/fullchain.pem or install a certificate first." >&2
  exit 1
fi

if [[ ! -f "${SSL_CERT_KEY}" ]]; then
  echo "SSL certificate key not found: ${SSL_CERT_KEY}" >&2
  echo "Set SSL_CERT_KEY=/path/privkey.pem or install a certificate first." >&2
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

  if [[ "${server_addr}" =~ ^:([0-9]+)$ ]]; then
    echo "127.0.0.1:${BASH_REMATCH[1]}"
    return
  fi

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
    listen 443 ssl http2;
    server_name ${DOMAIN};

    ssl_certificate ${SSL_CERT_FILE};
    ssl_certificate_key ${SSL_CERT_KEY};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;

    client_max_body_size 10m;

    location = /alipay/healthz {
        proxy_pass http://${UPSTREAM}/healthz;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
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
        proxy_set_header X-Forwarded-Proto https;

        # Preserve AI 收 protocol headers.
        proxy_set_header Payment-Proof \$http_payment_proof;
        proxy_pass_header Payment-Needed;

        proxy_connect_timeout 10s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
EOF

if [[ "${ENABLE_HTTP_REDIRECT}" == "1" ]]; then
cat >> "${CONF_FILE}" <<EOF

server {
    listen 80;
    server_name ${DOMAIN};
    return 301 https://\$host\$request_uri;
}
EOF
fi

nginx -t
systemctl reload nginx 2>/dev/null || nginx -s reload

echo "Installed ${CONF_FILE}"
echo "Public prefix: https://${DOMAIN}/alipay"
echo "ENV_FILE: ${ENV_FILE}"
echo "Upstream: http://${UPSTREAM}"
echo "SSL certificate: ${SSL_CERT_FILE}"
echo "SSL key: ${SSL_CERT_KEY}"
