---
name: mycelium
description: "Use when working with a Mycelium persistent memory store: reading or writing durable agent memory, resuming from a Mycelium mount, recording project conventions in MYCELIUM_MEMORY.md, resolving Mycelium CAS conflicts, inspecting _activity history, or checking/setup of the mycelium CLI and MYCELIUM_MOUNT environment."
---

# Mycelium

Mycelium is shell-first persistent memory for coding agents: a mounted folder,
safe mutations, and a searchable JSONL activity log. Use the `mycelium` CLI
through the normal shell; do not invent adapter APIs.

## Start Of Session

1. If the binary or mount is uncertain, run `scripts/doctor`.
2. Read the conventions file once:

```sh
mycelium read MYCELIUM_MEMORY.md --format json
```

If the file is missing, report that or create it only when the user asks you to
initialize memory. Do not broad-search for a replacement conventions file.

## Operating Rules

- Use raw filesystem commands for inspection only (`cat`, `ls`, `rg`, `tar`).
- Use `mycelium write`, `edit`, `rm`, and `mv` for live-store mutations.
- Never write under a root path beginning with `_`; `_activity/` is history.
- Use `--expected-version` for edits, deletes based on observed content, and
  revisions to files read earlier in the session.
- Use `--rationale` when the operation carries reasoning a future reviewer
  would need.
- Update `MYCELIUM_MEMORY.md` in the same session when a durable convention,
  user preference, repeated mistake, useful index, archive policy, or open
  question emerges.

## References

- Need exact command syntax or setup checks: read `references/commands.md`.
- Need to handle exit 64 or destination collisions: read `references/conflicts.md`.
- Need to decide what to store where or how to revise conventions: read
  `references/memory-guidance.md`.
- Need to inspect prior activity or reason about adapter events: read
  `references/activity-events.md`.

## Setup Scripts

- `scripts/doctor` is read-only. It checks `mycelium` on `PATH`,
  `MYCELIUM_MOUNT`, and basic command availability.
- `scripts/setup` is a local helper for creating/exporting a mount after user
  approval. It does not download binaries; when `mycelium` is absent, it prints
  install guidance.
