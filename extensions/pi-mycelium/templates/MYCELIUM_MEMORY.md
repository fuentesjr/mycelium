# MYCELIUM_MEMORY.md

This is your starter orientation for the mount at `$MYCELIUM_MOUNT`. You can
edit this file. Replace the conventions below as you find shapes that fit
your work better.

## Conventions

_(Empty. Append entries here as you adopt them — small, prose, dated.
Use `mycelium evolve convention --target <path-or-scope> --rationale "..."`
to also record the event in the activity log.)_

## Current rules in effect

Run `mycelium evolution --active` for the canonical list of active conventions,
indexes, archives, lessons, and questions. This file is a prose companion —
when it diverges from the activity log, **the activity log wins**.

For the divergence policy in full, see ADR-0001 in the mycelium repo:
<https://github.com/fuentesjr/mycelium/blob/main/docs/adr/0001-self-evolution-as-first-class-concept.md>

To record a self-evolution event:

```sh
mycelium evolve convention --target <path-or-scope> --rationale "..."
mycelium evolve lesson     --target <source>        --rationale "..."
mycelium evolve question   --target <topic>         --rationale "..."
# See: mycelium evolution --kinds --format json   for all available kinds
```

## What lives where

A reasonable starting layout — none of this is enforced, replace it as you
find better shapes:

- `agents/{agent_id}/` — your in-flight notes; other agents can read but
  typically don't edit
- `memories/` — durable cross-session memory you accumulate about the user,
  the project, or recurring patterns
- `shared/` — collaborative notes
- `learnings/` — durable lessons you want to keep across sessions
- `INDEX.md` — a hand-maintained map you build as patterns emerge

You will likely invent your own directories ad-hoc — that's expected. When
a new shape stabilizes, record it with `mycelium evolve convention` and
update this file.

## Reading your own activity

The activity log at `_activity/YYYY/MM/DD/{agent_id}.jsonl` is plain JSONL.
Standard tools work; `mycelium grep`/`ls`/`glob` work on the same files.

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
abandoned notes, or stale conventions. The system makes this possible; you
do the reflection.

## Edit me

This file has no special status. When the conventions above stop matching how
you actually work, rewrite them.
