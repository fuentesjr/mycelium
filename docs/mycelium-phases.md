# Mycelium: Phased Rollout

**Status:** Roadmap, companion to `mycelium-design.md`.

This document phases the Mycelium design into shippable units. Each phase ends in a working, useful system — not a half-built one. The cuts are made by principle, not convenience; where a phase defers a feature, the deferral is justified and the criterion for revisiting it is named.

The phasing rule: **every phase's scope must independently validate or extend the core bet.** If a phase doesn't either prove the system works or unlock new use cases, it's misshapen.

Mycelium is LocalFS/POSIX-native. The roadmap below keeps that guarantee set instead of promising storage portability that would weaken the design.

---

## Phase 1 — MVP: prove the bet under realistic conditions

**Goal.** Validate the central thesis on a real Frontier model, in conditions that match how the system will actually be used: across sessions, with multiple agents able to share the same local store, with the agent able to reflect on its own behavior over time. A one-shot single-agent MVP would validate a scenario nobody runs in production; the bet has to survive resumption, concurrency, and self-revision from day one or it isn't really validated.

**Prerequisites.** The Phase 1 task suite and scoring rubric lives at `docs/benchmarks/phase-1.md`. It specifies one multi-session research task (T1), one seeded self-evolution scenario (T2), failure-mode detectors (T3), and target Frontier models.

**In scope.**

- CLI surface: `mycelium read`, `write`, `edit`, `ls`, `grep`, `rm`, `mv`, `log`. Eight visible subcommands invoked through the agent harness's existing shell tool. A hidden transitional `evolve` stub points old callers at `MYCELIUM_MEMORY.md`.
- `read --format json` for CAS-safe UTF-8 reads that return `path`, `version`, and `content` in one envelope. Raw `cat` remains fine for inspection, including non-UTF-8 bytes.
- Every content-mutating subcommand accepts optional `--expected-version`; conflict recovery is first-class from v1.
- `mycelium grep` flags: `--pattern`, optional `--path`, `--regex`, `--format text|json`, and `--limit`. Implementation is pure Go: one search implementation and one regex dialect across machines.
- One storage contract: **LocalFS on a local POSIX filesystem**, with conditional writes implemented via `flock`-guarded version checks and content-hash version tokens.
- Atomic single-file ops via write-to-temp-then-rename, atomic rename for `mv`, and destination-collision protection.
- Authoritative activity log at `_activity/YYYY/MM/DD/{agent_id}.jsonl`. Every successful state-changing operation (`write`, `edit`, `rm`, `mv`, `log`) produces a durable JSONL entry. Reads are not logged.
- Durable mutation boundary: a command returns success only after the content change and activity entry are durable. A post-commit log append failure exits non-zero; a power loss in that narrow window may leave the final mutation unlogged.
- Mount and identity via environment variables: `MYCELIUM_MOUNT` is required; `MYCELIUM_AGENT_ID` defaults to `agent`; `MYCELIUM_SESSION_ID` is auto-generated per CLI process when absent. Harnesses can set stable agent/session ids for clearer timelines.
- Typed conflict errors. When `--expected-version` doesn't match, `mycelium` exits 64 and prints structured JSON to stderr containing the current version token. Recovery is re-read, merge, retry.
- Raw-read/raw-write boundary: raw filesystem reads are allowed; raw filesystem writes are unsupported. All live-store mutations go through `mycelium`.
- Reserved `_` prefix. `mycelium` rejects agent mutations under any `_`-prefixed root path; currently `_activity/` and `_lock` are system-owned, with legacy `_tx/` detected for compatibility.
- Reference agent harness: pi.dev extension. It registers no tools — the agent invokes `mycelium` via pi's built-in `bash` tool — and contributes only env-var setup, a small system-prompt block, starter convention seeding, and memory-relevant portable activity logging (`session_*`, `context_checkpoint`, `compaction`).
- Documentation for starter conventions, self-evolution recipes, and conflict-resolution conventions.

**Out of scope, with reasons.**

- _Non-POSIX storage._ The v1 guarantees depend on local filesystem semantics. A storage portability layer would either weaken the contract or require a separate design with different primitives. Out of scope.
- _Protocol wrapper._ The primary surface is shell plus the `mycelium` binary. A future harness without shell access can wrap the binary, but the core bet should not depend on a protocol binding.
- _Git/jj integration._ Useful ergonomics, not a correctness feature. Files on disk plus the activity log are sufficient to inspect, diff, and share. Phase 3.
- _Historical reads (`mycelium read --version=...`)._ Useful for archaeology but requires a versioned history source. Phase 3 with git/jj integration.
- _Activity log retention policy._ Default to no truncation in MVP. Retention is a later operational concern.
- _Token-budget enforcement on `mycelium read`._ `grep` has a mandatory `--limit`; read caps wait for benchmark evidence.
- _Templates repository and `mycelium init`._ Scaffolding ships as documentation and extension templates first. A CLI initializer is opt-in later.
- _Binary blobs._ Plain text covers everything needed to validate the bet.

**Acceptance criteria.** Phase 1 is done when:

