# ADR 0003: Optional rationale on rationale-bearing operations

- **Status:** Accepted
- **Date:** 2026-05-12
- **Deciders:** Sal Fuentes Jr.

> Filename retained as `0003-optional-rationale-on-write-and-edit.md` for
> stable linking. The accepted decision is broader than the original
> proposal: rationale is captured on `write`, `edit`, `rm`, `mv`, and
> `log` — and propagated through the CAS conflict envelope.

## Context

Mycelium asks agents to capture rationale at the moment of decision. For
structural decisions, the `evolve` command enforces this: `--rationale`
is required, and the binary errors out (`mycelium evolve: --rationale is
required`) when it is missing (`internal/mycelium/evolve.go`).

For per-note content and other mutations, no such mechanism exists. The
README's "What agents record" section frames rationale capture as a
discipline that applies to both surfaces — file contents _and_ `evolve`
events — but at the time of this ADR, `internal/mycelium/write.go`,
`mutate_tx.go`, `mv.go`, and `log.go` contained no references to rationale.
A note may be written, an edit applied, a file removed, or a path renamed
with no reasoning at all, and the binary will accept it.

This creates an asymmetry:

- _Why-this-pattern_ (structural decisions) → guaranteed in the activity
  log via `evolve`.
- _Why-this-thing_ (per-note reasoning, deletions, renames, signal
  entries) → present only by convention, in the body of whatever note
  the agent happens to write.

A reader auditing a mount cannot tell, from the activity log alone, why
any particular mutation happened. For long-running mounts and
cross-agent handoff — the cases mycelium is built for — that gap is
real.

The naive fix is to _require_ `--rationale` on every mutation. This was
rejected. Many legitimate operations have no separable rationale:
appending to a running TODO list, updating a status file, saving a
downloaded artifact, rebuilding an index, removing a clearly-obsolete
temp file. Forcing rationale on these produces placeholder text
("updating file", "syncing notes"), which is worse than no field — it
gives false confidence that reasoning is being captured when it isn't.

## Decision

Add an **optional** `--rationale` flag to every rationale-bearing CLI
verb: `write`, `edit`, `rm`, `mv`, and `log`. When supplied, the
rationale is captured into the corresponding activity log entry as a
new top-level field, parallel to how `evolve` records rationale today.
When absent, behavior is unchanged and the field is omitted from the
log entry.

The flag is intentionally symmetric across all rationale-bearing
operations. An agent that learns to type `--rationale` for one verb
can use the same flag everywhere a mutation or signal carries
operational meaning.

Rationale also propagates through the **CAS conflict envelope**: when a
write or edit loses a `--expected-version` race, the JSON envelope
emitted to stderr includes the losing caller's `rationale` field
alongside `current_version`. The reviewer or retrying agent sees both
sides' intent, not just the winning version hash.

The note-body discipline ("write the _why_ into the note itself")
remains a documented convention. Documentation describes it as a craft
norm — distinct from the operational rationale captured on the activity
log line — and stops framing it as a property the binary guarantees.

Harness adapters are encouraged to nudge agents toward supplying
`--rationale` when operational reasoning exists. The reference
adapter, `pi-mycelium`, includes a one-line recommendation in its
injected system-prompt block beginning with v0.2.0.

### Schema

`LogEntry` in `internal/mycelium/log.go` gains:

```go
Rationale string `json:"rationale,omitempty"`
```

`conflictEnvelope` in `internal/mycelium/write.go` gains:

```go
Rationale string `json:"rationale,omitempty"`
```

- Maximum size: 64 KiB, matching `maxRationaleSize` from
  `internal/mycelium/evolve.go`. Oversize input is rejected before the
  mutation runs with exit code `ExitReservedPrefix` (65), matching
  `evolve`'s existing convention for rationale-validation failures.
- `omitempty` ensures both fields are absent from log entries and
  envelopes when no rationale is supplied — existing log readers and
  JSONL fixtures remain valid without migration.
- The activity log field appears on the same JSONL line as the
  mutation entry, so `tail -f`, `grep`, and `mycelium grep --path
_activity` surface it without indirection.

### CLI surface

```
mycelium write notes/incident-2026-05-12.md \
  --rationale "API began returning 503 at 14:22; recording symptoms before mitigation closes the window."

mycelium edit notes/runbook.md \
  --old-string "..." --new-string "..." \
  --rationale "Removing the manual restart step; the operator-init bug it worked around was fixed in v0.1.7."

mycelium rm notes/spikes/2026-Q1/deprecated.md \
  --rationale "Spike concluded; superseded by notes/decisions/2026-04-cache-layer.md."

mycelium mv notes/draft.md notes/incidents/2026-05-12-cache-stampede.md \
  --rationale "Promoting from draft once symptoms confirmed the cache stampede hypothesis."

mycelium log decision \
  --rationale "Choosing Redis over Memcached for the cache layer; cluster mode and persistence outweigh the marginal latency cost." \
  --payload-json '{"chosen":"redis","rejected":["memcached","dragonfly"]}'
```

