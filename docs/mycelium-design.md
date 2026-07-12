# Mycelium: Pi Agent Memory System

**Status:** Design draft
**Project:** Mycelium

A persistent memory substrate for pi coding agents, on the bet that **a real filesystem driven by general file tools outperforms specialized memory infrastructure as models improve**. Mycelium is model-flexible inside pi, but pi is the sole supported coding-agent harness.

---

## 1. Goals and Non-Goals

### Goals

- Give the agent durable memory by mounting a local directory and exposing file operations through a small CLI.
- Let the agent own the schema. The agent decides what to save, how to name it, and how to organize it.
- Keep memory human-interpretable end to end — exportable as a directory, diffable as text, shareable as a tarball.
- Support multiple agents concurrently using the same local POSIX store with predictable conflict semantics.
- Support pi coding agents through the `pi-mycelium` extension, which invokes the bundled `mycelium` binary through pi's shell tool.
- Stay filesystem-native: LocalFS, `flock`, atomic rename, `O_APPEND`, and `fsync` are the guarantee set.
- Stay useful on Frontier models without compensating logic that becomes a ceiling on the next generation.
- Equip the agent to **observe and revise its own memory practices over time** — using the same general file tools and a conventions file.

### Non-Goals

- A memory API. There is no `remember(fact)` or `recall(query)`. Memory is a directory; the operations are file-shaped CLI commands.
- A backend-agnostic storage layer. Mycelium's v1 contract is a local POSIX filesystem contract.
- Automatic extraction, summarization, or compression of agent output.
- Embedding-based retrieval as the primary access path.
- Tiered memory (working / episodic / archival) maintained by infrastructure rather than by the agent.
- Schema enforcement on agent-authored content.
- System-driven reflection or self-evolution. The system _enables_ it; the agent _does_ it.
- Compensating for limited model judgment with infrastructure features. If the model is making a mess, the answer is a stronger model or a better prompt.

### Target model tier

Mycelium targets **Frontier-class production models running in pi** — top-of-class capability defined by the behaviors the rest of this design assumes: self-organizing a filesystem store from empty, reflecting on its own activity log, revising its own conventions without operator scaffolding (Claude Opus 4.7, GPT-5.5, and equivalent successors). Models below Frontier are out of scope; baking in compensating logic for tiers below the floor would mask, with infrastructure heuristics, exactly the self-evolution behavior this design exists to capture.

---

## 2. Design Principles

### As simple as possible, but not simpler

> "Everything should be made as simple as possible, but not simpler." — Einstein (apocryphal)

This is the project's first principle and the one every other principle answers to. Every tool, parameter, on-disk structure, and guarantee earns its complexity against "could this be expressed in terms of something already here?" — and is removed when the answer turns out to be yes. When in doubt, fewer.

The "but not simpler" half is load-bearing. A system that fakes simplicity by hiding necessary complexity behind opaque abstractions, or that drops a guarantee the bet depends on, has failed this principle as surely as one that piles on unnecessary layers. Atomicity, conflict visibility, reserved-prefix protection, and durable activity logging cannot be removed without breaking the bet — they're as simple as the bet allows, and no simpler.

**Simplicity budget.** Any new command, flag, metadata path, persisted concept,
or prompt requirement must either remove an equal or larger concept elsewhere or
explicitly justify why the core bet fails without it. Features are grouped by
mental-model tier: everyday surface first, occasional operations second,
metadata/internals last. If a guarantee needs machinery, the machinery stays in
implementation docs and diagnostics rather than the default agent/user model.

### General tools scale with intelligence; specialized infrastructure caps it

A specialized memory API encodes assumptions: what gets saved, how it's indexed, what counts as "relevant," when to summarize. As models improve, those heuristics become drag — the system forces the agent into compression and ranking policies the model could now beat unaided.

General tools (read, write, list, edit, grep) have no such ceiling. The pi agent invokes them through pi's shell — `mycelium read` sits in the same Bash tool as `git log` and `rg` — and the `mycelium` CLI is the smallest engine that earns its keep: atomic conditional writes, an authoritative activity log, no policy about what to save, how to name, or what's relevant. A Frontier model uses them with judgment indistinguishable from a thoughtful engineer keeping a working notebook, and the same surface gets _more_ useful — not less — as the next generation arrives. This is the central bet, and every other decision is downstream of it.

