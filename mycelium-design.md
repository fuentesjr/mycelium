# Mycelium: A Model-Agnostic Agent Memory System

**Status:** Design draft
**Project:** Mycelium

A persistent memory substrate for autonomous agents, built on the bet that **a real filesystem driven by general file tools outperforms specialized memory infrastructure as models improve** — and that this bet should be cashed in without taking a dependency on any single model provider.

---

## 1. Goals and Non-Goals

### Goals

- Give an agent durable memory by mounting a directory into its working environment and exposing standard file operations as tools.
- Let the agent own the schema. The agent decides what to save, how to name it, and how to organize it.
- Keep memory human-interpretable end to end — exportable as a directory, diffable as text, shareable as a tarball.
- Support multiple agents concurrently mounting the same store with consistent, predictable semantics on conflict.
- Run on any function-calling-capable model via a standard tool schema (MCP and/or OpenAI tool-call format).
- Run against any storage backend — local filesystem or S3-compatible object store — behind one interface.
- Stay useful today on Mid-High models without baking in compensating logic that becomes overhead, or worse, a ceiling, on Frontier-class models.
- Equip the agent to **observe and revise its own memory practices over time** — using the same general tools it uses for everything else.

### Non-Goals

- A memory API. There is no `remember(fact)` or `recall(query)`. Memory is a directory; the operations are the file tools.
- Automatic extraction, summarization, or compression of agent output into "memories."
- Embedding-based retrieval as the primary access path.
- Tiered memory (working / episodic / archival) maintained by infrastructure rather than by the agent.
- Schema enforcement on agent-authored content. The agent writes whatever it wants, however it wants.
- System-driven reflection or self-evolution. The system *enables* self-evolution by giving the agent the right primitives; it does not *perform* self-evolution on the agent's behalf.
- Compensating for limited model judgment with infrastructure features. If the model is making a mess, the answer is a stronger model or a better prompt — not a feature in this system.

### Supported model tiers

Mycelium targets two model tiers:

- **Mid-High.** Production models with strong general reasoning and tool-use, but not the top of class (Claude Sonnet 4.x, GPT-4o, Gemini Pro, equivalent open-weights). The floor — these models drive the central bet's "stays useful today" half.
- **Frontier.** Top-of-class production models (Claude Opus 4.x, the leading GPT-5 tier, Gemini Ultra, equivalent). The natural ceiling — these models drive the bet's "scales with intelligence" half.

Models below Mid-High are out of scope. The system is forward-looking; baking in compensating logic for a tier already on its way out would create a permanent ceiling on the tiers that matter.

---

## 2. Design Principles

### General tools scale with intelligence; specialized infrastructure caps it

A specialized memory API encodes assumptions: what gets saved, how it's indexed, what counts as "relevant," when to summarize. Each assumption is a heuristic that made sense at the model capability level it was designed for. As models improve, those heuristics become drag — the system is forcing the agent into compression policies and retrieval rankings the model could now beat unaided.

General tools (read, write, list, edit, glob, grep) have no such ceiling. A Mid-High model uses them workmanlike; a Frontier model uses them with judgment indistinguishable from a thoughtful engineer keeping a working notebook. The same surface scales across the supported range, and grows with each generation. This is the central bet of the system, and every other decision is downstream of it.

### Simplicity is a virtue. Every primitive must earn its complexity.

Every tool, every parameter, every backend method has been checked against the question "could this be expressed in terms of something already here?" The ones that survived are the ones that genuinely couldn't, or whose ergonomics, atomicity guarantees, or token economy made the consolidation worse than the duplication. When in doubt, fewer.

Corollary: when an idea looks like it needs a new tool or a new abstraction, the first move is to ask whether an existing primitive plus a convention covers it. The activity log used to be a special tool (`query_activity_log`); it's now ordinary files at a reserved path that the agent reads with `read_file` and `grep`. That's the pattern.

### Files are the unit. Directories are the structure. The agent owns both.

The agent's filesystem is its workspace, not a managed resource. The system does not move files around behind the agent's back, deduplicate them, prune them, or rewrite them. If the agent creates `notes/2025-04-26/scratch.md`, that file stays exactly where the agent put it, with exactly the content the agent wrote, until the agent changes it.

### Human-interpretable wins

Every byte stored is plain content the user can open, read, and edit by hand. There is no opaque embedding index, no proprietary serialization, no metadata sidecar that's authoritative over the visible file. If a person can't read and reason about the store, the agent will eventually mis-edit it — and the user will have no way to recover.

### Hints over enforcement; conventions over schemas

Where the system has opinions about how a particular tier of model should organize memory, it expresses them as **prompt fragments and starter files inside the store** — not as protocol features, not as enforced layouts, not as middleware that rewrites the agent's calls. Hints are removable. Schemas are sticky.

