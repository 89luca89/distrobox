#!/usr/bin/env bash
# Tiny assertion helpers used by bench/lib/*_test.sh

assert_eq() {
    local expected="$1" actual="$2" msg="${3:-}"
    if [ "$expected" != "$actual" ]; then
        printf 'FAIL %s\n  expected: %q\n  actual:   %q\n' \
            "${msg:-assert_eq}" "$expected" "$actual" >&2
        return 1
    fi
}

assert_contains() {
    local needle="$1" haystack="$2" msg="${3:-}"
    case "$haystack" in
        *"$needle"*) return 0 ;;
        *)
            printf 'FAIL %s\n  needle:   %q\n  haystack: %q\n' \
                "${msg:-assert_contains}" "$needle" "$haystack" >&2
            return 1
            ;;
    esac
}

assert_file_exists() {
    local path="$1" msg="${2:-}"
    if [ ! -e "$path" ]; then
        printf 'FAIL %s\n  missing: %s\n' "${msg:-assert_file_exists}" "$path" >&2
        return 1
    fi
}

assert_exit_code() {
    local expected="$1"; shift
    local actual=0
    "$@" >/dev/null 2>&1 || actual=$?
    if [ "$expected" != "$actual" ]; then
        printf 'FAIL assert_exit_code\n  expected: %s\n  actual:   %s\n  cmd: %s\n' \
            "$expected" "$actual" "$*" >&2
        return 1
    fi
}

mktempdir() {
    mktemp -d "${TMPDIR:-/tmp}/bench-test-XXXXXX"
}
