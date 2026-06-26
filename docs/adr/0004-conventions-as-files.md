# ADR 0004: Conventions as files, not evolve events

- **Status:** Accepted
- **Date:** 2026-06-11
- **Deciders:** Sal Fuentes Jr.

## Context

`mycelium evolve` and the `op: "evolve"` event stream made conventions durable but not easy for future-self continuity.

The model currently expected three separate projections to infer what is active:

- `mycelium evolve --active` for active conventions.
- `mycelium evolve --kinds` for convention vocabulary.
- `mycelium log` + `--rationale` for the why-history.

That split created extra command surface, vocabulary coupling in code, and a durable-record-centric migration path for mutable state that should be edited as text.

The simplification review approved a simpler rule on 2026-06-11: conventions are authored policy content in one file, and the log remains history.

## Decision

`MYCELIUM_MEMORY.md` is the single source of truth for conventions in-band.

The adapter and harness guidance now point to:

- read `MYCELIUM_MEMORY.md` at startup for active conventions,
- edit `MYCELIUM_MEMORY.md` with rationale when conventions change,
- rely on the existing activity log (`mycelium log` with rationale) for historical introspection.

`mycelium evolve` is removed as a functional command in Stage 3.

Compatibility behavior:

- existing `op: "evolve"` events remain valid history lines and are not treated as active source of truth for runtime behavior;
- readers must tolerate the historical op as legacy signal, in the same spirit as adapter portability from ADR-0002.

To support mounts still mentioning evolve, Stage 3 adds a hidden transitional diagnostic stub:

- `mycelium evolve` exits 1 with exactly one line: `evolve was removed in 0.4.0; record conventions in your conventions file (see MYCELIUM_MEMORY.md)`.
- the stub is hidden from normal help output.
- it is removed at 1.0.

ADR-0001 is superseded by this decision, because the archaeological, command-driven convention model is replaced by file-owned convention state.

## Consequences

### Positive

- Fewer command contracts to learn and maintain.
- Mutable convention state is now versionable and reviewable as ordinary markdown text.
- `MYCELIUM_MEMORY.md` and activity-log history stay separate by design: current policy in-file, rationale/history in-log.

### Negative

- Human-readable policy state can drift from log history.
- Any automation that expected live `op: "evolve"` state must shift to file-based reads.

### Neutral

- No new migration is required for existing mounts; historical evolve events are tolerated as legacy context only.
- `MYCELIUM_MEMORY.md` is now explicit source of truth without introducing schema changes.

## Open questions

- None after 2026-06-11 approval.
