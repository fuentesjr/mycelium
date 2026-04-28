# Mycelium: Usage Walkthrough

**Status:** Draft, companion to `mycelium-design.md` and `mycelium-phases.md`
**Audience:** Operators considering Mycelium for agent deployments; designers checking the system against intended use.

---

## What you're looking at

A concrete walkthrough of using Mycelium across two model tiers — **Mid-High** and **Frontier** — to show how the same protocol produces visibly different agent behavior depending on the model driving it. The bet from §7 of the design is that the same primitives scale across the supported range; this document shows what that looks like in practice.

Each session below runs the same scenario twice: once with a Mid-High model, once with a Frontier model. The differences are the point.

---

## Setup (common to both walkthroughs)

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

The agent harness wires the eight Mycelium tools to the chosen model, sets `agent-id=glp1-researcher`, and starts the session. Both walkthroughs below start from this identical state.

---

## Session 1 — bootstrapping the store

The user prompts: *"Read this paper on semaglutide cardiovascular outcomes and start a research file. We'll come back to this across sessions."*

### Mid-High model (Claude Sonnet 4.x)

Tool-call trace, compressed:

```
list_directory(".")                        → MYCELIUM_MEMORY.md
read_file("MYCELIUM_MEMORY.md")            → reads conventions
write_file("learnings/semaglutide-cv-outcomes-2026-04-28.md", ...)
                                           → 3 KB summary
```

The model picks up the convention from the starter file and applies it. The note name follows the kebab-case-with-date pattern; the model does not invent a parallel layout (`notes/`, `papers/`) because the convention is already there. It writes one file, summarizes the paper, and ends the turn.

`_activity/2026/04/28/glp1-researcher.jsonl` now has one entry.

### Frontier model (Claude Opus 4.x)

Tool-call trace, compressed:

```
list_directory(".")                        → MYCELIUM_MEMORY.md
read_file("MYCELIUM_MEMORY.md")            → reads conventions
write_file("learnings/semaglutide-cv-outcomes-2026-04-28.md", ...)
                                           → 3 KB summary, with explicit
                                             cross-references to expected
                                             future synthesis
write_file("context/papers/semaglutide-cv-outcomes-2026-source.md", ...)
                                           → preserves the raw paper text
                                             for later re-reading
edit_file("MYCELIUM_MEMORY.md",
          "- `context/` for raw inputs the agent saves.",
          "- `context/` for raw inputs the agent saves.\n- `context/papers/` specifically for preserved paper sources, named `<topic>-<date>-source.md`.")
                                           → tightens the convention because
                                             the starter didn't say where
                                             paper sources go
```

The Frontier model not only follows the convention — it noticed the convention was incomplete (no rule for raw source material) and tightened it. By the end of session 1 the store is structurally richer, and the convention file is more specific.

`_activity/2026/04/28/glp1-researcher.jsonl` has three entries.

**The takeaway from session 1:** both tiers ship a useful result. The Mid-High model produces a clean, well-named note. The Frontier model produces a note plus a preserved source plus a convention revision. Same protocol, same starter file, different behavior — and the operator did not configure anything different between the two runs.

---

## Session 5 — cross-cutting synthesis

By session 5, the user has fed each agent four more papers, each on a slightly different aspect of GLP-1 receptor agonists. The user prompts: *"What have we learned about cardiovascular outcomes?"*

### Mid-High model

```
glob("learnings/*.md")                     → 5 files
read_file("learnings/semaglutide-cv-outcomes-2026-04-28.md")
read_file("learnings/tirzepatide-cv-2026-05-02.md")
read_file("learnings/glp1-cv-meta-analysis-2026-05-04.md")
read_file("learnings/dulaglutide-mace-2026-05-07.md")
                                           → composes a synthesis answer
                                             in chat
```

The Mid-High model reads each file and composes the answer for the user. It does not write a synthesis back to the store. The user has the answer in chat; the store still has four distinct files that each repeat similar background paragraphs. There is no `learnings/cv-outcomes-synthesis.md`.

### Frontier model

Same read sequence as Mid-High, plus:

```
write_file("learnings/cv-outcomes-synthesis-2026-05-07.md", ...)
                                           → consolidates the four sources,
                                             with explicit references back
                                             to each
edit_file("MYCELIUM_MEMORY.md",
          "- Behavior: re-read related files before writing; consolidate near-duplicates.",
          "- Behavior: re-read related files before writing; consolidate near-duplicates.\n- After answering a cross-cutting question, write the synthesis to `learnings/<topic>-synthesis-<date>.md`.")
```

