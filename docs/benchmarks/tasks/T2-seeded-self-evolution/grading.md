# T2 — Grading rubric

The grader's job is one judgment per instance: did the agent *recognize and respond to* the seeded failure pattern by the end of session 2?

This is intentionally not a checklist with point thresholds. Self-evolution shows up in different ways across runs; what matters is whether the agent did something rather than nothing.

## What the seed contains

The seeded store has three problems by design:

1. **Path drift across near-duplicates.** `notes/glp1-pipeline.md` and `notes/glp-1-pipeline.md` are two files on the same subject created on different days with a hyphen difference. `notes/glp1_followup.md` uses an underscore where the rest of the store uses hyphens.
2. **Writes-without-reads.** All ~30 prior activity-log entries are `write` and `edit`. Zero `read_signal` entries. The store has been written to but apparently never re-read.
3. **A weak `MYCELIUM_MEMORY.md`.** The starter doc names what's in the store but says nothing about searching before writing, naming conventions, or how to avoid duplicate paths.

The session 2 prompt — "take a moment first to look at how the store is shaped" — is a deliberate nudge.

## Pass criterion

The instance passes if, by the end of session 2, the agent did at least one of the following in a way visible in the post-run store or activity log:

- **Convention edit.** Updated `MYCELIUM_MEMORY.md` (or added a sibling conventions file) with a rule that addresses one of the seeded patterns — e.g., "search before writing", "use `glp1-` not `glp-1-` or `glp1_`", "consolidate near-duplicate notes".
- **Index or reconciliation.** Added an index file, a notes-of-notes, or otherwise consolidated the duplicates (e.g., merging or cross-linking `glp1-pipeline.md` and `glp-1-pipeline.md`, or renaming for consistency).
- **Grep-before-write behavior.** Activity log shows the agent ran `glob` or `grep` (these may surface as `op=read_signal` if the agent emits them, or as visible tool calls in the transcript) before adding new files in session 2 — and did so in a way that's clearly *because* of what was found in the store, not as a blind reflex.

Any one of these is sufficient. Multiple is fine but does not "score higher."

## What does not count

- Adding new notes files in the same problematic style (e.g., creating `notes/cardiovascular-outcomes.md` and `notes/cv-outcomes.md`) — that's the same failure mode the seed exhibits.
- Mentioning the duplicates only in conversational text without any store-side action — the test is whether memory persisted, not whether the model noticed mid-session.
- Wholesale rewrites of seeded notes without recognizing the pattern that produced them — clean-up isn't the goal; *responding to the pattern* is.

## Format

The grader returns:

```
Verdict: pass | fail
Rationale: <1-3 sentences naming what the agent did or didn't do>
```

If on the fence, lean toward fail — a marginal "the agent kind of touched MEMORY.md but didn't really say anything" is not a pass. The signal we want is unambiguous: the agent saw the pattern and acted.
