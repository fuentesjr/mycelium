# Strategy: Focus Mycelium on pi Coding Agents

**Status:** Approved
**Date:** 2026-07-12
**Decision owner:** Sal Fuentes Jr.

## Executive summary

Mycelium should become a product built, documented, tested, and supported only
for pi coding agents.

This is a narrowing of the product contract, not a rewrite of the storage
engine. The Go CLI remains the extension's small, testable implementation
boundary for safe filesystem mutations and activity logging. Its plain-file
format and shell interface remain useful because they make memory inspectable,
recoverable, and easy to test—not because Mycelium promises integration with
every shell-capable agent harness.

The intended public model becomes:

> **Persistent memory for pi coding agents: a journal folder, safe mutations,
> and a searchable activity log.**

Other harnesses may happen to be able to invoke the binary, but they are not a
supported use case. The project will not ship harness-neutral integration
guidance, adapter conventions, compatibility fixtures, or roadmap commitments.

## Why narrow now

Mycelium is pre-1.0, and pi is already the only integration with a complete
installation path, lifecycle adapter, starter journal, system prompt, bundled
binary, and test suite. The broader portability promise currently creates more
surface than value:

- the README and design documents describe hypothetical harnesses;
- the portable skill duplicates guidance already delivered by the pi extension;
- portable activity-event conventions and fixtures outlive the reference
  adapter behavior that originally justified them;
- the roadmap includes integrations that have no committed user or maintainer;
- design decisions must account for harnesses the project does not test.

Narrowing now makes the supported product match the product actually being
built. It also lets future design decisions use pi's real lifecycle and
extension APIs instead of preserving abstractions for speculative adapters.

## Strategic decision

### Supported product

Mycelium supports:

- pi coding agents;
- installation through the `pi-mycelium` npm extension;
- global and project-local pi extension scopes;
- the platforms for which the extension publishes and tests bundled binaries;
- the journal format, memory conventions, and lifecycle behavior supplied by
  the pi extension;
- direct CLI use for inspection, diagnostics, development, and the agent's
  normal shell-based memory operations inside pi.

### Unsupported product surface

Mycelium will no longer claim or plan support for:

- Claude Code, Codex, Hermes, generic scripts, or other agent harnesses;
- a portable Agent Skill as an alternative integration;
- a cross-harness adapter API or event vocabulary;
- compatibility guarantees for third-party adapters;
- benchmark or acceptance matrices spanning multiple harnesses.

Unsupported does not mean intentionally blocked. It means the project does not
document, test, release, or make design tradeoffs for those integrations.

## Principles for the transition

1. **Narrow promises before changing architecture.** First remove unsupported
   product commitments. Do not rewrite working internals merely to make them
   look pi-specific.
2. **Keep boundaries that earn their keep.** The Go binary, plain files, CAS,
   durable activity logging, and environment-based extension-to-binary identity
   remain unless a concrete pi-only design is demonstrably simpler.
3. **Delete speculative abstractions.** Portable adapter conventions, generic
   harness examples, and planned integrations should not survive as active
   product concepts.
4. **Preserve user data.** Existing journals and historical activity entries
   remain readable. The transition must not require a journal migration.
5. **Keep history honest.** Existing ADRs remain in the repository as historical
   records, but a new ADR supersedes the portability decisions that are no
   longer active.
6. **Avoid replacement complexity.** Do not introduce a pi-specific protocol,
   TypeScript storage implementation, or new metadata schema unless it removes
   more complexity than it adds.

## Target architecture

The target remains two layers:

```text
pi coding agent
    |
    | system prompt + lifecycle events + environment
    v
pi-mycelium extension
    |
    | invokes bundled mycelium binary
    v
mycelium CLI
    |
    | safe mutations + durable activity entries
    v
local journal directory
```

The responsibilities become explicit:

- **pi-mycelium extension:** installation, scope detection, mount selection,
  identity, journal bootstrap, prompt guidance, and pi lifecycle events.
- **mycelium CLI:** filesystem safety, atomic mutations, CAS, search, and durable
  activity appends.
- **journal directory:** agent-owned durable memory plus system-owned activity
  history.

The CLI is an internal product component with a documented operational surface,
not a promise that every shell-capable harness is supported.

## Workstreams

### 1. Record the decision

Add an ADR that:

- declares pi the sole supported coding-agent harness;
- distinguishes supported integration from incidental CLI usability;
- supersedes ADR-0002's active cross-harness adapter-convention decision;
- revises ADR-0006's remaining references to other adapters;
- states that no journal migration is required;
- records why the CLI remains separate from the extension.

Existing ADRs should be marked superseded where appropriate rather than rewritten
to erase the project's history.

### 2. Reframe the product documentation

Update the public narrative around pi:

- replace the multi-harness README diagram and examples with the pi extension
  flow;
- lead installation with `pi install npm:pi-mycelium`;
- describe source builds and direct CLI use as development, diagnostics, and
  advanced operation rather than a separate integration path;
- change "model-agnostic agent memory system" language to pi-focused language;
- remove Claude, Codex, Hermes, scripts, and "any harness" claims;
- make support boundaries explicit in the README and FAQ;
- ensure release and troubleshooting docs describe only supported pi paths.

The core design principles about plain files and general file operations should
remain, but their justification should be inspectability and model capability
inside pi—not harness portability.

### 3. Remove the portable integration surface

Delete the unsupported integration package and its references:

- remove `skills/mycelium/`;
- remove setup/doctor scripts that exist only for generic skill installation;
- remove OpenAI/Codex skill metadata;
- remove README and changelog instructions that present the skill as a current
  installation option;
- preserve historical release notes as history, adding a new entry that clearly
  records removal.

