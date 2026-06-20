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
