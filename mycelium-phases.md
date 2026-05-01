# Mycelium: Phased Rollout

**Status:** Roadmap, companion to `mycelium-design.md`. Phase 1 binary shipped at v0.3.0.

This document phases the Mycelium design into shippable units. Each phase ends in a working, useful system — not a half-built one. The cuts are made by principle, not convenience; where a phase defers a feature, the deferral is justified and the criterion for revisiting it is named.

The phasing rule: **every phase's scope must independently validate or extend the core bet.** If a phase doesn't either prove the system works or unlock new use cases, it's misshapen.

---

## Phase 1 — MVP: prove the bet under realistic conditions

**Goal.** Validate the central thesis on a real Frontier model, in conditions that match how the system will actually be used: across sessions, with multiple agents able to share the same store, with the agent able to reflect on its own behavior over time. A one-shot single-agent MVP would validate a scenario nobody runs in production; the bet has to survive resumption, concurrency, and self-revision from day one or it isn't really validated.

**Prerequisites (before any runtime code).** The Phase 1 task suite and scoring rubric is drafted at `docs/benchmarks/phase-1.md`. Acceptance criteria 1, 4, and 5 below are unfalsifiable without it. The rubric specifies one multi-session research task (T1), one seeded self-evolution scenario (T2), the failure-mode detectors (T3), and the two target models: Claude Opus 4.7 and GPT-5.5. Per-task content under `docs/benchmarks/tasks/T<n>-<slug>/` is the remaining drafting work before runs are executable.

**In scope.**

- CLI surface: `mycelium read`, `write`, `edit`, `ls`, `glob`, `grep`, `rm`, `mv`, `log`. Nine subcommands invoked through the agent harness's existing shell tool. `log` is the signal collector — content mutations log themselves automatically, and explicit non-mutation signals (harness observations, agent annotations) land via `mycelium log`. Every content-mutating subcommand accepts an optional `--expected-version` flag; concurrency is first-class from v1.
- `mycelium grep` flags: `--pattern`, optional `--path`, `--regex`, `--file-type`, `--format` (`text` default, `json` for a `{matches, truncated, next_cursor}` envelope where each match is `{path, line, text}`), `--limit` (default 1000, hard-capped to prevent context overflow on log queries), `--cursor` accepted for forward compatibility but pagination not implemented in Phase 1; revisit in Phase 3 alongside log retention. Backend implementation prefers ripgrep, falls back to grep, falls back to Go-native scan.
- One backend: **LocalFS**, with conditional writes implemented via `flock`-guarded version checks (content-hash version tokens). Atomic single-file ops via write-to-temp-then-rename.
- Distribution: a single `mycelium` binary published for Linux/macOS. The agent harness places it on `$PATH`; the agent invokes it through its existing shell tool. No protocol layer.
- **Agent identity and session identity** travel via environment variables (`MYCELIUM_AGENT_ID`, `MYCELIUM_SESSION_ID`) set once by the harness. Both are recorded in the activity log; both are visible to the conflict-error path; both are filterable by grepping the log.
- **Typed conflict error.** When `--expected-version` doesn't match, `mycelium` exits with code 64 and prints structured JSON to stderr containing the current version token (and, opt-in via `--include-current-content`, the current content). This is the multi-agent and multi-session coordination primitive.
- **Activity log at reserved path `_activity/YYYY/MM/DD/{agent_id}.jsonl`.** Every successful mutating operation produces a JSONL entry on the writing agent's daily file. Reads are not logged. `mycelium log` appends a signal entry to the same file, with any caller-supplied payload inlined as the entry's `payload` field. `mycelium` rejects agent writes under any `_`-prefixed root path — the one principled enforcement exception in the design. Agent reads `_activity/` via `mycelium read`/`glob`/`grep` (or raw `cat`/`ls`/`rg`) like any other content. Durability contract per design section 5.
- **Reference agent harness — pi.dev extension.** Phase 1 ships a TypeScript pi.dev extension as the reference integration. It registers no tools — the agent invokes `mycelium` via pi's built-in `bash` tool — and contributes only a system-prompt block (via `before_agent_start`, introducing the conventions and the nine subcommands), env-var setup on `session_start` (`MYCELIUM_AGENT_ID` defaulting to `pi-agent`, `MYCELIUM_SESSION_ID` from `ctx.sessionManager.getLeafId()`, minting new on fork), and observation of the `context` event recorded to the activity log via `mycelium log context_signal`. Default mount at `.pi/mycelium/store/` (project-scoped); switches to `~/.pi/mycelium/store/` when extension config sets `scope: "global"` (cross-project shared memory). One mode per session in Phase 1; layered project-over-global is a Phase 3 feature. Multi-agent and multi-session scenarios are exercised via pi's session/branching model plus concurrent extension instances against the same store.
- Documentation only for starter conventions and self-evolution patterns. The `MYCELIUM_MEMORY.md` template lives in `docs/templates/`. A `docs/self-evolution.md` documents the convention-bootstrap, convention-revision, self-built-index, and archiving patterns from the design's section 7, with concrete `mycelium glob` + `mycelium grep --format=json` query recipes against the activity log. All copied by hand. No `mycelium init` yet.
- Documented conflict-resolution conventions (re-read, merge, retry) in `docs/`. Documentation only — not enforced, not injected into prompts.

