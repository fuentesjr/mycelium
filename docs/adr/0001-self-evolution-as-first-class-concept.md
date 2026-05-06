# ADR 0001: Self-evolution as a first-class concept

- **Status:** Accepted
- **Date:** 2026-05-01
- **Deciders:** Sal Fuentes Jr.

## Context

Mycelium's original design treated agent self-evolution — adopting and retiring conventions, building self-built indices, archiving stale regions of the store, recording lessons — as **emergent agent behavior** rather than a first-class concept. The activity log captured raw operations (`write`, `edit`, `rm`, `mv`, `context_signal`, `session_*`) but had no notion of _evolution events_. The expectation was that an evaluator or future-self agent could derive evolution patterns post-hoc from raw traces: "first write under `notes/incidents/` = a new convention", "writes to `MYCELIUM_MEMORY.md` = a policy change", "`mv` into `archive/` = archiving".

In practice this fails on three axes:

1. **Cross-session blindness.** When a new session starts, the agent has no structured way to discover what conventions its prior selves adopted. It must re-read prose and reconstruct the rules.
2. **Benchmark un-scoreability.** Seeded self-evolution tasks need to detect whether an agent extends, retires, or replaces conventions. Reconstructing that from arbitrary file edits requires fragile bespoke parsing.
3. **Lossy archaeology.** A `write` to `MYCELIUM_MEMORY.md` records that a file changed, but not _why_. The rationale is exactly the signal future-self agents and evaluators need.

The user directive: self-evolution should be a first-class concept in Mycelium and part of the design.

## Decision

Mycelium models self-evolution as explicit activity-log metadata and exposes it through a single CLI command: `mycelium evolve`.

The command has record and query modes:

```sh
# Record
mycelium evolve <kind> [--target <str>] [--supersedes <id>] \
  [--kind-definition "..."] --rationale "..."

# Query
mycelium evolve --list   [--kind X] [--since DATE] [--format json|text]
mycelium evolve --active [--kind X] [--since DATE] [--format json|text]
mycelium evolve --kinds  [--format json|text]
```

There is no separate `mycelium evolution` command. Keeping record and query under one verb matches the model: evolution is one system-owned metadata surface, not two concepts.

### Activity-log schema addition

A new `op` value `evolve` with payload:

```json
{
  "op": "evolve",
  "kind": "<string; either a built-in kind or an agent-introduced one>",
  "target": "<optional opaque agent-chosen string scoping the evolution>",
  "rationale": "<required free-text explanation, max 64 KB>",
  "supersedes": "<optional ULID of a prior evolve event this replaces>",
  "id": "<ULID, minted on write>",
  "kind_definition": "<required on the first use of a non-builtin kind; declares the kind's meaning>"
}
```

Field semantics:

