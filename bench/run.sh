#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/lib/common.sh"
. "${SCRIPT_DIR}/lib/container.sh"
. "${SCRIPT_DIR}/lib/hyperfine.sh"
. "${SCRIPT_DIR}/lib/time.sh"
. "${SCRIPT_DIR}/lib/perf.sh"

# ── Arguments ────────────────────────────────────────────────────────────────
EXEC=""
LABEL=""
SCENARIOS_FILTER=""
DO_PROFILE=0
DO_CLEAN_ORPHANS=0

usage() {
    cat >&2 <<EOF
Usage: $0 <executable> [--label NAME] [--scenarios LIST]
                       [--profile] [--clean-orphans]

  <executable>        path to distrobox binary or shell script
  --label NAME        results directory name (default: basename of executable)
  --scenarios LIST    comma-separated names or shell glob (default: all)
  --profile           also run 'perf record' per scenario
  --clean-orphans     remove containers labelled distrobox.bench.run.* before run
EOF
    exit 2
}

while [ $# -gt 0 ]; do
    case "$1" in
        --label)         LABEL="$2"; shift 2 ;;
        --scenarios)     SCENARIOS_FILTER="$2"; shift 2 ;;
        --profile)       DO_PROFILE=1; shift ;;
        --clean-orphans) DO_CLEAN_ORPHANS=1; shift ;;
        -h|--help)       usage ;;
        -*)              log_err "unknown flag: $1"; usage ;;
        *)
            if [ -z "$EXEC" ]; then EXEC="$1"; shift
            else log_err "unexpected arg: $1"; usage; fi
            ;;
    esac
done

[ -n "$EXEC" ] || usage
[ -x "$EXEC" ] || die "executable not found or not executable: $EXEC"
EXEC="$(readlink -f "$EXEC")"
[ -z "$LABEL" ] && LABEL="$(basename "$EXEC")"

# ── Preflight ────────────────────────────────────────────────────────────────
preflight_check_tools >/dev/null
export DBX_CONTAINER_MANAGER=podman

if [ "$DO_CLEAN_ORPHANS" -eq 1 ]; then
    log_info "cleaning orphans from previous runs"
    container_orphan_clean
fi

orphans="$(container_orphan_scan)"
if [ -n "$orphans" ]; then
    die "orphan containers from a previous bench run exist (label distrobox.bench.run). re-run with --clean-orphans to remove."
fi

# Pre-pull image if needed
IMAGE="docker.io/library/alpine:latest"
if ! podman image exists "$IMAGE"; then
    log_info "pulling $IMAGE"
    podman pull "$IMAGE" >/dev/null
fi

# ── Run setup ────────────────────────────────────────────────────────────────
export RUN_ID="$(date +%s)-$$"
RESULTS_DIR="${SCRIPT_DIR}/results/${LABEL}/${RUN_ID}"
mkdir -p "${RESULTS_DIR}/scenarios"
tracker_init "${RESULTS_DIR}/containers.list"

# Always clean up tracker on exit
trap 'tracker_cleanup' EXIT

# Meta
write_meta() {
    local started_at="$1"
    local finished_at="$2"
    local scenarios_run_csv="$3"
    local scenarios_skipped_csv="$4"
    local cpu_model
    cpu_model="$(awk -F': ' '/model name/ {print $2; exit}' /proc/cpuinfo 2>/dev/null || echo unknown)"
    jq -n \
        --arg label "$LABEL" \
        --arg run_id "$RUN_ID" \
        --arg started "$started_at" \
        --arg finished "$finished_at" \
        --arg executable "$EXEC" \
        --arg sha256 "$(sha256sum "$EXEC" | awk '{print $1}')" \
        --argjson size "$(stat -c %s "$EXEC")" \
        --arg kernel "$(uname -r)" \
        --arg cpu_model "$cpu_model" \
        --arg podman_version "$(podman --version | awk '{print $3}')" \
        --argjson perf_paranoid "$(cat /proc/sys/kernel/perf_event_paranoid 2>/dev/null || echo -1)" \
        --arg scenarios_run "$scenarios_run_csv" \
        --arg scenarios_skipped "$scenarios_skipped_csv" \
        '{
            label: $label,
            run_id: $run_id,
            started: $started,
            finished: $finished,
            executable: $executable,
            executable_sha256: $sha256,
            executable_size_bytes: $size,
            host: {
                kernel: $kernel,
                cpu_model: $cpu_model,
                podman_version: $podman_version,
                perf_event_paranoid: $perf_paranoid
            },
            scenarios_run: ($scenarios_run | split(",") | map(select(. != ""))),
            scenarios_skipped: ($scenarios_skipped | split(",") | map(select(. != "")))
        }' > "${RESULTS_DIR}/meta.json"
}

