#!/bin/bash
# compare-podman.sh — Run the compatibility suite using podman.

set -eu
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export CONTAINER_MANAGER=podman
exec "${SCRIPT_DIR}/compare.sh" "$@"
