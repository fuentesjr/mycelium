# Mycelium: A Model-Agnostic Agent Memory System

**Status:** Design draft
**Project:** Mycelium

A persistent memory substrate for autonomous agents, on the bet that **a real filesystem driven by general file tools outperforms specialized memory infrastructure as models improve** — and that this should be cashed in without a dependency on any single model provider.

---

## 1. Goals and Non-Goals

### Goals

- Give the agent durable memory by mounting a directory and exposing standard file operations as tools.
- Let the agent own the schema. The agent decides what to save, how to name it, and how to organize it.
- Keep memory human-interpretable end to end — exportable as a directory, diffable as text, shareable as a tarball.
- Support multiple agents concurrently mounting the same store with predictable conflict semantics.
- Run on any agent harness with a shell tool — the agent invokes a `mycelium` binary alongside its other shell calls.
- Run against any storage backend — local filesystem or S3-compatible object store — behind one interface.
- Stay useful on Frontier models without compensating logic that becomes a ceiling on the next generation.
- Equip the agent to **observe and revise its own memory practices over time** — using the same general tools.

### Non-Goals

- A memory API. There is no `remember(fact)` or `recall(query)`. Memory is a directory; the operations are the file tools.
- Automatic extraction, summarization, or compression of agent output.
- Embedding-based retrieval as the primary access path.
- Tiered memory (working / episodic / archival) maintained by infrastructure rather than by the agent.
- Schema enforcement on agent-authored content.
- System-driven reflection or self-evolution. The system *enables* it; the agent *does* it.
- Compensating for limited model judgment with infrastructure features. If the model is making a mess, the answer is a stronger model or a better prompt.

### Target model tier

Mycelium targets **Frontier-class production models** — top-of-class capability defined by the behaviors the rest of this design assumes: self-organizing a filesystem store from empty, reflecting on its own activity log, revising its own conventions without operator scaffolding (Claude Opus 4.7, GPT-5.5, and equivalent successors). Models below Frontier are out of scope; baking in compensating logic for tiers below the floor would mask, with infrastructure heuristics, exactly the self-evolution behavior this design exists to capture.

---

## 2. Design Principles

### As simple as possible, but not simpler

> "Everything should be made as simple as possible, but not simpler." — Einstein (apocryphal)

This is the project's first principle and the one every other principle answers to. Every tool, parameter, backend method, and on-disk structure earns its complexity against "could this be expressed in terms of something already here?" — and is removed when the answer turns out to be yes. When in doubt, fewer. The activity log was once a special tool; it's now ordinary files at a reserved path the agent reads like any other.

The "but not simpler" half is load-bearing. A system that fakes simplicity by hiding necessary complexity behind opaque abstractions, or that drops a guarantee the bet depends on, has failed this principle as surely as one that piles on unnecessary layers. Atomicity, conflict visibility, and the reserved-prefix protection cannot be removed without breaking the bet — they're as simple as the bet allows, and no simpler. Every other decision in this document is the application of this principle to a specific question.

### General tools scale with intelligence; specialized infrastructure caps it

A specialized memory API encodes assumptions: what gets saved, how it's indexed, what counts as "relevant," when to summarize. As models improve, those heuristics become drag — the system forces the agent into compression and ranking policies the model could now beat unaided.

General tools (read, write, list, edit, glob, grep) have no such ceiling. The agent invokes them through its existing shell — `mycelium read` sits in the same Bash tool as `git log` and `rg` — and `mycelium` is the smallest adapter that earns its keep: atomic conditional writes, an automatic activity log, no policy about what to save, how to name, or what's relevant. A Frontier model uses them with judgment indistinguishable from a thoughtful engineer keeping a working notebook, and the same surface gets *more* useful — not less — as the next generation arrives. This is the central bet, and every other decision is downstream of it.

### Files are the unit. Directories are the structure. The agent owns both.

The filesystem is the agent's workspace, not a managed resource. The system does not move files behind the agent's back, deduplicate them, prune them, or rewrite them. If the agent creates `notes/2025-04-26/scratch.md`, that file stays exactly where the agent put it until the agent changes it.

