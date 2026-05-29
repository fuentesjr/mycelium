# ADR 0006: Reference adapter emits only memory-relevant events

- **Status:** Accepted
- **Date:** 2026-06-11
- **Deciders:** Sal Fuentes Jr.

## Context

`pi-mycelium` previously emitted a broad portable-events set.

Operationally, the memory loop uses only session-level state changes and compacted checkpoints. A large portion of `turn_*` and `tool_*` telemetry does not feed memory continuity and adds noise, size, and integration surface.

ADR-0002 already defined portable event names as adapter conventions, not binary-enforced schema.

## Decision

After Stage 5, the reference pi adapter should emit only:

- session-boundary events (`session_startup`, `session_shutdown`, and related),
- `compaction`,
- deduped `context_checkpoint` events.

It should not emit `turn_start`, `turn_end`, `tool_start`, or `tool_end`.

`context_checkpoint` emissions continue with deduplication via checkpoint fingerprinting to prevent duplicate telemetry.

## Consequences

### Positive

- reduced adapter volume and hook footprint,
- fewer semantically irrelevant events in the memory log,
- adapter logic stays focused on memory loop value.

### Negative

- some downstream consumers lose optional telemetry if they rely specifically on reference adapter turn/tool events.

### Neutral

- ADR-0002 portable vocabulary remains unchanged and still applies to all adapters.
- other adapters may continue to emit full vocabulary (including turn/tool events) per their own needs; this is not a cross-adapter prohibition.

## Open questions

- None.
