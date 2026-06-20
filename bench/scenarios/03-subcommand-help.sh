SCENARIO_DESCRIPTION="distrobox create --help: subcommand dispatch cost"
SCENARIO_WARMUP=3
SCENARIO_RUNS=50

scenario_setup()   { :; }
scenario_command() { printf '%s create --help\n' "$EXEC"; }
scenario_cleanup() { :; }
