SCENARIO_DESCRIPTION="distrobox list with 20 dummy distrobox-labelled containers added"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

LIST_MANY_COUNT=20
LIST_MANY_IMAGE="docker.io/library/alpine:latest"

scenario_setup() {
    local i name
    for i in $(seq 1 "$LIST_MANY_COUNT"); do
        name="dbx-bench-${RUN_ID}-list-many-$(printf '%02d' "$i")"
        container_create_podman_direct "$name" "$LIST_MANY_IMAGE"
    done
    # Confirm distrobox sees at least LIST_MANY_COUNT containers
    local visible
    visible=$($EXEC list 2>/dev/null | tail -n +2 | wc -l | tr -d ' ')
    if [ "$visible" -lt "$LIST_MANY_COUNT" ]; then
        die "expected ≥${LIST_MANY_COUNT} containers visible to distrobox list, got ${visible}"
    fi
}

scenario_command() { printf '%s list\n' "$EXEC"; }

scenario_cleanup() {
    # Per-iteration containers go to global tracker; cleaned by EXIT trap.
    # Nothing to do here.
    :
}