### Out of scope

- A higher-level `mycelium note` verb that _requires_ rationale for
  rationale-bearing directories. Deferred until there is evidence of
  which directories want enforcement; this ADR does not preclude it.
- Lint-style audits over rationale density in chosen directories
  (e.g., `mycelium audit notes/incidents/`). Deferred; orthogonal to
  the activity-log gap this ADR closes.

## Consequences

### Positive

- **Activity log becomes self-sufficient for many review tasks.** A
  reader can reconstruct _why-this-operation_ by reading the log line,
  without opening every note. Operational reasoning travels with the
  operation.
- **Symmetry across the CLI.** Every rationale-bearing op accepts the
  same flag with the same size bound and the same on-disk shape. Agents
  learn one pattern, not five.
- **Conflict resolution gains context.** Both sides' rationale surface
  on the envelope, so the retrying agent can merge intent, not just
  bytes.
- **No coercion of low-signal operations.** Routine writes, appends,
  artifact saves, and cleanups carry no rationale and produce no
  placeholder text. The schema reflects the actual epistemic state.
- **Adapter alignment.** Harness adapters can nudge agents toward
  supplying rationale; pi-mycelium ships this as a default behavior
  in v0.2.0.

### Negative

- **Surface area on five verbs.** Each rationale-bearing op gains a
  flag. The flag is optional, so existing callers continue to work,
  but help text, documentation, and examples grow correspondingly.
- **No enforcement is still no guarantee.** Agents that omit
  `--rationale` produce log entries without reasoning, just as today.
  Adoption depends on harness prompts and operator discipline.
- **Activity log entry size grows when used.** With a 64 KiB cap and
  rationales expected to run a few hundred bytes in practice, the
  impact on log growth is modest but nonzero.
- **Tooling must tolerate the new field.** Anything that consumes
  `LogEntry` JSON or the conflict envelope should treat `rationale`
  as optional. Since both fields are `omitempty`, strict-parsing
  readers that ignore unknown fields are unaffected.

### Neutral

- **No migration.** Existing activity log entries and conflict
  envelopes remain valid; new entries optionally include the field.
- **Schema versioning.** This change lands before Phase 2's activity
  log schema is frozen, so the field is part of v1 rather than a v2
  addition.

## Alternatives considered

1. **Require `--rationale` on every mutation.** Rejected: produces
   placeholder rationale on legitimate rationale-free operations,
   which is worse than no field. Garbage in the rationale slot makes
   the slot meaningless across the corpus.
2. **Add `--rationale` to `write` and `edit` only.** This was the
   original proposal. Rejected during acceptance review: an
   asymmetric flag across rationale-bearing verbs trains the agent
   inconsistently and leaves `rm`, `mv`, and `log` operations
   without a rationale path even when they have one. The symmetric
   form is a small additional implementation cost for a large
   coherence gain.
3. **`payload.rationale` inside `mycelium log <op> --payload-json`
   only.** Rejected: forces every adapter and agent to remember a
   nested-JSON convention rather than the flag they already use on
   `write`/`edit`/`rm`/`mv`. Top-level `--rationale` is uniform.
4. **New `mycelium note` higher-level verb that requires rationale.**
   Deferred (out of scope above). Useful if usage shows agents
   skipping rationale on the things that need it most, but premature
   without evidence.
5. **Lint-style audit (`mycelium audit`).** Deferred (out of scope
   above). Useful for retrospective review; orthogonal to the
   activity-log gap.
6. **Embed rationale only in note content, never in the log.**
   Status quo. Leaves the activity-log gap unaddressed and forces
   every review path through opening individual files.

## Resolved on acceptance

The original Proposed draft included three open questions. They were
resolved during acceptance:

- **CAS conflict envelope:** _Yes_, propagate. Both sides' rationale
  appears in the envelope so the retrying agent can merge intent, not
  just bytes. Implemented as an `omitempty` field on
  `conflictEnvelope`.
- **`mycelium log` flag:** _Yes_, accept `--rationale` as a top-level
  flag on `mycelium log <op>` for symmetry with the mutation verbs.
  An adapter that wants structured rationale-adjacent metadata can
  still use `--payload-json`; the top-level flag avoids forcing every
  consumer through nested JSON.
- **Harness adapter coordination:** _Yes_, encourage. `pi-mycelium`
  v0.2.0 adds a one-line recommendation to its injected system-prompt
  block urging agents to supply `--rationale` on rationale-bearing
  operations. Future adapters are encouraged to follow suit. This is
  documented in `docs/portable-activity-events.md`; it is not enforced
  by the core binary.