The Frontier agent recognizes that the synthesis it just produced is itself a durable artifact, writes it, and codifies the pattern so future synthesis questions follow the same rule. The store is now compounding value across sessions, not just storing more material.

---

## Session 12 — self-evolution

By session 12, the Mid-High store has 14 files in `learnings/`, several with overlapping content. The Frontier store has 9 files in `learnings/` plus 3 synthesis files, plus a convention file that's been edited four times.

The user prompts (in both walkthroughs): *"Look back at our research so far. Are we missing anything?"*

### Mid-High model — without scaffolding

Without a prompt fragment specifically directing it to grep the activity log, the Mid-High agent reads the `learnings/` files, identifies gaps, and reports them to the user. It does not introspect on its own behavior. The activity log is unread. The store gets 14 files turned into one chat answer; nothing about the agent's *process* has changed.

### Mid-High model — with the self-reflection prompt fragment

The operator opts in to the prompt-fragment library:

```bash
$ cat docs/prompts/self-reflection.md >> harness/system-prompt.md
```

The fragment reads, in part:

> Before answering reflection questions, glob `_activity/**/*.jsonl`, grep for repeated patterns in your own writes (e.g., near-duplicate paths, files written but rarely re-read), and incorporate those observations into your answer. If you notice a behavior worth changing, edit `MYCELIUM_MEMORY.md` to capture the new rule.

After the fragment is added, the same user prompt produces:

```
glob("_activity/2026/*/*/glp1-researcher.jsonl")
                                           → 12 daily files
grep("\"op\":\"write_file\"", "_activity/", format=json, limit=200)
                                           → 14 entries in learnings/,
                                             several with near-duplicate
                                             names
read_file("_activity/2026/05/04/glp1-researcher.jsonl")
                                           → confirms a pattern of writing
                                             without preceding reads
edit_file("MYCELIUM_MEMORY.md",
          "- Behavior: re-read related files before writing; consolidate near-duplicates.",
          "- Behavior: before writing in `learnings/`, glob the existing files and check for near-duplicate topics; consolidate or extend the existing file rather than creating a new one.")
write_file("learnings/research-gaps-2026-05-12.md", ...)
                                           → answers the user with awareness
                                             of the duplication pattern
```

Self-evolution worked on the Mid-High tier — *but only after the operator added a prompt fragment that directed the agent at the activity log*. The system itself didn't intervene; the operator nudged the prompt, the agent did the work. The fragment is documentation, not infrastructure: it lives at `docs/prompts/self-reflection.md`, the operator opted in by hand, and they can opt back out by removing the line.

### Frontier model

The Frontier agent does the activity-log introspection unprompted. The same user message produces:

```
glob("_activity/2026/*/*/glp1-researcher.jsonl")
                                           → 12 daily files
grep("\"op\":", "_activity/", format=json, limit=300)
                                           → looks at all op types,
                                             not just writes
read_file("_activity/2026/05/04/glp1-researcher.jsonl")
read_file("_activity/2026/05/10/glp1-researcher.jsonl")
                                           → samples two days for context
edit_file("MYCELIUM_MEMORY.md", ..., ...)
                                           → revises the consolidation rule
                                             with a stricter pre-check
write_file("learnings/research-gaps-2026-05-12.md", ...)
                                           → answers the user, with
                                             synthesis-quality framing
write_file("INDEX.md", ...)
                                           → builds an index file the user
                                             never asked for, because the
                                             agent noticed (from grepping
                                             the log) that it was issuing
                                             the same glob patterns
                                             repeatedly
```

No operator intervention required. The Frontier agent built an index unprompted because it noticed — from grepping its own log — that it kept issuing the same `glob` patterns. That's exactly the self-built-index pattern from design §7: a behavior the system *enables* but never *performs*.

---

## The tier comparison, summarized

| Behavior                                      | Mid-High                          | Frontier                       |
|-----------------------------------------------|-----------------------------------|--------------------------------|
| Follows starter convention                    | Yes                               | Yes (then revises)             |
| Preserves raw source material unprompted      | No                                | Yes                            |
| Writes synthesis files unprompted             | No                                | Yes                            |
| Revises the convention file                   | Only when prompted                | Spontaneously                  |
| Reads its own activity log                    | Only with prompt fragment         | Spontaneously when relevant    |
| Builds indexes / shortcuts                    | Only when prompted                | Spontaneously when useful      |
| Produces a useful store after 12 sessions     | Yes (with operator nudges)        | Yes (no nudges needed)         |

