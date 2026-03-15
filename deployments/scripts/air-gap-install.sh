#!/bin/bash
# Air-gap installation script for Argus
# This script installs Argus from pre-loaded container images.
#
# NOTE: Make this script executable before running:
#   chmod +x deployments/scripts/air-gap-install.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGES_DIR="${SCRIPT_DIR}/images"

echo "=== Argus Air-Gap Installation ==="
echo ""

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Error: docker is required"; exit 1; }
command -v docker compose >/dev/null 2>&1 || { echo "Error: docker compose is required"; exit 1; }

# Load pre-built images
if [ -d "${IMAGES_DIR}" ]; then
    echo "Loading container images..."
    for img in "${IMAGES_DIR}"/*.tar; do
        echo "  Loading $(basename "$img")..."
        docker load -i "$img"
    done
fi

echo ""
echo "Starting Argus stack..."
docker compose -f "${SCRIPT_DIR}/../docker/docker-compose.yml" up -d

echo ""
echo "=== Argus is starting ==="
echo "Dashboard: http://localhost:8080"
echo "API:       http://localhost:8080/api/v1/"
