# Memory Guidance

## What Goes Where

- Task-specific content: write a note at a meaningful path.
- Durable behavioral guidance: edit `MYCELIUM_MEMORY.md`.
- Index locations, archive policy, and naming rules: record in
  `MYCELIUM_MEMORY.md`.
- Point-in-time signals that should remain history but not become current
  rules: use `mycelium log decision|agent_note --rationale "..."`.

## Session Discipline

At session start, read `MYCELIUM_MEMORY.md` exactly once unless the task calls
for a fresh reread. Consult task-relevant files by path. Avoid broad prefetching.

Before finishing, ask whether the session exposed a repeated mistake, durable
user preference, naming rule, useful index, stale region, or open question. If
yes, update `MYCELIUM_MEMORY.md` with `--rationale` before ending the work.

## Revising Conventions

Edit the relevant prose entry or add a dated replacement that states what it
replaces. The conventions file is the active rule set; `_activity/` is the
history of how it changed.

Example:

```sh
mycelium edit MYCELIUM_MEMORY.md \
  --old "## Conventions" \
  --new "## Conventions

- 2026-05-12: Use notes/incidents/<date>-<slug>.md for incident notes." \
  --rationale "Adopting chronological incident filenames after duplicate paths emerged."
```

## Naming

Prefer names that describe the content (`auth/session-token-rotation.md`) over
opaque timestamps (`2026-05-12-note.md`) unless the conventions file says
otherwise. Never create agent-authored content under a root `_` path.

## Self-Built Indexes

Build indexes only after observed search/navigation friction:

```sh
mycelium grep --path _activity --pattern 'glp1' --format json --limit 500
mycelium write notes/_index/glp1.md \
  --rationale "Creating a hand-built GLP-1 TOC after repeated searches across the same cluster."
```

Then record the index path and refresh rule in `MYCELIUM_MEMORY.md`.

## Archiving And Pruning

When stale paths or activity show a region is no longer active, move it with
`mycelium mv` and rationale:

```sh
mycelium mv notes/old-protocol.md archive/2025-q4/old-protocol.md \
  --rationale "Pre-2026 protocol note moved out of active notes; retained for reference."
```

If the move establishes a policy, add the archive rule to
`MYCELIUM_MEMORY.md`.

## Lessons And Open Questions

For durable lessons, add prose to `MYCELIUM_MEMORY.md` or a linked lessons
file. For unresolved questions, keep a visible `Open Questions` section or
linked file. When a question resolves, edit it into a lesson with rationale so
the file holds the current state and `_activity/` holds the transition.
