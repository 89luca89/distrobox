SCENARIO_DESCRIPTION="distrobox --help: binary load + arg parser cost"
SCENARIO_WARMUP=3
SCENARIO_RUNS=50

scenario_setup()   { :; }
scenario_command() { printf '%s --help\n' "$EXEC"; }
scenario_cleanup() { :; }
