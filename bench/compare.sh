#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "${SCRIPT_DIR}/lib/common.sh"
. "${SCRIPT_DIR}/lib/perf.sh"

# Source-only mode for tests
: "${COMPARE_SOURCE_ONLY:=0}"

# compare_ratio <a> <b> → echoes b/a as "X.XX" (2 decimal places), or "n/a"
compare_ratio() {
    local a="$1" b="$2"
    case "$a" in null|n/a|"") echo "n/a"; return ;; esac
    case "$b" in null|n/a|"") echo "n/a"; return ;; esac
    awk -v a="$a" -v b="$b" 'BEGIN {
        if (a + 0 == 0) { print "n/a"; exit }
        printf "%.2f\n", b / a
    }'
}

# compare_marker <mean_a> <mean_b> <stddev_a> → "within noise" / "Nx faster" / "Nx slower"
compare_marker() {
    local a="$1" b="$2" sd="$3"
    case "$a$b$sd" in *null*|*n/a*) echo "n/a"; return ;; esac
    awk -v a="$a" -v b="$b" -v sd="$sd" 'BEGIN {
        diff = (b > a) ? (b - a) : (a - b)
        if (sd + 0 > 0 && diff < (sd + 0)) { print "within noise"; exit }
        if (a + 0 == 0) { print "n/a"; exit }
        if (b < a) {
            printf "%.2fx faster\n", a / b
        } else {
            printf "%.2fx slower\n", b / a
        }
    }'
}

compare_main() {
    local dir_a="" dir_b="" allow_drift=0
    while [ $# -gt 0 ]; do
        case "$1" in
            --allow-engine-drift) allow_drift=1; shift ;;
            -h|--help)
                cat >&2 <<EOF
Usage: $0 <result-dir-A> <result-dir-B> [--allow-engine-drift]
EOF
                exit 2 ;;
            -*) die "unknown flag: $1" ;;
            *)
                if [ -z "$dir_a" ]; then dir_a="$1"
                elif [ -z "$dir_b" ]; then dir_b="$1"
                else die "unexpected arg: $1"; fi
                shift ;;
        esac
    done
    [ -n "$dir_a" ] && [ -n "$dir_b" ] || die "two result dirs required"

    local meta_a="${dir_a}/meta.json" meta_b="${dir_b}/meta.json"
    [ -e "$meta_a" ] || die "meta.json missing: $meta_a"
    [ -e "$meta_b" ] || die "meta.json missing: $meta_b"

    local label_a label_b podman_a podman_b
    label_a=$(jq -r .label "$meta_a")
    label_b=$(jq -r .label "$meta_b")
    podman_a=$(jq -r .host.podman_version "$meta_a")
    podman_b=$(jq -r .host.podman_version "$meta_b")

    if [ "$podman_a" != "$podman_b" ] && [ "$allow_drift" -eq 0 ]; then
        die "podman version differs ($podman_a vs $podman_b). Pass --allow-engine-drift to override."
    fi

    local out_dir="${SCRIPT_DIR}/results/comparisons"
    mkdir -p "$out_dir"
    local out_file="${out_dir}/${label_a}-vs-${label_b}-$(date -u +%Y%m%dT%H%M%SZ).md"

    {
        printf '# Comparison: %s vs %s\n\n' "$label_a" "$label_b"
        printf '| | A (%s) | B (%s) |\n' "$label_a" "$label_b"
        printf '|---|---|---|\n'
        printf '| Executable | `%s` | `%s` |\n' \
            "$(jq -r .executable "$meta_a")" "$(jq -r .executable "$meta_b")"
        printf '| Binary sha256 | `%s` | `%s` |\n' \
            "$(jq -r .executable_sha256 "$meta_a")" "$(jq -r .executable_sha256 "$meta_b")"
        printf '| Binary size (bytes) | %s | %s |\n' \
            "$(jq -r .executable_size_bytes "$meta_a")" "$(jq -r .executable_size_bytes "$meta_b")"
        printf '| Podman | %s | %s |\n' "$podman_a" "$podman_b"
        printf '| Kernel | %s | %s |\n' \
            "$(jq -r .host.kernel "$meta_a")" "$(jq -r .host.kernel "$meta_b")"
        printf '\n'

        printf '## Scenarios\n\n'
        printf '| Scenario | Mean A | Mean B | Ratio (B/A) | Marker | RSS A | RSS B | RSS ratio | Instr A | Instr B | Instr ratio |\n'
        printf '|---|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|\n'

        local scenarios_a scenario
        scenarios_a=$(ls "${dir_a}/scenarios/"*.hyperfine.json 2>/dev/null \
            | xargs -n1 basename | sed 's/\.hyperfine\.json$//' | sort)

        for scenario in $scenarios_a; do
            local ja="${dir_a}/scenarios/${scenario}.hyperfine.json"
            local jb="${dir_b}/scenarios/${scenario}.hyperfine.json"
            local ta="${dir_a}/scenarios/${scenario}.time.json"
            local tb="${dir_b}/scenarios/${scenario}.time.json"
            local pa="${dir_a}/scenarios/${scenario}.perf-stat.csv"
            local pb="${dir_b}/scenarios/${scenario}.perf-stat.csv"
            [ -e "$jb" ] || continue

            local mean_a mean_b sd_a rss_a rss_b instr_a instr_b
            mean_a=$(jq -r '.results[0].mean' "$ja")
            mean_b=$(jq -r '.results[0].mean' "$jb")
            sd_a=$(jq -r '.results[0].stddev' "$ja")
            rss_a=$([ -e "$ta" ] && jq -r .peak_rss_kb "$ta" || echo "n/a")
            rss_b=$([ -e "$tb" ] && jq -r .peak_rss_kb "$tb" || echo "n/a")

            if [ -e "$pa" ]; then
                instr_a=$(perf_stat_metric "$pa" instructions)
            else instr_a="n/a"; fi
            if [ -e "$pb" ]; then
                instr_b=$(perf_stat_metric "$pb" instructions)
            else instr_b="n/a"; fi

            printf '| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n' \
                "$scenario" \
                "$mean_a" "$mean_b" "$(compare_ratio "$mean_a" "$mean_b")" "$(compare_marker "$mean_a" "$mean_b" "$sd_a")" \
                "$rss_a" "$rss_b" "$(compare_ratio "$rss_a" "$rss_b")" \
                "$instr_a" "$instr_b" "$(compare_ratio "$instr_a" "$instr_b")"
        done
    } | tee "$out_file"

    log_info "wrote $out_file"
}

if [ "$COMPARE_SOURCE_ONLY" = "0" ]; then
    compare_main "$@"
fi
