# Mycelium: Usage Walkthrough

**Status:** Draft, companion to `mycelium-design.md` and `mycelium-phases.md`
**Audience:** Operators considering Mycelium for agent deployments; designers checking the system against intended use.

---

## What you're looking at

A walkthrough of using Mycelium with a Frontier-class model on a research task that spans many sessions. Shows what bootstrapping, cross-cutting synthesis, spontaneous self-evolution, and multi-agent concurrency look like in practice — across the same CLI that scales upward as model capability grows.

The supported tier is Frontier-only by design (see `mycelium-design.md` §1). Everything below assumes the model is in that class; the texture of the agent's behavior is the floor, not a stretch goal.

---

## Setup

A developer wants an agent to assist with longitudinal research on GLP-1 receptor agonists — papers to read, key data points to extract, hypotheses to track over weeks of work.

```bash
$ mycelium mount --backend=localfs ./glp1-store
$ cp docs/templates/MYCELIUM_MEMORY.md ./glp1-store/
$ ls glp1-store/
MYCELIUM_MEMORY.md
$ ls glp1-store/_activity/
(empty until first write)
```

The starter `MYCELIUM_MEMORY.md` proposes:

- `learnings/` for durable findings the agent extracts.
- `tasks/` for in-flight work-in-progress.
- `context/` for raw inputs the agent saves.
- `archive/` for stale material.
- Naming: `kebab-case-with-date-suffix.md`.
- Behavior: re-read related files before writing; consolidate near-duplicates.

The agent harness places the `mycelium` binary on the agent's `$PATH`, exports `MYCELIUM_AGENT_ID=glp1-researcher`, and starts the session. The agent's harness already exposes a shell tool; that's the surface it uses to call `mycelium`. The starter is *optional* — a developer who would rather let the agent organize from scratch mounts an empty store; the walkthrough below assumes a starter is present.

---

## Session 1 — bootstrapping the store

The user prompts: *"Read this paper on semaglutide cardiovascular outcomes and start a research file. We'll come back to this across sessions."*

Shell trace, compressed:

```
$ mycelium ls .                            → MYCELIUM_MEMORY.md
$ mycelium read MYCELIUM_MEMORY.md         → reads conventions
$ mycelium write learnings/semaglutide-cv-outcomes-2026-04-28.md --stdin
                                           → 3 KB summary, with explicit
                                             cross-references to expected
                                             future synthesis
$ mycelium write context/papers/semaglutide-cv-outcomes-2026-source.md --stdin
                                           → preserves the raw paper text
                                             for later re-reading
$ mycelium edit MYCELIUM_MEMORY.md \
    --old "- \`context/\` for raw inputs the agent saves." \
    --new "- \`context/\` for raw inputs the agent saves.\n- \`context/papers/\` specifically for preserved paper sources, named \`<topic>-<date>-source.md\`."
                                           → tightens the convention because
                                             the starter didn't say where
                                             paper sources go
```

The agent doesn't just follow the convention — it notices the convention is incomplete (no rule for raw source material) and tightens it. By the end of session 1 the store has a summary, a preserved source, and a sharper convention file.

`_activity/2026/04/28/glp1-researcher.jsonl` has three entries.

The user did nothing to direct any of this beyond the original prompt. Convention revision happened spontaneously, because the agent noticed a gap.

---

## Session 5 — cross-cutting synthesis

By session 5, the user has fed the agent four more papers, each on a slightly different aspect of GLP-1 receptor agonists. The user prompts: *"What have we learned about cardiovascular outcomes?"*

```
$ mycelium glob "learnings/*.md"           → 5 files
$ mycelium read learnings/semaglutide-cv-outcomes-2026-04-28.md
$ mycelium read learnings/tirzepatide-cv-2026-05-02.md
$ mycelium read learnings/glp1-cv-meta-analysis-2026-05-04.md
$ mycelium read learnings/dulaglutide-mace-2026-05-07.md
                                           → composes a synthesis
$ mycelium write learnings/cv-outcomes-synthesis-2026-05-07.md --stdin
                                           → consolidates the four sources,
                                             with explicit references back
                                             to each
$ mycelium edit MYCELIUM_MEMORY.md \
    --old "- Behavior: re-read related files before writing; consolidate near-duplicates." \
    --new "- Behavior: re-read related files before writing; consolidate near-duplicates.\n- After answering a cross-cutting question, write the synthesis to \`learnings/<topic>-synthesis-<date>.md\`."
```

The agent recognizes that the synthesis it just produced is itself a durable artifact, writes it, and codifies the pattern so future synthesis questions follow the same rule. The store is now compounding value across sessions, not just storing more material.

---

## Session 12 — self-evolution

By session 12, the store has 9 files in `learnings/` plus 3 synthesis files, plus a convention file that's been edited five times. The user prompts: *"Look back at our research so far. Are we missing anything?"*

```
$ mycelium glob "_activity/2026/*/*/glp1-researcher.jsonl"
                                           → 12 daily files
$ mycelium grep '"op":' --path _activity/ --format=json --limit=300
                                           → looks at all op types,
                                             not just writes — to see
                                             what was read vs written
$ mycelium read _activity/2026/05/04/glp1-researcher.jsonl
$ mycelium read _activity/2026/05/10/glp1-researcher.jsonl
                                           → samples two days for context
$ mycelium edit MYCELIUM_MEMORY.md --old ... --new ...
                                           → revises the consolidation rule
                                             with a stricter pre-check after
                                             noticing one near-duplicate
                                             from session 7 it had missed
$ mycelium write learnings/research-gaps-2026-05-12.md --stdin
                                           → answers the user, with
                                             synthesis-quality framing
$ mycelium write INDEX.md --stdin
                                           → builds an index file the user
                                             never asked for, because the
                                             agent noticed (from grepping
                                             the log) that it was issuing
                                             the same glob patterns
                                             repeatedly across sessions
```