### Human-interpretable wins

Every byte stored is plain content the user can open, read, and edit by hand. No opaque embedding index, no proprietary serialization, no metadata sidecar authoritative over the visible file. If a person can't read and reason about the store, the agent will eventually mis-edit it — and the user will have no way to recover.

### Hints over enforcement; conventions over schemas

System opinions about how the agent should organize memory live as **starter files inside the store** — not binary features, not enforced layouts, not middleware that rewrites calls. Hints are removable. Schemas are sticky.

One principled exception (sections 4 and 8): the `_` prefix at the store root is reserved for system paths, and `mycelium` rejects agent writes under it. The activity log's integrity is load-bearing for self-evolution and debugging — if the agent could rewrite its own history, both break, and operators lose the ability to diagnose dysfunction. Reserving the prefix rather than just the current path (`_activity/`) prevents future namespace collisions.

### Concurrency is a property of the store

Multi-agent semantics are not handled with locks or queues. They're expressed as primitives on the storage backend (compare-and-swap writes, etag-conditioned puts) and surfaced honestly through `mycelium`'s exit codes and structured stderr. The model decides how to resolve conflicts; the system makes sure no write is silently lost.

### Observability instead of intervention

The system records what the agent did. It does not act on what the agent did. The activity log is plain JSONL at a reserved path; the agent reads it for self-introspection, operators tail it for monitoring, nothing about it feeds back into the agent's loop automatically. The agent decides when to look.

---

## 3. Architecture Overview

Two layers, with the activity log inside the store as a reserved-path convention:

```
  Frontier-class agent
       │
       │  invokes `mycelium <sub>` via its existing shell tool
       │  env: MYCELIUM_AGENT_ID, MYCELIUM_SESSION_ID
       ▼
  `mycelium` binary
       read · write · edit · ls · glob · grep · rm · mv · log
       — enforces the _-prefix reservation
       — surfaces backend errors via exit codes + structured stderr
       │
       │  Backend interface (Go)
       ▼
  Storage Backend (LocalFS │ S3-compatible)
       — appends operation entries to _activity/YYYY/MM/DD/{agent_id}.jsonl
       — agent reads them via `mycelium read` or raw `cat`; operators do
         the same with standard log tooling
```

The **agent surface** is the shell already in its environment. There's no separate protocol layer — `mycelium` is a binary on `$PATH`. Operators reach for `cat`, `ls`, `rg`, `jq` against the same files. One surface, two callers.

The **`mycelium` binary** is a thin shim. It validates arguments, translates them into backend operations, attaches identity from environment for audit, surfaces errors via exit codes and structured stderr, and enforces the reserved-path rule. It does not interpret content or make policy decisions.

The **Storage Backend** is an interface, not a service. Two reference implementations ship: LocalFS and S3-compatible. Each implements the same contract, and each appends to `_activity/YYYY/MM/DD/{agent_id}.jsonl` on every mutation.

---

## 4. CLI Surface

A single binary, `mycelium`, invoked through the agent's shell. Nine POSIX-shaped subcommands, each justified against "could this be fewer?"

