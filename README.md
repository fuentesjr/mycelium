# Mycelium

**Persistent memory for pi coding agents:** a journal folder, safe mutations, and a searchable activity log. The `pi-mycelium` extension installs the runtime guidance, journal template, lifecycle logging, and bundled Go CLI engine used from pi's shell.

```
pi coding agent
    │ system prompt + lifecycle events + env
    ▼
pi-mycelium extension
    │ invokes bundled `mycelium` binary
    ▼
local journal directory
    ├── agent-owned notes
    └── _activity/YYYY/MM/DD/<agent>.jsonl
```

Mycelium is pre-1.0 and supports pi as its sole coding-agent harness. Other programs may be able to invoke the binary, but non-pi harness integrations are not documented, tested, or supported.

## Why

AI coding agents lose context when a session ends. Mycelium gives pi agents a durable, inspectable filesystem they own: `cat` reads it, `grep` searches it, `git` can version it, and multiple pi agents can write to it concurrently without corrupting the journal.

## Public model

> **A folder + safe mutations + a searchable activity log.**

The everyday loop is: list or grep paths, read relevant files, write or edit a note, and inspect `_activity/` when you need history. CAS tokens, locks, fsync, and durable append boundaries are implementation details of the safe-mutation layer.

## Features

- **Pi extension install.** `pi install npm:pi-mycelium` sets the journal mount, identity, prompt guidance, starter `MYCELIUM_MEMORY.md`, and lifecycle activity entries.
- **Bundled Go CLI engine.** The extension invokes `mycelium` through pi's `bash` tool. The CLI remains a separate, testable engine for filesystem safety and diagnostics.
- **Atomic writes with optimistic concurrency.** Mutations can use `--expected-version`; conflicts return structured JSON and exit 64.
- **Crash-aware durable history.** Mutations commit atomically and append a durable JSONL activity entry before reporting success.
- **Multi-agent safe.** Mount-level locking plus CAS lets sibling pi agents share a local POSIX journal.
- **Plain files.** Notes and logs are ordinary files you can inspect, diff, copy, and back up.
- **Self-evolution.** Agents revise `MYCELIUM_MEMORY.md` as durable conventions, lessons, index locations, archive policy, and open questions emerge.

## Install

```bash
# Global — available in every pi session, journal at ~/.pi/agent/extensions/pi-mycelium/journal/
pi install npm:pi-mycelium

# Project-local — journal at <cwd>/.pi/pi-mycelium/journal/
pi install npm:pi-mycelium -l
```

Verify with `pi list`. Update with `pi update npm:pi-mycelium`.

The npm package depends on platform-specific `@fuentesjr/mycelium-cli-*` optional packages and selects the one matching your OS/architecture. No separate binary or PATH setup is needed for supported pi installs.

## Direct CLI use

The `mycelium` binary remains documented for development, diagnostics, advanced operation, and for pi agents' normal shell-invoked memory operations. A source build is not a supported generic harness integration path.

Requires Go 1.26.2+:

```bash
git clone https://github.com/fuentesjr/mycelium.git
cd mycelium
make build
MYCELIUM_MOUNT="$(mktemp -d)" ./cmd/mycelium/mycelium ls
```

For diagnostics outside pi, set `MYCELIUM_MOUNT` to a journal directory and optionally set `MYCELIUM_AGENT_ID` / `MYCELIUM_SESSION_ID`. Existing journals remain compatible.

## Subcommands

| Tier | Command | Purpose |
| --- | --- | --- |
| Everyday | `read` | Read a note, optionally as JSON with version metadata |
| Everyday | `write` | Safe write with optional CAS and rationale |
| Everyday | `edit` | Safe unique-substring replacement |
| Everyday | `ls` | List journal entries, optionally by pattern |
| Everyday | `grep` | Search content and activity logs |
| Occasional | `rm` | Remove a note |
| Occasional | `mv` | Move/rename a note |
| Metadata | `log` | Append an agent-authored signal such as `decision` or `agent_note` |

## Quick example

```bash
export MYCELIUM_MOUNT=$(pwd)/.pi/pi-mycelium/journal
export MYCELIUM_AGENT_ID=pi-agent

printf 'incident: query latency spike at 14:30\n' \
  | mycelium write notes/incident-2026-07-12.md \
      --rationale "Capture symptoms before mitigation closes the window."

mycelium read notes/incident-2026-07-12.md --format json
mycelium grep --pattern latency --format json
mycelium log decision --rationale "Treat cache eviction as leading hypothesis." \
  --payload-json '{"path":"notes/incident-2026-07-12.md"}'
mycelium grep --path _activity --pattern '"op":"write"' --format json
```

## Activity log

Activity is plain JSONL at `<journal>/_activity/YYYY/MM/DD/<agent_id>.jsonl`. The pi extension records session boundaries, `session_shutdown`, and `compaction`. The CLI records successful `write`, `edit`, `rm`, `mv`, and explicit `log` entries. Historical journals may contain older event names; readers should tolerate unknown operations. See [`docs/pi-activity-events.md`](docs/pi-activity-events.md).

## What agents record

- File contents carry per-note reasoning.
- `--rationale` captures why a specific operation happened and appears on the activity log line.
- `MYCELIUM_MEMORY.md` carries current durable conventions and lessons; `_activity/` preserves how those rules changed over time.

## Documentation

- [`docs/faq.md`](docs/faq.md) — adoption and operation questions.
- [`extensions/pi-mycelium/README.md`](extensions/pi-mycelium/README.md) — pi extension details.
- [`docs/pi-activity-events.md`](docs/pi-activity-events.md) — current pi activity contract and compatibility notes.
- [`docs/mycelium-design.md`](docs/mycelium-design.md) — design rationale and storage contract.
- [`docs/mycelium-phases.md`](docs/mycelium-phases.md) — pi-focused roadmap.
- [`docs/benchmarks/phase-1.md`](docs/benchmarks/phase-1.md) — model benchmark rubric run through pi.
- [`docs/adr/`](docs/adr/) — architecture decision records.
- [`CHANGELOG.md`](CHANGELOG.md) — release notes.
- [`docs/release-checklist.md`](docs/release-checklist.md) — release checklist.

## Repository layout

- `cmd/mycelium/` — Go CLI engine for safe mutations, search, and activity appends.
- `extensions/pi-mycelium/` — supported pi extension, prompt, tests, and journal template.
- `docs/` — design, roadmap, benchmarks, ADRs, release notes.

## Development

```bash
npm install --prefix extensions/pi-mycelium
make test
make build
make dist
make clean
```

The repository uses [Jujutsu](https://docs.jj-vcs.dev/) (`jj`) co-located with git. Either toolchain works against the same history. See `AGENTS.md` for repository conventions.

## License

MIT. See [`LICENSE`](LICENSE).
