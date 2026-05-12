# ADR 0003: Optional rationale on `write` and `edit`

- **Status:** Proposed
- **Date:** 2026-05-12
- **Deciders:** Sal Fuentes Jr.

## Context

Mycelium asks agents to capture rationale at the moment of decision. For
structural decisions, the `evolve` command enforces this: `--rationale` is
required, and the binary errors out (`mycelium evolve: --rationale is
required`) when it is missing (`cmd/mycelium/evolve.go:295`).

For per-note content, no such enforcement exists. The README's "What agents
record" section frames rationale capture as a discipline that applies to
both surfaces — file contents *and* `evolve` events — but `cmd/mycelium/write.go`
and `cmd/mycelium/edit.go` contain no references to rationale. A note may
be written with no reasoning at all, and the binary will accept it.

This creates an asymmetry:

- *Why-this-pattern* (structural decisions) → guaranteed in the activity
  log via `evolve`.
- *Why-this-thing* (per-note reasoning) → present only by convention, in
  the body of whatever note the agent happens to write.

A reader auditing a mount cannot tell, from the activity log alone, why
any particular write happened. They have to open the file and hope the
agent embedded reasoning in the body. For long-running mounts and
cross-agent handoff — the cases mycelium is built for — that gap is real.

The naive fix is to require `--rationale` on `write` and `edit`. This
was rejected. Many legitimate writes have no separable rationale:
appending to a running TODO list, updating a status file, saving a
downloaded artifact, rebuilding an index. Forcing rationale on these
produces placeholder text ("updating file", "syncing notes"), which is
worse than no field — it gives false confidence that reasoning is being
captured when it isn't.

## Decision

Add an **optional** `--rationale` flag to `mycelium write` and `mycelium
edit`. When supplied, it is captured into the corresponding activity log
entry as a new top-level field, parallel to how `evolve` records
rationale today. When absent, behavior is unchanged and the field is
omitted from the log entry.

The note-body discipline ("write the *why* into the note itself")
remains a documented convention. Documentation will stop framing it as a
property the binary guarantees and start framing it as a craft norm,
distinct from the operational rationale captured on the activity log
line.

### Schema

`LogEntry` in `cmd/mycelium/log.go` gains:

```go
Rationale string `json:"rationale,omitempty"`
```

- Maximum size: 64 KiB, matching `maxRationaleSize` from
  `cmd/mycelium/evolve.go`. Oversize input is rejected before the
  mutation runs (exit code `ExitUsage`).
- `omitempty` ensures the field is absent from log entries when no
  rationale is supplied — existing log readers and JSONL fixtures
  remain valid without migration.
- The field appears on the same JSONL line as the mutation entry, so
  `tail -f`, `grep`, and `mycelium grep --path _activity` surface it
  without indirection.

### CLI surface

```
mycelium write notes/incident-2026-05-12.md \
  --rationale "API began returning 503 at 14:22; recording symptoms before mitigation closes the window."

mycelium edit notes/runbook.md \
  --old-string "..." --new-string "..." \
  --rationale "Removing the manual restart step; the operator-init bug it worked around was fixed in v0.1.7."
```

### Out of scope

- `rm` and `mv` rationale capture. Likely desirable for consistency but
  lower priority; deferred to a follow-up ADR if usage shows a gap.
- A higher-level `mycelium note` verb that requires rationale for
  rationale-bearing directories. Deferred until there is evidence of
  which directories want enforcement; this ADR does not preclude it.
- Conflict envelope propagation. The losing agent's rationale on a CAS
  write conflict — whether it should appear in the conflict envelope —
  is an open question (see below).

## Consequences

### Positive

- **Activity log becomes self-sufficient for many review tasks.** A
  reader can reconstruct *why-this-write* by reading the log line,
  without opening every note. Operational reasoning travels with the
  operation.
- **Symmetry with `evolve`.** Rationale is a first-class field on every
  rationale-bearing op, with the same flag name, the same size bound,
  and the same on-disk shape.
- **No coercion of low-signal writes.** Routine writes (status updates,
  appends, artifacts) carry no rationale and produce no placeholder
  text. The schema reflects the actual epistemic state.
- **Reversible adoption.** Harness prompts can be tightened over time to
  encourage `--rationale` for specific directories without any
  binary-level coupling.

### Negative

- **Surface area on the two most-used verbs.** `write` and `edit` gain
  a new flag. The flag is optional, so existing callers continue to
  work, but documentation, help text, and examples grow.
- **No enforcement is still no guarantee.** Agents that omit
  `--rationale` produce log entries without reasoning, just as today.
  Adoption depends on harness prompts and operator discipline.
- **Activity log entry size grows when used.** With a 64 KiB cap and
  rationales expected to run a few hundred bytes in practice, the
  impact on log growth is modest but nonzero.
- **Tooling must tolerate the new field.** Anything that consumes
  `LogEntry` JSON should treat `rationale` as optional. Since the field
  is `omitempty`, strict-parsing readers that ignore unknown fields are
  unaffected.

### Neutral

- **No migration.** Existing activity log entries remain valid; new
  entries optionally include the field.
- **Schema versioning.** This change should land before Phase 2's
  activity log schema is frozen, so the field is part of v1 rather than
  a v2 addition.

## Alternatives considered

1. **Require `--rationale` on every `write`/`edit`.** Rejected:
   produces placeholder rationale on legitimate rationale-free writes,
   which is worse than no field. Garbage in the rationale slot makes
   the slot meaningless across the corpus.
2. **New `mycelium note` higher-level verb that requires rationale.**
   Deferred. Useful if usage shows agents skipping rationale on the
   things that need it most, but premature without evidence. Can be
   added later as a thin wrapper without breaking the primitives.
3. **Lint-style audit (`mycelium audit notes/incidents/`).** Deferred.
   Useful for after-the-fact review of rationale density in chosen
   directories, but orthogonal to the activity-log gap this ADR
   closes.
4. **Embed rationale only in note content, never in the log.** Status
   quo. Leaves the activity-log gap unaddressed and forces every
   review path through opening individual files.

## Open questions

- Should `--rationale` propagate to the CAS conflict envelope
  (`conflictEnvelope` in `cmd/mycelium/write.go`)? The losing agent's
  rationale could help whoever investigates the conflict understand
  what each side was trying to accomplish. Probably yes, but worth
  validating against a real conflict workflow before committing.
- Should `mycelium log <op> --rationale ...` accept the same flag for
  agent-authored signal entries, or is `payload.rationale` (inside the
  JSON payload) sufficient? Consistency argues for the top-level
  flag.
- How should harness adapters (pi-mycelium and successors) surface the
  rationale to the agent at decision points? Likely a prompt-side
  concern, not a binary concern, but worth coordinating before
  publishing.
