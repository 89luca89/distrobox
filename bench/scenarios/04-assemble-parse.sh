SCENARIO_DESCRIPTION="distrobox assemble create --dry-run on a 3-section ini"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

FIXTURE_PATH="${SCRIPT_DIR}/fixtures/assemble.ini"

scenario_supported() {
    "$EXEC" assemble --help 2>&1 | grep -q -- '--dry-run'
}

scenario_setup()   { :; }
scenario_command() {
    printf '%s assemble create --file %s --dry-run\n' "$EXEC" "$FIXTURE_PATH"
}
scenario_cleanup() { :; }
