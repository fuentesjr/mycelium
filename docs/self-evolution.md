# Self-Evolution

**Status:** Reference documentation. Concrete recipes for agents revising their
own memory practices through files. ADR-0004 supersedes the old `evolve`
mechanism: current rules live in `MYCELIUM_MEMORY.md`; the activity log records
the history of edits and rationale.

---

## The mechanism

Self-evolution is now file-based:

1. Read `MYCELIUM_MEMORY.md` at session start.
2. Notice patterns by reading notes and grepping `_activity/`.
3. Revise `MYCELIUM_MEMORY.md` or a sibling conventions file with
   `mycelium write` / `edit` and a clear `--rationale`.
4. Use `mycelium log decision|agent_note --rationale "..."` for point-in-time
   decisions that should remain history but not become standing guidance.

Be proactive. When a repeated pattern, mistake, durable user preference, naming
rule, useful index, stale region, or open question emerges, update the
conventions file in the same session. Do not leave the lesson implicit.

Existing historical `op:"evolve"` entries remain valid activity-log history,
but they are not the current source of truth.

---

## Pattern 1 — Convention bootstrap

At session start:

```sh
mycelium read MYCELIUM_MEMORY.md --format json
```

If the file is absent, report that instead of broad-searching for a replacement.
The pi-mycelium extension seeds it for fresh mounts.

---

## Pattern 2 — Adopting or revising a convention

After noticing duplicate filenames, violated naming rules, or near-duplicate
paths:

```sh
mycelium ls '_activity/2026/04/*/researcher-7.jsonl' --recursive
mycelium grep --pattern '"op":"write"' --path _activity --format json --limit 200
```

Edit `MYCELIUM_MEMORY.md`:

```sh
mycelium edit MYCELIUM_MEMORY.md \
  --old "## Conventions" \
  --new "## Conventions

- 2026-04-30: Use <date>-<slug>.md filenames under notes/incidents/ so incidents sort chronologically without a separate index." \
  --rationale "Adopting incident filenames after index.md drifted from reality within a week."
```

To revise the rule later, edit the same prose entry or add a dated replacement.
The current file is the active rule; the activity log preserves history.

---

## Pattern 3 — Self-built indexes

Build indexes only after observed search/navigation friction:

```sh
mycelium grep --path _activity --pattern 'glp1' --format json --limit 500
mycelium write notes/_index/glp1.md \
  --rationale "Creating a hand-built GLP-1 TOC after repeated searches across the same cluster."
```

Then record the index location and refresh rule in `MYCELIUM_MEMORY.md`, for
example:

```md
- 2026-05-12: `notes/_index/glp1.md` maps common GLP-1 queries to canonical
  files. Regenerate it when `notes/glp1-*` changes substantially.
```

---

## Pattern 4 — Archiving and pruning

Triggered by stale paths or activity that shows a region is no longer active:

```sh
mycelium ls --recursive
mycelium grep --pattern 'notes/old-protocol' --path _activity --format json --limit 100
mycelium mv notes/old-protocol.md archive/2025-q4/old-protocol.md \
  --rationale "Pre-2026 protocol note moved out of active notes; retained for reference."
```

If this establishes a durable policy, add it to `MYCELIUM_MEMORY.md`:

```md
- 2026-05-12: Move inactive protocol notes older than one quarter to
  `archive/<year>-q<quarter>/`; keep them searchable but out of active notes.
```

---

## Pattern 5 — Lessons and open questions

For durable lessons, add prose to `MYCELIUM_MEMORY.md` or a linked lessons file:

```md
- 2026-05-12: For library-internals questions, prefer source permalinks over
  secondary summaries.
```

For open questions, keep them in a visible section or file:

```md
## Open Questions

- 2026-05-12: Does the GLP-1 cardio-protection effect generalize to
  non-diabetic populations? Need 3+ independent studies before treating this as
  a lesson.
```

When a question resolves, edit the entry into a lesson with rationale. The file
contains the current state; the activity log contains the transition.

---

## When this doc goes stale

These recipes assume the current public model: a folder, safe mutations, and a
searchable activity log. If `MYCELIUM_MEMORY.md`, `--rationale`, or `_activity/`
semantics change, update these recipes with the new mechanics.
