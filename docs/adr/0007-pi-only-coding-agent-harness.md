# ADR 0007: pi is the sole supported coding-agent harness

- **Status:** Accepted
- **Date:** 2026-07-12
- **Deciders:** Sal Fuentes Jr.
- **Supersedes:** ADR-0002, ADR-0006

> Operational clarification (2026-07-17): pi lifecycle activity writes and
> first-file bootstrap are best-effort so hook failures do not prevent pi from
> starting. This does not change the pi-only support decision.

## Context

Mycelium was originally described as a harness-neutral memory substrate with a portable Agent Skill and cross-harness activity-event conventions. In practice, the only complete integration is `pi-mycelium`: it installs through pi, selects the journal mount, ships the starter template, injects the runtime prompt, records pi lifecycle events, and bundles the Go CLI engine.

Keeping generic harness promises adds product surface the project does not test or maintain. The Go CLI and plain journal format still earn their place because they provide safe mutations, CAS, durable activity entries, local inspection, and repeatable tests. They do not imply support for every shell-capable agent harness.

## Decision

Mycelium supports pi coding agents only. The supported product path is:

```text
pi coding agent
    -> pi-mycelium extension
    -> bundled mycelium Go CLI engine
    -> local journal directory
```

The project will document, test, and release the pi extension and the Go CLI engine needed by that extension. Direct CLI use remains documented for development, diagnostics, advanced operation, and pi's shell-invoked memory commands. Other harnesses may happen to invoke the binary, but Claude Code, Codex, Hermes, generic scripts, custom harnesses, and third-party adapters are unsupported product surfaces.

The portable Agent Skill and portable adapter/event vocabulary are removed from active documentation. The pi activity contract is session-boundary entries and `compaction` from the extension, core mutation entries from the CLI, and optional agent-authored `decision` / `agent_note` signals through `mycelium log`. Activity readers and docs remain tolerant of historical and unknown operations; no journal migration is required.

The Go CLI remains separate from the TypeScript extension because filesystem safety, CAS, conflict envelopes, durable append semantics, and packageable platform binaries are clearer and more testable at that boundary.

## Consequences

### Positive

- Product promises now match the integration that is actually installed, tested, and maintained.
- The project can use pi lifecycle and extension APIs directly without preserving abstractions for speculative adapters.
- Existing journals, activity entries, and standalone binaries remain readable and useful for diagnostics.

### Negative

- Users of the removed portable skill must migrate any local guidance into their own prompts or use pi.
- Downstream tooling based on the old portable fixture cannot treat that fixture as an active compatibility target.

### Neutral

- The binary still accepts generic `mycelium log <op>` entries; constraining operation names would reduce useful agent-authored signals without simplifying storage.
- Historical ADRs, changelog entries, and activity logs may mention the former portability direction. They remain history and are marked superseded where appropriate.
