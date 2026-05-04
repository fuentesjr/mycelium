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
- `mycelium evolution` subcommand: query the evolution timeline. `--active` returns current rules in effect per `(kind, target)` pair; `--kinds` enumerates available kinds (built-in plus agent-introduced).
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
- Reserved `_`-prefix on top-level paths protects backend metadata (`_activity/`, `_lock`) from agent writes; rejected with usage-error exit 65.
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

- `mycelium read` does not surface the current version token; agents obtain it from the conflict envelope on a failed CAS write rather than via a pre-read.
- `mycelium ls` ignores positional path arguments (always lists from mount root) and is non-recursive by default; use `--recursive` to walk subdirectories.