- **`mycelium read <path> [--offset N] [--limit N]`** — print file contents. Optional offset and limit for paginated reads of large files.
- **`mycelium write <path> [--content STR | --stdin] [--expected-version SHA]`** — create or overwrite. With `--expected-version`, conditional on the current version; otherwise unconditional. Prints the new version on success.
- **`mycelium edit <path> --old STR --new STR [--expected-version SHA]`** — find-and-replace a unique substring. Fails if `--old` is absent or non-unique. *Earns its complexity:* token economy on large files, diff quality under git/jj, and the unique-substring constraint catches stale-view errors a full overwrite would silently paper over.
- **`mycelium ls <path> [--recursive]`** — list entries with sizes and mtimes. Bounded result count; cursor pagination. *Earns its complexity:* `glob` returns paths, not metadata.
- **`mycelium glob <pattern>`** — print paths matching a glob (`**/*.md`, `notes/2025-*/*.txt`, `_activity/2026/04/*/*.jsonl`).
- **`mycelium grep <pattern> [--path PATH] [--regex] [--file-type T] [--format text|json] [--limit N] [--cursor C]`** — print matching lines with paths and line numbers. `--format=text` is `path:line:text`; `--format=json` returns `{matches: [{path, line, text}, ...], truncated, next_cursor?}`. `--limit` caps results (default 1000, hard ceiling); `--cursor` paginates. Backend prefers ripgrep, falls back to grep, then to scan. *Earns its complexity:* JSON and type filter make the activity log usable through general tools; the `--limit` cap keeps log-reflection from overflowing context.
- **`mycelium rm <path> [--expected-version SHA]`** — remove. *Earns its complexity:* not expressible as `write` — empty content creates an empty file, not a deletion.
- **`mycelium mv <src> <dst>`** — atomic rename within the store. *Earns its complexity:* read+write+delete is not atomic; emulating rename loses the guarantee.
- **`mycelium log <op> [--path PATH] [--payload-json STR | --stdin]`** — append a non-mutation signal entry to `_activity/YYYY/MM/DD/{agent_id}.jsonl`. The system fills `ts`, `agent_id`, `session_id`; the caller supplies `op` (a non-mutation tag like `context_signal`, `compaction`, or an agent annotation), an optional `--path` recorded on the entry, and an optional payload via either `--payload-json` (inline JSON, for callers without easy stdin access) or `--stdin` (for bash pipelines). The payload, if present, lands inline as the entry's `payload` field. Silent on success. *Earns its complexity:* harness observations, compaction markers, and agent intent annotations all need to land in the same JSONL stream as mutations so existing reads (`grep --format=json`) work without specializing on signal type. Recommendation: keep payloads small (under 4 KB) so entries stay within POSIX `O_APPEND` atomicity; for larger signals, write a regular file via `mycelium write` and reference it via `--path`.

**Failure modes:**

- exit 0 — success
- exit 1 — generic error (path not found, malformed args)
- exit 64 — CAS conflict; stderr is JSON `{"error":"conflict","current_version":"sha256:..."}`. With `--include-current-content`, also `"current_content": "..."`.
- exit 65 — protocol violation (write under reserved `_` prefix)

A successful `write` or `edit` prints `{"version":"sha256:..."}` on stdout. `rm`, `mv`, and `log` are silent on success.

What's *not* here:

- **No specialized log query DSL.** Reading the activity log uses the same tools as reading any file: `mycelium read` (or `cat`), `mycelium glob` (or `ls`) for time windows, `mycelium grep --format=json` (or `rg --json`) for filtering. Writes go through `mycelium log` (above) for explicit signals, or happen automatically as a side effect of content mutations. (See section 8.)
- **No `summarize`, `index`, `embed`, `tag`, `pin`, `archive`, `recall`.** If the agent wants any of those, it implements them by writing files.
- **No `exists` subcommand.** `mycelium read` exits non-zero with a typed not-found message.

Three contract notes:

**Conditional writes are first-class.** Every content-mutating subcommand (`write`, `edit`, `rm`, `mv`) accepts optional `--expected-version`. This is how concurrency surfaces (section 6); a single-agent store can ignore it and the system behaves like a regular filesystem. `mycelium log` appends to the agent's daily log file unconditionally — concurrent appends are safe under POSIX `O_APPEND` (section 8).

**The `_` prefix is reserved at the store root.** `mycelium` rejects `write`, `edit`, `rm`, and `mv` whose target is under any path beginning with `_`. Currently `_activity/`; future system paths (`_schema/`, `_config/`) inherit the same protection without binary changes.

**Identity travels via environment.** The harness sets `MYCELIUM_AGENT_ID` and (optionally) `MYCELIUM_SESSION_ID` once. Every invocation reads them; every log entry records them. Standard Unix request identity.

Reference invocation:

