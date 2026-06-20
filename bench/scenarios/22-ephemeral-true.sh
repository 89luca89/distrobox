SCENARIO_DESCRIPTION="distrobox ephemeral alpine -- /bin/true (full lifecycle)"
SCENARIO_WARMUP=1
SCENARIO_RUNS=5

EPHEMERAL_IMAGE="docker.io/library/alpine:latest"

scenario_setup()   { :; }
scenario_command() {
    printf '%s ephemeral --image %s --yes -- /bin/true\n' "$EXEC" "$EPHEMERAL_IMAGE"
}
scenario_cleanup() { :; }
