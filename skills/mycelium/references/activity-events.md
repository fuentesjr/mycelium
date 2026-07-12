# Activity Events

The activity log is plain JSONL under:

```text
_activity/YYYY/MM/DD/<agent_id>.jsonl
```

Every successful `write`, `edit`, `rm`, `mv`, and `log` appends an entry. Reads
are not logged.

## Inspecting History

```sh
mycelium ls '_activity/2026/05/*/*.jsonl' --recursive
mycelium grep --path _activity --pattern 'MYCELIUM_MEMORY.md' --format json --limit 200
cat "$MYCELIUM_MOUNT"/_activity/2026/05/10/*.jsonl
```

Use `_activity/` to reconstruct timelines, find prior paths, review rationale,
and spot repeated behavior. Do not treat it as the current conventions source;
read `MYCELIUM_MEMORY.md` for current rules.

## Common Ops

Core mutation ops: `write`, `edit`, `rm`, `mv`.

Portable adapter ops include:

- `session_startup`, `session_reload`, `session_new`, `session_resume`,
  `session_fork`, `session_shutdown`
- `context_checkpoint`, `compaction`
- richer adapters may also use `turn_start`, `turn_end`, `tool_start`,
  `tool_end`
- agent-authored signals such as `decision` and `agent_note`

The reference `pi-mycelium` adapter emits session boundaries and `compaction`.
It does not emit turn/tool/context telemetry.

## Payloads

Payloads are metadata-only by default: counts, roles, sequence numbers, and
adapter identity. Larger or sensitive details belong in normal files referenced
with `--path`. See `docs/portable-activity-events.md` for adapter vocabulary
details.
