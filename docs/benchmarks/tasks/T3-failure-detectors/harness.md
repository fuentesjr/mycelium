# T3 — Failure-mode detectors

**Acceptance criterion:** #4 (failure-mode observability).
**Model-independent.** No real model runs; T3 is pure trajectory classification.

## What this task validates

Two detectors operating on activity-log content alone must distinguish a healthy trajectory from each named failure mode:

| Detector               | Fires when                                                                    |
| ---------------------- | ----------------------------------------------------------------------------- |
| `near_duplicate_paths` | ≥3 Levenshtein-1 path collisions among `op=write` entries in a single session |
| `thrashing`            | ≥50 activity-log entries in a single session                                  |

The earlier `writes_without_reads` detector was removed because its denominator, `op=read_signal`, is not emitted by Mycelium and is not in the pi activity contract.

Implementation under `docs/benchmarks/tasks/T3-failure-detectors/tool/`. Fixtures under `docs/benchmarks/tasks/T3-failure-detectors/tool/testdata/trajectories/`:

- `healthy.jsonl` — two well-behaved sessions; trips no detector.
- `unhealthy-duplicate-paths.jsonl` — one session writing `notes/glp1.md`, `notes/glp-1.md`, `notes/glp1_.md`, `notes/glp10.md`; trips the near-duplicate detector.
- `unhealthy-thrashing.jsonl` — one session with 55 entries; trips the thrashing detector.

## Run protocol

```
go test -v ./docs/benchmarks/tasks/T3-failure-detectors/tool
```

Pass condition: every fixture is classified as expected by both detectors. The test table is in `docs/benchmarks/tasks/T3-failure-detectors/tool/detect_test.go` (`expectedVerdicts`).

## Why no `task.md` or `held-out.md`

T3 has no agent under test. There's nothing for a model to do, no transcript to read, no grader to consult. The trajectories are the input; the detectors are the output. Per `docs/benchmarks/phase-1.md`, the broader human-judgment validation (30 trajectories, ≥90% inter-rater agreement) is deferred to Phase 2 once real run data is available to calibrate against. T3 in Phase 1 is the smoke test: do the detectors at minimum classify deliberately-constructed examples correctly.

## Adding a new failure mode

1. Add the detector function to `docs/benchmarks/tasks/T3-failure-detectors/tool/detect.go`. Keep it a pure function over `[]LogEntry`.
2. Add it to `RunDetectors` in the same file.
3. Add a fixture under `docs/benchmarks/tasks/T3-failure-detectors/tool/testdata/trajectories/unhealthy-<mode>.jsonl` that trips the new detector and no others.
4. Extend `expectedVerdicts` in `docs/benchmarks/tasks/T3-failure-detectors/tool/detect_test.go` with one new column entry per fixture (existing fixtures should report `true` for the new detector).
5. Update `docs/benchmarks/phase-1.md` T3 section and this harness doc.
