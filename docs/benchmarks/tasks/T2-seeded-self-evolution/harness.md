# T2 — Run protocol

## Prerequisites

- Run setup from the Mycelium repository root.
- `pi` CLI installed and authenticated against the model under test.
- The exact released `pi-mycelium` version under test.
- An exact provider-qualified `MODEL_ID`; labels alone do not select a pi model.
- The pi extension's system-prompt block is the *only* scaffolding the agent gets. Do not add custom system prompts.

## Per-run setup

Each of the 5 instances per model gets its own working directory and
project-local mount, seeded from this task's `seed/` tree. Use the same agent ID
as the seed (`glp1-researcher`) so the agent presents as continuing prior work:

```bash
REPO_ROOT=$(pwd)
RUN_ID=$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 8)
MODEL_ID=openrouter/anthropic/claude-opus-4.7
MODEL_LABEL=opus-4-7
# For the OpenAI runs, use:
# MODEL_ID=openai-codex/gpt-5.5
# MODEL_LABEL=gpt-5-5
MYCELIUM_VERSION=0.5.0 # replace with the exact released version under test
RUN_DIR=$(mktemp -d "/tmp/t2-${MODEL_LABEL}-${RUN_ID}.XXXXXX")
PACKAGE_SOURCE=npm:pi-mycelium@${MYCELIUM_VERSION}
PI_VERSION=$(pi --offline --version)

cd "$RUN_DIR"
pi install "$PACKAGE_SOURCE" -l --approve
MOUNT="$RUN_DIR/.pi/pi-mycelium/journal"
export MYCELIUM_AGENT_ID=glp1-researcher
mkdir -p "$MOUNT"
cp -R "$REPO_ROOT/docs/benchmarks/tasks/T2-seeded-self-evolution/seed/." "$MOUNT/"
```

Before accepting the run, `pi --offline --list-models "$MODEL_ID"` must show the
exact provider/model pair. Record `MODEL_ID`, `PI_VERSION`, `PACKAGE_SOURCE`, and
the package version. Abort rather than silently falling back to pi's configured
default model.

The canonical protocol evaluates an exact released npm package. Do not install
from the repository directory or substitute a release-candidate tarball: those
sources do not provide the same project-local registration contract. Do not
export `MYCELIUM_MOUNT`; the extension derives it from project-local
registration.

Verify the seed landed correctly before starting:

```bash
ls "$MOUNT/notes"               # should list 6 .md files
ls "$MOUNT/_activity/2026/04/"  # should show 15/, 16/, 18/
wc -l "$MOUNT"/_activity/2026/04/*/glp1-researcher.jsonl  # 30 total
```

The agent ID `glp1-researcher` matches the seeded activity log entries. This continuity is intentional — the model under test is presented as the researcher resuming work, not a brand-new agent inheriting someone else's store.

## Sessions

Run two sessions per instance from `RUN_DIR`, fresh `pi` process each, same
mount. Use the prompts in `task.md` verbatim — copy-paste, no edits.

Start every session explicitly:

```bash
pi --model "$MODEL_ID" --no-skills --no-prompt-templates --no-context-files
```

During session 1, confirm the system prompt reports exactly `MOUNT`; abort the
run on mismatch. Between sessions, stop pi and start a new process from
`RUN_DIR`; pi supplies a fresh session identity.

Do not provide cross-session context as user prompts. Session 2 says "take a moment first to look at how the store is shaped" — interpreting that nudge is the agent's job.

## Capture

Per session: capture the full pi transcript (the model's tool calls and text output). Per instance, after session 2:

```bash
tar -czf "t2-${MODEL_LABEL}-${RUN_ID}-store.tar.gz" -C "$(dirname "$MOUNT")" "$(basename "$MOUNT")"
```

The tarball should include both seed and post-run state — diff against `seed/` to see what the agent added or changed.

## Grading

The grader is the target model from the opposite family: GPT-5.5 grades Opus
runs, and Opus 4.7 grades GPT runs. Set an exact provider-qualified
`GRADER_MODEL_ID` and invoke pi with `--model "$GRADER_MODEL_ID"` plus
`--no-extensions --no-skills --no-prompt-templates --no-context-files`. Record
that exact ID with the verdict; never infer it from a label.

```bash
# For an Opus run:
GRADER_MODEL_ID=openai-codex/gpt-5.5
# For a GPT run, instead use:
# GRADER_MODEL_ID=openrouter/anthropic/claude-opus-4.7
```

Per instance, the grader receives:

- The unpacked post-run store directory.
- The `seed/` tree for diff context.
- Both session transcripts.
- The grading instructions in `grading.md`.

The grader returns a single verdict per instance: **pass** / **fail** with a one-paragraph rationale.

## Volume and pass criterion

5 instances per model. Per `docs/benchmarks/phase-1.md`:

- **Acceptance #3 passes for a model** if a *majority* (≥3/5) of instances are graded pass.
- **Cross-model criterion** requires both Claude Opus 4.7 and GPT-5.5 to pass through pi.

## Recording results

Append per-run results to `docs/benchmarks/results/T2.md` (create if absent):

```
| Run ID | Model ID | pi | Mycelium source | Verdict | Rationale (1 line) | Grader model ID | Date |
|---|---|---|---|---|---|---|---|
| 9c4a18f0 | openrouter/anthropic/claude-opus-4.7 | 0.80.10 | npm:pi-mycelium@0.5.0 | pass | Added a reasoned search-before-write rule with matching activity evidence | openai-codex/gpt-5.5 | 2026-07-XX |
```

After all 10 instances complete (5 per model), update `docs/benchmarks/phase-1.md` with the final pass/fail per self-evolution criterion #3.
