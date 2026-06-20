SCENARIO_DESCRIPTION="distrobox --version: alternate startup path"
SCENARIO_WARMUP=3
SCENARIO_RUNS=50

scenario_setup()   { :; }
scenario_command() { printf '%s --version\n' "$EXEC"; }
scenario_cleanup() { :; }
