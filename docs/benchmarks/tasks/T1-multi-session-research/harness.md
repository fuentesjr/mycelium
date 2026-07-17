# T1 — Run protocol

## Prerequisites

- Run setup from the Mycelium repository root.
- `pi` CLI installed and authenticated against the model under test.
- The released `pi-mycelium` package, or an absolute local package path supplied
  through `PACKAGE_SOURCE`.
- The pi extension's system prompt and canonical starter template are the only
  scaffolding the agent gets. Do not add custom system prompts or seed files.

## Per-run setup

Each of the 5 instances per model gets its own working directory, project-local
extension registration, derived journal mount, and unique agent ID:

```bash
REPO_ROOT=$(pwd)
RUN_ID=$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 8)
MODEL=opus-4-7   # or gpt-5-5
RUN_DIR=$(mktemp -d "/tmp/t1-${MODEL}-${RUN_ID}.XXXXXX")
PACKAGE_SOURCE=${PACKAGE_SOURCE:-npm:pi-mycelium}

cd "$RUN_DIR"
pi install "$PACKAGE_SOURCE" -l --approve
MOUNT="$RUN_DIR/.pi/pi-mycelium/journal"
export MYCELIUM_AGENT_ID=t1-${MODEL}-${RUN_ID}
```

For a local checkout, set `PACKAGE_SOURCE` to the absolute path
`$REPO_ROOT/extensions/pi-mycelium`. Do not export `MYCELIUM_MOUNT`; the
extension owns it and derives `MOUNT` from project-local registration.

The operator adds no seed. On the first session the shipped extension
bootstraps its canonical `MYCELIUM_MEMORY.md`; that product behavior is part of
the benchmark.

## Sessions

Run three sessions per instance from `RUN_DIR`, fresh `pi` process each, same
project-local mount. Use the prompts in `task.md` verbatim — copy-paste, no
edits.

After session 1, assert that `MOUNT/MYCELIUM_MEMORY.md` exists and that the
captured system prompt reports exactly `MOUNT`. Abort the run if either check
fails. Between sessions, stop pi and start a new process from `RUN_DIR`; pi
provides a fresh session identity while the mount and agent identity persist.

Do not provide cross-session context as user prompts. Session 2 says "continuing from the prior session" — discovering and re-reading prior content is the agent's job.

## Capture

Per session: capture the full pi transcript (the model's tool calls and text output). Per instance, after session 3:

```bash
tar -czf "t1-${MODEL}-${RUN_ID}-store.tar.gz" -C "$(dirname "$MOUNT")" "$(basename "$MOUNT")"
```

Save the session-3 transcript separately for the comparison-run grading step.

## Comparison run (no-memory baseline)

Per instance, after the three Mycelium sessions complete: run the same task once
more in a separate working directory, single session, same model. Disable all
extension discovery so a globally installed Mycelium cannot contaminate the
control.

```bash
BASELINE_DIR=$(mktemp -d "/tmp/t1-baseline-${MODEL}-${RUN_ID}.XXXXXX")
cd "$BASELINE_DIR"
env -u MYCELIUM_MOUNT -u MYCELIUM_AGENT_ID -u MYCELIUM_SESSION_ID \
  pi --no-extensions
# Feed task.md session 1 + session 2 + session 3 prompts concatenated.
```

Capture the transcript only. There is no store to tar.

## Grading

The grader is a Frontier model from the *opposite* provider — GPT-5.5 grades Opus runs, Opus 4.7 grades GPT runs. This minimizes single-vendor bias.

Per Mycelium run, the grader receives:

- The unpacked store directory.
- The grading instructions in `held-out.md`.

Per comparison run, the grader receives:

- The single concatenated transcript.
- The grading instructions in `held-out.md` (with the "Mycelium pass" criterion replaced by the comparison criterion below).

### Mycelium-run pass

≥4 of 5 questions in `held-out.md` answered correctly *and* traceable to specific notes in the store.

### Comparison

For each instance, the grader reads both the Mycelium run's store-derived answers and the comparison run's transcript-derived answers. The grader judges whether the Mycelium run is "more substantively grounded" — concretely: does it cite specific facts/sources/version numbers/operational details that the comparison run lacks?

## Volume and pass criterion

5 instances per model. Per `docs/benchmarks/phase-1.md`:

- **Acceptance #1 passes for a model** if ≥3/5 instances clear the Mycelium-run pass criterion AND the grader judges the Mycelium run more grounded than the comparison run in ≥3/5 instances.
- **Cross-model criterion** requires both Claude Opus 4.7 and GPT-5.5 to pass through pi.

## Recording results

Append per-run results to `docs/benchmarks/results/T1.md` (create if absent):

```
| Run ID | Model | Mycelium pass | Comparison verdict | Grader | Date |
|---|---|---|---|---|---|
| 7e15bf2d | opus-4-7 | 4/5 | more grounded | gpt-5-5 | 2026-05-XX |
```

After all 10 instances complete (5 per model), update `docs/benchmarks/phase-1.md` with the final pass/fail per acceptance criterion #1.