1. **Single-agent multi-session.** T1 task in `docs/benchmarks/phase-1.md` passes on target Frontier models.
2. **Multi-agent concurrency.** Two agents on the same LocalFS store can update overlapping files concurrently without silent loss. A benchmark exercises adversarial timing.
3. **Conflict recovery on real models.** When a conditional write fails, the model receives the typed conflict error and produces sensible recovery behavior (re-read, merge, retry) given only the error and no special prompting.
4. **Self-evolution through conventions-file edits and activity-log evidence.** T2 task passes. Self-evolution is the floor behavior the supported tier is defined by; failure here is a Phase 1 blocker.
5. **Failure-mode observability.** T3 detectors distinguish dysfunctional traces from healthy use by reading the activity log alone.
6. **Activity log integrity.** Every successful state-changing operation has a durable activity entry; tests cover post-commit append failure, CAS behavior, and legacy `_tx/pending/*.json` blocking before further mutations.
7. **Reserved-path protection.** Property-based tests cover every mutating subcommand against `_`-prefixed paths.
8. **LocalFS correctness.** Property-based and sibling-process tests cover atomicity, destination collisions, and CAS semantics under concurrent writes.
9. **Human-readability.** A second engineer can take a tarball of the store and inspect content files and `_activity/` with standard tools and a text editor.

**What Phase 1 explicitly does not prove.** It does not prove every future ergonomic feature stays out of the way of Frontier models. That remains a decision gate after each later phase.

---

## Phase 2 — Distribution and operational polish

**Goal.** Make the LocalFS-native system easy to install, observe, and use across the agent harnesses engineers already run locally or on a single shared host.

**In scope.**

- Documented activity log schema. A versioned `_activity/SCHEMA.md` declares the entry shape and compatibility expectations.
- Legacy recovery diagnostics. A documented procedure explains how to handle pre-v0.3 `_tx/pending/*.json` records before using the current binary.
- Optional read-byte caps with explicit override, if benchmarks show large reads causing practical failures.
- Claude Code skill distribution bundling platform-specific binaries and the starter `MYCELIUM_MEMORY.md` template.
- Hermes memory plugin or equivalent harness integration that shells out to the binary and avoids auto-acting memory hooks.
- Installation, update, and troubleshooting docs for common LocalFS deployments.

**Out of scope.**

- _Git/jj-backed history._ Phase 3.
- _General retention policy._ Phase 3 unless log growth becomes a Phase 2 blocker.
- _Cross-store federation / mount manifests._ Out of scope until a concrete LocalFS use case earns it.

**Acceptance criteria.**

1. The activity log file format has a documented v1 contract that downstream tooling and agents can build against.
2. A multi-session research task completes end-to-end on at least two harness integrations with no extra prompt engineering beyond the shipped scaffolding.
3. Recovery diagnostics are understandable to an operator using only the filesystem, JSON, and docs.
4. Installation and update paths are repeatable on supported Linux/macOS platforms.

**Decision gate after Phase 2.** Run the benchmark suite against the strongest then-current Frontier model family on a long-running multi-agent task. If the system shows signs of capping the Frontier model — including the activity log shape, conventions-file prompt block, or recovery metadata — that feature is revised before any new one goes in.

---

## Phase 3 — Workflow integration

**Goal.** Fit Mycelium into normal engineering workflows while preserving the LocalFS contract.

**In scope.**

- **Git/jj integration.** Opt-in, off by default. When enabled, every mutating op produces a commit with the agent and session id in the message, and the activity log is committed alongside content. Composes with `git log`, `git diff`, `git blame`, and jj's working-copy semantics.
- **Historical reads.** `mycelium read <path> --version=...` for git/jj-backed stores. Lets the agent reconstruct prior states for richer self-evolution.
- **Activity log retention.** Configurable per mount, with a system-written policy file declaring the oldest available date so the agent knows its horizon.
- **Templates repository** with curated starter content (`MYCELIUM_MEMORY.md`, suggested layouts for common agent shapes — research, coding, project management).
- **`mycelium init` CLI** that copies a template into a fresh store. One command; entirely opt-in. An empty store remains valid.

**Out of scope.**

- _Binary blob support._ Later or never, depending on evidence.
- _Capability-tier eval harness as a public benchmark._ We continue to use it internally; productizing it is separate.
- _Symlink support._ Resolved by refusing or ignoring symlinks unless a future design proves a safe need.

**Acceptance criteria.**

1. A LocalFS-backed store under git/jj can be checked out by a teammate, inspected with normal VCS tools, and resumed by a different agent without state loss.
2. Historical reads work end-to-end: the agent can grep a version token from the activity log, pass it to `mycelium read --version=...`, and receive the corresponding historical content.
3. Templates are versioned independently of the runtime. A user can pull a newer template into an older Mycelium install or vice versa.
4. Retention policy is visible to the agent and operator as plain text.

---

## What stays absent across all phases

All anti-goals from `mycelium-design.md` section 9 hold across every phase. Phasing pressure is exactly when they get smuggled back in; if a phase ships any of them, the phase is wrong.

---

## Sequencing summary

| Phase | Theme                                             | Validates                                                               | Ships                                                        |
| ----- | ------------------------------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------ |
| 1     | MVP: multi-agent + multi-session + self-evolution | The core bet pays off under realistic LocalFS conditions                | LocalFS CLI + CAS + `_activity/` + pi.dev extension           |
| 2     | Distribution + operational polish                 | The LocalFS-native system is installable, observable, and diagnosable   | Schema docs + diagnostics + harness bundles                  |
| 3     | Workflow integration                              | The system fits normal engineering workflows without weakening the core | git/jj, historical reads, retention, templates               |

The decision gate after each phase is the most important rule in the roadmap. If a Frontier model is bottlenecked by anything Mycelium added, that thing comes out or gets redesigned before any new feature goes in.
