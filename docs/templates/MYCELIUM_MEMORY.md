# MYCELIUM_MEMORY.md

This is your starter orientation for the mount at `$MYCELIUM_MOUNT`.
You can edit this file. Replace the conventions below as you find shapes
that fit your work better.

## Current rules in effect

Run `mycelium evolution --active` for the canonical list of active conventions,
indexes, archives, lessons, and questions. This file is a prose companion —
when it diverges from the activity log, **the activity log wins**. See
[ADR-0001](../adr/0001-self-evolution-as-first-class-concept.md) for the
divergence policy.

To record a self-evolution event:

```sh
mycelium evolve convention --target <path-or-scope> --rationale "..."
mycelium evolve lesson     --target <source>        --rationale "..."
mycelium evolve question   --target <topic>         --rationale "..."
# See: mycelium evolution --kinds --format json   for all available kinds
```

## What the binary enforces

The mount has one rule: **any path whose first segment starts with `_` is
reserved for the binary**. Agent-facing writes (`mycelium write`, `edit`, `rm`,
`mv`) under such paths are rejected with `path uses reserved '_' prefix`.

Today, that rule reserves one tree:

- `_activity/YYYY/MM/DD/{agent_id}.jsonl` — your daily activity log. The
  binary appends one JSONL entry on every successful mutation and on every
  `mycelium log` call. Payloads from `mycelium log` are inlined on the entry
  as a `payload` field. You cannot write to it; you can read it freely
  (`mycelium read`/`ls`/`glob`/`grep`).

Everything else under the mount is yours.

## What lives where

- `_activity/` — see above. Binary-controlled metadata, append-only.

- Anywhere else — yours. A reasonable starting layout:
  - `AGENTS/{agent_id}/` — your in-flight notes; other agents can read but
    typically don't edit
  - `shared/` — collaborative notes
  - `learnings/` — durable lessons you want to keep across sessions
  - `INDEX.md` — a hand-maintained map you build as patterns emerge

  None of this is enforced. Replace it.

## Identity

Three environment variables travel with every command:

- `MYCELIUM_MOUNT` — the directory you operate on (required)
- `MYCELIUM_AGENT_ID` — your stable identity, recorded on every entry
- `MYCELIUM_SESSION_ID` — optional per-process scope

If `MYCELIUM_AGENT_ID` is unset, your activity entries land in
`_activity/YYYY/MM/DD/unspecified.jsonl`. Set it once at session start.

## Reading your own activity

The activity log is plain JSONL. General tools work; `mycelium grep`/`ls`/`glob`
work on the same files.

```sh
# Today's entries (your agent)
mycelium read _activity/$(date -u +%Y/%m/%d)/$MYCELIUM_AGENT_ID.jsonl

# This month, all agents
mycelium glob '_activity/2026/04/*/*.jsonl'

# Find write ops
mycelium grep --path _activity --pattern '"op":"write"' --format=json

# Find signal entries with payloads (payload is inline on each entry)
mycelium grep --path _activity --pattern context_signal --format=json
```

Grepping your own log between sessions is how you notice duplicated writes,
abandoned notes, or stale conventions. The system makes this possible; you do
the reflection.

## Edit me

This file has no special status. When the conventions above stop matching how
you actually work, rewrite them.
