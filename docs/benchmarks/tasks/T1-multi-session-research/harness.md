# T1 — Run protocol

## Prerequisites

- Run setup from the Mycelium repository root.
- `pi` CLI installed and authenticated against the model under test.
- The exact released `pi-mycelium` version under test.
- An exact provider-qualified `MODEL_ID`; labels alone do not select a pi model.
- A source-backed answer key available only to the separate grading
  environment. It must not be present in the repository or any filesystem
  namespace available to the model under test. It must answer all five
  questions in `held-out.md`, cite primary sources, and record the date each
  source was checked.
- The pi extension's system prompt and canonical starter template are the only
  scaffolding the agent gets. Do not add custom system prompts or seed files.

## Per-run setup

Each of the 5 instances per model gets its own working directory, project-local
extension registration, derived journal mount, and unique agent ID:

```bash
RUN_ID=$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 8)
MODEL_ID=openrouter/anthropic/claude-opus-4.7
MODEL_LABEL=opus-4-7
# For the OpenAI runs, use:
# MODEL_ID=openai-codex/gpt-5.5
# MODEL_LABEL=gpt-5-5
MYCELIUM_VERSION=0.5.0 # replace with the exact released version under test
RUN_DIR=$(mktemp -d "/tmp/t1-${MODEL_LABEL}-${RUN_ID}.XXXXXX")
PACKAGE_SOURCE=npm:pi-mycelium@${MYCELIUM_VERSION}
PI_VERSION=$(pi --offline --version)

cd "$RUN_DIR"
pi install "$PACKAGE_SOURCE" -l --approve
MOUNT="$RUN_DIR/.pi/pi-mycelium/journal"
export MYCELIUM_AGENT_ID=t1-${MODEL_LABEL}-${RUN_ID}
```

Before accepting the run, `pi --offline --list-models "$MODEL_ID"` must show the
exact provider/model pair. Record `MODEL_ID`, `PI_VERSION`, `PACKAGE_SOURCE`, and
the package version. Abort rather than silently falling back to pi's configured
default model.

The canonical protocol evaluates an exact released npm package. Do not install
from the repository directory or substitute a release-candidate tarball: those
sources do not provide the same project-local registration contract. Do not
export `MYCELIUM_MOUNT`; the extension owns it and derives `MOUNT` from
project-local registration.

The operator adds no seed. On the first session the shipped extension
bootstraps its canonical `MYCELIUM_MEMORY.md`; that product behavior is part of
the benchmark.

## Sessions

Run three sessions per instance from `RUN_DIR`, fresh `pi` process each, same
project-local mount. Use the prompts in `task.md` verbatim — copy-paste, no
edits.

Start every session explicitly:

```bash
pi --model "$MODEL_ID" --no-skills --no-prompt-templates --no-context-files
```

After session 1, assert that `MOUNT/MYCELIUM_MEMORY.md` exists and that the
captured system prompt reports exactly `MOUNT`. Abort the run if either check
fails. Between sessions, stop pi and start a new process from `RUN_DIR`; pi
provides a fresh session identity while the mount and agent identity persist.

Do not provide cross-session context as user prompts. Session 2 says "continuing from the prior session" — discovering and re-reading prior content is the agent's job.

## Capture

Per session: capture the full pi transcript (the model's tool calls and text output). Per instance, after session 3:

```bash
tar -czf "t1-${MODEL_LABEL}-${RUN_ID}-store.tar.gz" -C "$(dirname "$MOUNT")" "$(basename "$MOUNT")"
```

Save the session-3 transcript separately for the comparison-run grading step.

## Comparison run (no-memory baseline)

Per instance, after the three Mycelium sessions complete: run the same task once
more in a separate working directory, single session, same model. Disable all
extension discovery so a globally installed Mycelium cannot contaminate the
control.

```bash
BASELINE_DIR=$(mktemp -d "/tmp/t1-baseline-${MODEL_LABEL}-${RUN_ID}.XXXXXX")
cd "$BASELINE_DIR"
env -u MYCELIUM_MOUNT -u MYCELIUM_AGENT_ID -u MYCELIUM_SESSION_ID \
  pi --model "$MODEL_ID" --no-extensions --no-skills \
     --no-prompt-templates --no-context-files
# Feed task.md session 1 + session 2 + session 3 prompts concatenated.
```

Capture the transcript only. There is no store to tar.

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

Run grading only after every model-under-test and baseline pi process has
ended. Provision the private key at that point in a separate grading
container, VM, or account whose filesystem was not available to the evaluated
model:

```bash
ANSWER_KEY_PATH=/absolute/grader-only/path/t1-answer-key.md
case "$ANSWER_KEY_PATH" in /*) ;; *) exit 1 ;; esac
test -f "$ANSWER_KEY_PATH"
ANSWER_KEY_SHA256=$(shasum -a 256 "$ANSWER_KEY_PATH" | awk '{print $1}')
```

The grading environment receives copies of the captured artifacts and grading
instructions. Do not mount the answer-key location into an environment used to
run the model under test.

Per Mycelium run, the grader receives:

- The unpacked store directory.
- The grading instructions in `held-out.md`.
- The operator-held truth set at `ANSWER_KEY_PATH`.

Per comparison run, the grader receives:

- The single concatenated transcript.
- The baseline transcript grading instructions in `held-out.md`.
- The operator-held truth set at `ANSWER_KEY_PATH`.

Never copy the answer key into `RUN_DIR`, `BASELINE_DIR`, the mounted store, a
session prompt, the repository, or any other context visible to the model under
test. Archive the key separately and use `ANSWER_KEY_SHA256` to prove every run
in a campaign used the same truth set.

### Mycelium-run pass

≥4 of 5 questions in `held-out.md` answered correctly *and* traceable to specific notes in the store.

### Comparison

For each instance, the grader reads both the Mycelium run's store-derived
answers and the baseline's transcript-derived answers, then applies the exact
comparison rule in `held-out.md`. Do not restate or replace that rule in the
grader prompt.

## Volume and pass criterion

5 instances per model. Per `docs/benchmarks/phase-1.md`:

- **Acceptance #1 passes for a model** if ≥3/5 instances clear the Mycelium-run pass criterion AND the grader judges the Mycelium run more grounded than the comparison run in ≥3/5 instances.
- **Cross-model criterion** requires both Claude Opus 4.7 and GPT-5.5 to pass through pi.

## Recording results

Append per-run results to `docs/benchmarks/results/T1.md` (create if absent):

```
| Run ID | Model ID | pi | Mycelium source | Key SHA-256 | Mycelium pass | Baseline pass | Comparison verdict | Grader model ID | Date |
|---|---|---|---|---|---|---|---|---|---|
| 7e15bf2d | openrouter/anthropic/claude-opus-4.7 | 0.80.10 | npm:pi-mycelium@0.5.0 | `<digest>` | 4/5 | 2/5 | more grounded | openai-codex/gpt-5.5 | 2026-07-XX |
```

After all 10 instances complete (5 per model), update `docs/benchmarks/phase-1.md` with the final pass/fail per acceptance criterion #1.
