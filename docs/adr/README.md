# Architecture Decision Records

Significant design decisions for Mycelium are recorded here as ADRs using the [Michael Nygard format](https://github.com/joelparkerhenderson/architecture-decision-record/blob/main/locales/en/templates/decision-record-template-by-michael-nygard/index.md): **Status**, **Context**, **Decision**, **Consequences**.

ADRs are immutable once accepted. To revise a decision, write a new ADR that supersedes the old one and update the old ADR's status to `Superseded by ADR-NNNN`.

Filename convention: `NNNN-kebab-case-title.md`, sequentially numbered.

## Index

- ADR 0001 — Self-evolution as a first-class concept
- ADR 0002 — Portable activity events as adapter conventions (**superseded by ADR 0007**)
- ADR 0003 — Optional rationale on write and edit
- ADR 0004 — Conventions as files
- ADR 0005 — Activity log as durable history
- ADR 0006 — Reference adapter emits only memory-relevant events (**superseded by ADR 0007**)
- ADR 0007 — pi is the sole supported coding-agent harness
