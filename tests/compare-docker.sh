#!/bin/bash
# compare-docker.sh — Run the compatibility suite using docker.

set -eu
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export CONTAINER_MANAGER=docker
exec "${SCRIPT_DIR}/compare.sh" "$@"