Before deletion, compare the skill's operational guidance with the extension's
system prompt and bundled `MYCELIUM_MEMORY.md` template. Any guidance still
needed by pi users should have exactly one pi-owned home.

### 4. Collapse activity events to the pi contract

Replace the generic adapter vocabulary with the behavior the pi extension
actually supports:

- document pi session-boundary and compaction events directly;
- remove `docs/portable-activity-events.md` and the representative portable
  fixture, or replace them with a concise pi activity-events document if the
  information is not already covered elsewhere;
- remove the extension test that validates the generic portable fixture;
- remove language encouraging future adapters to emit optional turn, tool, or
  context events;
- retain tolerant reading of historical and unknown activity operations.

The binary's `mycelium log <op>` command should remain generic. It is also used
for agent-authored `decision` and `agent_note` signals, so constraining op names
to pi lifecycle events would reduce useful capability without simplifying the
system.

### 5. Replace the roadmap

Revise `docs/mycelium-phases.md` so future work serves the pi product:

- remove Claude Code, Hermes, and multi-harness acceptance criteria;
- define installation and lifecycle quality in terms of the npm extension;
- make pi session continuity and multi-agent journal use the benchmark target;
- evaluate future features against pi workflows rather than hypothetical
  adapter compatibility;
- retain the LocalFS guarantee and plain-journal design constraints.

This is also an opportunity to separate completed work from future roadmap
items so the roadmap describes decisions still ahead rather than re-documenting
shipped history.

### 6. Align release and maintenance policy

For the release containing this change:

- classify the support narrowing explicitly in the changelog;
- identify the last release that included the portable skill;
- state that existing standalone binaries and journals continue to work, but
  non-pi integrations are unsupported;
- publish and test only the artifacts required by `pi-mycelium`, unless separate
  binary archives are still useful for debugging or development;
- remove non-pi examples from release verification.

Because the project is pre-1.0, this can ship as the next minor release. The
exact version should be chosen when implementation scope is known rather than
encoded in this plan.

## Deliberate non-goals

This transition does **not** initially include:

- rewriting the Go CLI in TypeScript;
- embedding storage logic directly in the pi extension;
- changing the journal layout;
- renaming the `mycelium` binary or npm package;
- restricting `MYCELIUM_AGENT_ID` or `MYCELIUM_SESSION_ID` to pi-specific values;
- rejecting invocations made outside pi;
- deleting historical changelog entries or activity records;
- adding new pi features merely because the product scope changed.

Each could be evaluated later, but none is necessary to make the support
contract honest and substantially smaller.

## Execution sequence

Implement the transition as small, reviewable changes:

1. **Decision commit:** add the superseding ADR and final support-boundary text.
2. **Documentation commit:** reframe README, design, FAQ, and roadmap around pi.
3. **Integration cleanup commit:** remove the portable skill and stale references.
4. **Activity cleanup commit:** remove portable event docs/fixtures/tests and
   consolidate pi event documentation.
5. **Release cleanup commit:** update changelog, release checklist, packaging,
   and verification commands.
6. **Final consistency pass:** search for unsupported harness claims, run all
   Go and extension checks, and inspect packaged npm contents.

The commits may be combined if the changes are too interdependent to leave the
repository coherent between steps, but unrelated architectural refactors should
not be folded into this transition.

## Verification

The transition is complete when:

- the README identifies pi as the sole supported coding-agent harness;
- `pi install npm:pi-mycelium` is the primary installation path;
- no active documentation promises Claude, Codex, Hermes, generic-script, or
  third-party adapter support;
- `skills/mycelium/` and generic portable-event fixtures are absent;
- the active ADR set clearly supersedes the cross-harness strategy;
- pi lifecycle and activity behavior are documented in one place;
- existing journal fixtures remain readable without migration;
- `go test ./...` passes;
- `go test -race ./internal/mycelium` passes;
- `npm test --prefix extensions/pi-mycelium` passes;
- the npm package contains the extension, template, and expected bundled-binary
  dependencies without removed portable artifacts;
- a global and project-local pi installation both bootstrap and resume the
  expected journal path.

## Risks and mitigations

### Risk: narrowing support is mistaken for needless technical coupling

**Mitigation:** keep the CLI and journal format loosely coupled. Narrow the
support policy without adding pi dependencies to the storage engine.

### Risk: useful guidance disappears with the portable skill

**Mitigation:** inventory skill content before deletion and move only unique,
still-needed guidance into the pi system prompt, template, or extension README.
Do not preserve duplicate documents.

### Risk: historical journals contain now-undocumented events

**Mitigation:** document that activity readers tolerate unknown operations and
briefly identify legacy events in compatibility notes. Do not migrate or rewrite
append-only history.

### Risk: source installation becomes unclear

**Mitigation:** retain concise contributor instructions for building the binary
and extension. Remove only the claim that a source-built binary constitutes a
supported non-pi integration.

### Risk: the cleanup grows into a redesign

**Mitigation:** enforce the non-goals above. Complete the support and
narrative narrowing first; evaluate deeper pi-specific simplifications from the
resulting smaller system.

## Follow-up decision gates

After the transition ships, evaluate these separately using evidence from pi
usage:

1. Does publishing standalone binary archives provide enough debugging or
   contributor value to retain them?
2. Can extension-to-binary identity be simplified without weakening concurrent
   agent attribution?
3. Does the activity log need any lifecycle event beyond session boundaries and
   compaction to improve memory continuity?
4. Does the separate Go process create measurable installation, performance, or
   maintenance cost that justifies architectural consolidation?
5. Can the design and roadmap documents be collapsed further now that adapter
   portability is no longer a concern?

None of these questions blocks the initial strategy. The first objective is to
make the project's promises match its actual pi-focused product.
