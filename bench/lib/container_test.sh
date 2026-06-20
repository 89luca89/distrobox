#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/test_helpers.sh"
. "${SCRIPT_DIR}/common.sh"
. "${SCRIPT_DIR}/container.sh"

test_tracker_init_creates_empty_file() {
    local d; d=$(mktempdir)
    tracker_init "$d/c.list"
    assert_file_exists "$d/c.list" "tracker_init must create file"
    local got; got=$(wc -l < "$d/c.list" | tr -d ' ')
    assert_eq "0" "$got" "tracker file starts empty"
    rm -rf "$d"
}

test_tracker_add_appends() {
    local d; d=$(mktempdir)
    tracker_init "$d/c.list"
    tracker_add "alpha"
    tracker_add "beta"
    local got; got=$(tr '\n' ',' < "$d/c.list")
    assert_eq "alpha,beta," "$got" "tracker_add appends in order"
    rm -rf "$d"
}

test_tracker_cleanup_idempotent_when_empty() {
    local d; d=$(mktempdir)
    tracker_init "$d/c.list"
    assert_exit_code 0 tracker_cleanup
    rm -rf "$d"
}

test_tracker_cleanup_handles_missing_containers() {
    # Names that don't exist must not fail cleanup (podman rm -f is idempotent).
    local d; d=$(mktempdir)
    tracker_init "$d/c.list"
    tracker_add "dbx-bench-nonexistent-aaaaa"
    assert_exit_code 0 tracker_cleanup
    rm -rf "$d"
}

test_tracker_init_creates_empty_file
test_tracker_add_appends
test_tracker_cleanup_idempotent_when_empty
test_tracker_cleanup_handles_missing_containers
printf 'container_test.sh: ok\n'
