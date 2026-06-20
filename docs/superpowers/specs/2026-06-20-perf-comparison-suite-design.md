# Performance comparison suite (shell vs Go distrobox)

## Goal

Build a developer-local performance test suite that measures wall time, peak
memory, and CPU counters of a `distrobox` executable across a fixed set of
scenarios. The suite takes the executable path as an argument so the same
scenarios can be run against either the shell implementation (`main` branch's
`./distrobox`) or the Go rewrite (`next` branch's `./bin/distrobox`), producing
result directories that a separate comparison tool diffs into a markdown
report.

Out of scope: functional/behavioral testing (covered by `tests/compare.sh`),
CI integration, multi-host comparison, flamegraph rendering, Go pprof
profiling.

## Constraints / decisions already made

- **Scope:** pure-overhead scenarios AND end-to-end container lifecycle scenarios.
- **Profiling depth:** black-box (hyperfine, `/usr/bin/time -v`, strace optional)
  plus `perf stat` / `perf record` on both implementations. No language-specific
  profilers (no pprof, no `bash -x`).
- **Container engine:** podman only, pinned via `DBX_CONTAINER_MANAGER=podman`.
- **Test image:** `docker.io/library/alpine:latest`. Pre-pulled once, never
  removed by the suite.
- **Harness language:** POSIX shell (`/bin/sh`), mirroring the existing
  `tests/compare.sh` style. Tools assumed present: `hyperfine`, `jq`,
  `/usr/bin/time` (GNU), `perf`, `podman`.

## Repository layout

```
bench/
├── run.sh            # entry point: ./bench/run.sh <executable> [opts]
├── compare.sh        # ./bench/compare.sh <result-dir-A> <result-dir-B>
├── lib/
│   ├── hyperfine.sh  # hyperfine wrapper, JSON export
│   ├── time.sh       # /usr/bin/time -v wrapper + parser to JSON
│   ├── perf.sh       # perf stat + perf record wrappers
│   ├── container.sh  # podman setup/cleanup, tracker file ops
│   └── common.sh     # logging, scenario loader, preflight
├── scenarios/
│   ├── 01-startup-help.sh
│   ├── 02-startup-version.sh
│   ├── 03-subcommand-help.sh
│   ├── 04-assemble-parse.sh    # may be a no-op if --dry-run not supported; see §Open
│   ├── 10-list-empty.sh
│   ├── 11-list-many.sh
│   ├── 20-create-rm.sh
│   ├── 21-enter-exec.sh
│   └── 22-ephemeral-true.sh
├── fixtures/
│   └── assemble.ini
└── results/
    ├── <label>/<run-id>/...
    └── comparisons/<labelA>-vs-<labelB>-<timestamp>.md
```

`bench/results/` is gitignored.

## Scenarios

Each scenario file under `bench/scenarios/` defines three shell functions:

- `scenario_setup` — runs once before any measurement; can pre-create containers
  via `container_create_tracked` (defined in `lib/container.sh`)
- `scenario_command` — the command whose execution is measured. Echoed to
  stdout as a single line that `hyperfine` / `time` / `perf` consume.
- `scenario_cleanup` — runs once after all measurements; typically calls
  `container_cleanup_tracked`

Scenarios may also export `SCENARIO_WARMUP`, `SCENARIO_RUNS`, and
`SCENARIO_DESCRIPTION` variables read by the harness.

### Pure overhead

| # | Name | Command | What it isolates |
|---|------|---------|------------------|
| 01 | `startup-help` | `$EXEC --help` | Binary load + arg parser cost |
| 02 | `startup-version` | `$EXEC --version` | Same cost class, distinct codepath |
| 03 | `subcommand-help` | `$EXEC create --help` | Cost of dispatching into a subcommand |
| 04 | `assemble-parse` | `$EXEC assemble create --file fixtures/assemble.ini --dry-run` (if supported, else dropped — see §Open questions) | Pure `.ini` parsing cost |

Iteration target: `--warmup 3 --min-runs 50`.

### Engine-touching, no container created

| # | Name | Command | What it isolates |
|---|------|---------|------------------|
| 10 | `list-empty` | `$EXEC list` with zero `dbx-bench-*` containers present | Cost of engine-query path on empty result |
| 11 | `list-many` | `$EXEC list` with 20 pre-existing dummy distrobox-labelled containers (created by the harness via `podman create` directly to keep setup fast) | Cost of formatting/filtering many rows |

Iteration target: `--warmup 3 --min-runs 30`.

### Full lifecycle

| # | Name | Per-iteration | What it measures |
|---|------|---------------|------------------|
| 20 | `create-rm` | `create --image alpine --name $NAME` then `rm --force $NAME` | Full create+destroy cost |
| 21 | `enter-exec` | `enter --name bench -- /bin/true` against a container pre-created in setup | Per-exec overhead (most user-visible metric) |
| 22 | `ephemeral-true` | `ephemeral --image alpine -- /bin/true` | Full ephemeral lifecycle |

Iteration target: `--warmup 1 --runs 10` (per-iteration cost too high for 50).

## Measurement layers

Each scenario is invoked three times, once per layer, in this order, so layers
don't interfere with each other (notably `perf` can perturb timing):

| Layer | Tool & flags | Output | Captures |
|-------|--------------|--------|----------|
| Wall time | `hyperfine --export-json <out> --warmup N --runs M -- '<cmd>'` | `<scenario>.hyperfine.json` | mean, stddev, median, min, max wall clock |
| Resource accounting | `/usr/bin/time -v -o <out> <cmd>` (single run; see note) | `<scenario>.time.txt` (raw) + `.time.json` (parsed by `lib/time.sh`) | peak RSS, user/sys CPU, page faults, voluntary/involuntary context switches, FS inputs/outputs |
| CPU counters | `perf stat -x, -o <out> -e task-clock,cycles,instructions,cache-references,cache-misses,page-faults,context-switches <cmd>` (single run; see note) | `<scenario>.perf-stat.csv` | total instructions, IPC, cache-miss rate, syscall pressure proxies |
| CPU sampling (opt-in via `--profile`) | `perf record -F 99 -g -o <out> <cmd>` | `<scenario>.perf.data` (large, gitignored) | flamegraph-able call samples |

**Why single-run for time/perf-stat:** hyperfine already characterises
wall-time variance across many runs. Peak RSS, instructions, cache-miss rate,
and the other counters are dominated by deterministic per-invocation costs and
are typically stable across runs; the extra runs aren't worth the wall-clock
budget. If a scenario reveals instability, it can be re-run by hand under
`time -v` / `perf stat` to confirm.

Caveats called out in the suite's README and printed at run start:

- **`perf_event_paranoid`**: `perf stat` and user-space `perf record` work
  without sudo when the value is ≤ 2. The harness reads `/proc/sys/kernel/perf_event_paranoid`
  on start and warns (but proceeds without the perf layer) if it's higher.
- **Shell pipelines & `/usr/bin/time -v`**: peak RSS reports only the parent's
  RSS, not children. This understates absolute memory for the shell
  implementation but is consistent across runs, so ratios remain meaningful.
- **Shell perf-stat noise**: every `bash` invocation forks many short-lived
  utilities; the counters reflect that, which is part of what we're measuring,
  not a flaw.

## Container & engine integration

- All scenarios run with `DBX_CONTAINER_MANAGER=podman` exported by the harness
  so distrobox doesn't auto-detect.
- The harness pulls `docker.io/library/alpine:latest` once at suite start and
  asserts it's present; teardown does not remove it.
- **Run ID**: `RUN_ID="$(date +%s)-$$"` — timestamp plus PID, unique per run
  even if two run concurrently.
- **Naming**: containers created by the harness are named
  `dbx-bench-<run-id>-<scenario>-<iteration>`.
- **Tracker file**: `bench/results/<label>/<run-id>/containers.list`. Every
  name is appended the instant it's created (whether via `distrobox create` or
  the direct `podman create` shortcut used by `list-many` setup). Cleanup
  iterates the tracker and removes by **exact** name with `podman rm -f <name>`.
  Never relies on `podman ... --filter name=` (not a partial-match contract
  the harness should depend on for destructive ops).
