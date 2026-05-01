# Self-Evolution Patterns

**Status:** Documentation only. Concrete `mycelium` recipes for the patterns described in section 7 of `mycelium-design.md`. The design doc carries the rationale; this doc carries the shell.

---

## Pattern 1 — Convention bootstrap

Read the convention file at session start. The seeded template is a starter, not a contract.

```
mycelium read MYCELIUM_MEMORY.md
```

If the file doesn't exist, the agent is on a fresh mount.

---

## Pattern 2 — Convention revision

Triggered when the activity log surfaces a behavior the agent wants to address — duplicate writes, near-duplicate filenames, conventions edited but later violated.

```
mycelium glob '_activity/2026/04/*/researcher-7.jsonl'
mycelium grep --pattern '"op":"write"' --path _activity --format json --limit 200
```

The `json` envelope returns `{matches: [{path, line, text}], truncated}`. Each `text` is a JSONL entry with `ts`, `op`, `path`, `version`, `agent_id`, `session_id`. Group by `path` to spot near-duplicates.

Then edit the convention file:

```
mycelium edit MYCELIUM_MEMORY.md \
    --old "Naming: lowercase-with-dashes." \
    --new "Naming: lowercase-with-dashes. Search before writing — \`mycelium grep\` for any topic before creating a new file."
```

**Concrete example.** Three sessions produce `notes/glp1-pipeline.md`, `notes/glp-1-pipeline.md`, and `notes/glp1_followup.md`, none re-read before the next was written. The agent adds a "search before writing" rule to `MYCELIUM_MEMORY.md`, consolidates the three files, and writes a brief `notes/_index/glp1.md` mapping the canonical file. The convention edit and the consolidation are separate operations; either alone is partial.

---

## Pattern 3 — Self-built indexes

Triggered by repeated reads or repeated grep/glob with the same pattern. Reads aren't auto-logged — either notice mid-session, or log search signals explicitly:

```
mycelium log read_signal --payload-json '{"pattern":"glp1","scope":"notes/"}'
mycelium grep --pattern '"op":"read_signal"' --path _activity --format json --limit 500
```

When the same patterns recur, write the index:

```
mycelium write notes/_index/glp1.md   # contents map common queries to canonical paths
```

Don't create indexes preemptively. An index that isn't earned by observed search frequency is just another file to maintain.

---

## Pattern 4 — Archiving and pruning

Triggered by `mycelium ls --recursive` returning paths the agent doesn't recognize, or a periodic pass at session start. The activity log answers "when was this file last modified," not "when was it last read" — so for staleness, the question is mutation recency.

```
mycelium ls --recursive
mycelium grep --pattern 'notes/old-protocol' --path _activity --format json --limit 100
mycelium mv notes/old-protocol.md archive/2025-q4/old-protocol.md
```

Prefer `mv` to `archive/` over `rm` when the operator might still need the content; both log the operation either way.

---

## When this doc goes stale

These recipes are written against the Phase 1 binary. If `--format json` envelope shape, `--limit` semantics, or the `_activity/` path layout change, the recipes update with them.
