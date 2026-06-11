#!/bin/bash
set -e

CONTAINER_NAME="${CONTAINER_NAME:-alipay-ai-service}"
BINARY_PATH="${BINARY_PATH:-/root/alipay-ai-service}"
LOCAL_BINARY="${LOCAL_BINARY:-dist/alipay-ai-service}"

make build

docker compose up -d --no-build --force-recreate

docker cp "$LOCAL_BINARY" "$CONTAINER_NAME:$BINARY_PATH"
docker restart "$CONTAINER_NAME"

echo "Setup complete: built $LOCAL_BINARY, replaced $BINARY_PATH in $CONTAINER_NAME, and restarted the container."