### Files are the unit. Directories are the structure. The agent owns both

The filesystem is the agent's workspace, not a managed resource. The system does not move files behind the agent's back, deduplicate them, prune them, or rewrite them. If the agent creates `notes/2026-04-26/scratch.md`, that file stays exactly where the agent put it until the agent changes it through `mycelium`.

### Human-interpretable wins

Every byte stored is plain content the user can open, read, diff, copy, and export. No opaque embedding index, no proprietary serialization, no metadata sidecar authoritative over the visible file. If a person can't read and reason about the store, the agent will eventually mis-edit it — and the user will have no way to recover.

There is one operational boundary: **raw filesystem reads are allowed; raw filesystem writes are not. All live-store mutations go through `mycelium`.** Operators can inspect with `cat`, `ls`, `rg`, editors, and tarballs. But raw writes, edits, deletes, and renames inside the mounted store are unsupported because they bypass CAS and the authoritative activity log. If a human wants to change the live store, they use the same `mycelium write` / `edit` / `rm` / `mv` surface the agent uses.

### Hints over enforcement; conventions over schemas

System opinions about how the agent should organize memory live as **starter files inside the store** — not system features, not enforced layouts, not middleware that rewrites calls. Hints are removable. Schemas are sticky.

One principled exception (sections 4, 5, and 8): the `_` prefix at the store root is reserved for system paths, and `mycelium` rejects agent mutations under it. The activity log, lock file, and any future system metadata need a collision-free namespace. Reserving the prefix rather than just the current paths prevents future namespace collisions.

### Concurrency is a property of the store

Multi-agent semantics are implemented with local filesystem primitives (`flock`, atomic rename, content-hash versions) and surfaced honestly through `mycelium`'s exit codes and structured stderr. The model decides how to resolve conflicts; the system makes sure no write is silently lost when conditional writes are used.

The lock is an implementation detail, not the coordination model exposed to agents. Agents coordinate through version tokens, conflict envelopes, re-read/merge/retry, and conventions they choose to write down.

### Observability instead of intervention

The system records what the agent did. It does not act on what the agent did. The activity log is plain JSONL at a reserved path; the agent reads it for self-introspection, operators tail it for monitoring, nothing about it feeds back into the agent's loop automatically. The agent decides when to look.

---

## 3. Architecture Overview

Public model first: **a folder + safe mutations + a searchable activity log**.
Agents and operators should not need to think about CAS, `flock`, fsync, or the
durability boundary during normal use. Those details exist to make "safe
mutations" true.

Internally there are two layers, with system metadata inside the store under
reserved `_` paths:

```
  Frontier-class pi agent
       │
       │  pi-mycelium injects prompt guidance, env, and lifecycle logging
       │  agent invokes `mycelium <sub>` via pi's shell tool
       │
       │  raw reads are okay: cat, ls, rg, editor, tar
       │  raw writes are unsupported: all mutations go through mycelium
       ▼
  `mycelium` binary
       read · write · edit · ls · grep · rm · mv · log
       — enforces _-prefix reservation for agent mutations
       — provides CAS and structured conflict errors
       — writes authoritative activity entries
       ▼
  Local POSIX filesystem store
       agent-authored files and directories
       _activity/YYYY/MM/DD/{agent_id}.jsonl
```

Agents and operators share one readable surface — the filesystem. The binary is a thin shim over LocalFS for mutations and system metadata; it never interprets agent-authored content.

---

## 4. CLI Surface

A single binary, `mycelium`, invoked through the agent's shell. Eight visible subcommands: seven file/navigation operations plus one metadata operation.

### File and navigation operations

- **`mycelium read <path> [--format text|json]`** — print file contents. `text` is the default and emits raw bytes. `json` requires UTF-8 content and emits a CAS-safe envelope containing content and version from the same read:

  ```json
  { "path": "notes/foo.md", "version": "sha256:...", "content": "..." }
  ```

  Raw `cat` is fine for inspection; `read --format json` is the normal pre-read before a conditional mutation.

