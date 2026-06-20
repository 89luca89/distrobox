#!/usr/bin/env bash
# /usr/bin/time -v wrapper and parser.

time_run() {
    local out_txt="$1" cmd="$2"
    /usr/bin/time -v -o "$out_txt" sh -c "$cmd"
}

# Parse a /usr/bin/time -v output file into a JSON object.
# Fields that don't appear in the input become null.
time_parse_v() {
    local file="$1"
    awk '
    BEGIN {
        keys["peak_rss_kb"] = "null"
        keys["user_seconds"] = "null"
        keys["sys_seconds"] = "null"
        keys["wall_seconds"] = "null"
        keys["voluntary_ctx_switches"] = "null"
        keys["involuntary_ctx_switches"] = "null"
        keys["major_page_faults"] = "null"
        keys["minor_page_faults"] = "null"
        keys["fs_inputs"] = "null"
        keys["fs_outputs"] = "null"
    }
    function set(k, v) { keys[k] = v }
    function wall_to_seconds(s,    parts, n, h, m, sec) {
        # Accept h:mm:ss(.frac) or m:ss(.frac)
        n = split(s, parts, ":")
        if (n == 3)      { return parts[1]*3600 + parts[2]*60 + parts[3] + 0 }
        else if (n == 2) { return parts[1]*60 + parts[2] + 0 }
        else             { return s + 0 }
    }
    /Maximum resident set size/                   { set("peak_rss_kb", $NF + 0) }
    /User time/                                   { set("user_seconds", $NF + 0) }
    /System time/                                 { set("sys_seconds", $NF + 0) }
    /Elapsed \(wall clock\) time/                 {
        # Last field is the formatted time
        v = wall_to_seconds($NF)
        set("wall_seconds", v)
    }
    /Voluntary context switches/                  { set("voluntary_ctx_switches", $NF + 0) }
    /Involuntary context switches/                { set("involuntary_ctx_switches", $NF + 0) }
    /Major \(requiring I\/O\) page faults/        { set("major_page_faults", $NF + 0) }
    /Minor \(reclaiming a frame\) page faults/    { set("minor_page_faults", $NF + 0) }
    /File system inputs/                          { set("fs_inputs", $NF + 0) }
    /File system outputs/                         { set("fs_outputs", $NF + 0) }
    END {
        # Emit JSON. Numbers are emitted bare (jq -r will keep them as-is),
        # null literal stays null.
        printf "{"
        sep = ""
        # Stable key order:
        order = "peak_rss_kb user_seconds sys_seconds wall_seconds " \
                "voluntary_ctx_switches involuntary_ctx_switches " \
                "major_page_faults minor_page_faults fs_inputs fs_outputs"
        n = split(order, ord, " ")
        for (i = 1; i <= n; i++) {
            k = ord[i]
            v = keys[k]
            if (v == "null") {
                printf "%s\"%s\":null", sep, k
            } else {
                # User/system times are like "0.00"; keep as numeric literal
                printf "%s\"%s\":%s", sep, k, v
            }
            sep = ","
        }
        printf "}\n"
    }
    ' "$file"
}
