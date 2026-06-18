#!/usr/bin/env bash
set -euo pipefail

# Install an Nginx reverse proxy location that exposes this service under /alipay.
# Public path:  /alipay/v1/paid-resource/prepare
# Upstream path: /v1/paid-resource/prepare
#
# Usage:
#   sudo bash scripts/install-nginx-alipay-prefix.sh your-domain.example.com 127.0.0.1:8080
#
# Then test:
#   curl -i https://your-domain.example.com/alipay/healthz
#   curl -i https://your-domain.example.com/alipay/v1/paid-resource/prepare ...

DOMAIN="${1:-}"
UPSTREAM="${2:-127.0.0.1:8080}"
CONF_DIR="/etc/nginx/conf.d"
CONF_FILE="${CONF_DIR}/alipay-ai-service.conf"

if [[ -z "${DOMAIN}" ]]; then
  echo "Usage: sudo bash $0 <domain> [upstream_host:port]" >&2
  echo "Example: sudo bash $0 pay.example.com 127.0.0.1:8080" >&2
  exit 1
fi

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root, for example: sudo bash $0 ${DOMAIN} ${UPSTREAM}" >&2
  exit 1
fi

if ! command -v nginx >/dev/null 2>&1; then
  echo "nginx command not found. Please install nginx first." >&2
  exit 1
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
echo "Upstream: http://${UPSTREAM}"
