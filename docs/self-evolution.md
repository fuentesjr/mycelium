# Self-Evolution

**Status:** Reference documentation. Concrete `mycelium` recipes for self-evolution, updated for the `evolve` op added in 0.1.0. The design rationale lives in `mycelium-design.md` section 7; the `evolve` schema and supersession semantics live in [`docs/adr/0001-self-evolution-as-first-class-concept.md`](adr/0001-self-evolution-as-first-class-concept.md).

---

## The primary mechanism: `mycelium evolve`

Self-evolution events are recorded with `mycelium evolve`:

```
mycelium evolve <kind> [--target <str>] [--supersedes <id>] [--kind-definition "..."] --rationale "..."
```

This writes a structured entry to the activity log — `op: evolve` with `kind`, `target`, `rationale`, a minted ULID `id`, and an optional `supersedes` chain — and prints `{"id":"..."}` to stdout. The call is metadata-only: it records the decision but never mutates the store.

To view current rules in effect (the latest non-superseded entry per `(kind, target)` pair):

```
mycelium evolution --active
mycelium evolution --active --format json
```

To enumerate available kinds (built-ins plus any agent-introduced kinds):

```
mycelium evolution --kinds --format json
```

---

## Built-in kinds

Five kinds ship with the binary. No `--kind-definition` required on first use.

| kind | definition |
|------|------------|
| `convention` | A naming, layout, or structural pattern for organizing data in the store. |
| `index` | A derived or summary file the agent has built or regenerated over a region of the store. |
| `archive` | A region of the store the agent has marked as no-longer-active and moved out of working scope. |
| `lesson` | A distilled insight from past work, intended to inform future behavior. |
| `question` | An open unknown the agent is tracking, expected to resolve into a `lesson` (or be superseded as no-longer-relevant) later. |

Agents may introduce additional kinds by passing `--kind-definition` on first use. Built-in and agent-introduced kinds coexist on equal footing in the activity log.

---

## Pattern 1 — Convention bootstrap

At session start, inherit the current conventions:

```
mycelium evolution --active --format json
```

The pi-mycelium extension pre-surfaces this in the `before_agent_start` system prompt — a fresh session sees current rules without manually consulting `MYCELIUM_MEMORY.md`.

If you're on a fresh mount with no evolution history, `--active` returns nothing and the built-in kinds are available from `--kinds`.

`MYCELIUM_MEMORY.md` is a prose companion for editorialized summaries. When it diverges from the activity log, the activity log wins. See the ADR for the divergence policy.

---

## Pattern 2 — Adopting or revising a convention

After noticing a pattern in the activity log — duplicate filenames, violated naming rules, near-duplicate paths:

```
mycelium glob '_activity/2026/04/*/researcher-7.jsonl'
mycelium grep --pattern '"op":"write"' --path _activity --format json --limit 200
```

Record the new or revised convention:

```
mycelium evolve convention \
  --target notes/incidents/ \
  --rationale "Adopting <date>-<slug>.md filenames so incidents sort chronologically without a separate index."
# {"id":"01HXKP4Z9M8YV1W6E2RTSA9KFG"}
```

When the same `(kind, target)` pair already has an active entry, the binary fills in `supersedes` automatically — the prior convention is retired and the chain is recorded:

```
mycelium evolve convention \
  --target notes/incidents/ \
  --rationale "Switching to YYYY/MM/<slug>.md after the year wrapped — flat-date layout was getting unwieldy."
# {"id":"01HXM2C0JD7H9ASBQYNV6XGGT2","supersedes":"01HXKP4Z9M8YV1W6E2RTSA9KFG"}
```

The old entry is now superseded and will not appear in `--active` output. The full chain remains in the activity log for archaeology.

---

## Pattern 3 — Self-built indexes

Triggered by repeated reads or repeated grep/glob with the same pattern. First, confirm the pattern recurs:

```
mycelium grep --path _activity --pattern '"op":"context_signal"' --format json --limit 500
```

Build the index file:

```
mycelium write notes/_index/glp1.md   # contents map common queries to canonical paths
```

Then record the index as an `evolve` event so future sessions know it exists and who is responsible for regenerating it:

```
mycelium evolve index \
  --target notes/_index/glp1.md \
  --rationale "Hand-built TOC mapping GLP-1 queries to canonical files. Regenerate when notes/glp1-* changes significantly."
```

Don't create indexes preemptively. An index that isn't earned by observed search frequency is just another file to maintain.

---

## Pattern 4 — Archiving and pruning

Triggered by `mycelium ls --recursive` returning paths that haven't been touched recently:

```
mycelium ls --recursive
mycelium grep --pattern 'notes/old-protocol' --path _activity --format json --limit 100
```

Move the stale content:

```
mycelium mv notes/old-protocol.md archive/2025-q4/old-protocol.md
```

Then record the archival decision:

```
mycelium evolve archive \
  --target archive/2025-q4/ \
  --rationale "Pre-2026 protocol notes moved to archive/2025-q4/. Not expected to change; keep for reference."
```

`evolve archive` and `mycelium mv` are separate calls. The `evolve` event records the reasoning; the `mv` performs the move. Neither implies the other.

---

## Pattern 5 — Distilling lessons

After an incident or a completed investigation, record the insight:

```
mycelium evolve lesson \
  --target notes/incidents/2026-04-30-latency-spike.md \
  --rationale "Queries mentioning 'latency' correlate with deploy events 80% of the time. Check deploy calendar before investigating latency spikes."
```

---

## Pattern 6 — Tracking open questions

For unknowns that need resolution before you can close a thread:

```
mycelium evolve question \
  --target hypotheses/glp1-cardio.md \
  --rationale "Does the GLP-1 cardio-protection effect generalize to non-diabetic populations? Need 3+ independent studies before concluding."
```

When the question resolves, supersede it with a `lesson`:

```
mycelium evolve lesson \
  --target hypotheses/glp1-cardio.md \
  --rationale "GLP-1 cardio protection confirmed for non-diabetic populations across 4 independent studies (see notes/glp1-cardio-evidence.md)."
# supersedes the question automatically via (kind, target) matching? No — different kinds don't auto-supersede.
```

For cross-kind retirement, use `--supersedes <id>` explicitly:

```
mycelium evolve lesson \
  --target hypotheses/glp1-cardio.md \
  --supersedes <question-id> \
  --rationale "GLP-1 cardio protection confirmed for non-diabetic populations across 4 independent studies."
```

---

## Agent-introduced kinds

When the built-in kinds don't fit your domain, introduce a new one:

```
mycelium evolve experiment \
  --target hypotheses/glp1-cardio.md \
  --kind-definition "An in-progress hypothesis I'm actively testing against new evidence. Distinct from 'lesson' (closed-out insight) and 'question' (passive unknown)." \
  --rationale "Tracking the GLP-1 cardio-protection question as an open thread until I have N=3 independent supporting papers."
```

`--kind-definition` is required on the first use of any non-builtin kind. Subsequent uses may omit it. To redefine a kind, pass a new `--kind-definition` — the binary writes a `_kind_definition` supersession chain automatically.

---

## When this doc goes stale

These recipes are written against the 0.1.0 binary. If `--format json` envelope shape, flag names, or `_activity/` path layout change in a future release, the recipes update with them.
