# Self-Evolution Patterns

**Status:** Documentation only.
**Audience:** Operators integrating Mycelium, contributors reading the codebase, and agents that grep their own docs.
**Source:** Patterns originate in `mycelium-design.md` § 7. This document records the concrete recipes; the design doc records the rationale.

---

## The principle

> Scaffolding lives in prompts and conventions — mutable, optional, removable. It never lives in the binary, the storage, or the tool surface — immutable, mandatory, sticky.

Self-evolution is a behavior the agent performs using the same primitives as everything else. The system does not reflect, summarize, organize, or rewrite on the agent's behalf. It enables observation by recording every mutation to `_activity/`; it enables revision by making every convention an editable file. The patterns below describe how an agent uses those two affordances to improve its own working memory over time.

This doc is **not enforced and not auto-injected** into the agent's context. The pi.dev extension's system-prompt block summarizes the contract; this doc is for the operator who wants to understand the playbook in depth, and for the agent that decides to `mycelium read docs/self-evolution.md` after grepping for "convention".

---

## Pattern 1 — Convention bootstrap

**What it is.** At session start, the agent reads the convention file at the mount root and follows what it says. The file is `MYCELIUM_MEMORY.md` by default (see `docs/templates/MYCELIUM_MEMORY.md`), but the agent owns it and may rename or replace it.

**What triggers it.** Session start. The pi.dev extension's system-prompt block names the file directly; an agent that ignores the cue is silently choosing a different convention surface.

**Recipe.**

```
mycelium read MYCELIUM_MEMORY.md
```

If the file doesn't exist, the agent is on a fresh mount. The seeded template is a starter, not a contract: read it, decide whether the proposed layout fits the work, edit or replace as needed.

**Why it's a pattern, not a feature.** Nothing in the binary forces this read. A session that skips it works correctly; it just runs without any prior layout knowledge. The cost shows up later — duplicate paths, conventions reinvented, harder cross-session navigation.

---

## Pattern 2 — Convention revision

**What it is.** The agent edits its own convention file in response to a behavior it observed in the activity log. The signal that prompts the edit comes from the agent's own grep over `_activity/`, not from any external feedback.

**What triggers it.** Mid-session or between sessions, when the agent notices a pattern in its own behavior: duplicate writes to similar paths, near-duplicate filenames, conventions edited but later violated. The auto-logged operations are `write`, `edit`, `rm`, and `mv` — every successful mutation produces one JSONL entry. Reads, globs, and greps are not auto-logged; the agent can record them explicitly with `mycelium log read_signal --path X` if it wants reads to leave traces. The activity log is the substrate; the agent goes looking.

**Recipe.**

Find every write the agent made today and inspect for duplicates:

```
mycelium glob '_activity/2026/04/*/researcher-7.jsonl'
mycelium grep --pattern '"op":"write"' --path _activity --format json --limit 200
```

The `json` format returns `{matches: [{path, line, text}, ...], truncated, next_cursor}`. Each match's `text` is one JSONL entry containing `ts`, `op`, `path`, `version`, `prior_version`, `agent_id`, `session_id`. Group by `path` to spot files mutated repeatedly with similar but not identical paths.

Once a pattern is identified, the agent edits the convention file:

```
mycelium edit MYCELIUM_MEMORY.md \
    --old "Naming: lowercase-with-dashes." \
    --new "Naming: lowercase-with-dashes. Search before writing — \`mycelium grep\` for any topic before creating a new file."
```

**Concrete example.** An agent observes (by grep) that it has written `notes/glp1-pipeline.md`, `notes/glp-1-pipeline.md`, and `notes/glp1_followup.md` over three sessions, none of which were re-read before the next one was written. The agent adds a "search before writing" rule to `MYCELIUM_MEMORY.md`, consolidates the three files into one, and writes a brief `notes/_index/glp1.md` mapping the canonical file. The convention edit and the consolidation are separate operations; either alone is partial.

---

## Pattern 3 — Self-built indexes

**What it is.** The agent writes an index file when it notices that the same content is searched repeatedly. The index is just another file; it earns its place by being faster to consult than re-deriving the answer.

**What triggers it.** Repeated reads of the same file, or repeated grep/glob with the same pattern. Reads, globs, and greps are *not* auto-logged — only mutations are. So this pattern requires the agent to either notice the repetition mid-session (felt experience: "I just looked this up again"), or to have logged its own searches explicitly:

```
mycelium log search_signal --payload-json '{"pattern":"glp1","scope":"notes/"}'
```

If the agent has been logging searches, the trail is greppable:

```
mycelium grep --pattern '"op":"search_signal"' --path _activity --format json --limit 500
```

