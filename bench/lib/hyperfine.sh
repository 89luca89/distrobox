#!/usr/bin/env bash
# hyperfine invocation wrapper.

hyperfine_run() {
    local out_json="$1" warmup="$2" runs="$3" cmd="$4"
    hyperfine \
        --warmup "$warmup" \
        --runs "$runs" \
        --export-json "$out_json" \
        --shell=none \
        -- "$cmd" >/dev/null
}

hyperfine_mean_seconds() {
    jq -r '.results[0].mean' "$1"
}

hyperfine_stddev_seconds() {
    jq -r '.results[0].stddev' "$1"
}