# ── Scenario discovery ──────────────────────────────────────────────────────
all_scenarios="$(scenario_list "${SCRIPT_DIR}/scenarios")"
if [ -n "$SCENARIOS_FILTER" ]; then
    scenarios="$(printf '%s\n' "$all_scenarios" | scenario_filter "$SCENARIOS_FILTER")"
else
    scenarios="$all_scenarios"
fi
[ -n "$scenarios" ] || die "no scenarios matched"

log_info "label=${LABEL} run_id=${RUN_ID}"
log_info "executable=${EXEC}"
log_info "results=${RESULTS_DIR}"

PERF_USABLE=0
if perf_paranoid_ok; then PERF_USABLE=1
else log_warn "perf_event_paranoid > 2 — skipping perf layer"; fi

ran_csv=""
skipped_csv=""
started="$(date -u +%FT%TZ)"

# ── Per-scenario loop ──────────────────────────────────────────────────────
while IFS= read -r scenario <&3; do
    [ -n "$scenario" ] || continue
    log_info "scenario: $scenario"

    # Source scenario in a subshell so its functions/vars don't leak
    scenario_file="${SCRIPT_DIR}/scenarios/${scenario}.sh"

    # Defaults overridable by the scenario
    SCENARIO_WARMUP=3
    SCENARIO_RUNS=50
    SCENARIO_DESCRIPTION=""

    # shellcheck disable=SC1090
    . "$scenario_file"

    # Allow scenario to declare itself unsupported by this binary
    if declare -F scenario_supported >/dev/null; then
        if ! scenario_supported; then
            log_warn "  skipping (scenario reports unsupported)"
            skipped_csv="${skipped_csv}${scenario},"
            unset -f scenario_setup scenario_command scenario_cleanup scenario_supported 2>/dev/null || true
            continue
        fi
    fi

    if declare -F scenario_setup >/dev/null; then scenario_setup; fi

    cmd="$(scenario_command)"
    out_base="${RESULTS_DIR}/scenarios/${scenario}"

    # Layer 1: hyperfine
    hyperfine_run "${out_base}.hyperfine.json" "$SCENARIO_WARMUP" "$SCENARIO_RUNS" "$cmd"

    # Layer 2: /usr/bin/time
    time_run "${out_base}.time.txt" "$cmd"
    time_parse_v "${out_base}.time.txt" > "${out_base}.time.json"

    # Layer 3: perf stat
    if [ "$PERF_USABLE" -eq 1 ]; then
        perf_stat_run "${out_base}.perf-stat.csv" "$cmd"
    fi

    # Optional: perf record
    if [ "$DO_PROFILE" -eq 1 ] && [ "$PERF_USABLE" -eq 1 ]; then
        perf_record_run "${out_base}.perf.data" "$cmd"
    fi

    if declare -F scenario_cleanup >/dev/null; then scenario_cleanup; fi
    unset -f scenario_setup scenario_command scenario_cleanup scenario_supported 2>/dev/null || true

    ran_csv="${ran_csv}${scenario},"
done 3< <(printf '%s' "$scenarios"; echo)

finished="$(date -u +%FT%TZ)"

# ── Summary markdown ───────────────────────────────────────────────────────
summary="${RESULTS_DIR}/summary.md"
{
    printf '# Bench summary — %s\n\n' "$LABEL"
    printf 'Run ID: `%s`\n\n' "$RUN_ID"
    printf 'Executable: `%s`\n\n' "$EXEC"
    printf '| Scenario | Mean (s) | Stddev (s) | Peak RSS (KiB) | Instructions |\n'
    printf '|---|---:|---:|---:|---:|\n'
    for scenario in $(printf '%s' "$ran_csv" | tr ',' '\n'); do
        [ -n "$scenario" ] || continue
        ob="${RESULTS_DIR}/scenarios/${scenario}"
        mean="$(hyperfine_mean_seconds "${ob}.hyperfine.json")"
        sd="$(hyperfine_stddev_seconds "${ob}.hyperfine.json")"
        rss="$(jq -r '.peak_rss_kb' "${ob}.time.json")"
        if [ "$PERF_USABLE" -eq 1 ] && [ -e "${ob}.perf-stat.csv" ]; then
            instr="$(perf_stat_metric "${ob}.perf-stat.csv" instructions)"
        else
            instr="n/a"
        fi
        printf '| %s | %s | %s | %s | %s |\n' "$scenario" "$mean" "$sd" "$rss" "$instr"
    done
} > "$summary"

write_meta "$started" "$finished" "$ran_csv" "$skipped_csv"

log_info "summary written to $summary"
cat "$summary"
