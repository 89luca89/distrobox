SCENARIO_DESCRIPTION="distrobox enter -- /bin/true (warm container)"
SCENARIO_WARMUP=2
SCENARIO_RUNS=10

ENTER_EXEC_IMAGE="docker.io/library/alpine:latest"
ENTER_EXEC_NAME=""

scenario_setup() {
    ENTER_EXEC_NAME="dbx-bench-${RUN_ID}-enter-exec"
    container_create_distrobox "$ENTER_EXEC_NAME" "$ENTER_EXEC_IMAGE" "$EXEC"
    # Warm: first enter starts the container and runs init scripts
    "$EXEC" enter --name "$ENTER_EXEC_NAME" -- /bin/true >/dev/null 2>&1 || true
}

scenario_command() {
    printf '%s enter --name %s -- /bin/true\n' "$EXEC" "$ENTER_EXEC_NAME"
}

scenario_cleanup() {
    # Belt-and-suspenders: the EXIT trap also handles this.
    podman rm -f "$ENTER_EXEC_NAME" >/dev/null 2>&1 || true
}
