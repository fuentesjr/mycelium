# Architecture Decision Records

Significant design decisions for Mycelium are recorded here as ADRs using the [Michael Nygard format](https://github.com/joelparkerhenderson/architecture-decision-record/blob/main/locales/en/templates/decision-record-template-by-michael-nygard/index.md): **Status**, **Context**, **Decision**, **Consequences**.

An accepted ADR's decision, context, and consequences are immutable. To revise
a decision, write a new ADR that supersedes the old one and update the old
ADR's status to `Superseded by ADR-NNNN`. Dated factual errata and
current-implementation warnings may be appended. Factual inaccuracies and
broken command examples may also be corrected in place when the correction is
identified by a dated clarification and does not change the recorded decision.

Filename convention: `NNNN-kebab-case-title.md`, sequentially numbered.

## Index

- [ADR 0001](0001-self-evolution-as-first-class-concept.md) — Self-evolution as a first-class concept (**superseded by ADR 0004**)
- [ADR 0002](0002-portable-activity-events-as-adapter-conventions.md) — Portable activity events as adapter conventions (**superseded by ADR 0007**)
- [ADR 0003](0003-optional-rationale-on-write-and-edit.md) — Optional rationale on rationale-bearing operations
- [ADR 0004](0004-conventions-as-files.md) — Conventions as files
- [ADR 0005](0005-activity-log-as-durable-history.md) — Activity log as durable history
- [ADR 0006](0006-reference-adapter-memory-relevant-events.md) — Reference adapter emits only memory-relevant events (**superseded by ADR 0007**)
- [ADR 0007](0007-pi-only-coding-agent-harness.md) — pi is the sole supported coding-agent harness
