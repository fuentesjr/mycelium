# ADR 0005: Activity log as durable history, not transactional ledger

- **Status:** Accepted
- **Date:** 2026-06-11
- **Deciders:** Sal Fuentes Jr.

## Context

The transaction journal (`_tx/`) exists only to bridge a narrow failure window between content mutation and log append.

That guarantee was costly: extra code paths, additional fsyncs, and a frozen-mount failure mode on incomplete journal recovery, while no system state now depends on transactional replay after stage 3 removes `evolve`.

The simplification review approved removal of `_tx` and a clearer durability contract.

## Decision

`_tx/` is removed; the activity log remains append-only durable history, not a transactional replay system.

For mutations:

1. lock
2. CAS check
3. content mutation commit
4. durable log append attempt
5. return success

Failure behavior is explicit:

- if append fails after content commit, the command exits non-zero (fail-loud),
- no silent success after non-durable append, and no success masking of power-loss scenarios.

`tx_id` is retained on mutation JSONL entries for continuity. It is generated using stdlib randomness and time (`tx-<zero-padded-unix-nano>-<rand>`), not ULID. The timestamp component is fixed-width decimal so string sorting preserves timestamp order.

The default session id generator also moves off ULID to stdlib time/randomness (`auto-<zero-padded-unix-nano>-<rand>`) so all randomness and ordering assumptions stay in stdlib-backed primitives.

## Boundaries and compatibility

There is a bounded power-loss gap:

- if power fails after durable content write and before log append, the mount stays mutated with a missing event line.
- this gap is bounded to the single microsecond-to-millisecond ordering window, and is now documented as the durable-history contract instead of being hidden by transaction recovery.

v0.3 compatibility for leftover `_tx/pending/*.json` is required:

- a preflight compatibility check must block mutations on mounts that still contain pending records,
- the CLI must fail with actionable instructions to run documented recovery before normal operations,
- no automatic replay is performed after `_tx` removal.

## Consequences

### Positive

- lower complexity in command path and failure model,
- no journal recovery mode,
- reduced I/O and simpler crash behavior,
- preserved historical `tx_id` field shape for existing tooling.

### Negative

- operators now see occasional loud failures in the narrow post-commit append gap and must tolerate one unlogged but completed mutation in crash windows.
- compatibility work for legacy `_tx` artifacts is required at v0.3 boundary.

### Neutral

- no change to durable JSONL event shape beyond `tx_id` generation.
- no dependency on non-stdlib ULID generation in those fields.

## Open questions

None.