There is exactly one principled exception (§4 and §8): the `_` prefix at the store root is reserved for system paths, and the protocol rejects agent writes to anything under it. Currently the only such path is `_activity/`; future system-owned paths (e.g., `_schema/`, `_config/`) follow the same convention without further protocol changes. The integrity of the activity log is load-bearing for self-evolution and debugging; if the agent could rewrite its own history, it could launder out evidence of its own dysfunction, and the user would lose the ability to diagnose failure modes. Reserving the prefix rather than just the current path prevents future namespace collisions. This is the only case where enforcement beats convention.

### Concurrency is a property of the store

Multi-agent semantics are not handled at the memory protocol layer with locks or queues. They are expressed as primitives on the storage backend (compare-and-swap writes, etag-conditioned puts) and surfaced honestly to the agent through tool errors. The model decides how to resolve conflicts; the system makes sure no write is ever silently lost.

### Observability instead of intervention

The system records what the agent did. It does not act on what the agent did. The activity log is plain JSONL files at a reserved path; the agent reads it for self-introspection, operators tail it for monitoring, and nothing about it feeds back into the agent's loop automatically. The agent decides when to look.

---

## 3. Architecture Overview

Three layers, with the activity log living inside the store as a reserved-path convention rather than as a separate sidecar:

```
┌────────────────────────────────────────────────────────────┐
│  Model (any function-calling model: Claude, GPT-4o,        │
│  Llama-class, Qwen, Gemini, ...)                           │
└──────────────────────────┬─────────────────────────────────┘
                           │ tool calls (MCP / OpenAI schema)
┌──────────────────────────▼─────────────────────────────────┐
│  Tool Surface                                              │
│  read_file · write_file · edit_file · list_directory ·     │
│  glob · grep · delete_file · move_file                     │
└──────────────────────────┬─────────────────────────────────┘
                           │ Memory Protocol (thin)
                           │ enforces: writes rejected under _-prefixed paths
┌──────────────────────────▼─────────────────────────────────┐
│  Storage Backend (pluggable)                               │
│  LocalFS │ S3-compatible                                  │
│                                                             │
│  Backend writes operation entries to                       │
│   _activity/YYYY/MM/DD/{agent_id}.jsonl                    │
│  Agent reads them like any other file.                     │
│  Operators tail them with standard log tooling.            │
└────────────────────────────────────────────────────────────┘
```

The **Tool Surface** is the only thing the model sees. It speaks a standard tool-call schema; any function-calling model can drive it.

The **Memory Protocol** is a deliberately thin shim. It validates tool calls, translates them into backend operations, attaches agent identity for audit, surfaces backend errors honestly, and enforces the one reserved-path rule. It does not interpret content, transform it, or make policy decisions about it.

The **Storage Backend** is an interface, not a service. Two reference implementations ship: local filesystem and S3-compatible object store. Each implements the same minimal contract, and each is responsible for appending an entry to `_activity/YYYY/MM/DD/{agent_id}.jsonl` on every mutating operation.

---

## 4. Tool Surface and Schema

The tool surface is intentionally close to a working developer's command-line vocabulary. Eight tools, each justified against "could this be expressed with fewer?"

