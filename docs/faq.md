# Mycelium FAQ

> Quick answers for people considering, integrating, or operating mycelium. If you're an AI agent reading mycelium's docs, see [agent-faq.md](agent-faq.md) for the operational reference.

## Table of contents

- [What it is and isn't](#what-it-is-and-isnt)
  - [What is mycelium and what problem does it solve?](#what-is-mycelium-and-what-problem-does-it-solve)
  - [Who is this for?](#who-is-this-for)
  - [How does this differ from a vector store, RAG service, or memory MCP server?](#how-does-this-differ-from-a-vector-store-rag-service-or-memory-mcp-server)
  - [Does it lock me into a particular model or harness?](#does-it-lock-me-into-a-particular-model-or-harness)
- [Trust, safety, and the threat model](#trust-safety-and-the-threat-model)
  - [Can a misbehaving agent escape the mount or trash my filesystem?](#can-a-misbehaving-agent-escape-the-mount-or-trash-my-filesystem)
  - [What happens to a write if mycelium gets killed or the machine crashes?](#what-happens-to-a-write-if-mycelium-gets-killed-or-the-machine-crashes)
  - [Two agents writing the same file at the same instant — what happens?](#two-agents-writing-the-same-file-at-the-same-instant--what-happens)
  - [Will it work on iCloud / Dropbox / NFS / Windows?](#will-it-work-on-icloud--dropbox--nfs--windows)
- [Install and integration](#install-and-integration)
  - [How do I install it?](#how-do-i-install-it)
  - [Does it need a daemon, database, or network access?](#does-it-need-a-daemon-database-or-network-access)
  - [What does an agent harness have to configure?](#what-does-an-agent-harness-have-to-configure)
  - [Can I use it without pi.dev, or with a custom harness?](#can-i-use-it-without-pidev-or-with-a-custom-harness)
  - [Does it conflict with Claude Code, Cursor, or Codex built-in memory?](#does-it-conflict-with-claude-code-cursor-or-codex-built-in-memory)
- [Auditing and operations](#auditing-and-operations)
  - [How do I see what an agent has been doing in a mount?](#how-do-i-see-what-an-agent-has-been-doing-in-a-mount)
  - [What does mycelium record so a future reviewer can understand why a change happened?](#what-does-mycelium-record-so-a-future-reviewer-can-understand-why-a-change-happened)
  - [Should I commit a mount to git?](#should-i-commit-a-mount-to-git)
  - [How do I move a mount between machines?](#how-do-i-move-a-mount-between-machines)
  - [How big does the activity log get, and how do I prune it?](#how-big-does-the-activity-log-get-and-how-do-i-prune-it)
  - [How do I delete or roll back something an agent wrote?](#how-do-i-delete-or-roll-back-something-an-agent-wrote)
- [Status and maturity](#status-and-maturity)
  - [Is mycelium production-ready?](#is-mycelium-production-ready)
  - [Are benchmarks available yet?](#are-benchmarks-available-yet)
  - [What's on the roadmap?](#whats-on-the-roadmap)
- [Why these design choices](#why-these-design-choices)
  - [Why a directory of files instead of a database?](#why-a-directory-of-files-instead-of-a-database)
  - [Why JSONL for the activity log?](#why-jsonl-for-the-activity-log)

---

## What it is and isn't

### What is mycelium and what problem does it solve?

AI coding agents lose all context when a session ends. The common workarounds — stuffing everything into the system prompt, ad-hoc scratch files, vector stores — either don't survive across processes or hide context behind opaque retrieval that no human can inspect.

Mycelium is a CLI and on-disk format that gives agents a durable, inspectable store they own across sessions, processes, and concurrent runs. A mount is a plain directory plus a searchable `_activity/` log. The daily model is small: list or grep paths, read relevant files, safely write or edit notes, and inspect the log when you need history. The same files are readable with `cat`, searchable with `grep`, versionable with `git`, and shareable as a tarball — no special tooling required.

### Who is this for?

Teams running Frontier-class AI agents (Claude Opus, GPT-5.5, and their successors) on coding, research, or operations work where memory across sessions matters. It's a good fit if you want the agent's reasoning to persist in human-readable form and you want to audit what the agent actually did. It's probably not the right fit if you need embedding-based retrieval as the primary access path or if your agent runs on a harness with no shell tool.

### How does this differ from a vector store, RAG service, or memory MCP server?

The core differences are access model and observability. Mycelium is file-based, not embedding-based: the agent navigates with `ls`, `glob`, and `grep` and reads files directly — no opaque retrieval step, no relevance scoring, no index to maintain. Every byte is plain text you can read line by line; a vector store's internal state is not.

Because it's a directory, standard tools work on it without any mycelium-specific tooling: `tail -f` for the activity log, `rg` for search, `git diff` for changes. And it needs no daemon or network — just a local POSIX filesystem.

These approaches are complementary, not mutually exclusive. A team could pair mycelium with a vector store: mycelium holds the agent's working notes and decision history; the vector store handles retrieval over a larger external corpus.

### Does it lock me into a particular model or harness?

**No — mycelium is harness-neutral and does not lock you into a model or harness.** Any agent that can run shell commands and read files can use it. The `pi-mycelium` npm extension is one harness integration for pi.dev, but the core binary works from any shell. Configure `MYCELIUM_MOUNT` once; `MYCELIUM_AGENT_ID` defaults to `agent`, and `MYCELIUM_SESSION_ID` is generated per CLI process when absent. After that, every invocation is a plain CLI call. Portable activity events are documented conventions for observability signals, not schema enforced by the binary. See [portable activity events](portable-activity-events.md) and [ADR 0002](adr/0002-portable-activity-events-as-adapter-conventions.md).

---

## Trust, safety, and the threat model

### Can a misbehaving agent escape the mount or trash my filesystem?

**Mycelium does not sandbox the agent.** If an agent has shell access, it can touch files anywhere on the filesystem that its OS user permits — mycelium's `_` prefix reservation only prevents the agent from using `mycelium write` on reserved system paths inside the mount. The harness controls what commands the agent can run; mycelium's contract is integrity within the mount (atomic writes, conflict detection, durable log), not isolation from the broader system. Treat mount-level protection accordingly: scope the agent's shell permissions at the harness level, not at mycelium's level.

### What happens to a write if mycelium gets killed or the machine crashes?

Every content mutation is a small transaction:

1. Before changing a file, mycelium writes an internal pending record under the reserved `_` namespace.
2. The content change and activity log append are performed.
3. The pending record is removed — marking the transaction complete.
4. On restart after a crash, any remaining pending records are replayed or cleaned up before later mutations proceed.

The activity log is authoritative: a command returns success only after both the file and the log entry are durable on disk. See the [design doc](mycelium-design.md) section 5 for the full transaction protocol.

### Two agents writing the same file at the same instant — what happens?

One wins; the other gets a structured conflict rather than silent data loss. The winning write returns a new SHA-256 version token. The losing write — if it used `--expected-version` — exits with code 64 and a JSON envelope containing the winner's version token and, optionally, the winner's content. The losing agent re-reads, merges in memory, and retries with the fresh version. See [conflict resolution](conflict-resolution.md) for the re-read/merge/retry pattern and guidance on when unconditional writes are acceptable.

### Will it work on iCloud / Dropbox / NFS / Windows?

The honest answer is: probably not reliably, and the recommendation is not to try. Mycelium depends on `flock`, atomic rename within a directory, `O_APPEND`, and `fsync` — the POSIX local filesystem guarantee set. These are tested on macOS and Linux on local disks. iCloud, Dropbox, and OneDrive can mangle file locks and replicate `_*` directories unexpectedly. NFS has well-known flock semantics issues. Windows is untested. Keep the mount on a local POSIX disk.

---

## Install and integration

### How do I install it?

Two paths. Build from source (requires Go 1.26+):

```sh
git clone https://github.com/fuentesjr/mycelium.git
cd mycelium
make build
sudo install cmd/mycelium/mycelium /usr/local/bin/
```

Or via `go install ./cmd/mycelium`. For pi.dev, the `pi-mycelium` npm extension bundles the platform-matching binary and handles PATH setup automatically:

```sh
pi install npm:pi-mycelium        # global
pi install npm:pi-mycelium -l     # project-local
```

Full install instructions are in the [README](../README.md).

### Does it need a daemon, database, or network access?

**No — mycelium is a single stateless binary with no daemon, database, or network access.** Every invocation opens the mount directory, does its work, and exits. The entire system state is in a directory on your local disk.

### What does an agent harness have to configure?

One environment variable is required before the agent runs:

- `MYCELIUM_MOUNT` — absolute path to the mount directory (created if absent on first write).

Two identity variables are optional:

- `MYCELIUM_AGENT_ID` — stable identifier for the agent; defaults to `agent`.
- `MYCELIUM_SESSION_ID` — session grouping id; auto-generated per CLI process when absent.

That's it. Higher-fidelity observability (session boundaries, turn/tool events) is optional and documented in [portable activity events](portable-activity-events.md).

### Can I use it without pi.dev, or with a custom harness?

Yes. `pi-mycelium` is a convenience wrapper, not a requirement. Any agent harness that can run shell commands works at L0 (the baseline shell-command integration level): set `MYCELIUM_MOUNT` and let the agent invoke `mycelium` subcommands through its existing shell tool. Set `MYCELIUM_AGENT_ID` and `MYCELIUM_SESSION_ID` only when you need stable multi-agent or cross-process session grouping. A minimal L1 shell wrapper that emits `session_startup` and `session_shutdown` log entries is shown in [portable activity events](portable-activity-events.md).

### Does it conflict with Claude Code, Cursor, or Codex built-in memory?

No. They're orthogonal. Claude Code's `CLAUDE.md`, Cursor's rules, and Codex's session context are session-scoped prompt injections managed by those harnesses. Mycelium is a durable on-disk store the agent reads and writes. They coexist: the harness-level memory shapes how the agent behaves; mycelium preserves what the agent has written across sessions.

> **See also:** [README](../README.md) for full install details · [agent-faq.md](agent-faq.md) for the operational reference · [portable activity events spec](portable-activity-events.md) for advanced observability configuration.

---

## Auditing and operations

### How do I see what an agent has been doing in a mount?

The activity log is plain JSONL at `<mount>/_activity/YYYY/MM/DD/<agent_id>.jsonl`. Every write, edit, delete, move, and `evolve` event lands there automatically; the agent can also append manual signal entries with `mycelium log`. Read it with standard tools:

```sh
# All activity today, all agents
cat $MYCELIUM_MOUNT/_activity/2026/05/10/*.jsonl

# Tail a live session
tail -f $MYCELIUM_MOUNT/_activity/2026/05/10/coder.jsonl

# Find write events
mycelium grep --pattern '"op":"write"' --path _activity --format json
```

For structured decisions — conventions the agent adopted, lessons distilled, regions archived — query `mycelium evolve --active` to see what rules are currently in effect, or `mycelium evolve --list` for the full timeline.

### What does mycelium record so a future reviewer can understand why a change happened?

The diff is the cheap part — any version control system can tell you _what_ changed. The thing a future reviewer (or another agent) usually can't reconstruct is the _why_. Mycelium records that across two surfaces, both governed by the same discipline: capture the rationale at the moment of decision, and name what was rejected — not just what was chosen.

| Layer              | What it captures                                  | Where it lives                         |
| ------------------ | ------------------------------------------------- | -------------------------------------- |
| File content       | Per-note reasoning — trigger, hypothesis, outcome | The note file itself                   |
| `--rationale` flag | Why this specific operation was performed         | Activity log entry (`rationale` field) |
| `evolve` events    | Structural patterns/conventions the agent adopted | `_activity/` + evolve log              |
| Activity log       | Who did what and when (chain of custody)          | `_activity/YYYY/MM/DD/<agent>.jsonl`   |

**File contents carry the per-note reasoning.** When the agent writes an incident note, an investigation log, or a plan file, the _why_ lives in the note itself — the trigger, the hypothesis being tested, the alternatives considered and rejected. Same craft as a good commit message, applied to every note. A diff shows what changed; the note explains why. This is a convention; the binary does not enforce it.

**Operational rationale is now first-class on every rationale-bearing op.** `write`, `edit`, `rm`, `mv`, and `log` all accept an optional `--rationale "..."` flag. When supplied, rationale appears as a top-level field on the corresponding activity log entry (`omitempty` — absent when not supplied, so existing log readers are unaffected). On a CAS or destination-exists conflict, the losing caller's rationale also appears in the conflict envelope emitted to stderr, so the retrying agent can merge intent rather than just bytes. Maximum 64 KiB per rationale; oversize input is rejected with exit 65 before the mutation runs.

The note-body discipline and `--rationale` are complementary: note bodies hold _why-this-thing_ (per-note reasoning embedded in content), `--rationale` holds _why-this-operation_ (the reason for the mutation or signal, captured on the activity log line). Both can coexist on the same write.

**`evolve` events carry the structural decisions** — the patterns and rules the agent adopts for itself. When the agent picks a filename convention, builds an index, or archives a region, it records the choice with `mycelium evolve convention|index|archive --rationale "..."`. Rationale is required on `evolve`. `mycelium evolve --active` shows the rules currently in effect with their original reasoning attached; `mycelium evolve --list` gives the full timeline including superseded rules.

**The activity log is the chain of custody.** Every mutation lands in `<mount>/_activity/YYYY/MM/DD/<agent>.jsonl` with `agent_id`, `session_id`, timestamp, op kind, path, version hash, and (when supplied) `rationale`. With session-boundary entries (`session_startup`/`session_shutdown`, and optionally `turn_start`/`turn_end` from richer harnesses), you can group a burst of writes as one turn of one session rather than as scattered events, then cross-reference the note content and any `evolve` entries to recover the reasoning.

The split is deliberate: notes carry _why-this-thing_, the `--rationale` flag carries _why-this-operation_, `evolve` events carry _why-this-pattern_, and the activity log carries _who_ and _when_. A reviewer typically reads the notes for per-decision rationale, checks the activity log for operational context attached to individual mutations, runs `mycelium evolve --active` for the workspace's current rules, and consults the activity log to reconstruct timelines or find which session a given change belonged to.

### Should I commit a mount to git?

It's a legitimate option, not a requirement. A mount is a directory of plain text files and JSONL logs, so `git add` and `git commit` work naturally. The benefit is versioned history and the ability to share state between machines or teammates using normal git workflows. The tradeoff is that the `_activity/` log grows with every agent run and will produce large diffs. Phase 3 of the roadmap includes opt-in git/jj integration with proper commit authoring per agent operation; for now, manual commits work, but treat the activity log as append-only operational data rather than source code.

### How do I move a mount between machines?

`tar` it:

```sh
tar czf mount-backup.tar.gz .mycelium-store/
```

Copy the tarball to the other machine and extract it. Set `MYCELIUM_MOUNT` to the new path before running the agent. There is no registration, no daemon to inform, and no database to migrate.

### How big does the activity log get, and how do I prune it?

The log partitions by `_activity/YYYY/MM/DD/<agent_id>.jsonl`. In practice, growth depends on how many mutations the agent makes per day. There is no built-in retention policy in the current release — that is Phase 3 scope. Manual pruning works cleanly: archive or delete older date directories (e.g., anything under `_activity/2025/`) with normal filesystem tools. The agent and the binary only read from paths they discover at runtime, so removing old date trees does not break anything for ongoing work.

### How do I delete or roll back something an agent wrote?

There is no native undo command. To revert a file to a prior state, you need its prior content — either from your own memory of what it contained, from the git history if the mount is versioned, or by reading a saved copy. The activity log records every mutation with before/after version hashes, so you can see what changed and when; it does not store full snapshots of prior content.

For self-evolution decisions — conventions adopted, regions archived — `mycelium evolve --list` shows the full timeline, and `mycelium evolve --active` shows what's currently in effect. To retire a specific convention, record a superseding `evolve` event with `--supersedes <id>`.

---

## Status and maturity

### Is mycelium production-ready?

Not yet. Mycelium is pre-1.0, currently at v0.2.0 (early access). Phase 1 is feature-complete: atomic content-addressable storage (CAS), transaction-journal crash recovery, the activity log, `evolve`, and the on-disk format are all implemented and have property-based and concurrent-process test coverage. What is not yet complete is the full benchmark validation against frontier models (T1 multi-session synthesis and T2 self-evolution runs are drafted but awaiting published runs). The [roadmap](mycelium-phases.md) lays out what Phases 2 and 3 add.

The practical risk at this stage is not data loss — the core integrity primitives are solid — but rather that the API surface, on-disk format details, or activity log schema may still shift before 1.0.

### Are benchmarks available yet?

A rubric is fully defined in [benchmarks/phase-1.md](benchmarks/phase-1.md): three tasks (T1 multi-session research synthesis, T2 seeded self-evolution, T3 failure-mode detectors), target models (Claude Opus 4.7 and GPT-5.5), and pass/fail criteria. T3's failure-mode detectors are implemented and exercised by `go test -run TestDetectors ./internal/mycelium`. T1 and T2 task definitions and grading rubrics are drafted and ready to run. Published model runs against Opus 4.7 and GPT-5.5 are pending — results will land in `docs/benchmarks/` as they complete.

### What's on the roadmap?

Three phases. Phase 1 (current, feature-complete): the core CLI, CAS, activity log, `evolve`, crash recovery, and the pi.dev extension. Phase 2 (distribution and operational polish): a versioned activity log schema, recovery diagnostics, Claude Code skill distribution, a second harness integration (Hermes plugin or equivalent), optional read-byte caps if benchmarks call for them, and install/troubleshooting docs. Phase 3 (workflow integration): opt-in git/jj integration with per-operation commits, historical reads via `mycelium read --version=...`, configurable activity log retention, a curated templates repository, and a `mycelium init` CLI for template-based mount setup. See [the roadmap](mycelium-phases.md) for acceptance criteria per phase.

---

## Why these design choices

### Why a directory of files instead of a database?

The core bet is that general file tools scale with model intelligence, while specialized memory infrastructure caps it. A database or embedding index encodes assumptions about what gets saved, how it's indexed, and what counts as relevant. As models improve, those assumptions become drag. A filesystem has no such ceiling: `read`, `write`, `ls`, `grep` are the same interface regardless of model generation, and a Frontier model uses them with the same judgment a thoughtful engineer brings to a working notebook.

There are also practical benefits. Filesystems are durable, concurrent-safe with standard primitives (`flock`, atomic rename), inspectable with every text tool, and trivially exportable as a tarball. The agent owns the schema — directory structure, filenames, organization — and can revise it without a migration. See [ADR 0001](adr/0001-self-evolution-as-first-class-concept.md) and the [design doc](mycelium-design.md) section 2 for the full argument.

### Why JSONL for the activity log?

JSONL is append-only and line-oriented, which means it tolerates partial writes gracefully: a corrupt or incomplete last line does not poison the rest of the file. It is readable with `cat`, streamable with `tail -f`, parseable by every language without a special library, and searchable with `grep` or `rg --json`. Log shippers, text editors, and shell scripts all work against it without mycelium-specific tooling. The alternative — a structured database or binary format — would require a dedicated reader for every consumer and make the store opaque to the same standard tools that make the content files useful.