- **`mycelium write <path> [--expected-version SHA] [--rationale STR]`** — create or overwrite from stdin. With `--expected-version`, conditional on the current version; otherwise unconditional. Prints the new version on success. `--rationale` is captured as a top-level field on the activity log entry and in the conflict envelope on CAS failure.
- **`mycelium edit <path> --old STR --new STR [--expected-version SHA] [--rationale STR]`** — find-and-replace a unique substring. Fails if `--old` is absent or non-unique. _Earns its complexity:_ token economy on large files, diff quality under git/jj, and the unique-substring constraint catches stale-view errors a full overwrite would silently paper over.
- **`mycelium ls [pattern] [--recursive]`** — list paths under the mount, optionally filtered by a glob pattern. Without `--recursive`, only top-level files are listed or matched. _Earns its complexity:_ one survey/path-discovery verb covers both browsing and pattern matching.
- **`mycelium grep --pattern STR [--path PATH] [--regex] [--format text|json] [--limit N]`** — print matching lines with paths and line numbers. `--format=text` is `path:line:text`; `--format=json` returns `{matches: [{path, line, text}, ...], truncated}`. `--limit` caps results (default 1000, hard ceiling). Implementation is pure Go: one search path, one regex dialect, deterministic behavior across machines. _Earns its complexity:_ JSON output makes the activity log usable through general tools; the `--limit` cap keeps log-reflection from overflowing context.
- **`mycelium rm <path> [--expected-version SHA] [--rationale STR]`** — remove. _Earns its complexity:_ not expressible as `write` — empty content creates an empty file, not a deletion.
- **`mycelium mv <src> <dst> [--expected-version SHA] [--rationale STR]`** — atomic rename within the store. `--expected-version`, when supplied, checks the source version. The destination must not exist; destination collisions return a structured conflict. _Earns its complexity:_ read+write+delete is not atomic; emulating rename loses the guarantee.

### Metadata operations

- **`mycelium log <op> [--path PATH] [--payload-json STR | --stdin] [--rationale STR]`** — append a non-mutation signal entry to `_activity/YYYY/MM/DD/{agent_id}.jsonl`. The system fills `ts`, `agent_id`, `session_id`; the caller supplies `op` (a non-mutation tag like `context_checkpoint`, `compaction`, or an agent annotation), an optional `--path`, an optional JSON payload, and an optional `--rationale`. Silent on success.

**Failure modes:**

- exit 0 — success
- exit 1 — generic error (path not found, malformed args, log append failure after content commit, legacy pending transaction)
- exit 2 — usage error (bad flags, malformed args, invalid regex, invalid output format)
- exit 64 — CAS conflict; stderr is JSON `{"error":"conflict","op":"write","path":"...","current_version":"sha256:...","expected_version":"sha256:..."}`.
- exit 65 — protocol violation such as a reserved `_` path or oversize rationale.

A successful `write` or `edit` prints `{"version":"sha256:..."}` on stdout. `read --format json` prints a read envelope. `rm`, `mv`, and `log` are silent on success.

What's _not_ here:

- **No specialized content query DSL.** Reading the activity log uses the same tools as reading any file: `mycelium read` (or `cat`), `mycelium ls [pattern] --recursive` (or `ls`) for time windows, `mycelium grep --format=json` (or `rg --json`) for filtering.
- **No `summarize`, `index`, `embed`, `tag`, `pin`, `archive`, `recall`.** If the agent wants any of those, it implements them by writing files and may record the standing convention in `MYCELIUM_MEMORY.md`.
- **No `exists` subcommand.** `mycelium read` exits non-zero with a typed not-found message.

Five contract notes:

**Raw reads are allowed; raw writes are not.** `cat`, `ls`, `rg`, editors, `tar`, and `cp -r` are fine for inspection/export. Mutations inside the live mount go through `mycelium` so CAS and `_activity/` remain authoritative.

