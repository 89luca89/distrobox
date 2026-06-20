#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
pass=0
fail=0
failing_files=()

for t in "${SCRIPT_DIR}"/lib/*_test.sh; do
    [ -e "$t" ] || continue
    printf '── %s ──\n' "$(basename "$t")"
    if bash "$t"; then
        pass=$((pass + 1))
    else
        fail=$((fail + 1))
        failing_files+=("$(basename "$t")")
    fi
done

printf '\n=== %d passed, %d failed ===\n' "$pass" "$fail"
if [ "$fail" -gt 0 ]; then
    printf 'Failing files:\n'
    for f in "${failing_files[@]}"; do
        printf '  %s\n' "$f"
    done
    exit 1
fi
