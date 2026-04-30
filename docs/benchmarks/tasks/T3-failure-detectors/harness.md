# T3 — Failure-mode detectors

**Acceptance criterion:** #5 (failure-mode observability).
**Model-independent.** No real model runs; T3 is pure trajectory classification.

## What this task validates

Three detectors operating on activity-log content alone must distinguish a healthy trajectory from each of three named failure modes:

| Detector | Fires when |
|---|---|
| `writes_without_reads` | `(write+edit) / read_signal` ratio >0.7 across ≥3 consecutive sessions |
| `near_duplicate_paths` | ≥3 Levenshtein-1 path collisions among `op=write` entries in a single session |
| `thrashing` | ≥50 activity-log entries in a single session |

Implementation in `binary/detect.go`. Fixtures under `binary/testdata/trajectories/`:

- `healthy.jsonl` — two well-behaved sessions; trips no detector.
- `unhealthy-writes-without-reads.jsonl` — four sessions with all-mutations, zero read signals; trips detector 1.
- `unhealthy-duplicate-paths.jsonl` — one session writing `notes/glp1.md`, `notes/glp-1.md`, `notes/glp1_.md`, `notes/glp10.md`; trips detector 2.
- `unhealthy-thrashing.jsonl` — one session with 55 entries, all distinct write paths; trips detector 3.

## Run protocol

```
cd binary
go test -run TestDetectors -v
```

Pass condition: every fixture is classified as expected by all three detectors. The test table is in `binary/detect_test.go` (`expectedVerdicts`).

## Why no `task.md` or `held-out.md`

T3 has no agent under test. There's nothing for a model to do, no transcript to read, no grader to consult. The trajectories are the input; the detectors are the output. Per `docs/benchmarks/phase-1.md`, the broader human-judgment validation (30 trajectories, ≥90% inter-rater agreement) is deferred to Phase 2 once real run data is available to calibrate against. T3 in Phase 1 is the smoke test: do the detectors at minimum classify deliberately-constructed examples correctly.

## Adding a new failure mode

1. Add the detector function to `binary/detect.go`. Keep it a pure function over `[]LogEntry`.
2. Add it to `RunDetectors` in the same file.
3. Add a fixture under `binary/testdata/trajectories/unhealthy-<mode>.jsonl` that trips the new detector and no others.
4. Extend `expectedVerdicts` in `binary/detect_test.go` with one new column entry per fixture (existing fixtures should report `true` for the new detector).
5. Update `docs/benchmarks/phase-1.md` T3 section and this harness doc.
