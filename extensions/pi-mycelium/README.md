# Mycelium Pi.dev Extension

A pi.dev extension that wires [Mycelium](https://github.com/fuentesjr/mycelium)
agent memory into pi sessions. The mycelium binary is bundled per platform —
no manual install or PATH setup required.

## Install

Global install (mount at `~/.pi/agent/extensions/pi-mycelium/journal/`, available in every project):

```bash
pi install npm:pi-mycelium
```

Project-local install (mount at `<repo>/.pi/pi-mycelium/journal/`):

```bash
pi install npm:pi-mycelium -l
```

That's it. Run `pi` from any project and the agent gets persistent memory.
First invocation creates the mount directory automatically.

Verify with `pi list`. The platform-matching binary
(`@fuentesjr/mycelium-cli-<os>-<arch>`) is pulled in as an npm
optionalDependency on install — only the one that matches your platform
gets resolved.

## What it does

- **`session_start`** — resolves the bundled `mycelium` binary, sets
  `MYCELIUM_AGENT_ID` (default `pi-agent`), `MYCELIUM_SESSION_ID` (from
  `ctx.sessionManager.getLeafId()`), and `MYCELIUM_MOUNT` for the agent's
  bash invocations. Records a session-boundary entry in the activity log.
- **`session_shutdown`** — records a portable `session_shutdown` entry before
  the extension runtime is torn down.
- **`before_agent_start`** — appends a system-prompt block with the small public
  model — a folder, safe mutations, and a searchable activity log — then tiers
  the `mycelium` subcommands into everyday, occasional, and metadata commands.
  It also includes conventions, identity, conflict recovery, the project's
  evolution kinds, active evolution, and a rationale nudge for operations whose
  reasoning is worth preserving. Chains off `event.systemPrompt` so other
  extensions' contributions are preserved.
- **turn/tool/context events** — records `turn_start`, `turn_end`,
  `tool_start`, `tool_end`, `compaction`, and deduped `context_checkpoint`
  entries using the portable vocabulary in
  [`docs/portable-activity-events.md`](https://github.com/fuentesjr/mycelium/blob/main/docs/portable-activity-events.md),
  without modifying the agent's message stream.

The bundled binary writes durable history under `_activity/`. Other `_` paths
are internal implementation details used to keep content mutations and activity
entries consistent across crashes.

## What it does not do

- Registers no tools. The agent invokes `mycelium <sub>` through pi's
  built-in `bash` tool, the same way it runs `git`, `rg`, or any other shell
  command. This is intentional — see
  [`docs/mycelium-design.md`](https://github.com/fuentesjr/mycelium/blob/main/docs/mycelium-design.md)
  section 1 in the main repo.
- Does not prefetch, summarize, or auto-inject memory hints. Self-evolution
  is an agent behavior, not a system feature (see design section 7).

## Mount location

Auto-detected from where the extension is installed:

| Install scope | Extension path            | Mount path                                    |
| ------------- | ------------------------- | --------------------------------------------- |
| Global        | `~/.pi/agent/extensions/` | `~/.pi/agent/extensions/pi-mycelium/journal/` |
| Project       | `<repo>/.pi/extensions/`  | `<repo>/.pi/pi-mycelium/journal/`             |

Detection compares `import.meta.url` against `~/.pi/agent/extensions/`.
A locally-checked-out copy loaded via `pi -e ./path.ts` is treated as project.

## Identity

`MYCELIUM_AGENT_ID` defaults to `pi-agent`. Set it explicitly when running
multiple concurrent agents against the same store:

```bash
MYCELIUM_AGENT_ID=researcher pi
```

`MYCELIUM_SESSION_ID` is taken from `ctx.sessionManager.getLeafId()` when pi
provides one — forks mint new ids automatically. If pi does not provide one,
the extension generates a `pi-auto-*` id; the core CLI also has an `auto-*`
per-process fallback for non-pi shell use.

## Binary resolution

The extension prefers the bundled binary from the matching
`@fuentesjr/mycelium-cli-<platform>` optional dependency. If that's not
present (unsupported platform, or `--omit=optional` install), it falls
back to `which mycelium` on PATH. If neither is found, the system-prompt
block becomes a `UNAVAILABLE` notice — sessions continue normally without
memory.

## Development

Local checkout for hacking on the extension:

```bash
git clone https://github.com/fuentesjr/mycelium
cd mycelium/extensions/pi-mycelium
npm install
pi -e ./index.ts
```

A stub binary at `stub/mycelium` returns canned successful JSON for every
subcommand, useful for end-to-end testing without rebuilding the Go binary:

```bash
chmod +x stub/mycelium
ln -s "$(pwd)/stub/mycelium" ~/.local/bin/mycelium
```
