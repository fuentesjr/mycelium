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

### General tools scale with intelligence; specialized infrastructure caps it

A specialized memory API encodes assumptions: what gets saved, how it's indexed, what counts as "relevant," when to summarize. As models improve, those heuristics become drag — the system forces the agent into compression and ranking policies the model could now beat unaided.

General tools (read, write, list, edit, glob, grep) have no such ceiling. The agent invokes them through its existing shell — `mycelium read` sits in the same Bash tool as `git log` and `rg` — and `mycelium` is the smallest adapter that earns its keep: atomic conditional writes, an automatic activity log, no policy about what to save, how to name, or what's relevant. A Frontier model uses them with judgment indistinguishable from a thoughtful engineer keeping a working notebook, and the same surface gets *more* useful — not less — as the next generation arrives. This is the central bet, and every other decision is downstream of it.

### Simplicity is a virtue. Every primitive must earn its complexity.

Every tool, parameter, and backend method has been checked against "could this be expressed in terms of something already here?" The survivors are the ones that genuinely couldn't, or whose ergonomics, atomicity, or token economy made consolidation worse than duplication. When in doubt, fewer. The activity log was once a special tool; it's now ordinary files at a reserved path the agent reads like any other.

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
  Agent (Claude, GPT-5, Gemini, Llama-class, ...)
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
- **`mycelium log <op> [--path PATH] [--payload-json STR | --stdin]`** — record a non-mutation signal. Without a payload, appends a JSONL metadata entry to `_activity/YYYY/MM/DD/{agent_id}.jsonl`. With a payload, writes the payload bytes to `logs/YYYY/MM/DD/{agent_id}/<HHMMSS>.<nanos>-<op>.json` and appends a metadata entry to `_activity/...` referencing it via `signal_path`. The system fills `ts`, `agent_id`, `session_id`; the caller supplies `op` (a non-mutation tag like `context_signal`, `compaction`, or an agent annotation), optional `--path` recorded as metadata on the entry, and an optional payload via either `--payload-json` (inline JSON string, for harness-side callers without easy stdin access) or `--stdin` (for agent-side bash pipelines). Returns `{"log_status":"ok"}` on success, or `{"log_status":"missing"}` when the payload was stored but the metadata write failed. *Earns its complexity:* harness observations, compaction markers, and agent intent annotations all need to land in the same JSONL stream as mutations so existing reads (`grep --format=json`) work without specializing on signal type. Splitting payloads into `logs/` keeps `_activity/` strictly binary-controlled metadata; the agent reads (and may groom) its own signal files like any other content.

**Failure modes:**

- exit 0 — success
- exit 1 — generic error (path not found, malformed args)
- exit 64 — CAS conflict; stderr is JSON `{"error":"conflict","current_version":"sha256:..."}`. With `--include-current-content`, also `"current_content": "..."`.
- exit 65 — protocol violation (write under reserved `_` prefix)

A successful mutation prints `{"version":"sha256:...","log_status":"ok"}` on stdout.

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
{"version":"sha256:8c4d...","log_status":"ok"}
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

A mutation has succeeded only when both content change and log entry are durable. If either fails, the operation is reported as failed and partial state is rolled back.

**LocalFS** orders writes through the filesystem. For `write`: write content to `path.{nonce}.tmp` and `fsync`; append the log entry (with the post-rename `version`) to the agent's daily file (`O_APPEND`), then `fsync`; atomically rename. A crash between append and rename leaves an entry whose `version` doesn't match any current file content; on startup, the backend scans recent entries against current hashes and either replays the rename (if the temp file survived) or appends a follow-up entry marking the original aborted. Crashes earlier leave only the temp file; startup cleanup removes it. Multi-process LocalFS relies on `O_APPEND` atomicity for activity entries; metadata-only entries fit naturally inside POSIX `PIPE_BUF` (section 8), and the per-agent daily path keeps contention low.

**LocalFS portability.** Assumes a local POSIX filesystem with working `flock` and `O_APPEND` atomicity (Linux, macOS, BSD on local disk). NFS, SMB, FUSE not supported in MVP; distributed deployments use S3.

**S3 and other object stores** can't make two PUTs atomic. Log entries are best-effort with respect to crash atomicity; every operation result includes `log_status` (`"ok"` | `"deferred"` | `"missing"`) so divergence between content and log is *visible*. Phase 2 documents the failure model in detail; the design commits to the contract that any divergence is visible, not silent.

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

A `mycelium log` entry, where the agent supplied a payload:

