# Changelog

## [Unreleased]

### Changed

- Split fast local feedback from exhaustive deterministic property coverage with `make test`, `make test-full`, and `make test-race`, while retaining exhaustive and race gates in CI and releases.
- Made T1/T2 benchmark runs reproducible by pinning provider-qualified model IDs, pi/package metadata, isolated startup flags, exact grading inputs, and a checksum-pinned external T1 answer-key contract.
- Hardened releases with cross-file version validation, npm access preflight, provenance-enabled publishing, idempotent same-tag retries, and complete npm repository metadata.
- Aligned active documentation with the pi-only support boundary, supported platform matrix, best-effort lifecycle/bootstrap behavior, CLI edge cases, offline activity retention, and idempotent release recovery.

### Fixed

- Fixed `mycelium grep` so matching lines larger than 64 KiB are returned instead of silently skipped.
- Fixed project-local scope detection for version-pinned `npm:pi-mycelium@<version>` registrations.
- Clarified agent guidance so CAS conflicts use re-read/merge/retry while `mv destination_exists` requires an explicit destination-content decision.

## [0.5.0] - 2026-07-12

### Changed

- Narrowed the supported coding-agent harness to pi via the `pi-mycelium` extension. Direct CLI use remains available for pi shell operation, development, diagnostics, and advanced inspection; non-pi harness integrations are unsupported.
- Added ADR-0007 and marked ADR-0002 / ADR-0006 superseded by the pi-only support decision.
- Replaced the portable activity-event specification with concise pi activity-event documentation while preserving journal compatibility and tolerance for historical/unknown log operations.
- Kept platform CLI npm packages and binary archives as the way `pi-mycelium` ships and diagnoses its Go CLI engine.

### Removed

- Removed the portable Agent Skill under `skills/mycelium/`; v0.4.0 was the last release that included it.
- Removed the portable activity-event fixture and extension test that treated cross-harness event vocabulary as an active compatibility target.

## [0.4.0] - 2026-06-26

This release intentionally combines the core simplification work and the portable skill split into one pre-1.0 cut. No `v0.3.0` tag was published.

### Added

- Added `skills/mycelium/`, a portable Agent Skill with concise operating guidance, command/conflict/activity references, and read-only setup diagnostics for shell-based harnesses.

### Changed (mycelium binary)

- Folded `mycelium glob <pattern>` into `mycelium ls [pattern] [--recursive]`; `glob` is no longer a subcommand.
- Removed `mycelium grep --file-type`; use `--path` to scope searches.
- Removed `--include-current-content` from `write`, `edit`, `rm`, and `mv`. Conflict envelopes still include `current_version`; recover by re-reading with `mycelium read <path> --format json`, merging, and retrying with the fresh version token.
- Removed functional `mycelium evolve`; conventions, lessons, index locations, archive policy, and open questions now live in `MYCELIUM_MEMORY.md`. A hidden transitional `evolve` stub exits with guidance for old prompts and templates.
- Removed the `_tx/` transaction journal. Mutating commands now lock, check CAS, commit content, append the durable activity entry, and return success only after the append succeeds. If legacy `_tx/pending/*.json` records are present, current binaries block mutations with instructions to recover the mount using the last v0.2 binary.
- Mutation `tx_id` values now use stdlib time/randomness (`tx-<unix-nano>-<rand>`), and generated default session IDs use the matching `auto-<unix-nano>-<rand>` shape.

### Changed (pi-mycelium extension)

- Stopped emitting turn/tool telemetry from the reference adapter. It now records session boundaries, `session_shutdown`, and `compaction` only; the portable vocabulary remains available for richer adapters.
- Removed the reference adapter context hook and context-checkpoint fingerprint/dedupe machinery.

### Changed (docs)

- Consolidated agent-facing operational guidance into `skills/mycelium/`; removed the old `docs/agent-faq.md`, `docs/conflict-resolution.md`, and `docs/self-evolution.md` duplicates.

## [0.2.1] - 2026-05-29

### Fixed (pi-mycelium extension)

- System prompt now points agents at the exact `<mount>/MYCELIUM_MEMORY.md` path and tells them not to broad-search for required files.

## [0.2.0] - 2026-05-12

### Added (mycelium binary)

- `--rationale "..."` flag on `write`, `edit`, `rm`, `mv`, and `log`. When supplied, the rationale is captured as a top-level `rationale` field (`omitempty`) on the corresponding activity log entry. When absent, behavior is unchanged and the field is omitted.
- `rationale` field on `LogEntry` (`internal/mycelium/log.go`) — `string`, `json:"rationale,omitempty"`.
- `rationale` field on `conflictEnvelope` (`internal/mycelium/write.go`) — when a CAS or destination-exists conflict occurs, the losing caller's rationale is included alongside `current_version` so retrying agents retain the attempted intent while re-reading the winning content.
- Rationale size validation: input exceeding 64 KiB is rejected before the mutation runs with exit code 65 (`ExitReservedPrefix`) and message `mycelium <op>: --rationale exceeds N bytes`.

### Added (pi-mycelium extension)

