# ADR 0002: Portable activity events as adapter conventions

- **Status:** Accepted; reference adapter behavior revised by ADR-0006
- **Date:** 2026-05-07
- **Deciders:** Sal Fuentes Jr.

## Context

Mycelium's activity log is authoritative for core state changes (`write`,
`edit`, `rm`, `mv`, `log`, `evolve`) and is intentionally plain JSONL under
`_activity/`. Harness adapters can also append non-mutating observability
signals with `mycelium log <op> --payload-json ...`.

The first pi.dev adapter emitted low-detail `context_signal` events. Improving
those events is useful: operators and future-self agents benefit from knowing
session boundaries, turn boundaries, tool execution metadata, token usage, and
context checkpoints. But putting a pi.dev-specific context schema into the
Mycelium binary would violate the core bet:

- Mycelium must remain usable from any agent harness that can run shell
  commands.
- Core must not depend on one harness's message, tool, or context lifecycle.
- The activity log must stay human-readable and tolerant of adapter-specific
  payloads.

## Decision

Portable activity events are **documented adapter conventions**, not reserved
op names or schema enforced by the `mycelium` binary.

Core continues to guarantee only:

- local filesystem-backed store;
- CLI operations and environment identity;
- append-only activity JSONL;
- durable `mycelium log <op>` entries with optional JSON payloads.

Adapters may emit a shared vocabulary when they can observe the corresponding
event:

- `session_startup`, `session_reload`, `session_shutdown`
- optional harness-specific session boundaries such as `session_new`,
  `session_resume`, and `session_fork`
- `turn_start`, `turn_end`
- `tool_start`, `tool_end`
- `context_checkpoint`
- `agent_note`, `decision`, `compaction`

`context_signal` is a legacy alias for a low-detail checkpoint. New adapters
should emit `context_checkpoint`; readers should continue to tolerate
`context_signal`.

Payload fields use snake_case and are best-effort. Common portable fields
include:

```json
{
  "harness": "pi.dev",
  "adapter_version": "0.1.7",
  "seq": 42,
  "fingerprint": "sha256:...",
  "suppressed_duplicates": 3
}
```

Adapters should avoid full prompt, assistant, tool, file, and preview contents
by default. Counts, roles, tool names/ids, timings, token usage, error flags,
and stable fingerprints are appropriate defaults. If a future adapter supports
previews, they must be disabled by default, explicitly opt-in, bounded in length,
and documented by that adapter. Larger or sensitive details belong in normal
agent-authored files referenced by `--path`.

## Implementation

At acceptance, `pi-mycelium` provided the reference L3 adapter behavior:

- emits `context_checkpoint` instead of `context_signal` from the `context` hook;
- fingerprints checkpoint metadata and suppresses duplicates;
- emits `turn_start`/`turn_end`, `tool_start`/`tool_end`, `compaction`, and
  `session_shutdown` when pi.dev exposes those hooks;
- enriches payloads with generic fields such as message counts, role counts,
  usage/cost, tool ids, durations, and output sizes;
- originally kept a legacy helper export for low-detail context signals, but no
  longer used it from the extension entrypoint.

ADR-0006 later narrows the reference adapter to memory-relevant events only and
removes the legacy helper export. The portable vocabulary above remains
unchanged.

The adapter vocabulary and examples are documented in
[`docs/portable-activity-events.md`](../portable-activity-events.md).

## Consequences

### Positive

- **Core remains agent-agnostic.** L0 shell-only harnesses can use Mycelium
  unchanged; richer harnesses add observability without changing the binary.
- **Observability improves without transcript capture.** Operators can inspect
  turns, tools, usage, and checkpoints without logging prompt/tool contents by
  default.
- **Readers can degrade gracefully.** Unknown op names or payload fields are
  tolerated because adapter telemetry is convention, not core schema.
- **Duplicate checkpoint spam is reduced.** Fingerprint suppression prevents
  context hooks from becoming heartbeat logs.

### Negative

- **No binary-level conformance guarantee.** A poorly-written adapter can emit
  malformed or noisy payloads. Documentation plus the non-enforcing fixture at
  `docs/fixtures/portable-activity-events.jsonl`, not core validation, are the
  enforcement mechanism.
- **Vocabulary drift is possible.** Different adapters may invent harness-local
  event names. The shared vocabulary is a convention they are encouraged to use,
  not a reserved namespace enforced by the binary.
- **Consumers must remain liberal.** Downstream tooling should treat payload
  fields as optional and tolerate both `context_checkpoint` and legacy
  `context_signal`.

### Neutral

- **No migration required.** Existing `context_signal` entries remain valid log
  history. New pi.dev sessions append `context_checkpoint`.

## Open questions

- Should a future helper package expose payload shaping and dedupe for non-pi
  adapters?
