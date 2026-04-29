# Mycelium Pi.dev Extension

A pi.dev extension that wires Mycelium agent memory into pi sessions.

## What it does

- On `session_start`: detects whether the `mycelium` binary is on PATH; sets
  `MYCELIUM_AGENT_ID` (default `pi-agent`), `MYCELIUM_SESSION_ID` (from pi's
  session leaf id via `ctx.sessionManager.getLeafId()`), and `MYCELIUM_MOUNT`
  for the agent's bash invocations.
- On `before_agent_start`: appends a system-prompt block introducing the
  `mycelium` subcommands, conventions, identity, and conflict semantics.
  The block chains off `event.systemPrompt` so other extensions' contributions
  are preserved.
- On `context`: records a `context_signal` entry to the activity log via
  `mycelium log` without modifying the agent's message stream.

## What it does not do

- It registers no tools. The agent invokes `mycelium <sub>` through pi's
  built-in `bash` tool, the same way it runs `git`, `rg`, or any other shell
  command. This is intentional — see `mycelium-design.md` section 1.
- It does not prefetch, summarize, or auto-inject memory hints. Self-evolution
  is an agent behavior, not a system feature (see design section 7).

## Mount location

Auto-detected from where the extension is installed:

- **Project install** — extension placed under `<repo>/.pi/extensions/`:
  mounts at `<cwd>/.pi/mycelium/store/`.
- **Global install** — extension placed under `~/.pi/agent/extensions/`:
  mounts at `~/.pi/mycelium/store/`.

The detection compares `import.meta.url` against `~/.pi/agent/extensions/`.
A locally-checked-out copy loaded via `pi -e ./path.ts` is treated as project.

## Identity

`MYCELIUM_AGENT_ID` defaults to `pi-agent`. Set it explicitly in your shell
environment (e.g. `MYCELIUM_AGENT_ID=researcher pi`) when running multiple
concurrent agents against the same store.

`MYCELIUM_SESSION_ID` is always taken from `ctx.sessionManager.getLeafId()` —
forks mint new ids automatically.

## Parallel development with the stub binary

The `stub/mycelium` script returns canned successful JSON for every subcommand,
allowing the extension to run end-to-end before the real Go binary lands:

```bash
chmod +x stub/mycelium
ln -s "$(pwd)/stub/mycelium" ~/.local/bin/mycelium
```

Replace with the real binary once available.

## Binary missing

If `mycelium` is not on PATH at session start, the extension contributes a
`UNAVAILABLE` system-prompt block instead, telling the agent that persistent
memory is configured but inactive until the binary is installed. Sessions
continue normally without memory.