```bash
$ MYCELIUM_AGENT_ID=glp1-researcher mycelium write learnings/today.md --stdin <<EOF
Notes from this session.
EOF
{"version":"sha256:8c4d..."}
```

Conflicting edit:

```bash
$ mycelium edit learnings/today.md \
    --old "Notes from this session." \
    --new "Notes from this session.\n\nAdditional observation." \
    --expected-version sha256:abcd... --include-current-content
{"error":"conflict","current_version":"sha256:efgh...","current_content":"..."}
$ echo $?
64
```

**Agent and operator share the same surface.** There's no agent-side log API distinct from operator tooling. A future harness without shell access can wrap `mycelium` in a thin protocol adapter (section 9) — but the primary surface is shell, and the bet works without one.

---

## 5. Storage Backend Abstraction

A backend satisfies a small interface. In Go, roughly:

```go
type Backend interface {
    Get(ctx context.Context, path string) (data []byte, version string, err error)
    Put(ctx context.Context, path string, data []byte, expectedVersion *string) (version string, err error)
    Delete(ctx context.Context, path string, expectedVersion *string) error
    List(ctx context.Context, prefix string, cursor string, limit int) (entries []Entry, nextCursor string, err error)
    Search(ctx context.Context, opts SearchOptions) ([]Match, error)
}

type SearchOptions struct {
    Pattern  string
    Prefix   string
    FileType string  // "" = any
    Format   string  // "text" | "json"
    Regex    bool
}
```

Reference implementations:

- **LocalFS.** Files on disk. Versions are content hashes; CAS is write-to-temp-then-rename plus an `flock`-guarded version check. `Search` prefers ripgrep, falls back to grep, then to a Go-native scan. For single-host development and sandboxed agent processes.
- **S3-compatible.** AWS S3, Cloudflare R2, MinIO, Backblaze B2. Versions map to ETags. Conditional writes use `If-Match`. `List` maps to `ListObjectsV2`. `Search` does prefix-scoped client-side scan unless the backend offers something richer.

Every backend appends to `_activity/YYYY/MM/DD/{agent_id}.jsonl` on every successful mutation. This is the only "system writes data" behavior in the design — the price of an agent-readable activity log without a separate storage system for it.

### Activity log durability

Content commits first (LocalFS: write-temp-then-rename under `flock`; S3: PUT with `If-Match`); the activity-log append follows. On append failure the binary warns to stderr and exits 0 — the content mutation has already committed, and divergence is visible to the operator rather than silent. Multi-process LocalFS relies on `O_APPEND` atomicity for activity entries; metadata-only entries fit inside POSIX `PIPE_BUF`, and the per-agent daily path keeps contention low. LocalFS assumes a local POSIX filesystem with working `flock` and `O_APPEND` atomicity (Linux, macOS, BSD on local disk); NFS, SMB, FUSE not supported in MVP, distributed deployments use S3.

The Backend interface deliberately omits `BeginTransaction`, `Watch`, `Snapshot`, and log-write retry queues. Anything more ambitious belongs in a higher layer the agent constructs by writing files.

A backend can be **read-only** (flag at mount) — useful for sharing a curated knowledge directory across agents that should treat it as reference. A read-only mount has no `_activity/`.

A backend can be **layered**: writable LocalFS overlay on a read-only S3 base, copy-on-write semantics, behind the same interface. This is how "common organizational memory + per-agent scratch" works without new abstractions.

---

## 6. Concurrency and Multi-Agent Semantics

Multiple agents may mount the same store simultaneously. Guarantees:

1. **No silent loss.** Concurrent writes to the same path don't silently overwrite each other when conditional writes are used. Unconditional writes are documented as last-writer-wins.
2. **Visible conflicts.** A failed conditional write returns a typed error with the current version and (optionally) current content; the agent re-reads, merges, retries.
3. **Atomic single-file ops.** `write`, `edit`, `rm`, `mv` either fully apply or fully fail. No half-written files, no partial renames.
4. **Per-agent log ordering.** Each agent writes its own daily file at `_activity/YYYY/MM/DD/{agent_id}.jsonl`, append-ordered without coordination. Cross-agent total order is reconstructed by sorting on `ts` (assigned at commit). Two agents writing the same content path produce two log entries.