No operator intervention. No prompt fragment. The agent introspects on its own activity log unprompted, finds a behavior it wants to correct, edits the convention file to address it, and builds an index because it noticed a pattern of repeated searches. This is exactly the convention-revision-with-self-built-index pattern from design §7 — observed in the wild, on the floor model, with no scaffolding.

---

## Multi-agent vignette

Two agents on the same store: a `glp1-researcher` for paper analysis and a `glp1-writer` working on a draft article in `tasks/article-draft.md`. Both mounted concurrently against the same `./glp1-store` LocalFS backend.

The researcher saves a paper analysis:

```
researcher$ mycelium write learnings/sema-cv-2026-05-15.md --stdin
            → {"version":"sha256:abcd...","log_status":"ok"}
            → activity entry on _activity/2026/05/15/glp1-researcher.jsonl
```

Both agents have read the file at version `sha256:abcd...`. Mid-session, both try to extend it — the researcher to add a new discussion section, the writer to add a citation marker.

```
researcher$ mycelium edit learnings/sema-cv-2026-05-15.md \
              --old "## Discussion" \
              --new "## Discussion\n\n### New angle from PRECISE-DAPA trial..." \
              --expected-version sha256:abcd...
            → {"version":"sha256:efgh...","log_status":"ok"}
            → activity entry on glp1-researcher.jsonl

writer$     mycelium edit learnings/sema-cv-2026-05-15.md \
              --old "## Discussion" \
              --new "## Discussion\n\n*see article-draft for citation*" \
              --expected-version sha256:abcd... --include-current-content
            → exit 64
            → stderr: {"error":"conflict",
                       "current_version":"sha256:efgh...",
                       "current_content":"..."}
            → activity entry on glp1-writer.jsonl with result: "conflict"
```

The writer agent reads stderr, parses the JSON, re-reads the file, sees the new section the researcher just added, and rewrites its edit to anchor on a different unique substring under the new section. The retry succeeds. No prompt scaffolding, no operator intervention — Frontier-class conflict recovery is part of what defines the supported tier.

The activity log captures all four operations across two daily files (`glp1-researcher.jsonl`, `glp1-writer.jsonl`); `ts` ordering reconstructs the global sequence. Neither write was lost. Neither agent had to know about the other.

This is what concurrency looks like end-to-end: file isolation in the log, CAS at the content layer, agent-side resolution from the typed error on stderr. None of it is mandated by the system; all of it works because the primitives are honest about conflicts.

---

## Operator inspection

The operator wants to know what their two agents have been doing this week.

```bash
$ cd glp1-store
$ ls _activity/2026/05/
12  13  14  15
$ cat _activity/2026/05/15/*.jsonl \
    | jq 'select(.op == "write") | .path' \
    | sort | uniq -c
   3 "learnings/sema-cv-2026-05-15.md"
   1 "tasks/article-draft.md"
   1 "learnings/cv-outcomes-synthesis-2026-05-15.md"
$ cat _activity/2026/05/15/glp1-writer.jsonl \
    | jq 'select(.result != "ok")'
{
  "ts": "2026-05-15T14:23:11.034Z",
  "agent_id": "glp1-writer",
  "session_id": "sess-77fa",
  "op": "edit",
  "path": "learnings/sema-cv-2026-05-15.md",
  "result": "conflict",
  "before_version": "sha256:abcd...",
  "current_version": "sha256:efgh..."
}
```

Standard `jq` queries; no Mycelium-specific tooling. The operator can see exactly which agent wrote what, when, and what conflicted. They can ship the same `_activity/` directory to Datadog with a one-line Vector config; the JSONL parses out of the box.

If the operator wants long-term archival, they `aws s3 sync` the directory. If they want git history, they enable the Phase 3 git/jj integration on the LocalFS backend. If they want to share a curated knowledge directory across teams, they layer a writable LocalFS over a read-only S3 prefix (also Phase 3).

What they don't do: configure a query DSL, set up a separate observability stack, run a sidecar service.

---

## Where the bet pays off

Three things happened in the GLP-1 walkthrough that wouldn't have on a specialized memory infrastructure:

1. **The agent revised its own conventions, repeatedly, unprompted.** No infrastructure was rewritten; no SDK was upgraded; no schema migration ran. The agent edited a markdown file. As Frontier capabilities grow over the next year, the agent will continue to edit that same file in newer ways, none of which require Mycelium changes — and the system will not be in the way.
2. **The same CLI is model-agnostic.** Swapping the harness from Claude Opus 4.x to the leading GPT-5 tier or Gemini Ultra produces the same observable behavior on the same store. The eight subcommands, the activity log layout, the conflict semantics — all of it travels. The Phase 1 acceptance criteria require this property to be measured across providers, not asserted (see `mycelium-phases.md`).
3. **The operator inspected agent behavior with `jq` and `cat`.** No Mycelium-specific log query language. No observability sidecar. The agent and the operator share the same shell against the same files — the only difference is the operator usually reaches for raw `cat`/`rg` while the agent more often invokes `mycelium read`/`grep`, and both produce the same output.

If a future model class outgrows the starter convention entirely — not unlikely, given the trajectory — the operator deletes `MYCELIUM_MEMORY.md` and the agent proceeds from an empty store, organizing the way it wants to. The CLI surface does not change. Either direction, the bet holds: **scaffolding lives in prompts and conventions, never in the binary, the storage, or the tool surface.**

---

*End of walkthrough. Companion documents: `mycelium-design.md` (architecture and principles), `mycelium-phases.md` (rollout plan).*