**Conditional writes are first-class.** Every content-mutating subcommand (`write`, `edit`, `rm`, `mv`) accepts optional `--expected-version`. Version tokens are opaque strings to the caller; the LocalFS implementation uses content hashes (`sha256:...`) and the sentinel `sha256:absent` for a missing file. A single-agent store can ignore CAS and the system behaves like a regular filesystem with an audit log.

**Operational rationale is optional on every mutation and signal.** `write`, `edit`, `rm`, `mv`, and `log` all accept optional `--rationale "..."`. When supplied, rationale appears as `rationale` on the activity log entry and on the conflict envelope. Maximum 64 KiB; oversize input is rejected with exit 65 before the mutation runs.

**The `_` prefix is reserved at the store root.** `mycelium` rejects `write`, `edit`, `rm`, and `mv` whose target/source is under any path beginning with `_`. Currently `_activity/` and `_lock` are system-owned; legacy `_tx/` records are detected for compatibility. Future system paths inherit the same protection without code changes.

**Identity travels via environment.** The harness sets `MYCELIUM_MOUNT` (the store directory) once. `MYCELIUM_AGENT_ID` is optional and defaults to `agent`; `MYCELIUM_SESSION_ID` is optional and auto-generated per CLI process when absent. The pi extension provides stable agent/session ids for clearer multi-agent timelines. Every invocation reads identity from the environment/defaults; every log entry records the agent and session. Standard Unix request identity.

---

## 5. LocalFS Storage Contract

Mycelium is filesystem-native. The supported storage contract is a local POSIX filesystem with working:

- atomic rename within a directory,
- `flock` for mount-level mutation serialization,
- `O_APPEND` for JSONL activity appends,
- `fsync` on files and directories for durability boundaries,
- normal path and permission semantics.

NFS, SMB, FUSE, and non-POSIX synchronization layers are outside the v1 guarantee set unless they faithfully provide those semantics.

Implementation shape:

- Versions are content hashes. `read --format json` and conflict envelopes surface the current version.
- Conditional mutations acquire the mount lock, check the expected version against current bytes, perform the file operation atomically, then append the activity entry durably.
- `write` uses write-to-temp, fsync, rename, and directory fsync.
- `edit` reads, validates unique substring replacement, then uses the same atomic write path.
- `rm` captures the prior version before deletion.
- `mv` refuses destination collisions and uses atomic rename.
- `grep` is implemented in Go, using the same scanner and regex engine on every machine.

There is no backend interface in the v1 design. Any future storage change would need to re-prove every guarantee above rather than share this contract by assertion.

### Durable activity boundary

The activity log is authoritative, so Mycelium cannot treat log append failure as a warning after content has changed. But a local filesystem still cannot atomically mutate an arbitrary content file and append a different JSONL file as one hardware transaction. The current contract makes that boundary explicit rather than hiding it behind a recovery journal.

Every content mutation follows this sequence:

1. Acquire the mount lock.
2. Refuse to proceed if legacy `_tx/pending/*.json` records exist.
3. Compute the operation's precondition and postcondition (`prior_version`, `version`, paths).
4. Check `--expected-version`, when supplied.
5. Apply the content mutation atomically and fsync affected files/directories.
6. Append and fsync the activity entry.
7. Return success.

Failure examples:

**Failure before content changed.** The target file remains at its prior version. No activity entry is written, and the command exits non-zero.

**Power loss after content changed but before log append.** The target file may contain the final mutation without a matching `_activity/` entry. This is the bounded power-loss gap in the durable-history contract.

**Log append fails after content changed.** The command exits non-zero with a message that the log entry write failed after content commit. The content mutation remains committed. No `_tx/` record is created, so there is no automatic replay path.

**Legacy `_tx` records exist.** Current binaries do not replay the old transaction journal. If `_tx/pending/*.json` exists, mutating commands fail before content changes with instructions to run the last v0.2 binary on the mount to recover pending records, then retry.

Legacy recovery procedure:

1. Run the last v0.2 `mycelium` binary with `MYCELIUM_MOUNT` pointed at the affected mount.
2. Execute a harmless log append, for example: `mycelium log legacy_tx_recovery --payload-json '{"from":"legacy-tx-recovery"}'`. The v0.2 binary runs pending transaction recovery before appending the log entry.
3. If the command exits 0, verify `_tx/pending/` has no `*.json` files, then return to the current binary.
4. If v0.2 reports an unrecoverable pending record, inspect the named JSON file and content path manually before deleting or archiving the pending record.