No multi-file transactions. If the agent wants atomicity across files, it composes it from single-file operations — typically a composite file or an agent-chosen journaling pattern.

Locks are explicitly avoided as the primary coordination mechanism. They introduce timeouts, deadlocks, and lifecycle questions ("what if the agent crashes holding the lock?"). CAS via versioned writes degrades cleanly: a conflict is just an error to read, reason about, and handle.

**Identity** travels via `MYCELIUM_AGENT_ID` and (optionally) `MYCELIUM_SESSION_ID` set once by the harness, recorded in the log, visible to any reading agent. By default, identity isn't used for access control — every mounted agent has equal permissions and the same view of the log. Per-prefix ACLs are opt-in on backends that support them; punted in v1.

A small convention helps coordination without being mandatory: starter `MYCELIUM_MEMORY.md` proposes a top-level `AGENTS/` directory where each agent maintains its own subdirectory for in-flight work; shared work goes at root or in `shared/`. **Not enforced** — a hint the agent reads once and replaces as it sees fit.

---

## 7. Self-Evolution

A Frontier agent doesn't just use general file tools well — it *reflects on its own use of them and revises its approach*. Given an empty store, it self-organizes: extracts durable lessons, archives stale notes, names things consistently, deletes on purpose. Over sessions it edits its own convention files, builds indexes when patterns emerge, consolidates when the activity log shows duplication. This is the central observable behavior the supported tier is defined by — and the property the system has to avoid breaking.

The system *enables* this with primitives the agent already has, and is careful not to *do* it on the agent's behalf:

> **Scaffolding lives in prompts and conventions — mutable, optional, removable. It never lives in the binary, the storage, or the tool surface — immutable, mandatory, sticky.**

This is the single rule. Everything else is its application.

### Concrete applications

**Starter conventions are files inside the store, not code paths.** A new mount can optionally be initialized with `MYCELIUM_MEMORY.md` at the root proposing a default layout (`learnings/`, `tasks/`, `context/`, `archive/`, naming and dating conventions). The template's first paragraph tells the agent it owns the file and may revise, replace, or delete it:

> This is your working memory's convention file. You own it. The system never edits it; it's here for you to read at session start, follow when convenient, and revise (or replace, or delete) when you see a better way to organize what you're working with. Treat it like a notebook's table of contents — useful when accurate, worse than nothing when stale.

The agent reads it once, finds it adequate or replaces it, and proceeds. **A user who wants no scaffolding mounts an empty store.**

**No automatic injection of memory hints into agent context.** The model navigates the store on its own initiative, exactly as a developer navigates a codebase.

**No automatic intervention.** The system never summarizes, dedupes, organizes, prunes, or rewrites the agent's files. If the activity log shows behavior the operator dislikes, the lever is the prompt or the model — not a system feature.

**Every piece of scaffolding is removable.** Starter `MYCELIUM_MEMORY.md`, layout conventions, anything else can be deleted or ignored without breaking the runtime. If removing it breaks the system, it doesn't belong.

### How self-evolution is enabled

Three primitives, all from section 4:

1. **Behavioral awareness via the activity log.** Operations are JSONL at `_activity/YYYY/MM/DD/{agent_id}.jsonl`. Time windows scope with `mycelium glob` (`_activity/2026/04/*/*.jsonl` for the month, `_activity/2026/04/*/researcher-7.jsonl` for one agent); filter with `mycelium grep --format=json` (or `rg --json`). Both paths produce the same output. Patterns obvious in retrospect — duplicate creation, files written but never re-read, conventions edited but unfollowed — become visible without any new tool.
2. **State awareness and modification via standard file tools.** Self-evolution adds no new mutation verbs; it gives the agent reasons to use existing ones differently.
3. **Conventions-as-files.** Any scheme the agent follows lives in editable text (`MYCELIUM_MEMORY.md`, `INDEX.md`, an agent-written `ARCHIVE_POLICY.md`). The agent rewrites its own rules with the same edit primitives as everything else.