- System-prompt block now includes a one-line recommendation urging agents to supply `--rationale` on rationale-bearing operations (`write`, `edit`, `rm`, `mv`, `log`) when operational reasoning exists.

### Added (docs)

- ADR-0003 records the decision to add optional `--rationale` to all rationale-bearing mutation and log verbs, document its propagation through the CAS conflict envelope, and keep the note-body discipline as a complementary convention.

## [0.1.8] - 2026-05-08

### Added (mycelium binary)

- `mycelium read --format json` now returns `{path, version, content}` so agents can obtain content and a CAS token from the same read.
- `mycelium evolve` now owns the full evolution surface: recording events plus query modes `--list`, `--active`, and `--kinds`.
- `_tx/pending/{tx_id}.json` transaction records now make content mutations and activity-log appends recover together after crashes.

### Changed (mycelium binary)

- Removed the separate `mycelium evolution` command; use `mycelium evolve --list|--active|--kinds` instead.
- Implicit evolution supersession now applies only when `--target` is non-empty. Targetless lessons/questions are additive unless explicitly superseded.
- Explicit `--supersedes` may now retire an event of a different kind, supporting workflows such as resolving a `question` into a `lesson`.

### Changed (pi-mycelium extension)

- Active-evolution startup queries now call `mycelium evolve --active/--kinds`.
- System-prompt conflict guidance now points agents to `mycelium read <path> --format json` for CAS recovery.
- Starter memory template now documents targetless additive evolution entries and explicit `--supersedes` retirement.
- Activity logging now emits portable `context_checkpoint` entries with generic payload fields, fingerprint-based duplicate suppression, and turn/tool/compaction/session-shutdown events. The legacy `context_signal` helper remains for compatibility.
- The injected system prompt now documents the adapter-recorded activity events, metadata-only payload policy, and how agents should grep `context_checkpoint` history.

### Added (docs)

- `docs/portable-activity-events.md` documents the adapter event vocabulary, payload conventions, dedupe policy, shell/tool/pi adapter examples, and links a representative JSONL fixture.
- ADR-0002 records the decision to keep portable activity events as adapter conventions rather than binary-enforced schema.

## [0.1.7] - 2026-05-04

### Fixed (pi-mycelium extension)

- Session-start ordering: `recordSessionBoundary` now runs before `bootstrapMemoryFile`, so the journal shows the `session_*` boundary entry first and the seed `op:write MYCELIUM_MEMORY.md` second. Previously the seed appeared to predate the session, since bootstrap ran first.

### Changed (mycelium binary)

- Widened the `convention` kind's built-in definition from "A naming, layout, or structural pattern for organizing data in the store." to "A naming, layout, structural, or behavioral pattern for organizing or operating on the store." Conservative widening — calibrated to one observed agent stretch. Behavioral norms about how to use the store (e.g. "record preferences proactively") now fit the kind without an `experiment`/`policy` extension.
- ADR-0001, `docs/mycelium-design.md`, and `docs/self-evolution.md` updated to match.

### Changed (pi-mycelium extension)

- System-prompt block's `convention` bullet now shows two worked examples (path-scoped and behavior-scoped) and broadens the target placeholder from `<path-or-glob>` to `<path-or-glob-or-scope>`. First observation showed an agent picking the vague target `mycelium usage` for a behavioral norm; the example gives a concrete model.

## [0.1.6] - 2026-05-03

### Added (pi-mycelium extension)

- `MYCELIUM_MEMORY.md` is now auto-seeded into the mount on first `session_start` if it doesn't already exist. Routes through `mycelium write`, so the bootstrap appears in the journal as a normal `op:write`. Previously the system prompt told the agent "Read it once at session start" but no install step ever created the file — every fresh mount served a not-found on first read. Bundled template lives at `extensions/pi-mycelium/templates/MYCELIUM_MEMORY.md`.

### Changed (template)

- Moved canonical template from `docs/templates/MYCELIUM_MEMORY.md` to `extensions/pi-mycelium/templates/MYCELIUM_MEMORY.md` so it ships in the npm package.
- Replaced the broken relative ADR link with an absolute GitHub URL — relative links from `docs/` don't resolve when the file is dropped into a mount.
- Renamed suggested `AGENTS/{agent_id}/` directory to lowercase `agents/{agent_id}/` to avoid collision with the `AGENTS.md` instructions convention.
- Added `memories/` to the suggested starter layout (matching the directory the agent already invents on its own).
- Added an empty top-level `## Conventions` section so the agent has a concrete place for its first prose entry.
- Trimmed sections that duplicated the system-prompt block (subcommand list, identity env-var enumeration, reserved-prefix rule explanation). The template now focuses on persistent, agent-editable content; the system prompt carries the per-session fundamentals.

## [0.1.5] - 2026-05-03

### Fixed (pi-mycelium extension)

- `recordSessionBoundary` now logs all five `session_start` reasons (`startup`, `reload`, `new`, `resume`, `fork`) instead of dropping `startup` and `reload`. Previously the most common case — a fresh `pi` invocation, which fires `reason="startup"` — produced no boundary entry, leaving the journal with only `context_signal` rows and no record of when sessions began.

## [0.1.4] - 2026-05-02

