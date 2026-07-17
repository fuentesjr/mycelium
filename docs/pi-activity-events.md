# Pi activity events

Mycelium's active activity-event contract is pi-only. The log remains plain JSONL at `_activity/YYYY/MM/DD/{agent_id}.jsonl`, written by the `pi-mycelium` extension and the bundled `mycelium` CLI engine.

## Current pi extension entries

`pi-mycelium` attempts to record lifecycle signals that matter for memory continuity:

- `session_startup`, `session_reload`, `session_new`, `session_resume`, or
  `session_fork` when pi reports the corresponding session-start reason;
- `session_shutdown` before the extension runtime exits;
- `compaction` when pi reports a compaction event.

Payloads are metadata-only by default. The extension does not log full prompts, assistant messages, tool contents, or file previews.

Lifecycle writes are best-effort so logging trouble does not make pi unusable.
A missing boundary, shutdown, or compaction line can therefore mean the hook
could not invoke the CLI; it must not be interpreted as proof the lifecycle
event never occurred. Operators who depend on continuity should verify the
mount, `MYCELIUM_MEMORY.md`, and current-day activity file after installation.

## Core mutation entries

The Go CLI appends an activity entry for every successful state-changing command:

- `write`, `edit`, `rm`, `mv`;
- `log` for explicit non-mutation signals.

Reads (`read`, `ls`, `grep`) are not logged. Mutation entries include the timestamp, agent id, session id, operation, affected path(s), version information when applicable, and optional `rationale`.

## Agent-authored signals

Agents may use `mycelium log decision` and `mycelium log agent_note` for durable signals worth grepping later. Put large or sensitive details in normal journal files and reference them with `--path`; keep inline payloads metadata-sized.

## Compatibility

No journal migration is required. Historical logs may contain old operations such as `context_signal`, `context_checkpoint`, `turn_start`, `tool_start`, or other adapter-specific names. Readers should tolerate unknown `op` values and optional payload fields. Those legacy names are history, not active pi extension commitments.
