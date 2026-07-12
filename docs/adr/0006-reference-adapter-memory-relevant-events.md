# ADR 0006: Reference adapter emits only memory-relevant events

- **Status:** Superseded by ADR-0007
- **Date:** 2026-06-11
- **Deciders:** Sal Fuentes Jr.

> Supersession note: ADR-0007 keeps the pi extension lifecycle subset but removes this ADR's cross-adapter framing. The active pi contract is documented in `docs/pi-activity-events.md`.

## Context

`pi-mycelium` previously emitted a broad portable-events set.

Operationally, the memory loop uses only session-level state changes and compacted checkpoints. A large portion of `turn_*` and `tool_*` telemetry does not feed memory continuity and adds noise, size, and integration surface.

ADR-0002 already defined portable event names as adapter conventions, not binary-enforced schema.

## Decision

After Stage 5, the reference pi adapter should emit only:

- session-boundary events (`session_startup`, `session_shutdown`, and related),
- `compaction`.

It should not emit `turn_start`, `turn_end`, `tool_start`, `tool_end`, or
`context_checkpoint` events. Checkpoint volume and deduplication are left to
adapters that choose to register context hooks.

## Consequences

### Positive

- reduced adapter volume and hook footprint,
- fewer semantically irrelevant events in the memory log,
- adapter logic stays focused on memory loop value.

### Negative

- some downstream consumers lose optional telemetry if they rely specifically on reference adapter turn/tool events.

### Neutral

- ADR-0002 portable vocabulary remains unchanged and still applies to all adapters.
- other adapters may continue to emit full vocabulary (including turn/tool/context events) per their own needs; this is not a cross-adapter prohibition.
- checkpoint spam is a harness-volume tradeoff, not a core guarantee handled by the reference adapter.

## Open questions

- None.
