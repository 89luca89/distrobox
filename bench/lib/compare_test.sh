#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/test_helpers.sh"
COMPARE_SCRIPT="${SCRIPT_DIR}/../compare.sh"

# Source compare.sh's helpers without running its main()
COMPARE_SOURCE_ONLY=1
. "$COMPARE_SCRIPT"

test_ratio_simple() {
    local r; r=$(compare_ratio 0.10 0.05)
    assert_eq "0.50" "$r" "ratio 0.05/0.10"
}

test_ratio_handles_zero() {
    local r; r=$(compare_ratio 0 0.05)
    assert_eq "n/a" "$r" "ratio with zero denominator"
}

test_ratio_handles_nan() {
    local r; r=$(compare_ratio null 0.05)
    assert_eq "n/a" "$r" "ratio with null"
}

test_marker_within_noise() {
    local m; m=$(compare_marker 0.100 0.105 0.010)
    assert_eq "within noise" "$m" "Δ smaller than stddev"
}

test_marker_faster() {
    # B is 0.5x A → 2x faster
    local m; m=$(compare_marker 0.10 0.05 0.001)
    assert_contains "faster" "$m" "B faster"
}

test_marker_slower() {
    local m; m=$(compare_marker 0.05 0.10 0.001)
    assert_contains "slower" "$m" "B slower"
}

test_ratio_simple
test_ratio_handles_zero
test_ratio_handles_nan
test_marker_within_noise
test_marker_faster
test_marker_slower
printf 'compare_test.sh: ok\n'
