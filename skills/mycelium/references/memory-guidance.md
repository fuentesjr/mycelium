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