### Changed (pi-mycelium extension)

- Mount paths moved: global is now `~/.pi/agent/extensions/pi-mycelium/journal/` (was `~/.pi/mycelium/store/`); project-local is now `<cwd>/.pi/pi-mycelium/journal/` (was `<cwd>/.pi/mycelium/store/`). Co-locates the journal with the extension's own directory tree and renames `store` → `journal` to better describe its append-mostly nature.
- `session_start` now prepends the bundled binary's directory to `PATH` so the agent's `bash` invocations of `mycelium <sub>` resolve without a separate PATH setup step.

## [0.1.3] - 2026-05-02

### Fixed (pi-mycelium extension)

- Added the required `pi.extensions` manifest field to `package.json`. Without it, pi installed the package but never registered the `session_start`/`before_agent_start`/`context` hooks.

## [0.1.2] - 2026-05-02

### Fixed (pi-mycelium extension)

- Bundled-binary resolver now maps Node `process.arch` (`x64`, `arm64`) to Go `GOARCH` (`amd64`, `arm64`) before looking up the optional dependency. Previously failed on Intel Node with `EBADPLATFORM`.
- Scope detection no longer assumes the extension lives under `~/.pi/agent/extensions/` or `<repo>/.pi/extensions/`. `pi install` drops packages into a node_modules tree outside both roots; detection now consults pi's `settings.json` to decide global vs. project.

## [0.1.1] - 2026-05-02

### Changed (release pipeline)

- npm publish step is idempotent: re-running a release skips platform packages already published at the target version instead of failing the workflow.

## [0.1.0] - 2026-05-01

### Added

- `mycelium evolve <kind>` subcommand: record self-evolution events (conventions, indices, archives, lessons, questions, or agent-introduced kinds) with structured kind/target/rationale/supersession metadata. See [ADR-0001](docs/adr/0001-self-evolution-as-first-class-concept.md).
- `mycelium evolution` subcommand: query the evolution timeline. `--active` returns current rules in effect per `(kind, target)` pair; `--kinds` enumerates available kinds (built-in plus agent-introduced). Superseded by `mycelium evolve --list|--active|--kinds` in the next release.
- Five built-in kinds shipped with definitions: `convention`, `index`, `archive`, `lesson`, `question`. Agents may introduce additional kinds via `--kind-definition` on first use.
- pi-mycelium extension surfaces evolution kinds, active evolution, and recording instructions in the `before_agent_start` system prompt.

## [0.0.1] - 2026-05-01

Initial release. Phase 1 scope per [`docs/mycelium-phases.md`](docs/mycelium-phases.md).

### Added

#### `mycelium` binary (Go)

- Nine subcommands: `read`, `write`, `edit`, `ls`, `glob`, `grep`, `rm`, `mv`, `log`.
- Mount-level `flock` guarding compare-and-swap to close the read-then-write TOCTOU race across sibling processes.
- SHA-256 version tokens (`sha256:<hex>`) returned on every successful mutation; conflict envelope on CAS failure includes `current_version`, `expected_version`, optional `current_content`, and exits 64.
- JSONL activity log at `<mount>/_activity/YYYY/MM/DD/<agent>.jsonl`, split per agent and per UTC day.
- Reserved `_`-prefix on top-level paths protects system metadata (`_activity/`, `_lock`) from agent writes; rejected with usage-error exit 65.
- 338 tests including property tests on the activity log, T3 failure-mode detectors with hand-crafted trajectories, sibling-process CAS validation, and a tarball-roundtrip test pinning the "plain files plus JSONL" contract.

#### `pi-mycelium` extension (TypeScript / pi.dev)

- `session_start` hook: detects the `mycelium` binary on `PATH`, sets `MYCELIUM_AGENT_ID` (default `pi-agent`), `MYCELIUM_SESSION_ID` (from pi's session leaf id), and `MYCELIUM_MOUNT`.
- `before_agent_start` hook: appends a system-prompt block introducing subcommands, conventions, identity, and CAS semantics. Chains off `event.systemPrompt` so other extensions' contributions are preserved. Falls back to an `UNAVAILABLE` block when the binary is not on PATH.
- `context` hook: records `context_signal` activity-log entries on every context event without modifying the agent message stream.
- Mount auto-detected from install location: project install mounts at `<cwd>/.pi/pi-mycelium/journal/`; global install mounts at `~/.pi/agent/extensions/pi-mycelium/journal/`.
- 37 vitest tests across config, env, system-prompt, activity-log, and the index entry point.

#### Documentation

- Design rationale, phasing roadmap, conflict-resolution conventions, self-evolution patterns, Phase 1 benchmark rubric, `MYCELIUM_MEMORY.md` template.

### Distribution

Source build only. Pre-built binaries, npm publish, and Homebrew tap are Phase 2.

### Known limitations

- `mycelium read` does not surface the current version token; agents obtain it from the conflict envelope on a failed CAS write rather than via a pre-read. Superseded by `mycelium read --format json` in the next release.
- `mycelium ls` ignores positional path arguments (always lists from mount root) and is non-recursive by default; use `--recursive` to walk subdirectories.
