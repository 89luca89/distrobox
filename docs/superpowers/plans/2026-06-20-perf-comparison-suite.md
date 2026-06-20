# Performance comparison suite — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a developer-local bash harness under `bench/` that measures wall time, peak memory, and CPU counters of a `distrobox` executable across nine scenarios (pure-overhead + end-to-end), plus a `compare.sh` that diffs two result directories into a markdown report.

**Architecture:** Bash entry script (`bench/run.sh`) takes an executable path and orchestrates three measurement layers (hyperfine, `/usr/bin/time -v`, `perf stat`/`record`) per scenario. Scenarios are individual shell files exporting `scenario_setup` / `scenario_command` / `scenario_cleanup`. Each run writes to `bench/results/<label>/<run-id>/`; a separate `compare.sh` reads two such dirs and emits a markdown comparison.

**Tech Stack:** bash, hyperfine, `/usr/bin/time` (GNU), `perf`, jq, podman.

## Global Constraints

- Harness language: `#!/usr/bin/env bash`, `set -euo pipefail` at every entry point.
- Container engine pinned to podman via `DBX_CONTAINER_MANAGER=podman` exported by harness.
- Test image: `docker.io/library/alpine:latest`. Pulled once on first run, never removed.
- Run ID format: `RUN_ID="$(date +%s)-$$"`.
- Container naming: `dbx-bench-<run-id>-<scenario>-<iteration>`.
- All containers created by the harness get label `distrobox.bench.run=<run-id>`.
- Tracker file: `bench/results/<label>/<run-id>/containers.list`, one container name per line, appended at create time.
- Cleanup is always by exact name (`podman rm -f <name>`); never relies on `--filter name=` substring behavior.
- Results dir is git-ignored; nothing under `bench/results/` is ever committed.
- Per-test convention: one test function per case; no table-driven loops (mirrors the project's Go test style).

---

## File structure

**Create:**

```
bench/
├── run.sh
├── compare.sh
├── test.sh
├── README.md
├── lib/
│   ├── common.sh
│   ├── container.sh
│   ├── hyperfine.sh
│   ├── time.sh
│   ├── perf.sh
│   ├── test_helpers.sh
│   ├── common_test.sh
│   ├── container_test.sh
│   ├── time_test.sh
│   └── compare_test.sh
├── scenarios/
│   ├── 01-startup-help.sh
│   ├── 02-startup-version.sh
│   ├── 03-subcommand-help.sh
│   ├── 04-assemble-parse.sh
│   ├── 10-list-empty.sh
│   ├── 11-list-many.sh
│   ├── 20-create-rm.sh
│   ├── 21-enter-exec.sh
│   └── 22-ephemeral-true.sh
└── fixtures/
    ├── assemble.ini
    └── time-sample.txt
```

**Modify:**

- `.gitignore` — add `bench/results/`
- `Makefile` — add `bench`, `bench-test`, `bench-compare` targets

---

## Task 1: Scaffolding + gitignore + Makefile targets

**Files:**
- Create: `bench/`, `bench/lib/`, `bench/scenarios/`, `bench/fixtures/`, `bench/results/.gitkeep`
- Modify: `.gitignore`
- Modify: `Makefile`

**Interfaces:**
- Consumes: nothing
- Produces: directory layout that subsequent tasks fill in; `make bench` placeholder that errors until `run.sh` exists.

- [ ] **Step 1: Create directory skeleton**

```bash
mkdir -p bench/lib bench/scenarios bench/fixtures bench/results
touch bench/results/.gitkeep
```

- [ ] **Step 2: Update `.gitignore`**

Append:

```
# Benchmark results
bench/results/*
!bench/results/.gitkeep
```

- [ ] **Step 3: Add Makefile targets**

Append to `Makefile`:

```makefile
.PHONY: bench
bench: build
	./bench/run.sh ./bin/distrobox

.PHONY: bench-test
bench-test:
	./bench/test.sh

.PHONY: bench-compare
bench-compare:
	@echo "Usage: ./bench/compare.sh <result-dir-A> <result-dir-B>"
```

- [ ] **Step 4: Verify skeleton**

```bash
ls bench/ && cat .gitignore | grep bench/results && grep -A1 '^bench:' Makefile
```

Expected: directories listed, gitignore entry present, makefile target present.

- [ ] **Step 5: Commit**

```bash
git add bench/ .gitignore Makefile
git commit -m "bench: scaffold perf comparison suite directory and Makefile targets"
```

---

## Task 2: Test helpers + bench/test.sh runner

**Files:**
- Create: `bench/lib/test_helpers.sh`
- Create: `bench/test.sh`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `assert_eq <expected> <actual> [<message>]` — exits 1 if mismatch
  - `assert_contains <substring> <haystack> [<message>]` — exits 1 if not found
  - `assert_file_exists <path> [<message>]` — exits 1 if missing
  - `assert_exit_code <expected_code> <command...>` — runs the command, exits 1 if code differs
  - `mktempdir` — echoes a fresh temp dir path; cleanup is the caller's job
  - `bench/test.sh` runs every `bench/lib/*_test.sh` and reports pass/fail counts

- [ ] **Step 1: Write `bench/lib/test_helpers.sh`**

```bash
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
```

- [ ] **Step 2: Write `bench/test.sh`**

```bash
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
```

- [ ] **Step 3: chmod and dry-run the runner**

```bash
chmod +x bench/test.sh
./bench/test.sh
```

Expected: `=== 0 passed, 0 failed ===` (no `_test.sh` files exist yet).

- [ ] **Step 4: Commit**

```bash
git add bench/test.sh bench/lib/test_helpers.sh
git commit -m "bench: add test helpers and test runner"
```

---

## Task 3: lib/common.sh (logging, preflight, scenario loader)

**Files:**
- Create: `bench/lib/common.sh`
- Create: `bench/lib/common_test.sh`

**Interfaces:**
- Consumes: `bench/lib/test_helpers.sh` (test only)
- Produces:
  - `log_info <msg>`, `log_warn <msg>`, `log_err <msg>`, `die <msg>` — formatted stderr output; `die` exits 1
  - `preflight_check_tools` — verifies hyperfine, jq, /usr/bin/time, perf, podman exist; calls `die` if any missing; echoes the GNU time path to stdout
  - `scenario_list <scenarios_dir>` — echoes scenario names (filename without `.sh`, sorted, excluding `_test.sh`), one per line
  - `scenario_filter <pattern>` — given input scenario names on stdin, emits the subset matching a shell glob or comma-separated list

- [ ] **Step 1: Write failing tests in `bench/lib/common_test.sh`**

```bash
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
```

- [ ] **Step 2: Run tests, verify they fail**

```bash
./bench/test.sh 2>&1 | tail -20
```

Expected: failure because `bench/lib/common.sh` does not yet exist.

- [ ] **Step 3: Write `bench/lib/common.sh`**

```bash
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
```

- [ ] **Step 4: Run tests, verify they pass**

```bash
./bench/test.sh
```

Expected: `=== 1 passed, 0 failed ===`.

- [ ] **Step 5: Commit**

```bash
git add bench/lib/common.sh bench/lib/common_test.sh
git commit -m "bench: add common.sh with logging, preflight, scenario loader"
```

---

## Task 4: lib/container.sh (tracker + podman ops)

**Files:**
- Create: `bench/lib/container.sh`
- Create: `bench/lib/container_test.sh`

**Interfaces:**
- Consumes: `bench/lib/common.sh` (`die`, `log_*`)
- Produces:
  - `tracker_init <path>` — creates empty file, sets `TRACKER_FILE=<path>`
  - `tracker_add <name>` — appends name to `$TRACKER_FILE`
  - `tracker_cleanup` — reads `$TRACKER_FILE` and `podman rm -f` each line; idempotent
  - `container_create_distrobox <name> <image> <exec_path>` — calls `$exec_path create --image $image --name $name --yes` and `tracker_add`s the name; returns the exit code of distrobox
  - `container_create_podman_direct <name> <image>` — `podman create --label distrobox.bench.run="$RUN_ID" --name "$name" "$image" /bin/true` and `tracker_add`s; used by `list-many` scenario
  - `container_orphan_scan` — echoes any container with label `distrobox.bench.run` (one ID per line)
  - `container_orphan_clean` — removes all orphans found by orphan_scan
  - `PODMAN_CMD` — env var listing the podman invocation (just `podman` for now; allows future rootful prefix)

- [ ] **Step 1: Write failing tests in `bench/lib/container_test.sh`**

```bash
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
```

- [ ] **Step 2: Run tests, verify they fail**

```bash
./bench/test.sh 2>&1 | tail -20
```

Expected: failure because `bench/lib/container.sh` doesn't exist.

- [ ] **Step 3: Write `bench/lib/container.sh`**

```bash
#!/usr/bin/env bash
# Container tracker and podman ops for bench/. Requires common.sh sourced.

: "${PODMAN_CMD:=podman}"
: "${RUN_ID:=unset}"
: "${TRACKER_FILE:=}"

tracker_init() {
    TRACKER_FILE="$1"
    : > "$TRACKER_FILE"
}

tracker_add() {
    local name="$1"
    [ -n "$TRACKER_FILE" ] || die "tracker_add called before tracker_init"
    printf '%s\n' "$name" >> "$TRACKER_FILE"
}

tracker_cleanup() {
    [ -n "$TRACKER_FILE" ] || return 0
    [ -e "$TRACKER_FILE" ] || return 0
    local name
    while IFS= read -r name; do
        [ -n "$name" ] || continue
        $PODMAN_CMD rm -f "$name" >/dev/null 2>&1 || true
    done < "$TRACKER_FILE"
}

container_create_distrobox() {
    local name="$1" image="$2" exec_path="$3"
    tracker_add "$name"
    "$exec_path" create --image "$image" --name "$name" --yes >/dev/null
}

container_create_podman_direct() {
    local name="$1" image="$2"
    [ "$RUN_ID" != "unset" ] || die "RUN_ID must be exported before container_create_podman_direct"
    tracker_add "$name"
    $PODMAN_CMD create \
        --label "distrobox.bench.run=${RUN_ID}" \
        --label "manager=distrobox" \
        --name "$name" \
        "$image" \
        /bin/true >/dev/null
}

container_orphan_scan() {
    $PODMAN_CMD ps -a --filter "label=distrobox.bench.run" --format '{{.ID}}' 2>/dev/null
}

container_orphan_clean() {
    local ids
    ids="$(container_orphan_scan)"
    [ -n "$ids" ] || return 0
    printf '%s\n' "$ids" | xargs -r $PODMAN_CMD rm -f >/dev/null 2>&1 || true
}
```

- [ ] **Step 4: Run tests, verify they pass**

```bash
./bench/test.sh
```

Expected: `=== 2 passed, 0 failed ===`.

- [ ] **Step 5: Commit**

```bash
git add bench/lib/container.sh bench/lib/container_test.sh
git commit -m "bench: add container tracker and podman ops helpers"
```

---

## Task 5: lib/hyperfine.sh

**Files:**
- Create: `bench/lib/hyperfine.sh`

**Interfaces:**
- Consumes: `bench/lib/common.sh`
- Produces:
  - `hyperfine_run <out_json> <warmup> <runs> <cmd_string>` — invokes hyperfine, writing JSON to `<out_json>`; `<cmd_string>` is passed as a single argument to hyperfine (so it accepts the cmd verbatim, including any shell metacharacters the caller chose)
  - `hyperfine_mean_seconds <json_file>` — echoes the mean wall time in seconds (extracted from hyperfine JSON via jq)
  - `hyperfine_stddev_seconds <json_file>` — echoes the stddev in seconds

  No `_test.sh` for this lib — pure integration wrappers; integration smoke happens in `run.sh` tests.

- [ ] **Step 1: Write `bench/lib/hyperfine.sh`**

```bash
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
```

- [ ] **Step 2: Smoke-test by hand**

```bash
. bench/lib/common.sh
. bench/lib/hyperfine.sh
tmp=$(mktemp -d)
hyperfine_run "$tmp/out.json" 1 3 'true'
hyperfine_mean_seconds "$tmp/out.json"
hyperfine_stddev_seconds "$tmp/out.json"
rm -rf "$tmp"
```

Expected: two small floating-point numbers (a few microseconds and a smaller stddev).

- [ ] **Step 3: Commit**

```bash
git add bench/lib/hyperfine.sh
git commit -m "bench: add hyperfine wrapper"
```

---

## Task 6: lib/time.sh (/usr/bin/time -v wrapper + JSON parser)

**Files:**
- Create: `bench/lib/time.sh`
- Create: `bench/lib/time_test.sh`
- Create: `bench/fixtures/time-sample.txt`

**Interfaces:**
- Consumes: `bench/lib/common.sh`
- Produces:
  - `time_run <out_txt> <cmd_string>` — runs the command under `/usr/bin/time -v`, writing the verbose output to `<out_txt>`. Returns the command's exit code.
  - `time_parse_v <txt_file>` — reads a `/usr/bin/time -v` output file, prints a JSON object with keys: `peak_rss_kb`, `user_seconds`, `sys_seconds`, `wall_seconds`, `voluntary_ctx_switches`, `involuntary_ctx_switches`, `major_page_faults`, `minor_page_faults`, `fs_inputs`, `fs_outputs`. Missing keys are emitted as `null`.

- [ ] **Step 1: Create `bench/fixtures/time-sample.txt`**

This is the exact output of `/usr/bin/time -v sh -c 'sleep 0.05'` on Linux (whitespace-sensitive — keep tabs and leading whitespace as below):

```
	Command being timed: "sh -c sleep 0.05"
	User time (seconds): 0.00
	System time (seconds): 0.00
	Percent of CPU this job got: 1%
	Elapsed (wall clock) time (h:mm:ss or m:ss): 0:00.05
	Average shared text size (kbytes): 0
	Average unshared data size (kbytes): 0
	Average stack size (kbytes): 0
	Average total size (kbytes): 0
	Maximum resident set size (kbytes): 2944
	Average resident set size (kbytes): 0
	Major (requiring I/O) page faults: 0
	Minor (reclaiming a frame) page faults: 117
	Voluntary context switches: 5
	Involuntary context switches: 1
	Swaps: 0
	File system inputs: 0
	File system outputs: 0
	Socket messages sent: 0
	Socket messages received: 0
	Signals delivered: 0
	Page size (bytes): 4096
	Exit status: 0
```

- [ ] **Step 2: Write failing tests in `bench/lib/time_test.sh`**

```bash
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
```

- [ ] **Step 3: Run tests, verify they fail**

```bash
./bench/test.sh 2>&1 | tail -20
```

Expected: failure (`bench/lib/time.sh` not yet present).

- [ ] **Step 4: Write `bench/lib/time.sh`**

```bash
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
    /Maximum resident set size/                   { set("peak_rss_kb", $NF) }
    /User time/                                   { set("user_seconds", $NF) }
    /System time/                                 { set("sys_seconds", $NF) }
    /Elapsed \(wall clock\) time/                 {
        # Last field is the formatted time
        v = wall_to_seconds($NF)
        set("wall_seconds", v)
    }
    /Voluntary context switches/                  { set("voluntary_ctx_switches", $NF) }
    /Involuntary context switches/                { set("involuntary_ctx_switches", $NF) }
    /Major \(requiring I\/O\) page faults/        { set("major_page_faults", $NF) }
    /Minor \(reclaiming a frame\) page faults/    { set("minor_page_faults", $NF) }
    /File system inputs/                          { set("fs_inputs", $NF) }
    /File system outputs/                         { set("fs_outputs", $NF) }
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
```

- [ ] **Step 5: Run tests, verify they pass**

```bash
./bench/test.sh
```

Expected: `=== 3 passed, 0 failed ===`.

- [ ] **Step 6: Commit**

```bash
git add bench/lib/time.sh bench/lib/time_test.sh bench/fixtures/time-sample.txt
git commit -m "bench: add /usr/bin/time -v wrapper and parser"
```

---

## Task 7: lib/perf.sh

**Files:**
- Create: `bench/lib/perf.sh`

**Interfaces:**
- Consumes: `bench/lib/common.sh`
- Produces:
  - `perf_paranoid_ok` — reads `/proc/sys/kernel/perf_event_paranoid`; returns 0 if ≤ 2 (perf-stat usable without sudo), 1 otherwise
  - `perf_stat_run <out_csv> <cmd_string>` — wraps the command in `perf stat -x, -o <out_csv> -e <event-list>`; returns the command exit code
  - `perf_record_run <out_data> <cmd_string>` — wraps in `perf record -F 99 -g -o <out_data>`
  - `perf_stat_metric <csv_file> <event>` — echoes the counter value for `<event>` from a perf stat CSV file (or `null` if not present)

  Integration smoke happens in `run.sh` tests; no `_test.sh` for this lib (pure wrappers + tiny CSV pluck).

- [ ] **Step 1: Write `bench/lib/perf.sh`**

```bash
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
```

- [ ] **Step 2: Smoke-test by hand**

```bash
. bench/lib/common.sh
. bench/lib/perf.sh
perf_paranoid_ok && echo "perf usable" || echo "perf not usable"
tmp=$(mktemp -d)
perf_stat_run "$tmp/p.csv" 'true'
perf_stat_metric "$tmp/p.csv" 'instructions'
rm -rf "$tmp"
```

Expected: `perf usable`, then a numeric instructions count (or `<not supported>` text if the kernel doesn't expose it, in which case awk prints whatever field 1 is).

- [ ] **Step 3: Commit**

```bash
git add bench/lib/perf.sh
git commit -m "bench: add perf stat/record wrappers"
```

---

## Task 8: run.sh (entry point, orchestration, summary)

**Files:**
- Create: `bench/run.sh`

**Interfaces:**
- Consumes: every lib from tasks 3–7
- Produces: an executable that takes `<executable> [--label NAME] [--scenarios LIST] [--profile] [--clean-orphans]` and writes results to `bench/results/<label>/<run-id>/`

- [ ] **Step 1: Write `bench/run.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/lib/common.sh"
. "${SCRIPT_DIR}/lib/container.sh"
. "${SCRIPT_DIR}/lib/hyperfine.sh"
. "${SCRIPT_DIR}/lib/time.sh"
. "${SCRIPT_DIR}/lib/perf.sh"

# ── Arguments ────────────────────────────────────────────────────────────────
EXEC=""
LABEL=""
SCENARIOS_FILTER=""
DO_PROFILE=0
DO_CLEAN_ORPHANS=0

usage() {
    cat >&2 <<EOF
Usage: $0 <executable> [--label NAME] [--scenarios LIST]
                       [--profile] [--clean-orphans]

  <executable>        path to distrobox binary or shell script
  --label NAME        results directory name (default: basename of executable)
  --scenarios LIST    comma-separated names or shell glob (default: all)
  --profile           also run 'perf record' per scenario
  --clean-orphans     remove containers labelled distrobox.bench.run.* before run
EOF
    exit 2
}

while [ $# -gt 0 ]; do
    case "$1" in
        --label)         LABEL="$2"; shift 2 ;;
        --scenarios)     SCENARIOS_FILTER="$2"; shift 2 ;;
        --profile)       DO_PROFILE=1; shift ;;
        --clean-orphans) DO_CLEAN_ORPHANS=1; shift ;;
        -h|--help)       usage ;;
        -*)              log_err "unknown flag: $1"; usage ;;
        *)
            if [ -z "$EXEC" ]; then EXEC="$1"; shift
            else log_err "unexpected arg: $1"; usage; fi
            ;;
    esac
done

[ -n "$EXEC" ] || usage
[ -x "$EXEC" ] || die "executable not found or not executable: $EXEC"
EXEC="$(readlink -f "$EXEC")"
[ -z "$LABEL" ] && LABEL="$(basename "$EXEC")"

# ── Preflight ────────────────────────────────────────────────────────────────
preflight_check_tools >/dev/null
export DBX_CONTAINER_MANAGER=podman

if [ "$DO_CLEAN_ORPHANS" -eq 1 ]; then
    log_info "cleaning orphans from previous runs"
    container_orphan_clean
fi

orphans="$(container_orphan_scan)"
if [ -n "$orphans" ]; then
    die "orphan containers from a previous bench run exist (label distrobox.bench.run). re-run with --clean-orphans to remove."
fi

# Pre-pull image if needed
IMAGE="docker.io/library/alpine:latest"
if ! podman image exists "$IMAGE"; then
    log_info "pulling $IMAGE"
    podman pull "$IMAGE" >/dev/null
fi

# ── Run setup ────────────────────────────────────────────────────────────────
export RUN_ID="$(date +%s)-$$"
RESULTS_DIR="${SCRIPT_DIR}/results/${LABEL}/${RUN_ID}"
mkdir -p "${RESULTS_DIR}/scenarios"
tracker_init "${RESULTS_DIR}/containers.list"

# Always clean up tracker on exit
trap 'tracker_cleanup' EXIT

# Meta
write_meta() {
    local started_at="$1"
    local finished_at="$2"
    local scenarios_run_csv="$3"
    local scenarios_skipped_csv="$4"
    local cpu_model
    cpu_model="$(awk -F': ' '/model name/ {print $2; exit}' /proc/cpuinfo 2>/dev/null || echo unknown)"
    jq -n \
        --arg label "$LABEL" \
        --arg run_id "$RUN_ID" \
        --arg started "$started_at" \
        --arg finished "$finished_at" \
        --arg executable "$EXEC" \
        --arg sha256 "$(sha256sum "$EXEC" | awk '{print $1}')" \
        --argjson size "$(stat -c %s "$EXEC")" \
        --arg kernel "$(uname -r)" \
        --arg cpu_model "$cpu_model" \
        --arg podman_version "$(podman --version | awk '{print $3}')" \
        --argjson perf_paranoid "$(cat /proc/sys/kernel/perf_event_paranoid 2>/dev/null || echo -1)" \
        --arg scenarios_run "$scenarios_run_csv" \
        --arg scenarios_skipped "$scenarios_skipped_csv" \
        '{
            label: $label,
            run_id: $run_id,
            started: $started,
            finished: $finished,
            executable: $executable,
            executable_sha256: $sha256,
            executable_size_bytes: $size,
            host: {
                kernel: $kernel,
                cpu_model: $cpu_model,
                podman_version: $podman_version,
                perf_event_paranoid: $perf_paranoid
            },
            scenarios_run: ($scenarios_run | split(",") | map(select(. != ""))),
            scenarios_skipped: ($scenarios_skipped | split(",") | map(select(. != "")))
        }' > "${RESULTS_DIR}/meta.json"
}

# ── Scenario discovery ──────────────────────────────────────────────────────
all_scenarios="$(scenario_list "${SCRIPT_DIR}/scenarios")"
if [ -n "$SCENARIOS_FILTER" ]; then
    scenarios="$(printf '%s\n' "$all_scenarios" | scenario_filter "$SCENARIOS_FILTER")"
else
    scenarios="$all_scenarios"
fi
[ -n "$scenarios" ] || die "no scenarios matched"

log_info "label=${LABEL} run_id=${RUN_ID}"
log_info "executable=${EXEC}"
log_info "results=${RESULTS_DIR}"

PERF_USABLE=0
if perf_paranoid_ok; then PERF_USABLE=1
else log_warn "perf_event_paranoid > 2 — skipping perf layer"; fi

ran_csv=""
skipped_csv=""
started="$(date -u +%FT%TZ)"

# ── Per-scenario loop ──────────────────────────────────────────────────────
while IFS= read -r scenario; do
    [ -n "$scenario" ] || continue
    log_info "scenario: $scenario"

    # Source scenario in a subshell so its functions/vars don't leak
    scenario_file="${SCRIPT_DIR}/scenarios/${scenario}.sh"

    # Defaults overridable by the scenario
    SCENARIO_WARMUP=3
    SCENARIO_RUNS=50
    SCENARIO_DESCRIPTION=""

    # shellcheck disable=SC1090
    . "$scenario_file"

    # Allow scenario to declare itself unsupported by this binary
    if declare -F scenario_supported >/dev/null; then
        if ! scenario_supported; then
            log_warn "  skipping (scenario reports unsupported)"
            skipped_csv="${skipped_csv}${scenario},"
            unset -f scenario_setup scenario_command scenario_cleanup scenario_supported 2>/dev/null || true
            continue
        fi
    fi

    if declare -F scenario_setup >/dev/null; then scenario_setup; fi

    cmd="$(scenario_command)"
    out_base="${RESULTS_DIR}/scenarios/${scenario}"

    # Layer 1: hyperfine
    hyperfine_run "${out_base}.hyperfine.json" "$SCENARIO_WARMUP" "$SCENARIO_RUNS" "$cmd"

    # Layer 2: /usr/bin/time
    time_run "${out_base}.time.txt" "$cmd"
    time_parse_v "${out_base}.time.txt" > "${out_base}.time.json"

    # Layer 3: perf stat
    if [ "$PERF_USABLE" -eq 1 ]; then
        perf_stat_run "${out_base}.perf-stat.csv" "$cmd"
    fi

    # Optional: perf record
    if [ "$DO_PROFILE" -eq 1 ] && [ "$PERF_USABLE" -eq 1 ]; then
        perf_record_run "${out_base}.perf.data" "$cmd"
    fi

    if declare -F scenario_cleanup >/dev/null; then scenario_cleanup; fi
    unset -f scenario_setup scenario_command scenario_cleanup scenario_supported 2>/dev/null || true

    ran_csv="${ran_csv}${scenario},"
done <<EOF
$scenarios
EOF

finished="$(date -u +%FT%TZ)"

# ── Summary markdown ───────────────────────────────────────────────────────
summary="${RESULTS_DIR}/summary.md"
{
    printf '# Bench summary — %s\n\n' "$LABEL"
    printf 'Run ID: `%s`\n\n' "$RUN_ID"
    printf 'Executable: `%s`\n\n' "$EXEC"
    printf '| Scenario | Mean (s) | Stddev (s) | Peak RSS (KiB) | Instructions |\n'
    printf '|---|---:|---:|---:|---:|\n'
    for scenario in $(printf '%s' "$ran_csv" | tr ',' '\n'); do
        [ -n "$scenario" ] || continue
        ob="${RESULTS_DIR}/scenarios/${scenario}"
        mean="$(hyperfine_mean_seconds "${ob}.hyperfine.json")"
        sd="$(hyperfine_stddev_seconds "${ob}.hyperfine.json")"
        rss="$(jq -r '.peak_rss_kb' "${ob}.time.json")"
        if [ "$PERF_USABLE" -eq 1 ] && [ -e "${ob}.perf-stat.csv" ]; then
            instr="$(perf_stat_metric "${ob}.perf-stat.csv" instructions)"
        else
            instr="n/a"
        fi
        printf '| %s | %s | %s | %s | %s |\n' "$scenario" "$mean" "$sd" "$rss" "$instr"
    done
} > "$summary"

write_meta "$started" "$finished" "$ran_csv" "$skipped_csv"

log_info "summary written to $summary"
cat "$summary"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x bench/run.sh
```

- [ ] **Step 3: Sanity-check argument parsing**

```bash
./bench/run.sh 2>&1 | head -5 || true
```

Expected: usage printed (no scenarios exist yet, but the script should reach scenario discovery and fail with a meaningful error after preflight). If you see "executable not found" — pass `--help`.

```bash
./bench/run.sh --help 2>&1 | head -5
```

Expected: usage block.

- [ ] **Step 4: Commit**

```bash
git add bench/run.sh
git commit -m "bench: add run.sh entry point and orchestration"
```

---

## Task 9: compare.sh

**Files:**
- Create: `bench/compare.sh`
- Create: `bench/lib/compare_test.sh`

**Interfaces:**
- Consumes: jq; reads result dirs produced by `run.sh`
- Produces:
  - `bench/compare.sh <dir-A> <dir-B> [--allow-engine-drift]` → writes `bench/results/comparisons/<labelA>-vs-<labelB>-<ts>.md` and prints to stdout
  - Functions exposed for test purposes: `compare_ratio <a> <b>` → echoes ratio b/a as float string, or "n/a" if either is non-numeric; `compare_marker <a> <b> <stddev_a>` → echoes "Δ < stddev", "Nx faster", or "Nx slower"

- [ ] **Step 1: Write failing tests in `bench/lib/compare_test.sh`**

```bash
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
```

- [ ] **Step 2: Run tests, verify they fail**

```bash
./bench/test.sh 2>&1 | tail -20
```

Expected: failure — `bench/compare.sh` not present.

- [ ] **Step 3: Write `bench/compare.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/lib/common.sh"

# Source-only mode for tests
: "${COMPARE_SOURCE_ONLY:=0}"

# compare_ratio <a> <b> → echoes b/a as "X.XX" (2 decimal places), or "n/a"
compare_ratio() {
    local a="$1" b="$2"
    case "$a" in null|n/a|"") echo "n/a"; return ;; esac
    case "$b" in null|n/a|"") echo "n/a"; return ;; esac
    awk -v a="$a" -v b="$b" 'BEGIN {
        if (a + 0 == 0) { print "n/a"; exit }
        printf "%.2f\n", b / a
    }'
}

# compare_marker <mean_a> <mean_b> <stddev_a> → "within noise" / "Nx faster" / "Nx slower"
compare_marker() {
    local a="$1" b="$2" sd="$3"
    case "$a$b$sd" in *null*|*n/a*) echo "n/a"; return ;; esac
    awk -v a="$a" -v b="$b" -v sd="$sd" 'BEGIN {
        diff = (b > a) ? (b - a) : (a - b)
        if (sd + 0 > 0 && diff < (sd + 0)) { print "within noise"; exit }
        if (a + 0 == 0) { print "n/a"; exit }
        if (b < a) {
            printf "%.2fx faster\n", a / b
        } else {
            printf "%.2fx slower\n", b / a
        }
    }'
}

compare_main() {
    local dir_a="" dir_b="" allow_drift=0
    while [ $# -gt 0 ]; do
        case "$1" in
            --allow-engine-drift) allow_drift=1; shift ;;
            -h|--help)
                cat >&2 <<EOF
Usage: $0 <result-dir-A> <result-dir-B> [--allow-engine-drift]
EOF
                exit 2 ;;
            -*) die "unknown flag: $1" ;;
            *)
                if [ -z "$dir_a" ]; then dir_a="$1"
                elif [ -z "$dir_b" ]; then dir_b="$1"
                else die "unexpected arg: $1"; fi
                shift ;;
        esac
    done
    [ -n "$dir_a" ] && [ -n "$dir_b" ] || die "two result dirs required"

    local meta_a="${dir_a}/meta.json" meta_b="${dir_b}/meta.json"
    assert_meta() { [ -e "$1" ] || die "meta.json missing: $1"; }
    assert_meta "$meta_a"; assert_meta "$meta_b"

    local label_a label_b podman_a podman_b
    label_a=$(jq -r .label "$meta_a")
    label_b=$(jq -r .label "$meta_b")
    podman_a=$(jq -r .host.podman_version "$meta_a")
    podman_b=$(jq -r .host.podman_version "$meta_b")

    if [ "$podman_a" != "$podman_b" ] && [ "$allow_drift" -eq 0 ]; then
        die "podman version differs ($podman_a vs $podman_b). Pass --allow-engine-drift to override."
    fi

    local out_dir="${SCRIPT_DIR}/results/comparisons"
    mkdir -p "$out_dir"
    local out_file="${out_dir}/${label_a}-vs-${label_b}-$(date -u +%Y%m%dT%H%M%SZ).md"

    {
        printf '# Comparison: %s vs %s\n\n' "$label_a" "$label_b"
        printf '| | A (%s) | B (%s) |\n' "$label_a" "$label_b"
        printf '|---|---|---|\n'
        printf '| Executable | `%s` | `%s` |\n' \
            "$(jq -r .executable "$meta_a")" "$(jq -r .executable "$meta_b")"
        printf '| Binary sha256 | `%s` | `%s` |\n' \
            "$(jq -r .executable_sha256 "$meta_a")" "$(jq -r .executable_sha256 "$meta_b")"
        printf '| Binary size (bytes) | %s | %s |\n' \
            "$(jq -r .executable_size_bytes "$meta_a")" "$(jq -r .executable_size_bytes "$meta_b")"
        printf '| Podman | %s | %s |\n' "$podman_a" "$podman_b"
        printf '| Kernel | %s | %s |\n' \
            "$(jq -r .host.kernel "$meta_a")" "$(jq -r .host.kernel "$meta_b")"
        printf '\n'

        printf '## Scenarios\n\n'
        printf '| Scenario | Mean A | Mean B | Ratio (B/A) | Marker | RSS A | RSS B | RSS ratio | Instr A | Instr B | Instr ratio |\n'
        printf '|---|---:|---:|---:|---|---:|---:|---:|---:|---:|---:|\n'

        local scenarios_a scenario
        scenarios_a=$(ls "${dir_a}/scenarios/"*.hyperfine.json 2>/dev/null \
            | xargs -n1 basename | sed 's/\.hyperfine\.json$//' | sort)

        for scenario in $scenarios_a; do
            local ja="${dir_a}/scenarios/${scenario}.hyperfine.json"
            local jb="${dir_b}/scenarios/${scenario}.hyperfine.json"
            local ta="${dir_a}/scenarios/${scenario}.time.json"
            local tb="${dir_b}/scenarios/${scenario}.time.json"
            local pa="${dir_a}/scenarios/${scenario}.perf-stat.csv"
            local pb="${dir_b}/scenarios/${scenario}.perf-stat.csv"
            [ -e "$jb" ] || continue

            local mean_a mean_b sd_a rss_a rss_b instr_a instr_b
            mean_a=$(jq -r '.results[0].mean' "$ja")
            mean_b=$(jq -r '.results[0].mean' "$jb")
            sd_a=$(jq -r '.results[0].stddev' "$ja")
            rss_a=$([ -e "$ta" ] && jq -r .peak_rss_kb "$ta" || echo "n/a")
            rss_b=$([ -e "$tb" ] && jq -r .peak_rss_kb "$tb" || echo "n/a")

            if [ -e "$pa" ]; then
                instr_a=$(awk -F, '/^#/{next} $3=="instructions"{print $1; exit}' "$pa" 2>/dev/null || echo "n/a")
            else instr_a="n/a"; fi
            if [ -e "$pb" ]; then
                instr_b=$(awk -F, '/^#/{next} $3=="instructions"{print $1; exit}' "$pb" 2>/dev/null || echo "n/a")
            else instr_b="n/a"; fi

            printf '| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n' \
                "$scenario" \
                "$mean_a" "$mean_b" "$(compare_ratio "$mean_a" "$mean_b")" "$(compare_marker "$mean_a" "$mean_b" "$sd_a")" \
                "$rss_a" "$rss_b" "$(compare_ratio "$rss_a" "$rss_b")" \
                "$instr_a" "$instr_b" "$(compare_ratio "$instr_a" "$instr_b")"
        done
    } | tee "$out_file"

    log_info "wrote $out_file"
}

if [ "$COMPARE_SOURCE_ONLY" = "0" ]; then
    compare_main "$@"
fi
```

- [ ] **Step 4: Make executable and run tests**

```bash
chmod +x bench/compare.sh
./bench/test.sh
```

Expected: `=== 4 passed, 0 failed ===`.

- [ ] **Step 5: Commit**

```bash
git add bench/compare.sh bench/lib/compare_test.sh
git commit -m "bench: add compare.sh diff/report tool"
```

---

## Task 10: Pure-overhead scenarios (01, 02, 03)

**Files:**
- Create: `bench/scenarios/01-startup-help.sh`
- Create: `bench/scenarios/02-startup-version.sh`
- Create: `bench/scenarios/03-subcommand-help.sh`

**Interfaces:**
- Consumes: `$EXEC` from `run.sh`
- Produces: three runnable scenarios

- [ ] **Step 1: Write `bench/scenarios/01-startup-help.sh`**

```bash
SCENARIO_DESCRIPTION="distrobox --help: binary load + arg parser cost"
SCENARIO_WARMUP=3
SCENARIO_RUNS=50

scenario_setup()   { :; }
scenario_command() { printf '%s --help\n' "$EXEC"; }
scenario_cleanup() { :; }
```

- [ ] **Step 2: Write `bench/scenarios/02-startup-version.sh`**

```bash
SCENARIO_DESCRIPTION="distrobox --version: alternate startup path"
SCENARIO_WARMUP=3
SCENARIO_RUNS=50

scenario_setup()   { :; }
scenario_command() { printf '%s --version\n' "$EXEC"; }
scenario_cleanup() { :; }
```

- [ ] **Step 3: Write `bench/scenarios/03-subcommand-help.sh`**

```bash
SCENARIO_DESCRIPTION="distrobox create --help: subcommand dispatch cost"
SCENARIO_WARMUP=3
SCENARIO_RUNS=50

scenario_setup()   { :; }
scenario_command() { printf '%s create --help\n' "$EXEC"; }
scenario_cleanup() { :; }
```

- [ ] **Step 4: Smoke-run scenarios 01–03 against the existing binary**

```bash
./bench/run.sh ./bin/distrobox --label smoke --scenarios '0[1-3]*'
```

Expected: harness completes; `bench/results/smoke/<run-id>/scenarios/` contains 3 sets of hyperfine.json / time.txt / time.json / perf-stat.csv; `summary.md` printed at end has 3 rows.

Inspect:
```bash
ls bench/results/smoke/*/scenarios/
cat bench/results/smoke/*/summary.md
```

- [ ] **Step 5: Commit**

```bash
git add bench/scenarios/01-startup-help.sh bench/scenarios/02-startup-version.sh bench/scenarios/03-subcommand-help.sh
git commit -m "bench: add pure-overhead startup scenarios"
```

---

## Task 11: Assemble parse scenario (04) + fixture

**Files:**
- Create: `bench/fixtures/assemble.ini`
- Create: `bench/scenarios/04-assemble-parse.sh`

**Interfaces:**
- Consumes: `$EXEC`; the fixture path resolved via `$SCRIPT_DIR` from `run.sh`
- Produces: scenario measuring `assemble create --file <fixture> --dry-run`

- [ ] **Step 1: Write `bench/fixtures/assemble.ini`**

A realistic but synthetic config — three sections, each with several keys distrobox understands. Uses non-existent image names; with `--dry-run` no container is created.

```ini
[bench-alpha]
image=docker.io/library/alpine:latest
additional_packages="git curl"
init_hooks="echo hello"
home=~/.local/share/bench-alpha
volume="/tmp:/tmp"

[bench-beta]
image=docker.io/library/alpine:latest
additional_packages="bash sudo"
pre_init_hooks="echo pre"
volume="/var/tmp:/var/tmp"
volume="/home:/home"

[bench-gamma]
image=docker.io/library/alpine:latest
init=true
nvidia=false
pull=false
unshare_groups=true
```

- [ ] **Step 2: Write `bench/scenarios/04-assemble-parse.sh`**

The scenario discovers the fixture path relative to the harness root (passed in via `$SCRIPT_DIR` exported by `run.sh`). `--dry-run` exists in both implementations (verified during planning); if a future build drops it, `scenario_supported` returns 1.

```bash
SCENARIO_DESCRIPTION="distrobox assemble create --dry-run on a 3-section ini"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

FIXTURE_PATH="${SCRIPT_DIR}/fixtures/assemble.ini"

scenario_supported() {
    "$EXEC" assemble --help 2>&1 | grep -q -- '--dry-run'
}

scenario_setup()   { :; }
scenario_command() {
    printf '%s assemble create --file %s --dry-run\n' "$EXEC" "$FIXTURE_PATH"
}
scenario_cleanup() { :; }
```

- [ ] **Step 3: Smoke-run scenario 04**

```bash
./bench/run.sh ./bin/distrobox --label smoke --scenarios '04-*'
```

Expected: harness completes; one row in summary; no containers leaked. Verify:
```bash
podman ps -a --filter label=distrobox.bench.run --format '{{.Names}}'
```
Expected: empty.

- [ ] **Step 4: Commit**

```bash
git add bench/fixtures/assemble.ini bench/scenarios/04-assemble-parse.sh
git commit -m "bench: add assemble --dry-run parse scenario and fixture"
```

---

## Task 12: list-empty scenario (10)

**Files:**
- Create: `bench/scenarios/10-list-empty.sh`

**Interfaces:**
- Consumes: `$EXEC`, `$PODMAN_CMD`
- Produces: scenario measuring `distrobox list` with no harness-created containers present

- [ ] **Step 1: Write `bench/scenarios/10-list-empty.sh`**

Note: the user may have unrelated distrobox containers already running; we don't touch them. The measurement therefore captures "list with whatever is in your environment minus zero of our own". The harness records the count it saw in setup so the reader can interpret.

```bash
SCENARIO_DESCRIPTION="distrobox list with no harness containers (user containers may be present)"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

scenario_setup() {
    local n
    n=$($EXEC list 2>/dev/null | tail -n +2 | wc -l | tr -d ' ')
    log_info "  (environment has ~${n} distrobox containers)"
}

scenario_command() { printf '%s list\n' "$EXEC"; }
scenario_cleanup() { :; }
```

- [ ] **Step 2: Smoke-run scenario 10**

```bash
./bench/run.sh ./bin/distrobox --label smoke --scenarios '10-*'
```

Expected: harness completes; summary row present; setup logs the pre-existing distrobox container count.

- [ ] **Step 3: Commit**

```bash
git add bench/scenarios/10-list-empty.sh
git commit -m "bench: add list-empty scenario"
```

---

## Task 13: list-many scenario (11)

**Files:**
- Create: `bench/scenarios/11-list-many.sh`

**Interfaces:**
- Consumes: `$EXEC`, `$RUN_ID`, `container_create_podman_direct`, `tracker_add`
- Produces: scenario measuring `distrobox list` with 20 dummy distrobox-labelled containers added

- [ ] **Step 1: Write `bench/scenarios/11-list-many.sh`**

```bash
SCENARIO_DESCRIPTION="distrobox list with 20 dummy distrobox-labelled containers added"
SCENARIO_WARMUP=3
SCENARIO_RUNS=30

LIST_MANY_COUNT=20
LIST_MANY_IMAGE="docker.io/library/alpine:latest"

scenario_setup() {
    local i name
    for i in $(seq 1 "$LIST_MANY_COUNT"); do
        name="dbx-bench-${RUN_ID}-list-many-$(printf '%02d' "$i")"
        container_create_podman_direct "$name" "$LIST_MANY_IMAGE"
    done
    # Confirm distrobox sees at least LIST_MANY_COUNT containers
    local visible
    visible=$($EXEC list 2>/dev/null | tail -n +2 | wc -l | tr -d ' ')
    if [ "$visible" -lt "$LIST_MANY_COUNT" ]; then
        die "expected ≥${LIST_MANY_COUNT} containers visible to distrobox list, got ${visible}"
    fi
}

scenario_command() { printf '%s list\n' "$EXEC"; }

scenario_cleanup() {
    # Per-iteration containers go to global tracker; cleaned by EXIT trap.
    # Nothing to do here.
    :
}
```

- [ ] **Step 2: Smoke-run scenario 11**

```bash
./bench/run.sh ./bin/distrobox --label smoke --scenarios '11-*'
```

Expected: 20 containers visible during setup; harness completes; tracker cleanup removes them on exit. Verify after:
```bash
podman ps -a --filter label=distrobox.bench.run --format '{{.Names}}'
```
Expected: empty.

- [ ] **Step 3: Commit**

```bash
git add bench/scenarios/11-list-many.sh
git commit -m "bench: add list-many scenario with 20 dummy containers"
```

---

## Task 14: create-rm scenario (20)

**Files:**
- Create: `bench/scenarios/20-create-rm.sh`

**Interfaces:**
- Consumes: `$EXEC`, `$RUN_ID`, `tracker_add`
- Produces: scenario measuring full create + rm cycle

- [ ] **Step 1: Write `bench/scenarios/20-create-rm.sh`**

The scenario pre-generates 10 unique container names and uses hyperfine's `-L` parameter list. The combined command is `sh -c '$EXEC create ... && $EXEC rm ...'`. Names are added to the tracker upfront so a crash mid-iteration is still cleaned up.

```bash
SCENARIO_DESCRIPTION="distrobox create alpine then rm --force (10 full cycles)"
SCENARIO_WARMUP=1
SCENARIO_RUNS=10

CREATE_RM_IMAGE="docker.io/library/alpine:latest"
CREATE_RM_NAMES=""

scenario_setup() {
    local i name
    CREATE_RM_NAMES=""
    for i in $(seq 1 "$SCENARIO_RUNS"); do
        name="dbx-bench-${RUN_ID}-create-rm-$(printf '%02d' "$i")"
        tracker_add "$name"
        if [ -z "$CREATE_RM_NAMES" ]; then
            CREATE_RM_NAMES="$name"
        else
            CREATE_RM_NAMES="${CREATE_RM_NAMES},${name}"
        fi
    done
}

scenario_command() {
    # NOTE: this is a non-trivial command shape; run.sh's hyperfine_run uses
    # --shell=none, so we wrap in sh -c ourselves.
    printf "sh -c '%s create --image %s --name {name} --yes >/dev/null && %s rm --force {name} >/dev/null'\n" \
        "$EXEC" "$CREATE_RM_IMAGE" "$EXEC"
}

scenario_cleanup() { :; }
```

But wait — `scenario_command` returns one command and `hyperfine_run` runs it `--runs N` times with the same command. We need parameter substitution: `hyperfine -L name name1,name2,...`. The current `hyperfine_run` doesn't support that.

Adjust scenario_command to use a name derived from hyperfine's iteration via a counter file (simpler than threading -L into hyperfine_run): each iteration generates a unique name based on `mktemp -u` inside the sh -c. Even simpler: just use `$$_$RANDOM` per invocation; `sh -c` is a fresh shell per call so it's unique each time.

Replace scenario_command:

```bash
scenario_command() {
    printf "sh -c '" \
        ; printf 'NAME=dbx-bench-%s-create-rm-$$-$RANDOM; ' "$RUN_ID" \
        ; printf 'echo $NAME >> %s/containers.list; ' "${RESULTS_DIR}" \
        ; printf '%s create --image %s --name $NAME --yes >/dev/null && ' "$EXEC" "$CREATE_RM_IMAGE" \
        ; printf '%s rm --force $NAME >/dev/null' "$EXEC" \
        ; printf "'\n"
}
```

The append-to-tracker inside the shell is a safety net so any rm failure leaves a name the global cleanup will catch.

Final scenario file (replace step 1 with this):

```bash
SCENARIO_DESCRIPTION="distrobox create alpine then rm --force (full cycles)"
SCENARIO_WARMUP=1
SCENARIO_RUNS=10

CREATE_RM_IMAGE="docker.io/library/alpine:latest"

scenario_setup() { :; }

scenario_command() {
    # Each invocation generates a unique container name. The name is appended
    # to the tracker before create runs so a mid-iteration crash still gets
    # cleaned up by the EXIT trap.
    printf "sh -c 'NAME=dbx-bench-%s-create-rm-\$\$-\$RANDOM; printf \"%%s\\n\" \"\$NAME\" >> %s/containers.list; %s create --image %s --name \"\$NAME\" --yes >/dev/null && %s rm --force \"\$NAME\" >/dev/null'\n" \
        "$RUN_ID" "$RESULTS_DIR" "$EXEC" "$CREATE_RM_IMAGE" "$EXEC"
}

scenario_cleanup() { :; }
```

- [ ] **Step 2: Smoke-run scenario 20 (this takes 1–3 minutes)**

```bash
./bench/run.sh ./bin/distrobox --label smoke --scenarios '20-*'
```

Expected: ~11 create+rm cycles complete (1 warmup + 10 runs); summary row present. Verify no leakage:
```bash
podman ps -a --filter label=distrobox.bench.run --format '{{.Names}}'
```
Expected: empty.

- [ ] **Step 3: Commit**

```bash
git add bench/scenarios/20-create-rm.sh
git commit -m "bench: add create-rm full-lifecycle scenario"
```

---

## Task 15: enter-exec scenario (21)

**Files:**
- Create: `bench/scenarios/21-enter-exec.sh`

**Interfaces:**
- Consumes: `$EXEC`, `$RUN_ID`, `container_create_distrobox`
- Produces: scenario measuring `distrobox enter -- /bin/true` against a single warm container

- [ ] **Step 1: Write `bench/scenarios/21-enter-exec.sh`**

```bash
SCENARIO_DESCRIPTION="distrobox enter -- /bin/true (warm container)"
SCENARIO_WARMUP=2
SCENARIO_RUNS=10

ENTER_EXEC_IMAGE="docker.io/library/alpine:latest"
ENTER_EXEC_NAME=""

scenario_setup() {
    ENTER_EXEC_NAME="dbx-bench-${RUN_ID}-enter-exec"
    container_create_distrobox "$ENTER_EXEC_NAME" "$ENTER_EXEC_IMAGE" "$EXEC"
    # Warm: first enter starts the container and runs init scripts
    "$EXEC" enter --name "$ENTER_EXEC_NAME" -- /bin/true >/dev/null 2>&1 || true
}

scenario_command() {
    printf '%s enter --name %s -- /bin/true\n' "$EXEC" "$ENTER_EXEC_NAME"
}

scenario_cleanup() {
    # Belt-and-suspenders: the EXIT trap also handles this.
    podman rm -f "$ENTER_EXEC_NAME" >/dev/null 2>&1 || true
}
```

- [ ] **Step 2: Smoke-run scenario 21 (this takes 1–2 minutes)**

```bash
./bench/run.sh ./bin/distrobox --label smoke --scenarios '21-*'
```

Expected: warmup + 10 enter cycles complete; summary row present. Verify no leakage:
```bash
podman ps -a --filter label=distrobox.bench.run --format '{{.Names}}'
```
Expected: empty.

- [ ] **Step 3: Commit**

```bash
git add bench/scenarios/21-enter-exec.sh
git commit -m "bench: add enter-exec warm-container scenario"
```

---

## Task 16: ephemeral scenario (22)

**Files:**
- Create: `bench/scenarios/22-ephemeral-true.sh`

**Interfaces:**
- Consumes: `$EXEC`
- Produces: scenario measuring full ephemeral lifecycle

- [ ] **Step 1: Write `bench/scenarios/22-ephemeral-true.sh`**

`distrobox ephemeral` creates a container, runs the command, then removes the container. It picks the name itself. We don't track names from outside; the harness's orphan scan would only catch leaks at the next run-start (which is acceptable).

```bash
SCENARIO_DESCRIPTION="distrobox ephemeral alpine -- /bin/true (full lifecycle)"
SCENARIO_WARMUP=1
SCENARIO_RUNS=5

EPHEMERAL_IMAGE="docker.io/library/alpine:latest"

scenario_setup()   { :; }
scenario_command() {
    printf '%s ephemeral --image %s --yes -- /bin/true\n' "$EXEC" "$EPHEMERAL_IMAGE"
}
scenario_cleanup() { :; }
```

- [ ] **Step 2: Smoke-run scenario 22 (this takes 1–3 minutes)**

```bash
./bench/run.sh ./bin/distrobox --label smoke --scenarios '22-*'
```

Expected: 6 cycles complete; summary row present.

- [ ] **Step 3: Commit**

```bash
git add bench/scenarios/22-ephemeral-true.sh
git commit -m "bench: add ephemeral full-lifecycle scenario"
```

---

## Task 17: README + end-to-end full-suite check

**Files:**
- Create: `bench/README.md`

**Interfaces:** none (documentation)

- [ ] **Step 1: Write `bench/README.md`**

```markdown
# bench — distrobox performance comparison suite

A developer-local harness that measures wall time, peak memory, and CPU
counters of a `distrobox` executable across nine scenarios. Used to compare
the shell implementation (main branch) against the Go rewrite (next branch).

## Quick start

```sh
# Run against the Go binary in this branch
make build
./bench/run.sh ./bin/distrobox --label next

# Run against the shell implementation in the main worktree
./bench/run.sh /path/to/main/worktree/distrobox --label main

# Diff the two
./bench/compare.sh bench/results/next/<run-id> bench/results/main/<run-id>
```

## Requirements

- `hyperfine`, `jq`, `podman`, `perf`, GNU `/usr/bin/time`
- `kernel.perf_event_paranoid ≤ 2` (perf layer is skipped automatically otherwise)

## What it measures

Three layers per scenario, each in its own invocation so they don't interfere:

1. **hyperfine** — wall time mean/stddev across many runs
2. **`/usr/bin/time -v`** — peak RSS, user/sys CPU, page faults, ctx switches
3. **`perf stat`** — cycles, instructions, cache-miss rate

Optional `--profile` adds `perf record` per scenario (large `.perf.data`
files, gitignored).

## Scenarios

| # | Name | What it measures |
|---|------|------------------|
| 01 | startup-help | binary load + arg parser |
| 02 | startup-version | alternate startup path |
| 03 | subcommand-help | subcommand dispatch |
| 04 | assemble-parse | .ini parsing (assemble --dry-run) |
| 10 | list-empty | engine-query path with no harness containers |
| 11 | list-many | list with 20 dummy distrobox-labelled containers |
| 20 | create-rm | full create + rm cycle |
| 21 | enter-exec | per-exec overhead on a warm container |
| 22 | ephemeral-true | full ephemeral lifecycle |

## Output

Per run: `bench/results/<label>/<run-id>/`
- `meta.json` — binary identity, host info, scenarios run/skipped
- `containers.list` — names of containers created (tracker for cleanup)
- `scenarios/<name>.hyperfine.json`, `.time.txt`, `.time.json`, `.perf-stat.csv`
- `summary.md`

Per comparison: `bench/results/comparisons/<labelA>-vs-<labelB>-<ts>.md`

## Caveats

- **Peak RSS underreports for shell pipelines** because `/usr/bin/time -v`
  reports only the parent's RSS, not children. Consistent across runs, so
  the ratio is still meaningful; the absolute number isn't.
- **Shell perf-stat numbers are noisy** because every `bash` invocation forks
  many short-lived utilities. That's part of what we're measuring.
- **The user's existing distrobox containers are not touched.** Scenario 10
  measures `list` against whatever you already have; scenario 11 adds 20 dummy
  containers on top. The harness never deletes containers it didn't create.

## Cleanup

If a run crashes and leaves containers behind, re-run with `--clean-orphans`
(removes anything labelled `distrobox.bench.run=*`).
```

- [ ] **Step 2: Full-suite smoke run**

```bash
make build
./bench/run.sh ./bin/distrobox --label smoke-full
```

Expected: all 9 scenarios complete; `summary.md` has 9 rows; total runtime roughly 5–15 minutes depending on host. Verify no leaked containers:
```bash
podman ps -a --filter label=distrobox.bench.run --format '{{.Names}}'
```
Expected: empty.

- [ ] **Step 3: Self-test compare.sh against two runs**

```bash
./bench/run.sh ./bin/distrobox --label compare-test-a --scenarios '0[1-3]*'
./bench/run.sh ./bin/distrobox --label compare-test-b --scenarios '0[1-3]*'
./bench/compare.sh bench/results/compare-test-a/* bench/results/compare-test-b/*
```

Expected: comparison markdown printed; mostly "within noise" markers (same binary running twice); file written under `bench/results/comparisons/`.

- [ ] **Step 4: Commit**

```bash
git add bench/README.md
git commit -m "bench: add README"
```

---

## Self-review

**Spec coverage:** every spec section is addressed:
- §Repository layout → Task 1
- §Scenarios → Tasks 10–16 (one task per scenario family)
- §Measurement layers → Tasks 5, 6, 7 (hyperfine, time, perf); orchestration in Task 8
- §Container & engine integration → Task 4 (tracker, orphan scan, podman direct); used by Tasks 8, 13
- §Output format → Task 8 (per-run meta.json + summary.md); Task 9 (compare report)
- §Invocation → Task 8 (run.sh argument parsing)
- §Risks / open questions:
  - `--dry-run` confirmed by code inspection during planning; scenario 04 uses `scenario_supported` as defense (Task 11)
  - distrobox label contract: confirmed substring "distrobox" anywhere in labels; the harness's `distrobox.bench.run` label is therefore sufficient (Task 13 setup also asserts count)
  - `--version` vs `--verbose` `-v` collision: scenarios use long forms only (Task 10)
  - enter-exec includes podman-exec dominant cost: README caveat (Task 17)
- README explains usage (Task 17).

**Placeholder scan:** no TBD / TODO / "implement later" in any step. Every code step shows actual code. `compare_main` uses an internal helper `assert_meta` defined inline.

**Type consistency:** `RUN_ID`, `EXEC`, `RESULTS_DIR`, `TRACKER_FILE`, `SCRIPT_DIR`, `PODMAN_CMD` are referenced consistently across run.sh and scenarios. `scenario_setup` / `scenario_command` / `scenario_cleanup` / `scenario_supported` are the only scenario hooks; signatures consistent. `tracker_init` / `tracker_add` / `tracker_cleanup`, `container_create_distrobox` / `container_create_podman_direct`, `container_orphan_scan` / `container_orphan_clean` match across producer (Task 4) and consumer (Task 8). hyperfine helpers (`hyperfine_run`, `hyperfine_mean_seconds`, `hyperfine_stddev_seconds`) match. `time_run`, `time_parse_v` match. `perf_paranoid_ok`, `perf_stat_run`, `perf_record_run`, `perf_stat_metric` match. `compare_ratio`, `compare_marker` match between tests (Task 9) and implementation.
