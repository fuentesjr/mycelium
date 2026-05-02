## [0.1.0] - 2026-05-01

### Added
- `mycelium evolve <kind>` subcommand: record self-evolution events (conventions, indices, archives, lessons, questions, or agent-introduced kinds) with structured kind/target/rationale/supersession metadata. See [ADR-0001](docs/adr/0001-self-evolution-as-first-class-concept.md).
- `mycelium evolution` subcommand: query the evolution timeline. `--active` returns current rules in effect per `(kind, target)` pair; `--kinds` enumerates available kinds (built-in plus agent-introduced).
- Five built-in kinds shipped with definitions: `convention`, `index`, `archive`, `lesson`, `question`. Agents may introduce additional kinds via `--kind-definition` on first use.
- pi-mycelium extension surfaces evolution kinds, active evolution, and recording instructions in the `before_agent_start` system prompt.

## [0.0.1] - 2026-05-01

Initial release. Phase 1 scope per [`mycelium-phases.md`](mycelium-phases.md).

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
- Mount auto-detected from install location: project install (under `<repo>/.pi/extensions/`) mounts at `<cwd>/.pi/mycelium/store/`; global install (under `~/.pi/agent/extensions/`) mounts at `~/.pi/mycelium/store/`.
- 37 vitest tests across config, env, system-prompt, activity-log, and the index entry point.

#### Documentation

- Design rationale, phasing roadmap, conflict-resolution conventions, self-evolution patterns, Phase 1 benchmark rubric, `MYCELIUM_MEMORY.md` template.

### Distribution

Source build only. Pre-built binaries, npm publish, and Homebrew tap are Phase 2.

### Known limitations

- `mycelium read` does not surface the current version token; agents obtain it from the conflict envelope on a failed CAS write rather than via a pre-read.
- `mycelium ls` ignores positional path arguments (always lists from mount root) and is non-recursive by default; use `--recursive` to walk subdirectories.
