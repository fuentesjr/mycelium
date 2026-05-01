# T2 — Run protocol

## Prerequisites

- `mycelium` binary on `$PATH`.
- `pi` CLI installed and authenticated against the model under test.
- `extensions/pi-mycelium/` symlinked into `.pi/extensions/pi-mycelium`. Verify: `ls .pi/extensions/pi-mycelium`.
- The pi extension's system-prompt block is the *only* scaffolding the agent gets. Do not add custom system prompts.

## Per-run setup

Each of the 5 instances per model gets its own mount, seeded from this task's `seed/` tree. Use the same agent ID as the seed (`glp1-researcher`) so the agent presents as continuing prior work:

```
RUN_ID=$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 8)
MODEL=opus-4-7   # or gpt-5-5
export MYCELIUM_MOUNT=/tmp/t2-${MODEL}-${RUN_ID}
export MYCELIUM_AGENT_ID=glp1-researcher
mkdir -p $MYCELIUM_MOUNT
cp -R docs/benchmarks/tasks/T2-seeded-self-evolution/seed/. $MYCELIUM_MOUNT/
```

Verify the seed landed correctly before starting:

```
ls $MYCELIUM_MOUNT/notes               # should list 6 .md files
ls $MYCELIUM_MOUNT/_activity/2026/04/  # should show 15/, 16/, 18/
wc -l $MYCELIUM_MOUNT/_activity/2026/04/*/glp1-researcher.jsonl  # 30 total
```

The agent ID `glp1-researcher` matches the seeded activity log entries. This continuity is intentional — the model under test is presented as the researcher resuming work, not a brand-new agent inheriting someone else's store.

## Sessions

Run two sessions per instance, fresh `pi` process each, same mount. Use the prompts in `task.md` verbatim — copy-paste, no edits.

Between sessions: kill the pi process, start a new one, re-export the env vars (`MYCELIUM_MOUNT`, `MYCELIUM_AGENT_ID`, plus `MYCELIUM_SESSION_ID` if your harness sets one).

Do not provide cross-session context as user prompts. Session 2 says "take a moment first to look at how the store is shaped" — interpreting that nudge is the agent's job.

## Capture

Per session: capture the full pi transcript (the model's tool calls and text output). Per instance, after session 2:

```
tar -czf t2-${MODEL}-${RUN_ID}-store.tar.gz -C $(dirname $MYCELIUM_MOUNT) $(basename $MYCELIUM_MOUNT)
```

The tarball should include both seed and post-run state — diff against `seed/` to see what the agent added or changed.

## Grading

The grader is a Frontier model from the *opposite* provider — GPT-5.5 grades Opus runs, Opus 4.7 grades GPT runs.

Per instance, the grader receives:
- The unpacked post-run store directory.
- The `seed/` tree for diff context.
- Both session transcripts.
- The grading instructions in `grading.md`.

The grader returns a single verdict per instance: **pass** / **fail** with a one-paragraph rationale.

## Volume and pass criterion

5 instances per model. Per `docs/benchmarks/phase-1.md`:

- **Acceptance #4 passes for a model** if a *majority* (≥3/5) of instances are graded pass.
- **Model-agnostic claim** requires both Claude Opus 4.7 and GPT-5.5 to pass.

## Recording results

Append per-run results to `docs/benchmarks/results/T2.md` (create if absent):

```
| Run ID | Model | Verdict | Rationale (1 line) | Grader | Date |
|---|---|---|---|---|---|
| 9c4a18f0 | opus-4-7 | pass | Edited MYCELIUM_MEMORY.md to add a search-before-writing rule, then ran grep before adding the new file | gpt-5-5 | 2026-05-XX |
```

After all 10 instances complete (5 per model), update `docs/benchmarks/phase-1.md` with the final pass/fail per acceptance criterion #4.