- **`read_file(path, offset?, limit?)`** — return the contents of a file. Optional byte offset and limit for paginated reads of large files.
- **`write_file(path, content, expected_version?)`** — create or overwrite. If `expected_version` is supplied, the write is conditional on the current version matching; otherwise unconditional. Returns the new version.
- **`edit_file(path, old_str, new_str, expected_version?)`** — find-and-replace a unique substring. Fails if `old_str` is absent or non-unique. Same conditional semantics as `write_file`. *Earns its complexity:* token economy on large files (a 5 KB file with a one-line change doesn't need to be retransmitted), diff quality when wired to git/jj, and the unique-substring constraint catches stale-view errors that a full overwrite would silently paper over.
- **`list_directory(path, recursive?)`** — list entries with sizes and modification times. Bounded result count; pagination via cursor. *Earns its complexity:* `glob` returns paths, not metadata; the agent often wants to see when files were last touched.
- **`glob(pattern)`** — return paths matching a glob pattern (`**/*.md`, `notes/2025-*/*.txt`, `_activity/2026/04/*/*.jsonl`).
- **`grep(pattern, path?, regex?, file_type?, format?, limit?, cursor?)`** — return matching lines with file paths and line numbers. `file_type` narrows by extension or named type (e.g., `json`, `md`). `format` is `"text"` (default, line-oriented `path:line:text`) or `"json"` (an envelope `{matches: [{path, line, text}, ...], truncated: bool, next_cursor?: string}` where `text` is the matched line returned verbatim — models do their own JSON parsing of matched JSONL content). `limit` caps `matches.length` (default 1000, hard ceiling); `cursor` paginates forward via `next_cursor`. The backend implements this with ripgrep when available, falls back to grep, and ultimately to a scan. *Earns its complexity:* the JSON output and type filter are what make the activity log usable through general tools; both are useful for non-log work too. The mandatory `limit` cap is what keeps log-reflection queries from overflowing the model's context window.
- **`delete_file(path, expected_version?)`** — remove a file. *Earns its complexity:* not expressible as `write_file` — empty content creates an empty file, not a deleted one.
- **`move_file(src, dst)`** — atomic rename within the store. *Earns its complexity:* read+write+delete is not atomic; backends provide rename as a primitive, and emulating it loses the guarantee.

What's *not* here, and why:

- **No `query_activity_log`.** The activity log is JSONL files at `_activity/YYYY/MM/DD/{agent_id}.jsonl`. The agent reads recent activity with `read_file`, scopes time windows with `glob`, and filters by op or agent or path with `grep --format=json`. One existing primitive — `grep` with a few new parameters — does the work of a special tool. (See §8.)
- **No `summarize`, `index`, `embed`, `tag`, `pin`, `archive`, or `recall`.** If the agent wants any of those behaviors, it implements them by writing files. A `tags.json` is the agent's own choice, not a system feature.
- **No `exists` tool.** `read_file` returns a typed not-found error; that's enough.

Two schema notes:

**Conditional writes are first-class.** Every mutating operation accepts an optional `expected_version`. This is how concurrency surfaces to the agent (see §6). It is *optional* on every call — a single-agent store can ignore it entirely and the system behaves like a regular filesystem.

**The `_` prefix at the store root is reserved.** The protocol rejects `write_file`, `edit_file`, `delete_file`, and `move_file` calls whose target (or, for moves, source or destination) is under any path beginning with `_` at the store root. Currently this means `_activity/`; future system-owned paths (e.g., `_schema/`, `_config/`) inherit the same protection without further protocol changes. This is the one principled exception to "hints over enforcement." The activity log's integrity is the substrate self-evolution and debugging both run on; if the agent could rewrite it, both break — and reserving the prefix rather than just one path prevents future namespace collisions.

A reference JSON schema for `write_file`:

```json
{
  "name": "write_file",
  "description": "Create or overwrite a file at the given path within the memory store. If expected_version is provided, the write succeeds only if the file's current version matches; this enables safe concurrent writes across agents. Writes under any _-prefixed root path (currently _activity/) are rejected.",
  "input_schema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Path relative to the memory store root, e.g. 'notes/today.md'"
      },
      "content": {
        "type": "string",
        "description": "UTF-8 file contents."
      },
      "expected_version": {
        "type": "string",
        "description": "Optional version token from a prior read; if present and stale, the write fails with a conflict error."
      }
    },
    "required": ["path", "content"]
  }
}
```

The tool surface is published as both an MCP server manifest and an OpenAI tool-call array, generated from a single source schema. Neither format is privileged; both are first-class.

---

## 5. Storage Backend Abstraction

A backend is anything that satisfies a small interface. In Go, roughly:

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

- **LocalFS.** Files on disk. Versions are computed as content hashes; CAS is implemented with a write-to-temp-then-rename plus an `flock`-guarded version check. `Search` prefers ripgrep, falls back to grep, falls back to a Go-native scan. Suitable for single-host development and sandboxed agent processes.
- **S3-compatible.** Works against AWS S3, Cloudflare R2, MinIO, Backblaze B2. Versions map to ETags. Conditional writes use `If-Match` (now standard on S3). `List` maps to `ListObjectsV2`. `Search` does a prefix-scoped client-side scan unless the backend offers something richer.

Every backend is responsible for appending an entry to `_activity/YYYY/MM/DD/{agent_id}.jsonl` on every successful mutating operation. This is the only "system writes data" behavior in the design, and it's the price of having an agent-readable activity log without inventing a separate storage system for it.

### Activity log durability

A mutating operation has succeeded only when both the content change and its log entry are durable. If either fails, the operation is reported as failed and any partial state is rolled back.

**LocalFS** orders writes through the filesystem. For `write_file`: write content to a temp file `path.{nonce}.tmp` and `fsync`; append the log entry (with the post-rename `after_version`) to `_activity/YYYY/MM/DD/{agent_id}.jsonl` opened with `O_APPEND`, then `fsync`; atomically rename `path.{nonce}.tmp` → `path`; return success. A crash between log append and rename leaves a log entry whose `after_version` does not match any current file content; on startup, the backend scans recent log entries against current file hashes and either replays the rename (if the temp file survived) or appends a compensating `result: "aborted"` entry. Crashes earlier than the log append leave only the temp file, which startup cleanup removes. Multi-process LocalFS deployments rely on `O_APPEND` atomicity for log entries ≤4KB (enforced at the protocol layer; see §8) and on the per-agent daily file path to prevent inter-agent contention.

**LocalFS portability.** The backend assumes a local POSIX filesystem with working `flock` and `O_APPEND` atomicity (Linux, macOS, BSD on local disk). NFS, SMB, and FUSE filesystems are not supported in MVP; distributed deployments use the S3 backend.

**S3 and other object stores** cannot make two PUTs atomic. The contract relaxes accordingly: log entries are best-effort with respect to crash atomicity, and every operation result includes a `log_status` field (`"ok"` | `"deferred"` | `"missing"`) so an agent or operator can detect divergence between content history (visible via S3 versioning / ETag list) and the log. The Phase 2 implementation will document the failure model in detail; the design commits only to the contract that any divergence is *visible* rather than silent.

The Backend interface deliberately does not include log-write retry queues, replication, or transactional grouping. Backends needing richer durability (a log replicator to a separate bucket, syslog mirroring) build it as wrappers, not as core methods.

The interface is deliberately narrow. There is no `BeginTransaction`, no `Watch`, no `Snapshot`. Anything more ambitious belongs in a higher layer that the agent itself constructs by writing files.

A backend can be **read-only** (set a flag at mount time) — useful for sharing a curated knowledge directory across agents that should treat it as reference, not scratch. A read-only mount has no `_activity/` since there are no mutations to log.

A backend can also be **layered**: a writable LocalFS overlay on top of a read-only S3 bucket, copy-on-write semantics, all behind the same interface. This is how the system supports patterns like "common organizational memory + per-agent scratch" without any new abstractions.

---

## 6. Concurrency and Multi-Agent Semantics

Multiple agents may mount the same store simultaneously. The system guarantees:

1. **No silent loss.** Concurrent writes to the same path will not silently overwrite each other when the agent uses conditional writes. Unconditional writes are last-writer-wins and are documented as such.
2. **Visible conflicts.** When a conditional write fails, the agent receives a typed error containing the current version and (optionally) the current content, so it can re-read, merge, and retry.
3. **Atomic single-file operations.** A `write_file`, `edit_file`, `delete_file`, or `move_file` either fully applies or fully fails. No half-written files. No partial renames.
4. **Activity log entries are durable and per-agent ordered.** Each agent writes its own daily log file at `_activity/YYYY/MM/DD/{agent_id}.jsonl`, so per-agent streams are append-ordered without coordination. Total order across agents is reconstructed by sorting on `ts`, which is assigned by the backend at commit time. Two agents writing the same content path produce two log entries (one per agent's stream); their relative ordering is the timestamp comparison. The log is the only authoritative source of the operation sequence.

There are no multi-file transactions. If the agent wants atomicity across files, it composes it from single-file operations — typically by writing a single composite file or by using a journaling pattern the agent itself chooses to apply. This is consistent with the principle of letting the agent own structure.

Locks are explicitly avoided as the primary coordination mechanism. They introduce timeouts, deadlocks, and lifecycle questions ("what if the agent crashes holding the lock?") that distract from the model's actual job. CAS via versioned writes degrades cleanly: a conflict is just an error the agent reads, reasons about, and handles.

**Agent identity** travels with each call as a header (`agent-id`, optionally `session-id`). It is recorded in the activity log and visible to any agent reading the log, which is what makes the log usable as a coordination signal across agents and sessions. `session-id` is set by the caller (typically the harness) and recorded without interpretation; the agent identifies prior sessions either by querying its own log entries or by maintaining a session index in convention files of its own choosing. By default, identity is *not* used for access control — every mounted agent has the same permissions and the same view of the log. Per-prefix ACLs are an opt-in feature on backends that support them; we punt on this in v1.

A small convention helps multi-agent coordination without being mandatory: the system documents (in starter conventions) a top-level `AGENTS/` directory where each agent maintains its own subdirectory for in-flight work. Shared work goes at the root or in `shared/`. **None of this is enforced by the protocol** — it's a hint in the starter `MYCELIUM_MEMORY.md` that Mid-High agents tend to follow and that Frontier agents adapt or replace.

---

## 7. Capability-Tier Strategy and Self-Evolution

This section addresses the design tension head-on, and then lays out how the same primitives that handle the tension also let the agent evolve its memory practices over time.

### The tension

Mid-High and Frontier models both manage a filesystem competently, but they don't manage it the same way. Given an empty store, a Mid-High agent will produce a workable layout and follow it consistently — *if* given a sane starter convention to anchor on. Without one, it tends to drift: file-naming conventions diverge across sessions, near-duplicates accumulate when re-reading would have caught them, and the activity log gets used reactively (when the agent is asked to look) rather than proactively (when the agent notices it should). A Frontier model, given the same empty store, self-organizes — extracts durable lessons into a `learnings.md`, archives stale notes, names things consistently, deletes on purpose, and notices its own drift before anyone has to point it out.

The naive responses are both wrong:

- **Compensate in infrastructure** (auto-dedup, auto-summarize, scheduled compaction). This stabilizes Mid-High behavior but becomes a hard ceiling on Frontier. The Frontier model is now fighting the framework, which is rewriting its files based on heuristics that were a fit for last year's capabilities.
- **Refuse to compensate** (ship the bare protocol with no starter convention, no prompt fragments). This is honest about the trajectory but gives Mid-High users a system that fails to coalesce around any organizational principle out of the box — every store grows differently, no across-deployment learnings carry over.

### The principle

> **Scaffolding lives in prompts and conventions — mutable, optional, removable. It never lives in the protocol, the storage, or the tool surface — immutable, mandatory, sticky.**

This is the single rule. Everything in this section is its application.

### Concrete applications

**Starter conventions ship as files inside the store, not as code paths.** A new mount can optionally be initialized with a `MYCELIUM_MEMORY.md` at the root that describes a default layout (`learnings/`, `tasks/`, `context/`, `archive/`, naming and dating conventions, when to consolidate, when to delete). Mid-High agents read it and follow it. Frontier agents read it once, find it adequate or replace it, and proceed accordingly. **A user who wants no scaffolding mounts an empty store.** The convention is data, not code; it cannot lock anyone in.

**Prompt fragments are a library, not a default.** We ship a curated set of system-prompt addenda — one for "model tends to create duplicates," one for "model fails to re-read before writing," one for multi-agent coordination — that users can opt into per deployment. They are not injected. They are not on by default. They live in documentation.

**No automatic injection of memory hints into agent context.** The system does not stuff a "current memory state" block into the model's context window on every turn. The model navigates the store with its tools, on its own initiative, exactly as a developer navigates a codebase.

**No automatic intervention on the store.** The system never summarizes, dedupes, organizes, prunes, or rewrites the agent's files. If the activity log shows a Mid-High agent accumulating near-duplicates without consolidating, the user's signal is to add the relevant prompt fragment, revise the starter convention, or move to a stronger model. The system's only job at that moment is to make the failure visible.

**The compensating mechanisms remain removable.** Every piece of scaffolding the system offers — starter `MYCELIUM_MEMORY.md`, prompt fragments, layout conventions — can be deleted or ignored without breaking anything. There is no piece of the runtime that depends on the agent following a convention. This is the test: if removing it breaks the system, it doesn't belong.

### Self-evolution as an emergent property

Frontier models don't just use general file tools well — they *reflect on their own use of them and revise their approach*. That's what we mean by self-evolution: the agent observes its own behavior, decides something isn't working, and changes how it organizes memory, without anyone (system or operator) telling it to. Mid-High models do this when their prompt directs them to (which is what the prompt-fragment library exists for); Frontier models do it spontaneously when given the right activity-log access.

The system enables this with primitives the agent already has:

1. **Behavioral awareness via reading the activity log.** Operations are recorded as JSONL at `_activity/YYYY/MM/DD/{agent_id}.jsonl`. The agent uses `glob` to scope a time window and an agent set (`_activity/2026/04/*/*.jsonl` for everyone this month, `_activity/2026/04/*/researcher-7.jsonl` for one agent), `grep --format=json` to filter by op or path, and `read_file` to load a specific day's log. Patterns that are obvious in retrospect — duplicate file creation, files written but never re-read, conventions edited but not followed — become visible without any new tool.
2. **State awareness and modification via the standard file tools.** The agent already had these. Self-evolution doesn't add new mutation verbs; it just gives the agent a reason to use the existing ones differently.
3. **Conventions-as-files.** Any organizational scheme the agent follows lives in editable text files (`MYCELIUM_MEMORY.md` at the root, `INDEX.md` for a hand-built lookup, an `ARCHIVE_POLICY.md` the agent wrote for itself). Because these are ordinary files, they're subject to the same edit primitives as everything else. The agent can rewrite its own rules.

Concrete patterns that emerge — *not* features the system implements:

- **Convention bootstrap.** At session start, the agent reads `MYCELIUM_MEMORY.md` and applies its rules to subsequent reads and writes.
- **Convention revision.** After observing in the activity log that it has created twelve near-duplicate notes despite a "consolidate before creating" rule, the agent edits `MYCELIUM_MEMORY.md` to add a stricter pre-write check, or adds an `INDEX.md` to make the right file findable. The next session reads the updated rules.
- **Self-built indexes.** The agent notices, by grepping its own log for repeated `glob`/`grep` patterns, that it keeps searching for the same content. It writes an `INDEX.md` that maps common queries to the relevant file paths, short-circuiting future searches.
- **Archiving and pruning.** The agent uses `list_directory` to find paths not touched in a long time, cross-references the activity log to confirm they're stale, and consolidates or deletes accordingly.

What the system does *not* do:

- It does not run a "reflection step" between turns. There is no scheduled introspection.
- It does not analyze the agent's patterns for it. There is no "drift detector" that flags duplicate-creation behavior to the agent.
- It does not maintain or update `MYCELIUM_MEMORY.md` (or any convention file) on the agent's behalf. Convention drift is the agent's problem to notice and the agent's problem to fix.
- It does not enforce that the agent reads its conventions before acting. That's a prompt-level concern.

The same setup runs across both supported tiers. A Mid-High model, given a starter convention and a prompt fragment that directs it to grep its log periodically, produces a store that stays organized across sessions and improves measurably with operator nudges. A Frontier model, given the same setup or no setup at all, evolves its own conventions and produces a store that gets *more* useful over time without operator nudges at all. **Same protocol, same primitives, different observed behavior** — exactly the property we wanted from the central bet.

If the system tried to drive evolution itself — auto-revising `MYCELIUM_MEMORY.md`, auto-detecting "drift," auto-rebuilding indexes — it would re-introduce all the capability-tier coupling §7's principle exists to reject. Self-evolution is something the system makes *possible*, not something it *does*.

---

## 8. Observability and Export

Observability is achieved by writing operations to plain JSONL files at a reserved path. There is no separate sidecar service, no audit-only API, no operator-vs-agent split in the data model. One source of truth, two readers, one writer.

### The activity log

Every *mutating* operation — `write_file`, `edit_file`, `delete_file`, `move_file` — produces a JSON entry, appended to `_activity/YYYY/MM/DD/{agent_id}.jsonl`. Reads (`read_file`, `list_directory`, `glob`, `grep`) are not logged; the log records what changed, not what was looked at. Skipping read entries keeps the log small enough to grep, and the failure modes that matter (duplicate creation, write-without-consolidate, never-revising-conventions) are all detectable from writes alone.

```json
{
  "ts": "2026-04-26T18:42:11.034Z",
  "agent_id": "researcher-7",
  "session_id": "sess-9b2f",
  "op": "write_file",
  "path": "learnings/glp1-pipeline.md",
  "before_version": "sha256:1f2a...",
  "after_version": "sha256:8c4d...",
  "bytes": 4127,
  "result": "ok"
}
```

**Path layout:** `_activity/YYYY/MM/DD/{agent_id}.jsonl`. Each agent writes its own daily log; concurrency between agents is handled by file isolation rather than coordinated appends. Date-partitioned, agent-segregated layout means time-windowed queries collapse to glob patterns: `_activity/2026/04/*/*.jsonl` for "this month so far across all agents," `_activity/2026/04/26/*.jsonl` for "today across all agents," `_activity/2026/04/26/glp1-research.jsonl` for "today, this agent." The granularity is daily; deployments needing finer can configure hourly partitioning, but daily is the default because it composes well with retention policies and human inspection. Total order across agents is reconstructed by sorting on `ts`. Entries are bounded at 4KB per line; large content artifacts are referenced by version rather than embedded inline, which is what keeps `O_APPEND` atomicity guarantees honest on POSIX (§5) and keeps the log greppable.

**Two readers, one writer:**

- **The agent** reads it like any other file. `glob` for time windows, `grep --format=json` for filtering, `read_file` for a full day. This is the substrate self-evolution runs on. There is no separate query tool because none is needed.
- **Operators** tail the same files with whatever tooling they like — `tail -f`, `aws s3 sync` to a local copy, Vector or Filebeat shipping to Splunk or Datadog. The format is plain JSONL; standard tools work without any Mycelium-specific configuration.
- **The system itself** is the only writer. The protocol rejects agent writes under any `_`-prefixed root path — see §4. Writes happen as a side effect of every successful mutation, ordered consistently with the mutations themselves.

### Native git/jj support

A LocalFS backend can be initialized inside a git or jj repository, and every operation can optionally produce a commit. This gives the user `git log`, `git diff`, `git blame` over the agent's memory for free — and makes the store a normal artifact in any version-controlled workflow. The activity log itself is committed alongside content, which means git history and the activity log are two views of the same truth: the activity log is faster to grep, and git history is richer for content diffs.

### Export

Export is `tar` (or `aws s3 sync`, or `cp -r`). There is no proprietary export format because there is no proprietary storage format. A directory of UTF-8 files is the export format. The activity log comes along automatically because it's just files in the directory.

### Diff and share

Two agents' stores can be diffed with `diff -r`. A team can share a knowledge directory by handing each other a tarball or a read-only S3 prefix. There is no impedance mismatch between "agent memory" and "files an engineer would email a colleague."

---

## 9. Anti-Goals

Each of the following is a feature commonly found in competing systems. Each is rejected here, by name, with reasoning.

### Automatic memory extraction at session end

**Where it appears:** mem0, parts of LangChain, frameworks like AutoGen and CrewAI that ship "memory modules."

**Why we reject it:** Extraction encodes a policy about what is salient. That policy was hand-tuned for a particular model class at a particular point in time. As models improve, the extractor becomes a lossy filter sitting between the agent and what the agent actually wanted to keep. The agent already has the tools to write what it wants to remember at the moment it knows it wants to remember it. Extraction at session end is solving the wrong problem.

### Vector retrieval over memory files

**Where it appears:** LangChain `VectorStoreRetrieverMemory`, most "RAG-as-memory" architectures.

**Why we reject it:** Embeddings drift on edit. They are not human-readable, not diffable, not portable across embedding models without re-indexing. They introduce a retrieval ranking the agent does not control and cannot reason about. They train users to think of memory as "a search engine over what the agent said," which is precisely the wrong frame: memory is a workspace, not a search corpus. `grep` and `glob` are *surprisingly good* on a well-organized directory, and they get better as models get better at organizing directories.

We are not against vector retrieval as a tool the agent might choose to invoke. If the agent wants similarity search over a knowledge base, it can call out to a vector store as a separate tool. We are against vector retrieval as the *primary access path to the agent's own memory*.

### MemGPT-style hierarchical memory tiers

**Where it appears:** MemGPT (now Letta), several derivative projects.

**Why we reject it:** Tiered memory makes the framework, not the agent, responsible for deciding what is in working memory vs. external memory vs. archival memory. The framework's tiering policy was a fit for a particular context window and a particular model's ability to manage attention. It is not the right policy for next year's models, and it is not adjustable by the agent in the moment. A flat directory the agent navigates by intent (read what you need, when you need it) collapses the tiering question into a tooling question — which the agent is already equipped to answer.

### Automatic summarization of long files or sessions

**Where it appears:** LangChain `ConversationSummaryMemory` and friends, most "long-running agent" frameworks.

**Why we reject it:** Same critique as extraction. Summarization is a compression policy. The system does not know what the agent's downstream task will require. The agent does. If the agent wants a summary, it writes one (using its own model, at its own discretion). If a file gets too long, the agent decides whether to split, condense, or leave it alone.

### Zep-style temporal knowledge graphs with auto-extraction

**Where it appears:** Zep, Graphiti.

**Why we reject it:** Knowledge graphs encode an ontology. Ontologies were designed for a particular use case at a particular time. As models improve, they become better at extracting structure on demand from unstructured text — which means a flat corpus of plain text becomes more capable, not less, over time, while a fixed knowledge graph becomes more rigid. We don't ship one.

### Tiered summary layers maintained by the framework

**Where it appears:** Most "agent memory" SaaS offerings, several open-source frameworks that maintain rolling summaries.

**Why we reject it:** Same root cause. Maintenance happens behind the agent's back, with policies the agent didn't set. The agent may rely on summaries that have drifted from the underlying files. The user may discover the system has been silently rewriting their notes. If summaries are wanted, the agent writes and maintains them as ordinary files.

### Embedding-based deduplication

**Where it appears:** Various memory frameworks that auto-deduplicate "similar" memories.

**Why we reject it:** Two near-duplicate notes may be intentional (different contexts, different angles). The system is not in a position to decide. The agent is.

### System-driven reflection or self-evolution

**Where it appears:** Some agent frameworks ship a "reflection" step that runs between turns, summarizing the agent's recent actions and feeding them back into context. Some maintain auto-updated "lessons learned" files on the agent's behalf.

**Why we reject it:** Reflection-as-a-feature is the same anti-pattern as auto-summarization, one level up. It encodes a policy about *when* and *how* the agent should review its own behavior. That policy is wrong for some models and unnecessary for others. The right answer is: give the agent the primitives to reflect (an activity log it can grep, editable convention files), let the agent's prompt determine when it does, and stay out of the loop. Self-evolution is an agent behavior, not a system feature — see §7.

### Specialized query languages or APIs for the activity log

**Where it appears:** Most observability systems ship a query language (LogQL, KQL, etc.) for their structured logs.

**Why we reject it:** The activity log is JSONL and the agent already has `grep --format=json`. Inventing a query DSL would add a tool, a parser, and a documentation surface to do work the existing primitive already does. JSONL plus ripgrep is the smallest thing that works.

---

## 10. Open Questions

The following are unresolved or deliberately deferred. Each is flagged because it might tempt a future maintainer to violate the design principles in §2 — and the answer in each case has to be worked out without doing that.

**Binary blobs.** The agent may want to save images, PDFs, audio. The tool surface as specified is text-oriented. Options: a separate `read_blob` / `write_blob` pair (probably the right call), or a base64-content convention on `write_file` (probably worse). How does an agent reason about a file it can't natively read? Likely answer: it reasons about the path, the filename, and a sibling text file with notes, exactly as a human would.

**Garbage collection of content files.** Stores will grow unbounded. Who decides what to prune? Per the design principles: **the agent**, when prompted to. This is unsatisfying in long-running deployments where no one is prompting the agent to clean up. We may need a documented "housekeeping prompt" that operators run periodically — but it must be a prompt to the agent, not a job that runs against the store.

**Activity log retention.** Now sharper because the log is visible files: `_activity/2024/*.jsonl` will sit there for years if nothing trims them. Options: (1) configurable retention with the system trimming old daily files (system writes — fine, since system already owns the path); (2) operator's problem, document the convention and let them ship logs out; (3) hybrid — system trims at a generous default, ops can override. Leaning toward (3). Whatever we choose, the agent needs to know the horizon — a `_activity/RETENTION.md` file at the root of the activity directory, written by the system, stating the policy and oldest available date, would let the agent reason about what it can and can't query.

**Token budget enforcement.** A model can `read_file` a 10 MB file and blow its context window. The tool surface exposes `offset` and `limit`, but the agent has to know to use them. `grep` already caps result counts via its mandatory `limit` (§4); should `read_file` similarly enforce a max-bytes-per-read with an explicit override? Probably yes — the asymmetry is awkward, and the failure mode (context overflow on a single read) is sharper than the case for `grep`.

**Cross-store federation.** Should an agent be able to mount multiple stores at different paths (e.g., `/team/` from a shared bucket, `/me/` from local)? Yes, almost certainly. The implementation is straightforward (the layered backend pattern handles it). The open question is how mount configuration is expressed without becoming complicated. Each mount has its own `_activity/`, which means cross-mount activity queries work via glob across mounts — natural, no new abstractions.

**Symlinks.** Almost certainly: refuse them. They are an attack surface, they break portability across backends, and the agent gets little from them that move/rename doesn't already provide.

**Activity log format versioning.** The first line of each daily file (or a sidecar `_activity/SCHEMA.md`) should declare the format version. We need this before downstream tooling — or the agent — builds rigid expectations against the current shape.

**Conflict resolution UX.** When a conditional write fails, what exactly does the agent get back? The current version token is required. The current content is *useful* but expensive on large files. Probably: return current version unconditionally, return current content opt-in via a flag, and let the agent decide whether to re-fetch.

**Backwards self-evolution and convention recovery.** At MVP the activity log records version hashes but not content, so an agent that destructively rewrites its own `MYCELIUM_MEMORY.md` (or any convention file) cannot recover the prior version through Mycelium primitives alone. The agent can detect *that* a write happened, *when*, and *by whom* — but cannot read the prior content. Phase 3's historical reads (`read_file(path, version=...)` on backends with version history — git/jj-backed LocalFS, versioned S3 buckets) close the gap. Until then, deployments wanting a backstop for backwards-self-evolution can opt into a git-backed LocalFS variant out of band; the design's stance remains that an agent that mangles its own conventions is a model-quality problem, not a system problem, and the right pressure-release is a stronger model. Documenting the gap is what keeps that stance honest at MVP.

**Default starter content.** Ship `MYCELIUM_MEMORY.md` populated, or leave the directory truly empty and let users opt in by copying from a templates repo? Leaning toward: ship empty (`_activity/` initializes lazily on first write), document the templates, make `mycelium init --template=default` a one-line opt-in. Empty stores keep the system honest about its principles.

**Read-only knowledge sharing.** The layered backend supports it; the protocol surfaces it. But the UX — how a user expresses "this prefix is read-only, this one is read-write" — needs design. Probably a mount manifest file. Possibly just per-mount config. Not a hard problem; just unspecified.

**External activity log sink for backend-level isolation.** The current design co-locates the activity log with content under `_activity/` in the same backend. This is a deliberate simplification: one storage system, one format, agent can read its own history through general tools, operators tail the same files. The trade-off is that a backend-level failure (S3 bucket corruption, accidental prefix deletion, regional outage) takes content and audit history down together. The original design had these in separate storage systems precisely to preserve audit history through backend-level incidents. We've deferred this for MVP because: no concrete deployment yet requires it; standard S3 practices (versioning, replication, separate prefix policies on `_activity/`) cover most of the gap; and the simpler design is easier to validate the central bet against. If a real high-assurance deployment surfaces a need for backend-level isolation, the path forward is optional log mirroring: the system continues to write to `_activity/` *and* tees to an external sink (file, append-only object store in a separate account, syslog endpoint, etc.). The agent's read path is unchanged; only operators see the mirrored copy. This adds about a hundred lines of code and a small amount of mount configuration. Not built now; documented so the option isn't lost.

---

*End of draft. Feedback welcome — particularly on §4 (the simplification pass and the one principled enforcement exception), §6 (concurrency primitives), and the activity-log retention question in §10.*
