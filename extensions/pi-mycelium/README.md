# Mycelium Pi Extension

A pi extension that wires [Mycelium](https://github.com/fuentesjr/mycelium) persistent memory into pi coding-agent sessions. The platform-matching Go CLI engine is bundled through npm optional dependencies; supported users do not need a separate binary install.

## Install

```bash
pi install npm:pi-mycelium        # global journal
pi install npm:pi-mycelium -l     # project-local journal
```

Global journals live at `~/.pi/agent/extensions/pi-mycelium/journal/`. Project-local journals live at `<repo>/.pi/pi-mycelium/journal/`. Verify with `pi list` and update with `pi update npm:pi-mycelium`.

## What it does

- **`session_start`** resolves the bundled `mycelium` binary, prepends it to PATH, sets `MYCELIUM_MOUNT`, `MYCELIUM_AGENT_ID`, and `MYCELIUM_SESSION_ID`, bootstraps `MYCELIUM_MEMORY.md` when absent, and records a session-boundary activity entry.
- **`before_agent_start`** contributes concise system-prompt guidance: the journal path, command tiers, conventions-file rule, conflict recovery, reserved `_` paths, rationale guidance, and activity-log notes.
- **`session_shutdown`** records a shutdown entry before teardown.
- **`compaction`** records pi compaction metadata when pi reports it.

The extension registers no custom tools. Pi agents invoke `mycelium <subcommand>` through pi's built-in `bash` tool, the same way they run `git` or `rg`.

## Activity contract

The active pi contract is session boundaries, `session_shutdown`, `compaction`, core CLI mutation entries, and agent-authored `decision` / `agent_note` signals. Historical journals may contain older event names; readers should tolerate unknown operations. See [`docs/pi-activity-events.md`](https://github.com/fuentesjr/mycelium/blob/main/docs/pi-activity-events.md).

## What it does not do

- Does not support non-pi coding-agent harnesses. Direct CLI use is for pi shell operation, development, diagnostics, and advanced inspection.
- Does not prefetch, summarize, rank, or auto-inject memory contents. The agent reads and updates the journal deliberately.
- Does not sandbox the agent's shell. Mycelium protects journal mutations made through the CLI; pi and the OS control broader permissions.

## Mount location

| Install scope | Extension path | Mount path |
| --- | --- | --- |
| Global | `~/.pi/agent/extensions/` | `~/.pi/agent/extensions/pi-mycelium/journal/` |
| Project | `<repo>/.pi/extensions/` | `<repo>/.pi/pi-mycelium/journal/` |

Detection compares `import.meta.url` against `~/.pi/agent/extensions/`. A locally checked-out copy loaded with `pi -e ./path.ts` is treated as project-local.

## Identity

`MYCELIUM_AGENT_ID` defaults to `pi-agent`. Agent IDs are filename-safe ASCII using letters, digits, `.`, `_`, or `-`. Set it explicitly when running multiple concurrent agents against the same journal:

```bash
MYCELIUM_AGENT_ID=researcher pi
```

`MYCELIUM_SESSION_ID` uses `ctx.sessionManager.getLeafId()` when pi provides one. If not, the extension generates a `pi-auto-*` id; the CLI still has an `auto-*` fallback for diagnostics.

## Binary resolution and diagnostics

The extension prefers the bundled `@fuentesjr/mycelium-cli-<platform>` package. If optional dependencies were skipped or the platform is unsupported, it falls back to `which mycelium` on PATH. If neither exists, the injected prompt reports Mycelium as unavailable and the pi session continues without persistent memory.

For troubleshooting: exit 64 is a CAS/destination conflict with a JSON envelope; exit 65 is a protocol violation such as reserved `_` path mutation or oversize rationale.

## Development

```bash
git clone https://github.com/fuentesjr/mycelium
cd mycelium/extensions/pi-mycelium
npm install
npm test
pi -e ./index.ts
```

A stub binary at `stub/mycelium` returns canned successful JSON for extension tests and local smoke checks.