`mycelium log` has no content mutation: its only state change is the activity append itself. It still acquires the mount lock, checks for legacy pending records, and returns success only after the activity entry is durable.

---

## 6. Concurrency and Multi-Agent Semantics

Multiple agents may mount the same local store simultaneously. Guarantees:

1. **No silent loss.** Concurrent writes to the same path don't silently overwrite each other when conditional writes are used. Unconditional writes are documented as last-writer-wins.
2. **Visible conflicts.** A failed conditional write returns a typed error with the current version; the agent re-reads, merges, retries.
3. **Atomic single-file ops.** `write`, `edit`, `rm`, `mv` either fully apply or fail. No half-written files, no partial renames.
4. **Authoritative mutation log.** Every successful mutation has a durable activity entry. If content commits but logging fails, the command exits non-zero and reports that the post-commit append failed.
5. **Per-agent log files.** Each agent writes its daily activity stream at `_activity/YYYY/MM/DD/{agent_id}.jsonl`. Cross-agent order can be reconstructed by sorting on `ts` or `tx_id`.

No user-facing multi-file transactions. If the agent wants atomicity across several agent-authored files, it composes it from single-file operations — typically a composite file or an agent-chosen journaling pattern.

Locks are not exposed as the primary coordination mechanism. They introduce timeouts, deadlocks, and lifecycle questions at the agent layer. CAS via versioned writes degrades cleanly: a conflict is just an error to read, reason about, and handle.

### Conflict recovery convention

The conflict recovery pattern is **re-read, merge, retry**:

1. Re-read the path with `mycelium read <path> --format json` so content and
   version come from the same observation.
2. Merge the intended change with the current state.
3. Retry with the fresh `version` token, or deliberately omit
   `--expected-version` only when last-writer-wins is acceptable.

Operation-specific guidance:

- `write`: apply the intended addition or rewrite against current bytes rather
  than overwriting the stale view.
- `edit`: re-locate the substring; if it is gone, decide whether the change is
  still semantically correct.
- `rm`: re-read before deleting when deletion is based on observed content.
- `mv destination_exists`: read the destination first; choosing a new path,
  deleting the destination, or aborting is a content decision.

Stop instead of looping when repeated conflicts show another active writer, the
current content contradicts the intended change, or a destination collision has
valuable content. Agent-facing quick guidance lives in the `pi-mycelium` system prompt and starter `MYCELIUM_MEMORY.md` template.

**Identity** is set once by the harness via the env vars in section 4 and recorded on every log entry. By default it isn't used for access control — every mounted agent has equal permissions and the same view of the log.

---

## 7. Self-Evolution

A Frontier agent doesn't just use general file tools well — it _reflects on its own use of them and revises its approach_. Given an empty pi journal, it self-organizes: extracts durable lessons, archives stale notes, names things consistently, deletes on purpose. Over sessions it edits its own convention files, builds indexes when patterns emerge, consolidates when the activity log shows duplication. This is the central observable behavior the supported tier is defined by — and the property the system has to avoid breaking.

The system _enables_ this with primitives the agent already has, and is careful not to _do_ it on the agent's behalf:

> **Scaffolding lives in prompts and conventions — mutable, optional, removable. It never lives as automatic content-retrieval or organization policy in the binary.**

### Concrete applications

**Starter conventions are files inside the store, not code paths.** A new mount can optionally be initialized with `MYCELIUM_MEMORY.md` at the root proposing a default layout (`agents/{agent_id}/`, `memories/`, `shared/`, `learnings/`, `INDEX.md`). The template tells the agent it owns the file and should revise it proactively as better conventions emerge.

**No automatic injection of retrieved memory content into agent context.** The pi extension surfaces minimal mount metadata and the exact conventions-file path at session start. It should not prefetch, rank, summarize, or inject arbitrary memory content.