- **Belt-and-suspenders label**: containers also get
  `--label distrobox.bench.run=<run-id>`. If the harness crashes and the
  tracker is incomplete, `podman ps -a --filter label=distrobox.bench.run=<run-id> -q | xargs -r podman rm -f`
  cleans up.
- **Pre-flight orphan scan**: at run start, look for any container with the
  `distrobox.bench.run` label from *any* previous run. If any exist, the
  harness aborts with a clear message; pass `--clean-orphans` to remove them
  first. The harness never silently destroys state it didn't create in the
  current invocation.
- **`list-many` setup detail**: 20 dummy containers are created via
  `podman create --label manager=distrobox --label distrobox.bench.run=<run-id>
  --name dbx-bench-<run-id>-list-many-NN alpine /bin/true`. This is fast (no
  start, no init-script) and is sufficient for `distrobox list` to find them.
  *Risk*: if the label distrobox actually uses for detection differs from
  `manager=distrobox`, the count won't match. The harness verifies the count
  before running the scenario; if it doesn't match, the scenario is marked
  skipped with a diagnostic. The exact label/filter contract is read from
  `pkg/commands/list.go` and `distrobox-list` during implementation.

## Output format

### Per-run directory

```
bench/results/<label>/<run-id>/
├── meta.json
├── containers.list
├── scenarios/
│   ├── 01-startup-help.hyperfine.json
│   ├── 01-startup-help.time.txt
│   ├── 01-startup-help.time.json
│   ├── 01-startup-help.perf-stat.csv
│   └── (one set per scenario, plus .perf.data if --profile)
└── summary.md
```

