SCENARIO_DESCRIPTION="distrobox list with no harness containers (user containers may be present)"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

scenario_setup() {
    local n
    n=$($EXEC list 2>/dev/null | tail -n +2 | wc -l | tr -d ' ')
    log_info "  (environment has ~${n} distrobox containers)"
}

scenario_command() { printf '%s list\n' "$EXEC"; }
scenario_cleanup() { :; }
