# Self-Evolution

**Status:** Reference documentation. Concrete `mycelium` recipes for self-evolution. The design rationale lives in `mycelium-design.md` section 7; the activity-log schema and rationale live in [`docs/adr/0001-self-evolution-as-first-class-concept.md`](adr/0001-self-evolution-as-first-class-concept.md).

---

## The mechanism: `mycelium evolve`

One command records and queries self-evolution metadata.

Record an evolution event:

```sh
mycelium evolve <kind> [--target <str>] [--supersedes <id>] \
  [--kind-definition "..."] --rationale "..."
```

This writes a structured `op: "evolve"` entry to the authoritative activity log with `kind`, optional `target`, `rationale`, a minted ULID `id`, and optional `supersedes`. The call is metadata-only: it records the decision but never mutates agent-authored files.

Query evolution state:

```sh
# Current rules/lessons/questions in effect
mycelium evolve --active
mycelium evolve --active --format json

# Full user-facing timeline
mycelium evolve --list
mycelium evolve --list --kind convention --since 2026-05-01 --format json

# Available vocabulary: built-ins plus agent-introduced kinds
mycelium evolve --kinds --format json
```

---

## Built-in kinds

Five kinds ship by default. No `--kind-definition` required on first use.

| kind         | definition                                                                                                               |
| ------------ | ------------------------------------------------------------------------------------------------------------------------ |
| `convention` | A naming, layout, structural, or behavioral pattern for organizing or operating on the store.                            |
| `index`      | A derived or summary file the agent has built or regenerated over a region of the store.                                 |
| `archive`    | A region of the store the agent has marked as no-longer-active and moved out of working scope.                           |
| `lesson`     | A distilled insight from past work, intended to inform future behavior.                                                  |
| `question`   | An open unknown the agent is tracking, expected to resolve into a `lesson` or be superseded as no-longer-relevant later. |

Agents may introduce additional kinds by passing `--kind-definition` on first use. Built-in and agent-introduced kinds coexist on equal footing in the activity log.

---

## Supersession rules

Targeted evolution forms automatic chains:

- If `target` is non-empty, a new event with the same `(kind, target)` as an active prior event supersedes that prior event automatically.
- The new event prints `{"id":"...","supersedes":"..."}` so the agent sees the chain it extended.

Targetless evolution is additive:

- If `target` is omitted or empty, no implicit supersession occurs.
- Use `--supersedes <id>` explicitly when a targetless event replaces a prior one.

Explicit supersession is allowed when the agent is intentionally retiring another event, including across kinds. The common case is resolving a `question` into a `lesson`.

---

## Pattern 1 — Convention bootstrap

At session start, inherit the current conventions and lessons:

```sh
mycelium evolve --active --format json
```

The pi-mycelium extension may pre-surface this active-evolution view in the `before_agent_start` system prompt. This is metadata, not retrieved memory content: a fresh session sees current rules without the harness prefetching arbitrary notes.

If you're on a fresh mount with no evolution history, `--active` returns nothing and the built-in kinds are available from `--kinds`.

`MYCELIUM_MEMORY.md` is a prose companion for editorialized summaries. When it diverges from the activity log, the activity log wins. See the ADR for the divergence policy.

---

## Pattern 2 — Adopting or revising a convention

After noticing a pattern in the activity log — duplicate filenames, violated naming rules, near-duplicate paths:

```sh
mycelium ls '_activity/2026/04/*/researcher-7.jsonl' --recursive
mycelium grep --pattern '"op":"write"' --path _activity --format json --limit 200
```

Record the new convention:

```sh
mycelium evolve convention \
  --target notes/incidents/ \
  --rationale "Adopting <date>-<slug>.md filenames so incidents sort chronologically without a separate index."
# {"id":"01HXKP4Z9M8YV1W6E2RTSA9KFG"}
```

Revise it later by reusing the same non-empty target:

```sh
mycelium evolve convention \
  --target notes/incidents/ \
  --rationale "Switching to YYYY/MM/<slug>.md after the year wrapped — flat-date layout was getting unwieldy."
# {"id":"01HXM2C0JD7H9ASBQYNV6XGGT2","supersedes":"01HXKP4Z9M8YV1W6E2RTSA9KFG"}
```