`meta.json` shape:

```json
{
  "label": "next",
  "run_id": "1718901234-12345",
  "started": "2026-06-20T12:30:00Z",
  "finished": "2026-06-20T12:33:21Z",
  "executable": "/abs/path/to/bin/distrobox",
  "executable_sha256": "…",
  "executable_size_bytes": 12742149,
  "host": {
    "kernel": "6.19.10-1-default",
    "cpu_model": "…",
    "podman_version": "5.8.1",
    "perf_event_paranoid": 2
  },
  "scenarios_run": ["startup-help", "startup-version", "..."],
  "scenarios_skipped": [{"name": "assemble-parse", "reason": "--dry-run not supported by this binary"}]
}
```

`summary.md` is a single markdown table: scenario × (mean ± stddev, peak RSS,
instructions). Generated unconditionally at the end of `run.sh`.

### Comparison directory

`bench/results/comparisons/<labelA>-vs-<labelB>-<timestamp>.md` produced by
`compare.sh`. Header section:

```
Binary A (label: next):  /abs/path  sha256:…  size: 12,742,149 bytes
Binary B (label: main):  /abs/path  sha256:…  size: …
Podman: 5.8.1 (matches)
Host kernel: 6.19.10-1-default (matches)
Run-id A: 1718901234-12345
Run-id B: 1718900000-12000
```

Per-scenario rows: mean wall time A, B, ratio (B/A), marker
(`Δ < stddev` = "within noise"; otherwise `Nx faster/slower`); peak RSS A, B,
ratio; instructions A, B, ratio.

Refuses to compare if podman versions differ. Pass `--allow-engine-drift` to
override (the difference is annotated in the report header).

## Invocation

```
./bench/run.sh <executable> [--label NAME] [--scenarios LIST]
                            [--profile] [--clean-orphans]
```

- `<executable>` — positional, required, absolute or relative path. The harness
  resolves it to an absolute path before running.
- `--label NAME` — directory name under `bench/results/`. Defaults to the
  executable basename (so `./bin/distrobox` becomes `distrobox`). Multiple runs
  with the same label coexist as sibling `<run-id>` subdirs; nothing is ever
  deleted by `run.sh`.
- `--scenarios LIST` — comma-separated names or shell glob (`startup-*`,
  `2*`). Default = all.
- `--profile` — enable `perf record` per scenario. Off by default
  (`.perf.data` files are large).
- `--clean-orphans` — only honoured at run start: removes containers from
  previous runs (matched by the `distrobox.bench.run` label) before pre-flight.

```
./bench/compare.sh <result-dir-A> <result-dir-B> [--allow-engine-drift]
```

## Risks / open questions

1. **`distrobox assemble create --dry-run`** — needs verification that this
   subcommand/flag exists on both implementations. If it doesn't, scenario 04
   is dropped and noted in the spec. Read `distrobox-assemble` and
   `pkg/commands/assemble.go` during implementation Phase 1.
2. **Label distrobox uses to identify its containers** — `list-many`'s direct
   `podman create` shortcut depends on the exact label/filter contract. Verify
   during implementation Phase 1 by reading `distrobox-list` and
   `pkg/commands/list.go`. If the contract is more involved (e.g., requires
   specific label *value* per-distrobox), the shortcut is replaced with a
   parallel `distrobox create --yes` loop, accepting the slower setup cost.
3. **`--version` flag collision** — `next`'s help shows `--version, -v` AND
   `--verbose, -v` sharing `-v`. The harness always uses long forms
   (`--version`, `--verbose`) to avoid ambiguity across implementations.
4. **Time/perf overhead of `enter-exec`** — `enter` may include
   significant podman-exec time that dominates the implementation difference.
   This is acceptable; the suite reports what it reports. The spec is honest
   about this in the README.

## Non-goals (reaffirmed)

- No CI wiring.
- No flamegraph generation (raw `perf.data` only).
- No Go pprof harness — if needed, a sibling `bench/pprof.sh`, not a feature
  of this suite.
- No automatic statistical significance testing beyond hyperfine's stddev /
  the "Δ < stddev → noise" marker.
