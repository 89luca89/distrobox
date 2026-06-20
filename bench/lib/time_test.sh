#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/test_helpers.sh"
. "${SCRIPT_DIR}/common.sh"
. "${SCRIPT_DIR}/time.sh"

FIXTURE="${SCRIPT_DIR}/../fixtures/time-sample.txt"

test_parse_peak_rss() {
    local got
    got=$(time_parse_v "$FIXTURE" | jq -r '.peak_rss_kb')
    assert_eq "2944" "$got" "peak_rss_kb"
}

test_parse_user_seconds() {
    # jq normalises JSON numbers: 0.00 → 0
    local got
    got=$(time_parse_v "$FIXTURE" | jq -r '.user_seconds')
    assert_eq "0" "$got" "user_seconds"
}

test_parse_wall_seconds_from_mm_ss() {
    # 0:00.05 → 0.05 seconds
    local got
    got=$(time_parse_v "$FIXTURE" | jq -r '.wall_seconds')
    assert_eq "0.05" "$got" "wall_seconds"
}

test_parse_minor_faults() {
    local got
    got=$(time_parse_v "$FIXTURE" | jq -r '.minor_page_faults')
    assert_eq "117" "$got" "minor_page_faults"
}

test_parse_voluntary_ctx() {
    local got
    got=$(time_parse_v "$FIXTURE" | jq -r '.voluntary_ctx_switches')
    assert_eq "5" "$got" "voluntary_ctx_switches"
}

test_parse_missing_field_is_null() {
    local d; d=$(mktempdir)
    printf '\tMaximum resident set size (kbytes): 1024\n' > "$d/partial.txt"
    local got
    got=$(time_parse_v "$d/partial.txt" | jq -r '.user_seconds')
    assert_eq "null" "$got" "missing field becomes null"
    rm -rf "$d"
}

test_time_run_captures_output() {
    local d; d=$(mktempdir)
    time_run "$d/out.txt" "true"
    assert_file_exists "$d/out.txt" "time_run writes output"
    assert_contains "Maximum resident set size" "$(cat "$d/out.txt")" "verbose output present"
    rm -rf "$d"
}

test_parse_peak_rss
test_parse_user_seconds
test_parse_wall_seconds_from_mm_ss
test_parse_minor_faults
test_parse_voluntary_ctx
test_parse_missing_field_is_null
test_time_run_captures_output
printf 'time_test.sh: ok\n'
