# T1 — Run protocol

## Prerequisites

- `mycelium` binary on `$PATH` (built from this repo or installed from a release).
- `pi` CLI installed and authenticated against the model under test.
- `extensions/pi-mycelium/` symlinked into `.pi/extensions/pi-mycelium` (project-scoped install). Verify: `ls .pi/extensions/pi-mycelium`.
- The pi extension's system-prompt block is the *only* scaffolding the agent gets. Do not add custom system prompts.

## Per-run setup

Each of the 5 instances per model gets its own mount and a unique agent ID:

```
RUN_ID=$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 8)
MODEL=opus-4-7   # or gpt-5-5
export MYCELIUM_MOUNT=/tmp/t1-${MODEL}-${RUN_ID}
export MYCELIUM_AGENT_ID=t1-${MODEL}-${RUN_ID}
mkdir -p $MYCELIUM_MOUNT
```

The store starts empty. Do not seed `MYCELIUM_MEMORY.md` — letting the agent organize from scratch is part of what's being measured.

## Sessions

Run three sessions per instance, fresh `pi` process each, same mount. Use the prompts in `task.md` verbatim — copy-paste, no edits.

Between sessions: kill the pi process, start a new one, re-export the env vars (`MYCELIUM_MOUNT`, `MYCELIUM_AGENT_ID`, `MYCELIUM_SESSION_ID` if your harness sets one). The new session's first activity-log entry will be in a fresh `_activity/YYYY/MM/DD/<agent>.jsonl` (or appended to today's, depending on date).

Do not provide cross-session context as user prompts. Session 2 says "continuing from the prior session" — discovering and re-reading prior content is the agent's job.

## Capture

Per session: capture the full pi transcript (the model's tool calls and text output). Per instance, after session 3:

```
tar -czf t1-${MODEL}-${RUN_ID}-store.tar.gz -C $(dirname $MYCELIUM_MOUNT) $(basename $MYCELIUM_MOUNT)
```

Save the session-3 transcript separately for the comparison-run grading step.

## Comparison run (no-memory baseline)

Per instance, after the three Mycelium sessions complete: run the same task once more, single session, no Mycelium mount, same model. The agent has only what it can do in one shot.

```
unset MYCELIUM_MOUNT MYCELIUM_AGENT_ID MYCELIUM_SESSION_ID
# then start pi and feed it: task.md session 1 + session 2 + session 3 prompts concatenated
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
- **Model-agnostic claim** requires both Claude Opus 4.7 and GPT-5.5 to pass.

## Recording results

Append per-run results to `docs/benchmarks/results/T1.md` (create if absent):

```
| Run ID | Model | Mycelium pass | Comparison verdict | Grader | Date |
|---|---|---|---|---|---|
| 7e15bf2d | opus-4-7 | 4/5 | more grounded | gpt-5-5 | 2026-05-XX |
```

After all 10 instances complete (5 per model), update `docs/benchmarks/phase-1.md` with the final pass/fail per acceptance criterion #1.