- **`target`** is an opaque agent-chosen string. The binary does not validate it, glob-expand it, or check that it refers to an existing path. Agents commonly use mount-relative paths or globs, but topic names, project slugs, and empty target are valid.
- **`id`** is a [ULID](https://github.com/ulid/spec) — sortable, 26 chars, mint-on-write.
- **`source`** is never stored in the event. It is a synthetic field appearing only in `mycelium evolve --kinds` output, derived by checking each kind name against the built-in registry baked into the binary.

The `kind` taxonomy is **open and agent-adaptable**, but ships with five built-in kinds that bootstrap the system so agents have a usable vocabulary out of the box without ceremony. Agents may use the built-ins as-is, ignore them, or introduce their own kinds. Built-in and agent-introduced kinds coexist on equal footing in the activity log; both are queryable via `mycelium evolve --kinds`, distinguished by the synthetic `source` field.

### Built-in kinds

| kind         | definition                                                                                                               | example                                                                                                                                                         |
| ------------ | ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `convention` | A naming, layout, structural, or behavioral pattern for organizing or operating on the store.                            | "Incidents go under `notes/incidents/<date>-<slug>.md`." Behavioral example: `--target memory-discipline --rationale "Record durable preferences proactively."` |
| `index`      | A derived or summary file the agent has built or regenerated over a region of the store.                                 | A TOC at `notes/incidents/INDEX.md` regenerated when incident notes change.                                                                                     |
| `archive`    | A region of the store the agent has marked as no-longer-active and moved out of working scope.                           | "Pre-2026 incidents moved to `archive/incidents-2025/`."                                                                                                        |
| `lesson`     | A distilled insight from past work, intended to inform future behavior.                                                  | "Queries mentioning `latency` often correlate with deploy events."                                                                                              |
| `question`   | An open unknown the agent is tracking, expected to resolve into a `lesson` or be superseded as no-longer-relevant later. | "Does the GLP-1 cardio effect generalize to non-diabetic populations?"                                                                                          |

Agent-introduced kinds (e.g. `experiment`, `hypothesis`, `policy`, `dead-end`, `decision`) must include `--kind-definition` on first use so the meaning is captured for future-self discovery.

### Supersession semantics

Supersession has two modes.

**Implicit targeted supersession.** If `target` is non-empty, emitting a new event with the same `(kind, target)` as an existing active entry automatically retires the prior entry. The new event records the prior id in `supersedes`; the CLI prints it so the agent sees the chain it extended.

**No implicit supersession for empty target.** Targetless evolution events are additive by default. This prevents unrelated targetless lessons or questions from replacing each other merely because both have `(kind, "")`.

**Explicit supersession.** `--supersedes <id>` may be passed when the agent intentionally retires a prior event. Explicit supersession may cross kind boundaries; this supports normal workflows such as resolving a `question` into a `lesson`. The referenced id must exist and must not create an invalid cycle. Non-existent ids are rejected with exit 65 and a clear error.

Supersession also governs kind-definition redefinition. When an agent redefines a kind by supplying a new `--kind-definition`, the taxonomy's evolution is recorded in the activity log so `mycelium evolve --kinds` can report the currently-active definition. The exact internal representation is system-owned metadata and hidden from user-facing `--list` output.

### Validation and limits

- `kind` must be a non-empty string ≤ 64 chars, matching `^[a-zA-Z0-9_-]+$`. `_`-prefixed kinds are reserved for binary-internal use; agent-issued events that try to use a `_`-prefixed `kind` are rejected with exit 65.
- `rationale` is required, non-empty, and capped at 64 KB. Larger payloads are rejected with exit 65.
- `kind_definition` is required on the first use of a non-builtin kind; subsequent uses may omit it or supply a new one to redefine the kind.
- `target` is unvalidated free-form, capped at 4 KB.

### Concurrency and durability

`mycelium evolve` acquires the mount-level `flock` used by other mutating commands. This guarantees monotonic ordering of supersession chains across concurrent agents on the same local store.

Evolution entries are activity-log entries, and the activity log is authoritative. `mycelium evolve` returns success only after its activity entry is durable.

### Example payloads

A new targeted convention:

```json
{
  "ts": "2026-05-02T14:00:00Z",
  "agent_id": "researcher-7",
  "session_id": "abc123",
  "op": "evolve",
  "id": "01HXKP4Z9M8YV1W6E2RTSA9KFG",
  "kind": "convention",
  "target": "notes/incidents/",
  "rationale": "Adopting <date>-<slug>.md filenames so incidents sort chronologically without a separate index."
}
```

A revision using implicit targeted supersession:

```json
{
  "ts": "2026-05-09T11:30:00Z",
  "agent_id": "researcher-7",
  "session_id": "def456",
  "op": "evolve",
  "id": "01HXM2C0JD7H9ASBQYNV6XGGT2",
  "kind": "convention",
  "target": "notes/incidents/",
  "supersedes": "01HXKP4Z9M8YV1W6E2RTSA9KFG",
  "rationale": "Switching to YYYY/MM/<slug>.md after the year wrapped — flat-date layout was getting unwieldy."
}
```

A targetless lesson, additive by default:

```json
{
  "ts": "2026-05-10T09:00:00Z",
  "agent_id": "researcher-7",
  "session_id": "def456",
  "op": "evolve",
  "id": "01HXM7FA0Y7C2S6FZWQ8W2X7E1",
  "kind": "lesson",
  "rationale": "For library-internals questions, prefer source permalinks over secondary summaries."
}
```

An agent-introduced kind on first use:

```json
{
  "ts": "2026-05-12T09:15:00Z",
  "agent_id": "researcher-7",
  "session_id": "ghi789",
  "op": "evolve",
  "id": "01HXN9QXRG6J3PTKB4WHCEM2YS",
  "kind": "experiment",
  "target": "hypotheses/glp1-cardio.md",
  "kind_definition": "An in-progress hypothesis I'm actively testing against new evidence. Distinct from `lesson` (closed-out insight) and `question` (passive unknown).",
  "rationale": "Tracking the GLP-1 cardio-protection question as an open thread until I have N=3 independent supporting papers."
}
```

## Cross-session continuity

The pi-mycelium extension's `before_agent_start` system-prompt block may:

1. Embed or summarize the output of `mycelium evolve --kinds --format json` so the agent sees the current vocabulary.
2. Pre-surface the active-evolution view, or a count plus pointer to `mycelium evolve --active`, so a fresh session inherits prior conventions without manually reconstructing them from prose.
3. Instruct the agent when to invoke `mycelium evolve`: adopting or retiring a convention, building or regenerating an index, archiving a region, distilling a lesson, tracking a question, or coining a new kind.

This is not automatic retrieval of memory content. It is a small system-metadata view whose purpose is continuity of the agent's own operating conventions.

## Boundaries

- **`evolve` is strictly metadata; it never mutates agent-authored files.** `mycelium evolve archive --target old-notes/` records the decision but does not move files. The agent runs `mycelium mv old-notes/ archive/old-notes/` as a separate, explicit call.
- **`evolve` and `MYCELIUM_MEMORY.md` are independent; the activity log is authoritative.** Updating `MYCELIUM_MEMORY.md` is not a required side effect of any `evolve` event, and vice versa. The two serve different purposes: the activity log is the immutable event record, while `MYCELIUM_MEMORY.md` is the agent's editorialized prose summary that it chooses what to elevate into. When the two diverge, the activity log wins by definition.

## Consequences

### Positive

- **Cross-session continuity becomes mechanical.** A new session can query `mycelium evolve --active` and see what conventions are currently in force.
- **The agent's taxonomy is itself first-class.** `mycelium evolve --kinds` exposes the vocabulary the agent has evolved — what categories of evolution it tracks, how it defines them, how often it uses them.
- **Future-self archaeology gets the rationale signal.** Why a convention was adopted or retired is captured at the moment of decision.
- **Open taxonomy aligns with the original "agents derive structure" philosophy.** Self-evolution becomes first-class without prescribing what evolution looks like.
- **One command avoids surface-area drift.** `evolve` as both recorder and projector avoids a near-duplicate `evolution` command.

### Negative

- **Cross-mount inconsistency in agent-introduced kinds.** Two stores may evolve incompatible custom kind taxonomies. The five built-ins provide a shared floor; divergence above that floor is agent-specific signal.
- **Adoption gap.** The op only matters if agents actually use it. Prompt instructions and benchmark rubrics must reinforce the habit.
- **Schema breakage risk.** Adding a new `op` is non-breaking for robust readers, but downstream tools that pattern-match on a fixed op set need updates.

### Neutral

- **No data migration.** Existing stores remain valid; `evolve` is additive activity metadata.

## Open questions

None. Resolved decisions: ULID ids, single `mycelium evolve` command, implicit supersession only with non-empty target, explicit cross-kind supersession allowed, and `MYCELIUM_MEMORY.md` remaining independent from the authoritative activity log.