**Out of scope, with reasons.**

- *S3-compatible backend.* LocalFS with `flock`-based CAS is sufficient to validate the system's multi-agent semantics: two processes on the same host mounting the same directory exercises every concurrency code path that matters. S3's specific behaviors (ETag conditional-put rollout, listing consistency, prefix-policy ACLs) are real but orthogonal to whether the *core bet* survives concurrency. They get a focused phase. **If "MVP" means "deployable for distributed teams from day one" rather than "validates the bet," pull S3 forward — but say so explicitly; it's a different goal.**
- *Protocol wrapper (MCP, OpenAI tool-call, or other).* The agent's surface is the shell plus the `mycelium` binary. A protocol wrapper is a hundred lines of code over the same binary if a future harness can't grant shell access — but every Frontier deployment we'd target has shell, and shipping a wrapper in MVP would conflate "does the bet work" with "does our protocol-binding work." Defer until concrete demand surfaces.
- *Git/jj integration.* Tempting and cheap to add, but it's an ergonomics feature, not a correctness feature. Files on disk plus the activity log are sufficient to inspect, diff, and share. Phase 3.
- *Historical reads (`mycelium read` with a `--version` flag).* Useful for self-evolution but requires a versioned backend. Lands in Phase 3 with git/jj integration where there's a natural version source. The Phase 1 activity log gives the agent enough behavioral awareness to self-evolve without it: the agent can see *that* a file was rewritten and *when* and *by whom*, even if it can't see the prior contents.
- *Activity log retention policy.* Default to no truncation in MVP. Retention is a Phase 3 concern when stores are deployed long enough for the log to actually grow.
- *Token-budget enforcement on `mycelium read`.* `mycelium grep` has its mandatory `--limit` cap in Phase 1 (the log-reflection failure mode is sharp enough to address pre-emptively); `mycelium read` byte cap is deferred until benchmarking shows it as a real failure mode. If it doesn't surface, we save the complexity.
- *Layered backends, per-prefix ACLs, mount manifests.* All Phase 3.
- *Templates auto-install, mycelium init CLI.* Both scaffolding-adjacent. Per the design's central principle, scaffolding ships as documentation first and gets promoted to code only if benchmarking shows users repeatedly need it.
- *Binary blobs.* Defer to Phase 4 or beyond. Plain text covers everything we need to validate the bet.

**Acceptance criteria.** Phase 1 is done when:

1. **Single-agent multi-session.** T1 task in `docs/benchmarks/phase-1.md` passes on Claude Opus 4.7 and GPT-5.5.
2. **Multi-agent concurrency.** Two agents on the same LocalFS store can update overlapping files concurrently without silent loss. A benchmark exercises this with adversarial timing (synchronized writes to the same path).
3. **Conflict recovery on real models.** When a conditional write fails, the model receives the typed conflict error and produces sensible recovery behavior (re-read, merge, retry) given only the error and no special prompting. Verified on Claude Opus 4.7 and GPT-5.5.
4. **Self-evolution via the activity log.** T2 task in `docs/benchmarks/phase-1.md` passes on Claude Opus 4.7 and GPT-5.5. Self-evolution is the floor behavior the supported tier is defined by; failure here is a Phase 1 blocker, not a calibration signal.
5. **Failure-mode observability.** T3 detectors in `docs/benchmarks/phase-1.md` distinguish dysfunctional traces (e.g., the "31 transcript files" pattern) from healthy use by reading the activity log alone.
6. **Activity log integrity.** `mycelium` rejects every attempted agent write under any `_`-prefixed root path (write, edit, rm, mv source, mv destination). Property-based tests cover all four mutating subcommands, and the test suite includes paths under `_activity/` as well as a synthetic `_test_reserved/` to exercise the prefix rule rather than the specific path.
7. **Backend correctness.** The LocalFS backend passes a property-based test suite covering atomicity and CAS semantics under concurrent writes from sibling processes.
8. **Human-readability.** A second engineer can take a tarball of the store and inspect both content files and the activity log with `cat`, `grep`, and a text editor — no Mycelium-specific tooling required.