```json
{
  "ts": "2026-04-26T18:43:02.117Z",
  "agent_id": "researcher-7",
  "session_id": "sess-9b2f",
  "op": "context",
  "signal_path": "logs/2026/04/26/researcher-7/184302.117000000-context.json"
}
```

**Path layout: `_activity/YYYY/MM/DD/{agent_id}.jsonl`.** Each agent writes its own daily file; cross-agent concurrency is handled by file isolation, not coordinated appends. Time-windowed queries collapse to glob: `_activity/2026/04/*/*.jsonl` (this month, all agents); `_activity/2026/04/26/*.jsonl` (today, all agents); `_activity/2026/04/26/glp1-research.jsonl` (today, one agent). Daily granularity is the default; deployments needing finer can configure hourly. Total order across agents is `ts`-sorted. Entries are intentionally metadata-only — agent-supplied payloads from `mycelium log` live in `logs/YYYY/MM/DD/{agent_id}/<HHMMSS>.<nanos>-<op>.json` and are referenced by `signal_path`. Keeping activity entries small lets them fit inside POSIX `O_APPEND` atomicity (`PIPE_BUF`, typically 4KB) without an explicit per-line cap, and keeps the log greppable.

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

Each is a feature commonly found in competing systems, rejected by name with reasoning.

### Automatic memory extraction at session end

**Where it appears:** mem0, parts of LangChain, AutoGen and CrewAI memory modules. **Why we reject it:** Extraction encodes a salience policy hand-tuned for a particular model class. As models improve, the extractor becomes a lossy filter between the agent and what it actually wanted to keep. The agent already has the tools to write what it wants to remember when it knows it wants to remember it.

### Vector retrieval over memory files

**Where it appears:** LangChain `VectorStoreRetrieverMemory`, most "RAG-as-memory" architectures. **Why we reject it:** Embeddings drift on edit. Not human-readable, not diffable, not portable across embedding models without re-indexing. They introduce a retrieval ranking the agent can't reason about. They train users to think of memory as "a search engine over what the agent said," which is the wrong frame: memory is a workspace, not a search corpus. `grep` and `glob` are surprisingly good on a well-organized directory and get better as models get better at organizing directories.

We aren't against vector retrieval as a tool the agent might choose to invoke against an external knowledge base — only against it as the *primary access path to the agent's own memory*.

### MemGPT-style hierarchical memory tiers

**Where it appears:** MemGPT (now Letta), several derivatives. **Why we reject it:** Tiered memory makes the framework, not the agent, decide what's in working / external / archival memory. The tiering policy was a fit for one context window and one model's attention management — not next year's models, and not adjustable in the moment. A flat directory the agent navigates by intent collapses tiering into a tooling question the agent already answers.

### Automatic summarization of long files or sessions

**Where it appears:** LangChain `ConversationSummaryMemory` and friends, most "long-running agent" frameworks. **Why we reject it:** Same critique as extraction. Summarization is a compression policy; the system doesn't know what the agent's downstream task needs, the agent does. If the agent wants a summary, it writes one.

### Zep-style temporal knowledge graphs with auto-extraction

**Where it appears:** Zep, Graphiti. **Why we reject it:** Knowledge graphs encode an ontology fixed at design time. As models improve they extract structure on demand from unstructured text — flat plain-text becomes more capable over time, while a fixed graph becomes more rigid.

### Tiered summary layers maintained by the framework

**Where it appears:** Most "agent memory" SaaS, several open-source frameworks with rolling summaries. **Why we reject it:** Maintenance happens behind the agent's back with policies the agent didn't set. The agent may rely on summaries that have drifted from the underlying files; the user may discover the system has been silently rewriting their notes. If summaries are wanted, the agent writes and maintains them as ordinary files.

### Embedding-based deduplication

**Where it appears:** Frameworks that auto-deduplicate "similar" memories. **Why we reject it:** Two near-duplicates may be intentional — different contexts, angles. The system isn't positioned to decide; the agent is.

### System-driven reflection or self-evolution

**Where it appears:** Frameworks that ship a reflection step between turns or maintain auto-updated "lessons learned" files. **Why we reject it:** Same anti-pattern as auto-summarization, one level up. It encodes a policy about *when* and *how* the agent should review its own behavior — wrong for some models, unnecessary for others. Right answer: give the agent the primitives (an activity log it can grep, editable convention files), let the prompt determine when reflection happens, and stay out of the loop. Self-evolution is an agent behavior, not a system feature (section 7).