Each match's `text` is a JSONL entry with `signal_path` pointing to the payload under `logs/`. If `glp1`, `pipeline`, and `followup` show up repeatedly, the agent writes an index that resolves the common queries:

```
mycelium write notes/_index/glp1.md
# stdin:
# # GLP-1 work
#
# - Pipeline design: notes/glp1-pipeline.md
# - Follow-ups: notes/glp1-followups.md
# - Open questions: tasks/glp1-open.md
```

**Why it's optional.** A new contributor reading the store doesn't need the index — they have `grep`. The index helps the agent because it short-circuits the search loop, and it helps a human because it announces what the agent considered worth indexing. Both audiences benefit, but neither requires it.

**What not to do.** Don't create indexes preemptively. An index that isn't earned by observed search frequency is just another file to maintain. The activity log is the trigger; without it, indexes drift.

---

## Pattern 4 — Archiving and pruning

**What it is.** The agent moves stale content to `archive/` (or deletes it, when nothing valuable remains) based on observed disuse. Archived content is recoverable by `mycelium read archive/...`; deleted content is gone except for the activity-log entry recording the deletion.

**What triggers it.** A periodic pass — typically at the start of a long session, or after the agent notices `mycelium ls --recursive` returning paths it doesn't recognize. The decision is "did I write or edit this recently, and if not, does it still earn its place?" (Reads aren't logged, so the question the activity log can answer cleanly is "when was this file last *modified*," not "when was it last *consulted*." If read-tracking matters for an archival decision, the agent can log read signals explicitly.)

**Recipe.**

List paths and cross-reference the log for last-modification time:

```
mycelium ls --recursive
mycelium grep --pattern 'notes/old-protocol' --path _activity --format json --limit 100
```

If the most recent activity-log entry for the path is months old (visible in the entry's `ts` field), the agent moves it:

```
mycelium mv notes/old-protocol.md archive/2025-q4/old-protocol.md
```

Or, if the content has been superseded by something the agent now considers canonical:

```
mycelium rm notes/old-protocol.md
```

Both `mv` and `rm` log the operation, so the archival decision is itself in the activity log — a future agent can reconstruct *that* the file existed and *when* it was archived, even though the prior content is no longer at its original path.

**Conflict-of-interest note.** The agent should not delete content the operator might need. When in doubt, `mv` to `archive/` rather than `rm`. The archive path is a hint to a human reader that the agent considered the content stale but did not destroy it.

---

## What the system does not do

These boundaries exist because crossing them would re-introduce the capability coupling the design rejects:

- **No reflection step between turns.** The system runs no analysis on the agent's behalf. If the agent doesn't grep the log, no patterns are surfaced.
- **No drift detection.** "This convention was added two sessions ago and has been violated three times" is a query the agent can run; it is not a notification the system sends.
- **No automatic convention updates.** `MYCELIUM_MEMORY.md` is edited only by the agent. The binary writes only to `_activity/` and `logs/`, and only as a side effect of operations the agent initiated.
- **No enforced read-before-write.** The reservation rule (`_`-prefix) is the *only* enforced policy. Conventions are not enforced; if the agent ignores its own rules, the binary never knows.

The system makes self-evolution **possible**. The agent **does it**.

---

## Reading the activity log

Two formats serve different purposes:

- **`mycelium grep --format text`** (default) — human-friendly, one match per line, suitable for an operator inspecting a tarball.
- **`mycelium grep --format json`** (recommended for the agent) — `{matches: [{path, line, text}], truncated, next_cursor}`. Each match's `text` is a full JSONL line that the agent can re-parse to extract `op`, `path`, `version`, `prior_version`, `agent_id`, `session_id`. The `truncated` flag and the hard `--limit 1000` cap exist specifically to prevent log-reflection from blowing the agent's context window.

The log is at `_activity/YYYY/MM/DD/{agent_id}.jsonl`. Agent-supplied payloads from `mycelium log <op> --stdin` live separately at `logs/YYYY/MM/DD/{agent_id}/<HHMMSS>.<nanos>-<op>.json`, referenced from the activity entry via `signal_path`. Both are agent-readable; only `_activity/` is reserved.

---

## When this doc goes stale

Self-evolution patterns evolve. If a Frontier model's behavior in benchmarks reveals patterns this doc doesn't cover — or contradicts ones it does — revise the doc. The patterns are descriptive, not prescriptive: they document what works on the supported model tier as observed, and they get updated when observation shows otherwise.

The activity-log query recipes here are written against the Phase 1 binary. If `--format json` envelope shape, `--limit` semantics, or the `_activity/` path layout change in a later phase, the recipes here need to update with them.
