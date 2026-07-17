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
- `INDEX.md` — a hand-maintained map you build as patterns emerge; record its refresh rule here once it earns maintenance

You will likely invent your own directories ad-hoc — that's expected. When
a new shape stabilizes, update this file. Prefer descriptive names over opaque timestamps. Archive stale regions with `mycelium mv` plus `--rationale`; if the move creates a durable policy, record the policy here.

## Reading your own activity

The activity log at `_activity/YYYY/MM/DD/{agent_id}.jsonl` is plain JSONL.
Successful `write`, `edit`, `rm`, `mv`, and `log` operations append entries;
reads are not logged. Standard tools work; `mycelium grep` and patterned
`mycelium ls --recursive` work on the same files. Other root paths beginning with `_` are internal; read
`_activity/` for history and leave the rest alone.

```sh
# Today's entries (your agent)
mycelium read _activity/$(date -u +%Y/%m/%d)/${MYCELIUM_AGENT_ID:-agent}.jsonl

# This month, all agents
mycelium ls "_activity/$(date -u +%Y/%m)/*/*.jsonl" --recursive

# Find write ops
mycelium grep --path _activity --pattern '"op":"write"' --format=json

# Find session lifecycle entries
mycelium grep --path _activity --pattern session_ --format=json
```

Grepping your own log between sessions is how you notice duplicated writes,
abandoned notes, stale conventions, and prior paths. Treat `_activity/` as
history; this file is the current rules source. When an open question resolves,
edit it into a current lesson or remove it so history and active guidance do
not conflict.

## Edit me

This file has no special status. When the conventions above stop matching how
you actually work, rewrite them.