**What Phase 1 explicitly does not prove.** It does not prove the system stays out of the way of Frontier models. That's a Phase 2+ question, and it gets a decision gate at the end of Phase 2 (below).

---

## Phase 2 — Durable cloud storage

**Goal.** Make the system deployable for distributed teams on real cloud storage and portable to additional harnesses.

**In scope.**

- **S3-compatible backend.** ETag as version token, `If-Match` for conditional puts, `ListObjectsV2` for `mycelium ls`. Tested against AWS S3, Cloudflare R2, and MinIO. Behavior under listing eventual consistency documented and surfaced honestly. Activity log entries written to `_activity/YYYY/MM/DD/{agent_id}.jsonl` as objects in the same bucket; per-agent path keeps log writes contention-free without coordinated appends. `Search` does prefix-scoped client-side scan with ripgrep-equivalent matching. On activity-log append failure the binary warns to stderr and exits 0 (the content PUT has already committed); benchmarks include forced-failure scenarios that verify content/log divergence is *visible* rather than silent.
- Activity log file format gets a documented v1 contract — a `_activity/SCHEMA.md` written by the system at mount initialization (or first write) declaring the entry shape and version. Stable interface for both downstream tooling and the agent.
- Optional read-bytes cap with explicit override flag for `mycelium read` and a result-count cap on `mycelium grep`. Default generous; configurable per mount.
- Backend-agnostic test suite: every concurrency, durability, and self-evolution test from Phase 1 runs against S3 with identical pass criteria.
- **Hermes memory plugin.** A Python `MemoryProvider` plugin (per Hermes' `agent/memory_provider.py` ABC) shipping at `plugins/memory/mycelium/`. Implements the required methods (`name`, `is_available`, `initialize`, `get_tool_schemas`, `get_config_schema`, `save_config`, `handle_tool_call`) plus `system_prompt_block` for prompt scaffolding; `handle_tool_call` shells out to the `mycelium` binary. Skips the auto-acting hooks (`prefetch`, `sync_turn`, `on_pre_compress`, `on_memory_write`, `on_session_end`) on principle. Default mount at `${hermes_home}/mycelium/store/`. Aligns with Hermes' Single Provider Rule.
- **Claude Code skill distribution.** A `mycelium` Claude Code skill bundling the platform-specific binaries under `scripts/`, the starter `MYCELIUM_MEMORY.md` template, and a `SKILL.md` that teaches the nine subcommands, the `_` prefix reservation, identity env vars, and conflict handling. `allowed-tools: Bash(${CLAUDE_SKILL_DIR}/scripts/mycelium *)` pre-approves invocation. Follows the Agent Skills open standard so the same bundle is portable to any harness that adopts it.

**Out of scope.**

- *Layered backends (read-only overlay over writable).* Phase 3.
- *Git/jj integration and historical reads.* Phase 3.
- *Per-prefix ACLs.* Phase 3, and only on backends that support it cleanly.
- *Cross-store federation / mount manifests.* Phase 3.
- *Activity log retention policy.* Phase 3.

**Acceptance criteria.**

1. Two agents on different hosts, mounted at the same S3 prefix, can update overlapping files concurrently without silent loss.
2. The same agent harness, with a single-line config change, runs against LocalFS and S3 with identical observable behavior on the full Phase 1 + Phase 2 task suite — including the self-evolution criterion.
3. The activity log file format has a documented v1 contract that downstream tooling and agents can build against.
4. A multi-session research task — same shape as Phase 1's T1, run against an S3-mounted store — completes end-to-end on Claude Code after the user drops the `mycelium` skill in `~/.claude/skills/`, with no further configuration.
5. The same task completes on Hermes after the user installs the `mycelium` Hermes memory plugin via Hermes' standard plugin install path. The plugin's `system_prompt_block` is sufficient scaffolding — no further prompt engineering required.

**Decision gate after Phase 2.** Before starting Phase 3, run the benchmark suite against the strongest then-current Frontier model family on a long-running multi-agent task. If the system shows signs of capping the Frontier model — e.g., the model is fighting any feature added in Phase 1 or Phase 2 — that feature comes out before any new ones go in. **Special attention to the activity log shape:** if a Frontier model is parsing entries in ways that suggest the schema is too narrow (or too wide), revise it before it ossifies.

---

## Phase 3 — Production hardening and workflow integration

**Goal.** Polish, ergonomics, and integration with the workflows engineers already use. By the end of Phase 3, deploying Mycelium should feel like deploying any other piece of infrastructure: version-controllable, composable with existing storage, and ergonomic against the supported backends (LocalFS, S3).

**In scope.**

- **Layered backends.** Writable LocalFS or S3 over a read-only S3 prefix; copy-on-write on first mutation. Enables shared knowledge directories without forking.
- **Git/jj integration on LocalFS.** Opt-in, off by default. When enabled, every mutating op produces a commit with the agent and session id in the commit message, and the activity log is committed alongside content. Composes with `git log`, `git diff`, `git blame`, and jj's working-copy semantics.
- **Historical reads.** `mycelium read <path> --version=...` for backends that support it (git/jj-backed LocalFS, versioned S3 buckets). Lets the agent reconstruct prior states for richer self-evolution: "what did `MYCELIUM_MEMORY.md` look like before I changed it?"
- **Activity log retention.** Configurable per mount, with the system writing a `_activity/RETENTION.md` declaring policy and oldest available date so the agent knows its horizon.
- **Per-prefix ACLs** on S3 (via prefix policies / IAM). Optional and per-mount; the default remains "every mounted agent has equal permissions."
- **Mount manifest.** A small config format for federated mounts (e.g., `/team/` from one backend, `/me/` from another).
- **Templates repository** with curated starter content (`MYCELIUM_MEMORY.md`, suggested layouts for common agent shapes — research, coding, project management).
- **`mycelium init` CLI** that copies a template into a fresh store. One command; entirely opt-in. An empty store remains the default.

**Out of scope.**

- *Binary blob support.* Phase 4.
- *Capability-tier eval harness as a public benchmark.* We continue to use it internally; productizing it is a separate piece of work.
- *Symlink support.* Resolved by refusing them; documented as a deliberate non-feature.

**Acceptance criteria.**

1. A team can deploy Mycelium against an S3-compatible bucket with no Mycelium-specific operations knowledge — install, configure storage URL and credentials, point an agent at it. The same store can be mounted concurrently from multiple hosts.
2. A LocalFS-backed store under git can be checked out by a teammate, inspected with normal git tools, and resumed by a different agent on different hardware without any state loss.
3. Layered backends work transparently: an agent reading from a layered mount cannot tell whether it's reading the read-only base or the writable overlay until it tries to mutate, at which point the COW happens silently.
4. Historical reads work end-to-end: the agent can grep an `after_version` token from the activity log, pass it to `mycelium read --version=...`, and receive the corresponding historical content.
5. The templates are versioned independently of the runtime. A user can pull a newer template into an older Mycelium install or vice versa.

---

## What stays absent across all phases

All anti-goals from `mycelium-design.md` section 9 hold across every phase. Phasing pressure is exactly when they get smuggled back in; if a phase ships any of them, the phase is wrong.

---

## Sequencing summary

| Phase | Theme                                             | Validates                                                | Ships                                              |
|-------|---------------------------------------------------|----------------------------------------------------------|----------------------------------------------------|
| 1     | MVP: multi-agent + multi-session + self-evolution | The core bet pays off under realistic conditions, including agent reflection | LocalFS + `mycelium` CLI + CAS + reserved `_activity/` + ripgrep + pi.dev extension |
| 2     | Durable storage + harness distribution            | The bet survives cloud storage and ships to general users | + S3 + log format versioning + Hermes plugin + Claude Code skill   |
| 3     | Production polish + integration                   | The system fits into normal engineering workflows        | + git/jj, layered, ACLs, historical reads          |

The decision gate at the end of Phase 2 is the most important moment in the roadmap. If a Frontier model is bottlenecked by anything Mycelium added — including the activity log shape — that thing comes out. The whole design only works if that rule is honored.