**No automatic intervention.** The system never summarizes, dedupes, organizes, prunes, or rewrites the agent's files. If the activity log shows behavior the operator dislikes, the lever is the prompt or the model — not a system feature.

**Every piece of scaffolding is removable.** Starter `MYCELIUM_MEMORY.md`, layout conventions, anything else can be deleted or ignored without breaking the runtime. If removing it breaks the system, it doesn't belong.

### How self-evolution is enabled

Three primitives, all from section 4:

1. **Behavioral awareness via the activity log.** Mutations and explicit signals are JSONL at `_activity/YYYY/MM/DD/{agent_id}.jsonl`. Time windows scope with `mycelium ls [pattern] --recursive`; filter with `mycelium grep --format=json` or raw `rg --json`. Patterns obvious in retrospect — duplicate creation, abandoned naming schemes, conventions edited but unfollowed, repeated writes into stale regions — become visible without a specialized memory API. Reads are not logged, so the log does not claim to know what the agent looked at.
2. **State awareness and modification via standard file tools.** Self-evolution adds no content mutation verbs; it gives the agent reasons to use existing ones differently.
3. **Conventions as files.** Current rules live in editable text (`MYCELIUM_MEMORY.md`, `INDEX.md`, an agent-written `ARCHIVE_POLICY.md`). Changing the file is the supersession mechanism. The activity log records each edit and its rationale.

Patterns that emerge — convention bootstrap, convention revision, self-built indexes, archiving and pruning — are documented for agents in the pi-owned starter `MYCELIUM_MEMORY.md` template.

What the system does _not_ do: run a reflection step between turns; analyze patterns or detect drift for the agent; maintain or update convention files on the agent's behalf; enforce that conventions are read before acting. Doing any of these would re-introduce the capability coupling this principle exists to reject. The system makes self-evolution _possible_; the agent _does_ it.

### Conventions as the active state

`MYCELIUM_MEMORY.md` is the current rule set. It may point to richer files
such as `INDEX.md`, `ARCHIVE_POLICY.md`, or `learnings/`, but a fresh session
can start by reading one known file. Agents should edit it when a durable rule,
lesson, useful index, archive policy, or open question emerges. `--rationale`
captures why the rule changed; the file contents capture what rule is now in
effect.

Historical `op:"evolve"` activity entries remain valid old log lines. They are
tolerated as history, not treated as the active source of truth. ADR-0004
records the decision to remove the functional command and make conventions
files authoritative.

---

## 8. Observability and Export

Observability is plain JSONL files at a reserved path. No sidecar service, no audit-only API, no operator-vs-agent split. One source, two readers, one writer.

### The activity log

Two paths produce entries in `_activity/YYYY/MM/DD/{agent_id}.jsonl`: every content-mutating subcommand (`write`, `edit`, `rm`, `mv`) appends automatically on commit, and `mycelium log` appends explicit signal entries. Reads (`read`, `ls`, `grep`) aren't logged; the log records what changed and what was observed, not what was looked at.

The activity log is authoritative for state-changing Mycelium operations: no mutating command returns success unless its activity entry is durable. If content changes but logging cannot complete, the command fails loudly after the content commit. A power loss in that same narrow window can leave the final content mutation unlogged; that bounded gap is part of the durable-history contract.

A mutation entry (`write`):

```json
{
  "ts": "2026-04-26T18:42:11.034Z",
  "agent_id": "researcher-7",
  "session_id": "sess-9b2f",
  "tx_id": "tx-1782468000000000000-4f8d2c1a9b0e7d33",
  "op": "write",
  "path": "learnings/glp1-pipeline.md",
  "prior_version": "sha256:abc...",
  "version": "sha256:8c4d...",
  "rationale": "Recording initial synthesis before the literature window closes."
}
```

`rationale` is an optional top-level field (`omitempty`) on every mutation and `log` entry. When the caller passes `--rationale "..."` to `write`, `edit`, `rm`, `mv`, or `log`, the text is stored here — maximum 64 KiB, enforced before the mutation runs (exit 65 on violation). When absent, the field is omitted; existing log readers and fixtures remain valid without migration.

