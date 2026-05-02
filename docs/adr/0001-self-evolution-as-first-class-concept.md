# ADR 0001: Self-evolution as a first-class concept

- **Status:** Accepted
- **Date:** 2026-05-01
- **Deciders:** Sal Fuentes Jr.

## Context

Mycelium's original design (see [`docs/self-evolution.md`](../self-evolution.md)) treated agent self-evolution — adopting and retiring conventions, building self-built indices, archiving stale regions of the store, recording lessons — as **emergent agent behavior** rather than a backend concept. The activity log captured raw operations (`write`, `edit`, `rm`, `mv`, `context_signal`, `session_*`) but had no notion of *evolution events*. The expectation was that an evaluator (or a future-self agent) could derive evolution patterns post-hoc from those raw traces: "first write under `notes/incidents/` = a new convention", "writes to `MYCELIUM_MEMORY.md` = a policy change", "`mv` into `archive/` = archiving".

In practice this fails on three axes:

1. **Cross-session blindness.** When a new pi session starts, the agent has no structured way to discover what conventions its prior selves adopted. It must re-read `MYCELIUM_MEMORY.md` and reconstruct the rules from prose, with no machine-readable "active conventions" surface. This makes convention drift across sessions invisible until something breaks.

2. **Benchmark un-scoreability.** Phase 1's T2 ("seeded self-evolution") rubric depends on detecting whether an agent extends, retires, or replaces seeded conventions. With evolution buried in raw `write`/`mv` traces, every benchmark run requires bespoke parsing heuristics to reconstruct what the agent decided. Scoring becomes fragile and reviewer-dependent rather than mechanical.

3. **Lossy archaeology.** A `write` to `MYCELIUM_MEMORY.md` records that the file changed, but not *why*: was a convention introduced, retired, or refined? Was a lesson distilled from a specific incident? The `rationale` is exactly the signal future-self agents and evaluators need, and it has no place to live in the current schema.

The user's directive (this conversation, 2026-05-01): "self-evolution should be a first-class concept in mycelium and part of the design."

## Decision

Mycelium will model self-evolution as an explicit, first-class concept in both the activity log schema and the CLI surface.

### Activity-log schema addition

A new `op` value `evolve` with payload:

```json
{
  "op": "evolve",
  "kind": "<string; either a built-in kind or an agent-introduced one>",
  "target": "<optional opaque agent-chosen string scoping the evolution>",
  "rationale": "<required free-text explanation, max 64 KB>",
  "supersedes": "<optional ULID of a prior evolve event this explicitly replaces>",
  "id": "<ULID, minted on write>",
  "kind_definition": "<required on the first use of a non-builtin kind; declares the kind's meaning>"
}
```

Field semantics:

- **`target`** is an **opaque agent-chosen string**. The binary does not validate it, glob-expand it, or check that it refers to an existing path. Agents will commonly use mount-relative paths or globs (e.g. `notes/incidents/`, `notes/incidents/*.md`) but other identifiers are valid (e.g. a topic name, a project slug, or empty for kinds that aren't path-scoped like `lesson`).
- **`id`** is a [ULID](https://github.com/ulid/spec) — sortable, 26 chars, mint-on-write. (Resolved from prior open question; ULID picked for monotonic sortability without coordinator.)
- **`source`** is **never stored in the event**. It is a synthetic field appearing only in `mycelium evolution --kinds` output, derived by checking each kind name against the built-in registry baked into the binary.

The `kind` taxonomy is **open and agent-adaptable**, but ships with five **built-in kinds** that bootstrap the system so agents have a usable vocabulary out of the box without ceremony. Agents may use the built-ins as-is, ignore them, or introduce their own kinds. Built-in and agent-introduced kinds coexist on equal footing in the activity log; both are queryable via `mycelium evolution --kinds`, distinguished by the synthetic `source` field. Supersession (chain semantics) is described below.

#### Built-in kinds (shipped with mycelium)

The five built-ins sit on different axes: `convention` describes current data structure, `index` describes derived data over that structure, `archive` describes deactivated regions, `lesson` describes accumulated knowledge, and `question` describes accumulated *not*-knowing — open unknowns the agent is tracking and expects to resolve into lessons (or supersede as no-longer-relevant) later.

| kind | definition | example |
|------|------------|---------|
| `convention` | A naming, layout, or structural pattern for organizing data in the store. | "Incidents go under `notes/incidents/<date>-<slug>.md`." |
| `index` | A derived or summary file the agent has built or regenerated over a region of the store. | A TOC at `notes/incidents/INDEX.md` regenerated nightly. |
| `archive` | A region of the store the agent has marked as no-longer-active and moved out of working scope. | "Pre-2026 incidents moved to `archive/incidents-2025/`." |
| `lesson` | A distilled insight from past work, intended to inform future behavior. | "Queries that mention `latency` correlate with deploy events 80% of the time." |
| `question` | An open unknown the agent is tracking, expected to resolve into a `lesson` (or be superseded as no-longer-relevant) later. | "Does the GLP-1 cardio effect generalize to non-diabetic populations? Need 3+ independent studies before concluding." |

These ship with mycelium itself; no `kind_definition` is required when an agent first uses them. Agent-introduced kinds (e.g. `experiment`, `hypothesis`, `policy`, `dead-end`, `decision`) **must** include `kind_definition` on first use so the meaning is captured for future-self discovery.

#### Supersession semantics

Supersession is **implicit by `(kind, target)` pair** — emitting a new `evolve` event with the same `kind` and `target` as an existing active entry automatically retires the prior entry. The new event records the prior id in `supersedes` (the binary fills it in). This is the common case and avoids forcing the agent to look up the prior id by hand.

`--supersedes <id>` may be passed explicitly when retiring an entry whose `(kind, target)` does not match — for example, "this lesson replaces an earlier different lesson at a different target." Explicit supersession overrides the implicit `(kind, target)` rule.

Supersession also governs **kind-definition redefinition**. When an agent redefines a kind (re-uses a kind name with a new `kind_definition`), the binary writes a synthetic `evolve` event of the reserved built-in kind `_kind_definition` with `target: "<kindname>"`, supersession-chained to any prior `_kind_definition` entry for the same target. This keeps the taxonomy's own evolution queryable without introducing a separate machinery for it.

`--supersedes` referencing a non-existent id, or one whose `kind` differs from the new event's `kind`, is rejected with exit 65 and a clear error.

#### Validation and limits

- `kind` must be a non-empty string ≤ 64 chars, matching `^[a-zA-Z0-9_-]+$`. (`_`-prefixed kinds are reserved for binary-internal use such as `_kind_definition`; agent-issued events that try to use a `_`-prefixed `kind` are rejected with exit 65.)
- `rationale` is required, non-empty, and capped at 64 KB. Larger payloads are rejected with exit 65.
- `kind_definition` is required on the first use of a non-builtin kind; subsequent uses may omit it (or supply a new one to redefine, which triggers the `_kind_definition` chain above).
- `target` is unvalidated free-form, capped at 4 KB.

#### Concurrency

`mycelium evolve` acquires the **mount-level `flock`** the same way `write`, `edit`, `rm`, and `mv` do. This guarantees monotonic ordering of supersession chains across concurrent agents on the same mount.

#### Example payloads

A new convention, no prior entry for this `(kind, target)`:

```json
{"ts":"2026-05-02T14:00:00Z","agent_id":"researcher-7","session_id":"abc123","op":"evolve","id":"01HXKP4Z9M8YV1W6E2RTSA9KFG","kind":"convention","target":"notes/incidents/","rationale":"Adopting <date>-<slug>.md filenames so incidents sort chronologically without a separate index."}
```

A retiring/replacement (implicit supersession on `(convention, notes/incidents/)`):

```json
{"ts":"2026-05-09T11:30:00Z","agent_id":"researcher-7","session_id":"def456","op":"evolve","id":"01HXM2C0JD7H9ASBQYNV6XGGT2","kind":"convention","target":"notes/incidents/","supersedes":"01HXKP4Z9M8YV1W6E2RTSA9KFG","rationale":"Switching to YYYY/MM/<slug>.md after the year wrapped — flat-date layout was getting unwieldy."}
```

An agent-introduced kind on first use:

```json
{"ts":"2026-05-12T09:15:00Z","agent_id":"researcher-7","session_id":"ghi789","op":"evolve","id":"01HXN9QXRG6J3PTKB4WHCEM2YS","kind":"experiment","target":"hypotheses/glp1-cardio.md","kind_definition":"An in-progress hypothesis I'm actively testing against new evidence. Distinct from `lesson` (closed-out insight) and `policy` (behavioral rule).","rationale":"Tracking the GLP-1 cardio-protection question as an open thread until I have N=3 independent supporting papers."}
```

### CLI surface addition

A new subcommand:

```
mycelium evolve <kind> [--target <str>] [--supersedes <id>] [--kind-definition "..."] --rationale "..."
```

Emits the activity-log entry and prints `{"id":"...","supersedes":"..."}` to stdout (the `supersedes` field is included whenever a prior entry was retired, whether implicitly via `(kind, target)` matching or explicitly via the flag — so the agent always sees the chain it just extended). No mutation to the store proper — `evolve` is metadata about the agent's reasoning, not data.

A read-side companion:

```
mycelium evolution [--kind X] [--since DATE] [--active] [--format json]
mycelium evolution --kinds [--format json]
```

Filters the activity log and returns the evolution timeline. `--active` returns only the latest non-superseded entry per `(kind, target)` pair — the "current rules in effect" view. `--kinds` returns the distinct kinds available in this mount: built-ins (always present, with their shipped definitions) plus any agent-introduced kinds (with their currently-active `kind_definition` and event-counts). Each entry carries the synthetic `source: "builtin" | "agent"` field so callers can distinguish the two; built-ins also carry `defined_at_version` recording the mycelium version that introduced them, so future built-in additions don't silently shadow earlier agent-introduced kinds of the same name (both rows continue to exist; the `source` field disambiguates).

### Cross-session continuity

The pi-mycelium extension's `before_agent_start` system-prompt block will:

1. **Embed the output of `mycelium evolution --kinds --format json`** at session start so the agent sees the current vocabulary (built-ins plus any agent-introduced kinds and their definitions). The block does not hardcode the built-in kind list as prose — that would let the binary and extension drift out of sync. Instead the binary is the single source of truth; the extension just renders what it returns.
2. Pre-surface the active-evolution view (or a count + pointer to `mycelium evolution --active`) so a fresh session inherits its predecessors' conventions without manually re-reading `MYCELIUM_MEMORY.md`.
3. Instruct the agent on when to invoke `mycelium evolve`: when adopting or retiring a convention, building or regenerating an index, archiving a region, recording a policy, distilling a lesson, or coining a new kind for behavior the built-ins don't cover.

### Boundaries

Two boundaries that scope what `evolve` is and is not:

- **`evolve` is strictly metadata; it never mutates the store.** `mycelium evolve archive --target old-notes/` records the decision but does not move files. The agent runs `mycelium mv old-notes/ archive/old-notes/` as a separate, explicit call. This preserves single responsibility (one op per concern), avoids partial-rollback complexity (no two-system atomicity), and matches the asymmetric reality that only some kinds (`archive`, `index`) even have an associated mutation. Mount-level `flock` already serializes paired calls, so the atomicity-via-coupling argument doesn't carry. A future sugar wrapper that combines paired calls is Phase 2 if demand emerges.
- **`evolve` and `MYCELIUM_MEMORY.md` are independent; the activity log is authoritative.** Updating `MYCELIUM_MEMORY.md` is not a required side effect of any `evolve` event, and vice versa. The two serve different purposes: the activity log is the immutable event record, while `MYCELIUM_MEMORY.md` is the agent's editorialized prose summary that it chooses what to elevate into. Forced synchronization breaks legitimate workflows (e.g. batching three related conventions into one prose paragraph after the fact). When the two diverge, the activity log wins by definition. Tooling may later surface divergence warnings as an observability concern; this ADR does not prescribe such tooling.

### Documentation alignment

- [`docs/self-evolution.md`](../self-evolution.md) is rewritten from "agents do this ad-hoc with plain writes" to "agents do this via `mycelium evolve`, supplemented by `MYCELIUM_MEMORY.md` for prose context."
- [`mycelium-design.md`](../../mycelium-design.md) gains a section on evolution as a first-class concept and the `kind` taxonomy.
- The pi-mycelium system-prompt block instructs agents on when and how to invoke `mycelium evolve`.

## Consequences

### Positive

- **Cross-session continuity becomes mechanical.** A new session can query `mycelium evolution --active` and see what conventions are currently in force, without re-reading prose.
- **The agent's taxonomy is itself first-class.** `mycelium evolution --kinds` exposes the vocabulary the agent has evolved — what categories of evolution it tracks, how it defines them, how often it uses them. This is meta-learning made legible.
- **T2 benchmark scoring shifts to kind-agnostic metrics.** Rather than checking specific kinds, scoring measures the *shape* of evolution: count of distinct kinds introduced, supersession-chain depth, ratio of `evolve` events to plain mutations, coherence between `MYCELIUM_MEMORY.md` references and backing `evolve` ids. These work regardless of taxonomy choice and reward genuine reflection over kind-checkbox-filling.
- **Future-self archaeology gets the rationale signal.** Why a convention was adopted/retired is captured at the moment of decision, not lost to whichever commit message or prose blob the agent happened to write at the time.
- **Open taxonomy aligns with the original "agents derive structure" philosophy.** Self-evolution becomes first-class without prescribing what evolution looks like — the machinery is fixed, the vocabulary stays the agent's.

### Negative

- **Cross-mount inconsistency in agent-introduced kinds.** Two agents on two mounts may evolve incompatible custom kind taxonomies, complicating any tooling that wants to compare across stores. Mitigation: the five built-in kinds provide a hard floor of shared vocabulary that all mounts have; cross-mount tooling can rely on them. Divergence only matters above that floor, which is also where the agent-specific signal lives.
- **Scoring mechanics must adapt.** Kind-agnostic metrics are robust but less expressive than "did the agent introduce a `convention`?" — some benchmark questions get harder. Mitigation: T2 rubric explicitly frames evolution-shape metrics, not kind-presence metrics.
- **Adoption gap.** The new op only matters if agents actually use it. The system-prompt instructions and benchmark rubric must reinforce the habit, or `evolve` becomes a museum-piece subcommand.
- **Schema breakage risk.** Adding a new `op` is non-breaking for readers (unknown ops can be skipped), but downstream tools that pattern-match on the existing op set will need to be updated.

### Neutral

- **No data migration.** Existing stores remain valid; `evolve` is purely additive.
- **Phase placement.** This is a Phase 1 amendment, not Phase 2 — it changes the binary surface and the system-prompt block, both of which were just released as 0.0.1. Either bump to 0.1.0 with `evolve` included or ship 0.0.2 as the amendment.

## Open questions

None. All open questions raised during ADR review have been resolved: `id` format (ULID), kind redefinition (folded into supersession via reserved `_kind_definition` kind), mutation coupling (strictly metadata — see Boundaries), and `MYCELIUM_MEMORY.md` reconciliation (independent — see Boundaries).
