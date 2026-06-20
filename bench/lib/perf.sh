#!/usr/bin/env bash
# perf wrappers for bench/.

PERF_EVENTS="task-clock,cycles,instructions,cache-references,cache-misses,page-faults,context-switches"

perf_paranoid_ok() {
    local v
    v=$(cat /proc/sys/kernel/perf_event_paranoid 2>/dev/null || echo 99)
    [ "$v" -le 2 ]
}

perf_stat_run() {
    local out_csv="$1" cmd="$2"
    perf stat -x, -o "$out_csv" -e "$PERF_EVENTS" -- sh -c "$cmd" >/dev/null 2>&1
}

perf_record_run() {
    local out_data="$1" cmd="$2"
    perf record -F 99 -g --quiet -o "$out_data" -- sh -c "$cmd" >/dev/null 2>&1
}

# perf_stat_metric <csv> <event> → numeric value or "null"
# perf stat -x, format: <value>,<unit>,<event>,<runtime>,<pct>,...
perf_stat_metric() {
    local csv="$1" event="$2"
    awk -F',' -v ev="$event" '
        # skip comment/header lines starting with #
        /^#/ { next }
        $3 == ev { print $1; found=1; exit }
        END { if (!found) print "null" }
    ' "$csv"
}
