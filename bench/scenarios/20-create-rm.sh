SCENARIO_DESCRIPTION="distrobox create alpine then rm --force (full cycles)"
SCENARIO_WARMUP=1
SCENARIO_RUNS=10

CREATE_RM_IMAGE="docker.io/library/alpine:latest"

scenario_setup() { :; }

scenario_command() {
    # Each invocation generates a unique container name. The name is appended
    # to the tracker before create runs so a mid-iteration crash still gets
    # cleaned up by the EXIT trap.
    printf "sh -c 'NAME=dbx-bench-%s-create-rm-\$\$-\$RANDOM; printf \"%%s\\n\" \"\$NAME\" >> %s/containers.list; %s create --image %s --name \"\$NAME\" --yes >/dev/null && %s rm --force \"\$NAME\" >/dev/null'\n" \
        "$RUN_ID" "$RESULTS_DIR" "$EXEC" "$CREATE_RM_IMAGE" "$EXEC"
}

scenario_cleanup() { :; }
