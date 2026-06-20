#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/test_helpers.sh"
. "${SCRIPT_DIR}/common.sh"

test_scenario_list_sorts_and_strips_suffix() {
    local d
    d=$(mktempdir)
    touch "$d/02-b.sh" "$d/01-a.sh" "$d/10-c.sh" "$d/a_test.sh"
    local got
    got=$(scenario_list "$d" | tr '\n' ',')
    rm -rf "$d"
    assert_eq "01-a,02-b,10-c," "$got" "scenario_list ordering"
}

test_scenario_filter_glob() {
    local got
    got=$(printf '01-a\n02-b\n10-c\n' | scenario_filter '0*' | tr '\n' ',')
    assert_eq "01-a,02-b," "$got" "scenario_filter glob"
}

test_scenario_filter_csv() {
    local got
    got=$(printf '01-a\n02-b\n10-c\n' | scenario_filter '02-b,10-c' | tr '\n' ',')
    assert_eq "02-b,10-c," "$got" "scenario_filter csv"
}

test_die_exits_one() {
    assert_exit_code 1 bash -c '. "'"${SCRIPT_DIR}"'/common.sh"; die "boom"'
}

test_scenario_list_sorts_and_strips_suffix
test_scenario_filter_glob
test_scenario_filter_csv
test_die_exits_one
printf 'common_test.sh: ok\n'
