#!/usr/bin/env bash
# Logging, preflight, scenario loading helpers for bench/.

log_info() { printf '[INFO]  %s\n' "$*" >&2; }
log_warn() { printf '[WARN]  %s\n' "$*" >&2; }
log_err()  { printf '[ERROR] %s\n' "$*" >&2; }
die()      { log_err "$*"; exit 1; }

# Find GNU time. /usr/bin/time is the conventional path on Linux.
# Returns the path on stdout; dies if not GNU time (must support -v).
preflight_check_tools() {
    local missing=()
    for tool in hyperfine jq podman perf; do
        command -v "$tool" >/dev/null 2>&1 || missing+=("$tool")
    done
    [ -x /usr/bin/time ] || missing+=("/usr/bin/time")
    if [ ${#missing[@]} -gt 0 ]; then
        die "missing required tools: ${missing[*]}"
    fi
    # Verify GNU time supports -v
    if ! /usr/bin/time -v true >/dev/null 2>&1; then
        die "/usr/bin/time does not support -v (need GNU time)"
    fi
    printf '/usr/bin/time\n'
}

# scenario_list <dir> — print scenario base names (sorted, excluding *_test.sh)
scenario_list() {
    local dir="$1"
    [ -d "$dir" ] || die "scenario dir not found: $dir"
    local f base
    for f in "$dir"/*.sh; do
        [ -e "$f" ] || continue
        base="$(basename "$f" .sh)"
        case "$base" in
            *_test) continue ;;
        esac
        printf '%s\n' "$base"
    done | sort
}

# scenario_filter <pattern> — filters stdin scenario names by glob or csv
scenario_filter() {
    local pattern="$1"
    local name match item
    while IFS= read -r name; do
        match=0
        case "$pattern" in
            *,*)
                # CSV
                local IFS=','
                for item in $pattern; do
                    if [ "$name" = "$item" ]; then match=1; break; fi
                done
                ;;
            *)
                # Glob
                # shellcheck disable=SC2254
                case "$name" in
                    $pattern) match=1 ;;
                esac
                ;;
        esac
        if [ "$match" -eq 1 ]; then
            printf '%s\n' "$name"
        fi
    done
}