Patterns that emerge — *not* features the system implements:

- **Convention bootstrap.** Read `MYCELIUM_MEMORY.md` at session start and apply.
- **Convention revision.** After observing in the log that "consolidate before creating" was violated, edit `MYCELIUM_MEMORY.md` with a stricter pre-write check, or add `INDEX.md` to make the right file findable.
- **Self-built indexes.** Notice (by grepping the log for repeated `glob`/`grep`) that the same content is searched repeatedly; write `INDEX.md` mapping common queries to file paths.
- **Archiving and pruning.** Use `mycelium ls --recursive` to find paths not touched in a long time, cross-reference the log to confirm staleness, consolidate or delete.

What the system does *not* do: run a reflection step between turns; analyze patterns or detect drift for the agent; maintain or update convention files on the agent's behalf; enforce that conventions are read before acting. Doing any of these would re-introduce the capability coupling this principle exists to reject. The system makes self-evolution *possible*; the agent *does* it.

---

## 8. Observability and Export

Observability is plain JSONL files at a reserved path. No sidecar service, no audit-only API, no operator-vs-agent split. One source, two readers, one writer.

### The activity log

Two paths produce entries in `_activity/YYYY/MM/DD/{agent_id}.jsonl`: every content-mutating subcommand (`write`, `edit`, `rm`, `mv`) appends automatically on success, and `mycelium log` appends explicit signal entries (harness observations like a pi.dev `context` event, compaction markers, agent intent annotations). Reads (`read`, `ls`, `glob`, `grep`) aren't logged; the log records what changed and what was observed, not what was looked at. Skipping reads keeps the log small enough to grep, and the failure modes that matter (duplicate creation, write-without-consolidate, never-revising-conventions) are detectable from the recorded entries alone.

A mutation entry (`write`, `edit`, `rm`, `mv`):

```json
{
  "ts": "2026-04-26T18:42:11.034Z",
  "agent_id": "researcher-7",
  "session_id": "sess-9b2f",
  "op": "write",
  "path": "learnings/glp1-pipeline.md",
  "version": "sha256:8c4d...",
  "prior_version": "sha256:1f2a..."
}
```

A `mycelium log` entry with an inline payload:

```json
{
  "ts": "2026-04-26T18:43:02.117Z",
  "agent_id": "researcher-7",
  "session_id": "sess-9b2f",
  "op": "context_signal",
  "payload": {"message_count": 42, "last_role": "assistant"}
}
```

**Path layout: `_activity/YYYY/MM/DD/{agent_id}.jsonl`.** Each agent writes its own daily file; cross-agent concurrency is handled by file isolation, not coordinated appends. Time-windowed queries collapse to glob: `_activity/2026/04/*/*.jsonl` (this month, all agents); `_activity/2026/04/26/*.jsonl` (today, all agents); `_activity/2026/04/26/glp1-research.jsonl` (today, one agent). Daily granularity is the default; deployments needing finer can configure hourly. Total order across agents is `ts`-sorted. Payloads from `mycelium log` are inlined on the entry — keeping all signal data in one file means agents reflect on the log with `mycelium grep --format=json` against a single stream rather than dispatching across schemas. Recommended payload size is under 4 KB so entries stay within POSIX `O_APPEND` atomicity (`PIPE_BUF`); larger signals belong in a regular file referenced via `--path`.

**Same shell, two callers, one writer:**

- **The agent** reads the log with `mycelium glob` / `mycelium grep --format=json` / `mycelium read` — or raw `ls` / `rg --json` / `cat`. Both run against the same files. This is the substrate self-evolution runs on.
- **Operators** tail the same files with whatever they like — `tail -f`, `aws s3 sync`, Vector or Filebeat to Splunk or Datadog. Plain JSONL; standard tools work without Mycelium-specific config.
- **The system** is the only writer. `mycelium` rejects agent writes under `_`-prefixed paths (section 4). Entries land via two paths — as a side effect of every successful content mutation, ordered consistently with them, or as explicit signal entries via `mycelium log` — both stamped with the same identity.