The old entry is now superseded and will not appear in `--active` output. The full chain remains in the activity log for archaeology.

---

## Pattern 3 — Self-built indexes

Triggered by repeated searches, repeated navigation to the same cluster of files, or recurring user questions. First, confirm the pattern is real from available evidence:

```sh
mycelium grep --path _activity --pattern 'glp1' --format json --limit 500
```

Build the index file:

```sh
mycelium write notes/_index/glp1.md   # contents map common queries to canonical paths
```

Then record the index as an evolution event so future sessions know it exists and when to regenerate it:

```sh
mycelium evolve index \
  --target notes/_index/glp1.md \
  --rationale "Hand-built TOC mapping GLP-1 queries to canonical files. Regenerate when notes/glp1-* changes significantly."
```

Don't create indexes preemptively. An index that isn't earned by observed search/navigation friction is just another file to maintain.

---

## Pattern 4 — Archiving and pruning

Triggered by `mycelium ls --recursive` returning paths that haven't been touched recently or that the activity log shows are no longer active:

```sh
mycelium ls --recursive
mycelium grep --pattern 'notes/old-protocol' --path _activity --format json --limit 100
```

Move the stale content:

```sh
mycelium mv notes/old-protocol.md archive/2025-q4/old-protocol.md
```

Then record the archival decision:

```sh
mycelium evolve archive \
  --target archive/2025-q4/ \
  --rationale "Pre-2026 protocol notes moved to archive/2025-q4/. Not expected to change; keep for reference."
```

`evolve archive` and `mycelium mv` are separate calls. The `evolve` event records the reasoning; the `mv` performs the move. Neither implies the other.

---

## Pattern 5 — Distilling lessons

After an incident or completed investigation, record the insight. Use a target when the lesson scopes to a file/topic and should replace future revisions automatically:

```sh
mycelium evolve lesson \
  --target notes/incidents/2026-04-30-latency-spike.md \
  --rationale "Queries mentioning 'latency' correlate with deploy events 80% of the time. Check deploy calendar before investigating latency spikes."
```

For broad standalone lessons where no natural target exists, omit `--target`. Targetless lessons are additive and will not accidentally supersede each other:

```sh
mycelium evolve lesson \
  --rationale "For library-internals questions, prefer source permalinks over secondary summaries."
```

If a targetless lesson later needs replacement, pass `--supersedes <id>` explicitly.

---

## Pattern 6 — Tracking open questions

For unknowns that need resolution before you can close a thread:

```sh
mycelium evolve question \
  --target hypotheses/glp1-cardio.md \
  --rationale "Does the GLP-1 cardio-protection effect generalize to non-diabetic populations? Need 3+ independent studies before concluding."
```

When the question resolves, supersede it explicitly with a lesson because the kind changes:

```sh
mycelium evolve lesson \
  --target hypotheses/glp1-cardio.md \
  --supersedes <question-id> \
  --rationale "GLP-1 cardio protection confirmed for non-diabetic populations across 4 independent studies."
```

Implicit supersession only matches the same `(kind, target)` pair. Cross-kind retirement is intentional and explicit.

---

## Agent-introduced kinds

When the built-in kinds don't fit your domain, introduce a new one:

```sh
mycelium evolve experiment \
  --target hypotheses/glp1-cardio.md \
  --kind-definition "An in-progress hypothesis I'm actively testing against new evidence. Distinct from 'lesson' (closed-out insight) and 'question' (passive unknown)." \
  --rationale "Tracking the GLP-1 cardio-protection question as an open thread until I have N=3 independent supporting papers."
```

`--kind-definition` is required on the first use of any non-builtin kind. Subsequent uses may omit it. To redefine a kind, pass a new `--kind-definition`; mycelium records the taxonomy's evolution in the activity log.

---

## When this doc goes stale

These recipes describe the current design. If the `--format json` envelope shape, flag names, or `_activity/` path layout change in a future release, update the recipes with them.
