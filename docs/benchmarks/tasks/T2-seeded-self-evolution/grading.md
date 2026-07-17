# T2 — Grading rubric

The grader's job is one judgment per instance: did the agent _recognize and respond to_ the seeded failure pattern by the end of session 2?

This is intentionally not a checklist with point thresholds. Self-evolution shows up in different ways across runs; what matters is whether the agent did something rather than nothing.

## What the seed contains

The seeded store has three problems by design:

1. **Path drift across near-duplicates.** `notes/glp1-pipeline.md` and `notes/glp-1-pipeline.md` are two files on the same subject created on different days with a hyphen difference. `notes/glp1_followup.md` uses an underscore where the rest of the store uses hyphens.
2. **Writes without recorded search context.** All ~30 prior activity-log entries are `write` and `edit`. Reads are not auto-logged, so the store shows repeated mutation with no reliable recorded evidence of search/re-read behavior.
3. **A weak `MYCELIUM_MEMORY.md`.** The starter doc names what's in the store but says nothing about searching before writing, naming conventions, or how to avoid duplicate paths.

The session 2 prompt — "take a moment first to look at how the store is shaped" — is a deliberate nudge.

## Pass criterion

The instance passes only when all three required signals are present by the end
of session 2:

1. **Current rule.** `MYCELIUM_MEMORY.md` contains a new or revised rule that
   addresses a seeded pattern — for example, search before writing, one canonical
   `glp1-` naming form, or reconciliation of near-duplicate notes.
2. **Operational rationale.** The mutation that introduced that rule has a
   non-empty `rationale` explaining why the store's observed shape warranted it.
3. **Durable evidence.** The post-run activity log contains the matching
   `write` or `edit` entry for `MYCELIUM_MEMORY.md` from this run.

The grader may use these as supporting evidence, but none is a substitute for
the required signals:

- an index or reconciliation of the duplicate notes;
- transcript/tool evidence that the agent ran `ls` or `grep` before writing and
  did so in response to what it found rather than as a blind reflex.

## What does not count

- Adding new notes files in the same problematic style (e.g., creating `notes/cardiovascular-outcomes.md` and `notes/cv-outcomes.md`) — that's the same failure mode the seed exhibits.
- Mentioning the duplicates only in conversational text without any store-side action — the test is whether memory persisted, not whether the model noticed mid-session.
- Wholesale rewrites of seeded notes without recognizing the pattern that produced them — clean-up isn't the goal; _responding to the pattern_ is.
- Adding an index, reconciling notes, or grepping without updating
  `MYCELIUM_MEMORY.md` with rationale and matching activity evidence.

## Format

The grader returns:

```
Verdict: pass | fail
Rationale: <1-3 sentences naming what the agent did or didn't do>
```

If on the fence, lean toward fail — a marginal "the agent kind of touched MEMORY.md but didn't really say anything" is not a pass. The signal we want is unambiguous: the agent saw the pattern and acted.
