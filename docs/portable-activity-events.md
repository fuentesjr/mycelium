# Portable activity events

**Status:** Reference documentation. Design rationale lives in
[ADR-0002](adr/0002-portable-activity-events-as-adapter-conventions.md).

Mycelium activity events are adapter conventions, not a schema enforced by the
`mycelium` binary. Core stays agent-agnostic: any harness that can run a shell
command can append observability signals with `mycelium log <op>
--payload-json ...`.

## Capability levels

| Level | Capability                                   | Examples                       |
| ----- | -------------------------------------------- | ------------------------------ |
| L0    | Agent can run `mycelium` commands in a shell | pi.dev, Codex CLI, Claude Code |
| L1    | Wrapper records session boundaries           | shell launch wrappers          |
| L2    | Adapter records turn/tool start and end      | harness hooks, tool wrappers   |
| L3    | Adapter records context checkpoints          | pi.dev-style context hooks     |

Mycelium works at L0. Higher levels only improve observability.

## Event vocabulary

Adapters should prefer these op names when they can observe the event:

- `session_startup`, `session_reload`, `session_shutdown`
- Optional harness-specific session boundaries such as `session_new`, `session_resume`, and `session_fork` when a generic startup/reload distinction loses useful information
- `turn_start`, `turn_end`
- `tool_start`, `tool_end`
- `context_checkpoint`
- `agent_note`, `decision`, `compaction`

`context_signal` is a legacy checkpoint name from early `pi-mycelium` releases.
Consumers should treat it as a low-detail `context_checkpoint`; new adapters
should emit `context_checkpoint`.

These names are documented conventions, not binary-reserved schema. The
`mycelium` binary accepts arbitrary `mycelium log <op>` values; adapter authors
should use the shared names above when they fit and keep any harness-local names
clearly documented.

## Payload conventions

Payloads are optional and best-effort. Use snake_case for portable fields and
avoid full prompt, assistant, tool, and file contents by default.

Common fields:

```json
{
  "harness": "pi.dev",
  "adapter_version": "0.2.0",
  "seq": 42,
  "fingerprint": "sha256:...",
  "suppressed_duplicates": 3
}
```

Session and turn fields:

```json
{
  "turn_index": 7,
  "message_count": 18,
  "message_delta": 2,
  "last_role": "toolResult",
  "role_counts": { "user": 4, "assistant": 7, "toolResult": 7 }
}
```

Tool fields:

```json
{
  "tool_call_id": "call_123",
  "tool_name": "bash",
  "is_error": false,
  "duration_ms": 384,
  "output_chars": 2048,
  "exit_code": 0
}
```

Assistant/model fields:

```json
{
  "provider": "anthropic",
  "model": "claude-sonnet-4",
  "stop_reason": "toolUse",
  "usage": { "input": 12345, "output": 678, "cache_read": 0, "cache_write": 0 },
  "cost": { "total": 0.0123 }
}
```

## Content policy

Default adapter payloads should be metadata-only:

- counts, roles, tool names/ids, timings, token usage, and error flags;
- stable fingerprints for dedupe and correlation;
- no prompt, assistant, tool, file, or preview text by default.

If a future adapter supports previews, previews must be disabled by default,
explicitly opt-in, bounded in length, and documented by that adapter. Put large
or sensitive details in normal agent-authored files and reference those files
with `mycelium log <op> --path <path>` when needed.

## Dedupe policy

Adapters should suppress repeated identical `context_checkpoint` events:

1. Compute a fingerprint from stable metadata such as roles, timestamps,
   tool ids/names, and message count.
2. If it matches the prior checkpoint, increment an in-memory duplicate counter
   and do not append a log entry.
3. On the next distinct checkpoint, include `suppressed_duplicates` in the
   payload.

This keeps checkpoints informative without turning `_activity/` into a heartbeat
log.

## Conformance fixture

A representative JSONL fixture lives at
[`docs/fixtures/portable-activity-events.jsonl`](fixtures/portable-activity-events.jsonl).
It is not a binary-enforced schema; it gives adapter authors and downstream
consumers concrete examples of the recommended op names and payload shapes.

## Adapter examples

### L1 shell wrapper

```sh
#!/usr/bin/env sh
set -eu
export MYCELIUM_MOUNT="${MYCELIUM_MOUNT:-$HOME/.mycelium}"
export MYCELIUM_AGENT_ID="${MYCELIUM_AGENT_ID:-shell-agent}"
export MYCELIUM_SESSION_ID="${MYCELIUM_SESSION_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"

mycelium log session_startup --payload-json '{"harness":"shell-wrapper","adapter_version":"0.1.0","seq":1}' || true
trap 'mycelium log session_shutdown --payload-json "{\"harness\":\"shell-wrapper\",\"adapter_version\":\"0.1.0\",\"seq\":2}" || true' EXIT

exec "$@"
```

### L2 generic tool wrapper

```sh
start_s=$(date +%s)
mycelium log tool_start --payload-json "{\"harness\":\"tool-wrapper\",\"adapter_version\":\"0.1.0\",\"tool_name\":\"$1\"}"
"$@"
code=$?
end_s=$(date +%s)
mycelium log tool_end --payload-json "{\"harness\":\"tool-wrapper\",\"adapter_version\":\"0.1.0\",\"tool_name\":\"$1\",\"exit_code\":$code,\"duration_ms\":$(((end_s-start_s)*1000))}"
exit "$code"
```

### L3 pi.dev adapter

`pi-mycelium` emits session boundary events, `turn_start`/`turn_end`,
`tool_start`/`tool_end`, `compaction`, and deduped `context_checkpoint` entries
using the vocabulary above. It still registers no tools: the agent invokes the
portable `mycelium` CLI through pi's built-in shell tool.
