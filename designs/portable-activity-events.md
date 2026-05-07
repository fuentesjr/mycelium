# Portable activity events for Mycelium

Date: 2026-05-07
Status: phase 1 implemented in `pi-mycelium`; adapter conventions promoted to `docs/portable-activity-events.md`

## Problem

`pi-mycelium` currently emits many low-signal `context_signal` events. Making those events richer would help agents and operators understand what happened, but the design must not make Mycelium depend on pi.dev's context model. Mycelium should remain usable from any agent harness that can run a shell command.

## Design principle

Keep Mycelium core agent-agnostic. The core contract remains:

- a local filesystem-backed store;
- `mycelium` CLI operations;
- environment identity (`MYCELIUM_MOUNT`, `MYCELIUM_AGENT_ID`, `MYCELIUM_SESSION_ID`);
- append-only activity JSONL under `_activity/`;
- arbitrary non-mutating signals via `mycelium log <op> --payload-json ...`.

Harness-specific telemetry belongs in optional adapters, not in the core binary.

## Compatibility levels

| Level | Capability                                     | Example support                                                  |
| ----- | ---------------------------------------------- | ---------------------------------------------------------------- |
| L0    | Agent can run `mycelium` commands in a shell   | pi.dev, Codex CLI, Claude Code, most coding agents               |
| L1    | Wrapper records session boundaries             | Most CLI agents via shell wrapper                                |
| L2    | Adapter records turn/tool start and end events | Harnesses with extension hooks or wrapper-visible tool execution |
| L3    | Adapter records context checkpoints            | pi.dev-like harnesses that expose message context                |

Mycelium must work at L0. Higher levels only improve observability.

## Portable event vocabulary

Adapters should prefer a small, generic vocabulary:

- `session_startup`, `session_reload`, `session_shutdown`
- optional harness-specific session boundaries such as `session_new`, `session_resume`, and `session_fork` when useful
- `turn_start`, `turn_end`
- `tool_start`, `tool_end`
- `context_checkpoint`
- `agent_note`
- `decision`
- `compaction`

Existing `context_signal` can remain as a legacy alias or be treated as a low-detail `context_checkpoint`.

## Generic payload conventions

Payloads are optional and best-effort. Adapters fill what they can.

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

Session/turn fields:

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

Model/assistant fields:

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

Default payloads should be metadata-only and avoid full prompt, assistant, tool, file, or preview contents.

Recommended defaults:

- store counts, roles, tool names, ids, timing, token usage, error flags;
- store stable fingerprints for dedupe and correlation;
- if a future adapter supports previews, require explicit opt-in, length bounds, and adapter documentation;
- put large or sensitive details in normal agent-authored files and reference them with `--path` if needed.

## Dedupe policy

Adapters should suppress repeated identical checkpoints.

A simple approach:

1. Compute a fingerprint from stable context metadata, e.g. roles, message ids/timestamps where available, tool ids, and message count.
2. If fingerprint matches the previous checkpoint, increment an in-memory duplicate counter and do not log immediately.
3. On the next distinct event, include `suppressed_duplicates` in the payload.

This keeps checkpoints informative without turning `_activity/` into a heartbeat log.

## Implementation sketch

Phase 1: pi adapter cleanup

- Rename or add `context_checkpoint` events.
- Add fingerprint-based duplicate suppression.
- Enrich checkpoint payload with generic fields available from pi's `ContextEvent`.
- Keep `context_signal` compatibility if needed.

Phase 2: portable adapter guidance

- Document the event vocabulary and payload conventions in Mycelium docs.
- Provide examples for:
  - shell-only wrapper (L1),
  - pi.dev extension (L3),
  - generic tool-wrapper adapter (L2).

Phase 3: optional helper library

- Add a tiny adapter-side helper package for payload shaping and dedupe.
- Do not add binary-level schema enforcement.

## Implementation notes (2026-05-07)

- `pi-mycelium` now emits `context_checkpoint` as the canonical context event.
- The exported legacy `recordContextSignal` helper remains and still writes `context_signal`, but the extension no longer uses it.
- Adapter guidance and examples live in `docs/portable-activity-events.md`.

## Non-goals

- No pi.dev-specific schema in Mycelium core.
- No required context checkpoint support.
- No automatic memory summarization or retrieval.
- No full transcript logging by default.
- No central daemon or service.

## Phase-1 resolved questions

- Migrate emitted pi events to `context_checkpoint`; keep `context_signal` as a legacy compatibility helper and documented alias.
- Keep event names as documented adapter conventions, not binary-reserved names or binary-enforced schema.
- Keep default payloads metadata-only. Preview text, if ever supported, must be explicitly opt-in, bounded, and adapter-documented.
- Provide `docs/fixtures/portable-activity-events.jsonl` as a non-enforcing adapter fixture.