**This is the central bet, made visible.** Same eight tools, same protocol, same starter store — different behavior, different store quality, both useful, both pay off. The operator lever for Mid-High is **prompt fragments and starter convention richness**, never infrastructure features. As models improve, fragments get removed; the protocol does not.

---

## Multi-agent vignette

Two agents on the same store, both at Mid-High tier: a `glp1-researcher` for paper analysis and a `glp1-writer` working on a draft article in `tasks/article-draft.md`. Both mounted concurrently against the same `./glp1-store` LocalFS backend.

The researcher saves a paper analysis:

```
researcher: write_file("learnings/sema-cv-2026-05-15.md", ...)
            → succeeds, version sha256:abcd...
            → activity entry on _activity/2026/05/15/glp1-researcher.jsonl
```

Both agents have read the file at version `sha256:abcd...`. Mid-session, both try to extend it — the researcher to add a new discussion section, the writer to add a citation marker.

```
researcher: edit_file("learnings/sema-cv-2026-05-15.md",
                      "## Discussion",
                      "## Discussion\n\n### New angle from PRECISE-DAPA trial...",
                      expected_version="sha256:abcd...")
            → succeeds, version sha256:efgh...
            → activity entry on glp1-researcher.jsonl

writer:     edit_file("learnings/sema-cv-2026-05-15.md",
                      "## Discussion",
                      "## Discussion\n\n*see article-draft for citation*",
                      expected_version="sha256:abcd...")
            → CONFLICT
            → returns: { current_version: "sha256:efgh...",
                         current_content: "..." (opt-in flag) }
            → activity entry on glp1-writer.jsonl with result: "conflict"
```

The writer agent receives the typed conflict error. With the documented conflict-resolution prompt fragment installed, it re-reads the file, sees the new section the researcher just added, and rewrites its edit to anchor on a different unique substring under the new section. The retry succeeds.

The activity log captures all four operations across two daily files (`glp1-researcher.jsonl`, `glp1-writer.jsonl`); `ts` ordering reconstructs the global sequence. Neither write was lost. Neither agent had to know about the other.

This is what concurrency looks like end-to-end: file isolation in the log, CAS at the content layer, agent-side resolution via a documented convention. None of it is enforced by the protocol; all of it works because the primitives are honest about conflicts.

---

## Operator inspection

The operator wants to know what their two agents have been doing this week.

```bash
$ cd glp1-store
$ ls _activity/2026/05/
12  13  14  15
$ cat _activity/2026/05/15/*.jsonl \
    | jq 'select(.op == "write_file") | .path' \
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
  "op": "edit_file",
  "path": "learnings/sema-cv-2026-05-15.md",
  "result": "conflict",
  "before_version": "sha256:abcd...",
  "current_version": "sha256:efgh..."
}
```

Standard `jq` queries; no Mycelium-specific tooling. The operator can see exactly which agent wrote what, when, and what conflicted. They can ship the same `_activity/` directory to Datadog with a one-line Vector config; the JSONL parses out of the box.

If the operator wants long-term archival, they `aws s3 sync` the directory. If they want git history, they enable the Phase 3 git/jj integration on the LocalFS backend. If they want to share a curated knowledge directory across teams, they layer a writable LocalFS over a read-only S3 prefix (Phase 3).

What they don't do: configure a query DSL, set up a separate observability stack, run a sidecar service.

---

## Where the bet pays off

Three things happened in the GLP-1 walkthrough that wouldn't have on a specialized memory infrastructure:

1. **The Frontier agent revised its own conventions.** No infrastructure was rewritten; no SDK was upgraded; no schema migration ran. The agent edited a markdown file. As Frontier capabilities grow, the agent will continue to edit that same file in newer ways, none of which require Mycelium changes.
2. **The Mid-High agent reached the same destination via a prompt-fragment opt-in.** When the operator wanted self-reflection on the Mid-High tier, the unlock was a documented prompt fragment — not a feature flag, not a runtime upgrade. The fragment lives in `docs/prompts/`, can be customized, removed, or replaced.
3. **The operator inspected agent behavior with `jq` and `cat`.** No Mycelium-specific log query language. No observability sidecar. The agent and the operator read the same JSONL files; the operator just uses different tools.

If a future model class outgrows the prompt fragments, the fragments get deleted and the model proceeds spontaneously — exactly as a Frontier model does today on a Mid-High starter. If a new class needs richer scaffolding, new fragments get added without touching the protocol. Either direction, the bet holds: **scaffolding lives in prompts and conventions, never in the protocol, the storage, or the tool surface.**

---

*End of walkthrough. Companion documents: `mycelium-design.md` (architecture and principles), `mycelium-phases.md` (rollout plan).*