### Native git/jj support

A LocalFS backend can be initialized inside a git or jj repo, and every operation can optionally produce a commit. This gives `git log`, `git diff`, `git blame` over the agent's memory for free, and makes the store a normal artifact in version-controlled workflows. The activity log is committed alongside content: git history and the log are two views of the same truth — log faster to grep, history richer for diffs.

### Export

Export is `tar` (or `aws s3 sync`, or `cp -r`). No proprietary format; a directory of UTF-8 files is the export format. The log comes along automatically.

### Diff and share

Two stores diff with `diff -r`. A team shares a knowledge directory by handing each other a tarball or a read-only S3 prefix. No impedance mismatch between "agent memory" and "files an engineer would email a colleague."

---

## 9. Anti-Goals

Frameworks in this space commonly ship features Mycelium deliberately omits: automatic memory extraction at session end (mem0), vector retrieval as the primary access path to memory, hierarchical tiered memory maintained by the framework (MemGPT/Letta), automatic summarization (`ConversationSummaryMemory` and friends), temporal knowledge graphs with auto-extraction (Zep, Graphiti), embedding-based deduplication of "similar" memories, system-driven reflection between turns, and specialized query DSLs over the activity log. Each encodes a salience, structure, or compression policy that ages out as models improve — wrong for some, unnecessary for others, never adjustable in the moment. The principle is the same in every case: the agent owns those decisions; the system gives it the primitives (read/write/edit, grep, an activity log it can re-read) and stays out of the loop.

Two clarifications worth naming. Vector retrieval against an *external* knowledge base is a tool the agent might choose to invoke; we reject it only as the primary access path to the agent's *own* memory. Specialized agent protocols (custom REST, MCP servers, framework-specific plugin contracts) are rejected as the *primary* surface — `mycelium read foo.md` and `cat foo.md` should produce the same bytes against the same files; an "agent surface" distinct from the "operator surface" reintroduces exactly the human-uninterpretable opacity Section 2's "human-interpretable wins" rules out. A future harness without shell access can still wrap the binary in a thin protocol adapter (a hundred lines, see Section 4), but the binary is the contract.

---

## 10. Open Questions

Each is unresolved or deferred, flagged because it might tempt a future maintainer to violate Section 2.

**Activity log retention.** Files accumulate forever without trimming. Leaning toward hybrid: system trims at a generous default, ops can override; a `_activity/RETENTION.md` declares the policy and oldest available date so the agent knows its horizon. Phase 3.

**Garbage collection of content files.** Stores grow unbounded. Per design principle: the agent prunes when prompted. Unsatisfying in long-running deployments where no one is prompting cleanup. A documented "housekeeping prompt" is the likely answer — a prompt, not a job.

**Token budget enforcement on `mycelium read`.** A read of a 10 MB file blows the context window. `--offset` / `--limit` exist; `mycelium grep` already enforces a hard `--limit` (section 4). Whether `read` should similarly enforce a max-bytes-per-read with explicit override is deferred until benchmarking shows it as a real failure mode. Phase 2 if surfaced.

**Activity log format versioning.** First line of each daily file (or `_activity/SCHEMA.md`) declares format version. Needed before downstream tooling or the agent builds rigid expectations. Phase 2 deliverable.

**External activity log sink for backend-level isolation.** Current design co-locates the log with content. A backend-level failure (S3 bucket corruption, prefix deletion, regional outage) takes content and audit history down together. Standard S3 practices (versioning, replication, separate prefix policies on `_activity/`) cover most of the gap. If a high-assurance deployment surfaces a need, the path is optional log mirroring: continue writing to `_activity/` *and* tee to an external sink. About a hundred lines and a small mount config.

---

*End of draft. Feedback welcome — particularly on section 4 (the simplification pass and the one principled enforcement exception), section 6 (concurrency primitives), and the activity-log retention question in section 10.*