### Specialized query languages or APIs for the activity log

**Where it appears:** Most observability systems ship a query DSL (LogQL, KQL). **Why we reject it:** The log is JSONL and the agent already has `grep --format=json`. Inventing a DSL adds a tool, parser, and documentation surface to do work the existing primitive already does.

### Specialized agent protocols where shell suffices

**Where it appears:** Most agent memory frameworks ship a custom protocol — proprietary REST, MCP servers, framework-specific plugin contracts — as the agent's interface to memory. **Why we reject it as the primary surface:** Frontier agents already have shell. Wrapping eight POSIX-shaped operations in a separate protocol adds a manifest, transport, schema generator, and tool dispatcher to do work the shell already does. The agent runs `mycelium read foo.md`; the operator runs `cat foo.md`; both see the same bytes against the same files. A separate protocol creates an "agent surface vs. operator surface" distinction "human-interpretable wins" exists to reject. If a future harness can't grant shell access, a thin MCP wrapper is a hundred lines over the same binary — but never the primary surface.

---

## 10. Open Questions

Each is unresolved or deferred, flagged because it might tempt a future maintainer to violate section 2.

**Binary blobs.** The CLI is text-oriented. Options: separate `mycelium read-blob` / `write-blob` (likely right) vs. base64 convention on `write` (worse). How does an agent reason about a file it can't natively read? By the path, filename, and a sibling text file with notes — exactly as a human would.

**Garbage collection of content files.** Stores grow unbounded. Per design principle: the agent prunes when prompted. Unsatisfying in long-running deployments where no one is prompting cleanup. We may need a documented "housekeeping prompt" — but it's a prompt, not a job.

**Activity log retention.** Files will accumulate forever without trimming. Leaning toward hybrid: system trims at a generous default, ops can override; a `_activity/RETENTION.md` declares the policy and oldest available date so the agent knows its horizon.

**Token budget enforcement.** `mycelium read` of a 10 MB file blows the context window. `--offset` / `--limit` exist; `mycelium grep` already enforces `--limit` (section 4). Should `mycelium read` similarly enforce a max-bytes-per-read with explicit override? Probably yes — the asymmetry is awkward and the failure mode sharper.

**Cross-store federation.** Almost certainly yes — mount multiple stores at different paths (`/team/`, `/me/`). Layered backend handles it; open question is mount config without becoming complicated. Each mount has its own `_activity/`, so cross-mount queries are a glob across mounts.

**Symlinks.** Almost certainly: refuse them. Attack surface, breaks portability, no value rename doesn't already provide.

**Activity log format versioning.** First line of each daily file (or `_activity/SCHEMA.md`) declares format version. Needed before downstream tooling or the agent builds rigid expectations.

**Conflict resolution UX.** Return current version unconditionally; current content opt-in via `--include-current-content`; the agent decides whether to re-fetch.

**Backwards self-evolution.** At MVP the log records version hashes but not content, so an agent that destructively rewrites its own `MYCELIUM_MEMORY.md` can't recover the prior version through Mycelium primitives. It can detect *that* the write happened, *when*, and *by whom* — not the prior content. Phase 3's historical reads (`mycelium read --version=...` on git/jj-backed LocalFS or versioned S3) close the gap. Until then, deployments wanting a backstop opt into a git-backed LocalFS variant out of band; the design's stance is that an agent that mangles its own conventions is a model-quality problem, not a system one.

**Default starter content.** Ship `MYCELIUM_MEMORY.md` populated, or empty? Leaning empty (`_activity/` initializes lazily on first write); document templates; make `mycelium init --template=default` a one-line opt-in.

**Read-only knowledge sharing.** Layered backend supports it. The UX — how a user expresses "this prefix read-only, this one read-write" — needs design. Probably a mount manifest. Not hard, just unspecified.

**External activity log sink for backend-level isolation.** Current design co-locates the log with content. A backend-level failure (S3 bucket corruption, prefix deletion, regional outage) takes content and audit history down together. Standard S3 practices (versioning, replication, separate prefix policies on `_activity/`) cover most of the gap. If a high-assurance deployment surfaces a need, the path is optional log mirroring: continue writing to `_activity/` *and* tee to an external sink (separate-account append-only object store, syslog endpoint, etc.). The agent's read path is unchanged; only operators see the mirror. About a hundred lines and a small mount config.

---

*End of draft. Feedback welcome — particularly on section 4 (the simplification pass and the one principled enforcement exception), section 6 (concurrency primitives), and the activity-log retention question in section 10.*