On a CAS or destination-exists conflict, the `rationale` field also appears on the conflict envelope emitted to stderr alongside `current_version`:

```json
{
  "error": "conflict",
  "op": "write",
  "path": "notes/foo.md",
  "current_version": "sha256:def...",
  "expected_version": "sha256:abc...",
  "rationale": "Adding GLP-1 cardio section — hypothesis confirmed across 4 studies."
}
```

This lets the retrying agent merge intent rather than just bytes when both sides supplied rationale.

A move entry:

```json
{
  "ts": "2026-04-26T18:45:00.000Z",
  "agent_id": "researcher-7",
  "session_id": "sess-9b2f",
  "tx_id": "tx-1782468300000000000-b03a8e4574c9f211",
  "op": "mv",
  "from": "notes/old.md",
  "path": "archive/old.md",
  "version": "sha256:8c4d..."
}
```

A `mycelium log` entry with an inline payload:

```json
{
  "ts": "2026-04-26T18:43:02.117Z",
  "agent_id": "researcher-7",
  "session_id": "sess-9b2f",
  "op": "decision",
  "payload": {
    "chosen": "redis",
    "rejected": ["memcached"]
  }
}
```

Pi lifecycle events and compatibility expectations are documented in [pi activity events](pi-activity-events.md). The binary does not enforce a closed set of `log` operation names.

**Path layout: `_activity/YYYY/MM/DD/{agent_id}.jsonl`.** Each agent writes its own daily file; `agent_id` must be filename-safe ASCII using letters, digits, `.`, `_`, or `-`. Cross-agent order is reconstructed by sorting on `ts` or `tx_id`. Time-windowed queries use path patterns with `mycelium ls --recursive`: `_activity/2026/04/*/*.jsonl` (this month, all agents); `_activity/2026/04/26/*.jsonl` (today, all agents); `_activity/2026/04/26/glp1-research.jsonl` (today, one agent). Payloads from `mycelium log` are inlined on the entry; larger signals belong in a regular file referenced via `--path`.

**Same files, two readers, one mutation path:**

- **The agent** reads the log with `mycelium ls --recursive` / `mycelium grep --format=json` / `mycelium read` — or raw `ls` / `rg --json` / `cat`. This is the substrate self-evolution runs on.
- **Operators** tail the same files with standard tools — `tail -f`, log shippers, text editors, shell scripts. Plain JSONL; standard tools work without Mycelium-specific config.
- **Mutations** go through `mycelium`. The binary writes `_activity/`; callers do not write reserved `_` paths directly.

### Export

Export is `tar` or `cp -r`. No proprietary format; a directory of UTF-8 files plus JSONL logs is the export format. If a legacy store still has `_tx/pending/*.json`, recover it with the last v0.2 binary before treating the export as clean.

---

## 9. Anti-Goals

Frameworks in this space commonly ship features Mycelium deliberately omits: automatic memory extraction at session end (mem0), vector retrieval as the primary access path to memory, hierarchical tiered memory maintained by the framework (MemGPT/Letta), automatic summarization (`ConversationSummaryMemory` and friends), temporal knowledge graphs with auto-extraction (Zep, Graphiti), embedding-based deduplication of "similar" memories, system-driven reflection between turns, and specialized query DSLs over agent-authored content. Each encodes a salience, structure, or compression policy that ages out as models improve — wrong for some, unnecessary for others, never adjustable in the moment. The principle is the same in every case: the agent owns those decisions; the system gives it the primitives (read/write/edit, grep, an activity log it can re-read) and stays out of the loop.

Two clarifications worth naming. Vector retrieval against an _external_ knowledge base is a tool the agent might choose to invoke; we reject it only as the primary access path to the agent's _own_ memory. Specialized agent protocols (custom REST, MCP servers, framework-specific plugin contracts) are rejected as the _primary_ surface — `mycelium read foo.md` and `cat foo.md` should produce the same bytes against the same files; an "agent surface" distinct from the "operator surface" reintroduces exactly the human-uninterpretable opacity Section 2's "human-interpretable wins" rules out. The binary and LocalFS store remain the implementation contract for the supported pi extension and for diagnostics.
