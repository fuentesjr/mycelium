# MYCELIUM_MEMORY.md

This is your starter orientation for the mount at `$MYCELIUM_MOUNT`. You can
edit this file. Replace the conventions below as you find shapes that fit
your work better.

## Conventions

_(Empty. Append dated prose entries here as you adopt conventions, lessons,
index locations, archive policy, or open questions. Revise by editing this file
with `--rationale`; the activity log records the change and why. Do this
proactively when a repeated pattern, mistake, durable user preference, naming
rule, or useful index emerges.)_

## Current rules in effect

This file is the canonical list of current rules in effect. To change a rule,
edit the rule here. To record a point-in-time decision that should remain log
history but not become a standing convention, use `mycelium log decision` or
`mycelium log agent_note` with `--rationale`.

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
a new shape stabilizes, update this file.

## Reading your own activity

The activity log at `_activity/YYYY/MM/DD/{agent_id}.jsonl` is plain JSONL.
Standard tools work; `mycelium grep` and patterned `mycelium ls --recursive`
work on the same files. Other root paths beginning with `_` are internal; read
`_activity/` for history and leave the rest alone.

```sh
# Today's entries (your agent)
mycelium read _activity/$(date -u +%Y/%m/%d)/${MYCELIUM_AGENT_ID:-agent}.jsonl

# This month, all agents
mycelium ls '_activity/2026/04/*/*.jsonl' --recursive

# Find write ops
mycelium grep --path _activity --pattern '"op":"write"' --format=json

# Find context checkpoints with inline payloads
mycelium grep --path _activity --pattern context_checkpoint --format=json
```

Grepping your own log between sessions is how you notice duplicated writes,
abandoned notes, or stale conventions. The system makes this possible; you
do the reflection.

## Edit me

This file has no special status. When the conventions above stop matching how
you actually work, rewrite them.
