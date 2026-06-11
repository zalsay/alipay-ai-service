#!/bin/bash
set -e

# Build static binary
make build

# Rebuild and restart docker-compose
docker compose down
docker compose up --build -d

echo "Setup complete: static binary built and docker-compose restarted."