#!/usr/bin/env bash
# Container tracker and podman ops for bench/. Requires common.sh sourced.

: "${PODMAN_CMD:=podman}"
: "${RUN_ID:=unset}"
: "${TRACKER_FILE:=}"

tracker_init() {
    TRACKER_FILE="$1"
    : > "$TRACKER_FILE"
}

tracker_add() {
    local name="$1"
    [ -n "$TRACKER_FILE" ] || die "tracker_add called before tracker_init"
    printf '%s\n' "$name" >> "$TRACKER_FILE"
}

tracker_cleanup() {
    [ -n "$TRACKER_FILE" ] || return 0
    [ -e "$TRACKER_FILE" ] || return 0
    local name
    while IFS= read -r name; do
        [ -n "$name" ] || continue
        $PODMAN_CMD rm -f "$name" >/dev/null 2>&1 || true
    done < "$TRACKER_FILE"
}

container_create_distrobox() {
    local name="$1" image="$2" exec_path="$3"
    tracker_add "$name"
    "$exec_path" create --image "$image" --name "$name" --yes >/dev/null
}

container_create_podman_direct() {
    local name="$1" image="$2"
    [ "$RUN_ID" != "unset" ] || die "RUN_ID must be exported before container_create_podman_direct"
    tracker_add "$name"
    $PODMAN_CMD create \
        --label "distrobox.bench.run=${RUN_ID}" \
        --label "manager=distrobox" \
        --name "$name" \
        "$image" \
        /bin/true >/dev/null
}

container_orphan_scan() {
    $PODMAN_CMD ps -a --filter "label=distrobox.bench.run" --format '{{.ID}}' 2>/dev/null
}

container_orphan_clean() {
    local ids
    ids="$(container_orphan_scan)"
    [ -n "$ids" ] || return 0
    printf '%s\n' "$ids" | xargs -r $PODMAN_CMD rm -f >/dev/null 2>&1 || true
}
