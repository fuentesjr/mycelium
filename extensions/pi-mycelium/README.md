# Mycelium Pi Extension

A pi extension that wires [Mycelium](https://github.com/fuentesjr/mycelium) persistent memory into pi coding-agent sessions. The platform-matching Go CLI engine is bundled through npm optional dependencies; supported users do not need a separate binary install.

## Install

```bash
pi install npm:pi-mycelium        # global journal
pi install npm:pi-mycelium -l     # project-local journal
```

Global journals live at `~/.pi/agent/extensions/pi-mycelium/journal/`. Project-local journals live at `<cwd>/.pi/pi-mycelium/journal/`. Verify with `pi list` and update with `pi update npm:pi-mycelium`.

## Support matrix

| Platform | Architecture | Bundled CLI |
| --- | --- | --- |
| macOS | `arm64`, `x64` | Supported |
| Linux | `arm64`, `x64` | Supported |
| Windows or other OS/architectures | Any | Not shipped or supported |

Use a local POSIX filesystem. Network and synchronization filesystems such as
NFS, SMB, FUSE, iCloud, Dropbox, and OneDrive are outside the storage guarantee.
The package's pi peer dependency is intentionally unpinned; each release is
verified against its development baseline, not every pi version.

## What it does

- **`session_start`** resolves the bundled `mycelium` binary, prepends it to PATH, sets `MYCELIUM_MOUNT`, `MYCELIUM_AGENT_ID`, and `MYCELIUM_SESSION_ID`, then attempts to bootstrap `MYCELIUM_MEMORY.md` when absent and record a session-boundary activity entry.
- **`before_agent_start`** contributes concise system-prompt guidance: the journal path, command tiers, conventions-file rule, conflict recovery, reserved `_` paths, rationale guidance, and activity-log notes.
- **`session_shutdown`** attempts to record a shutdown entry before teardown.
- **`compaction`** attempts to record pi compaction metadata when pi reports it.

The extension registers no custom tools. Pi agents invoke `mycelium <subcommand>` through pi's built-in `bash` tool, the same way they run `git` or `rg`.

## Activity contract

The active pi vocabulary is session boundaries, `session_shutdown`, `compaction`, core CLI mutation entries, and agent-authored `decision` / `agent_note` signals. Extension lifecycle writes and first-file bootstrap are best-effort: their absence signals a failed hook, not a promise that pi startup must fail. Historical journals may contain older event names; readers should tolerate unknown operations. See [`docs/pi-activity-events.md`](https://github.com/fuentesjr/mycelium/blob/main/docs/pi-activity-events.md).

## What it does not do

- Does not support non-pi coding-agent harnesses. Direct CLI use is for pi shell operation, development, diagnostics, and advanced inspection.
- Does not prefetch, summarize, rank, or auto-inject memory contents. The agent reads and updates the journal deliberately.
- Does not sandbox the agent's shell. Mycelium protects journal mutations made through the CLI; pi and the OS control broader permissions.

## Mount location

| Install scope | Mount path |
| --- | --- |
| Global | `~/.pi/agent/extensions/pi-mycelium/journal/` |
| Project | `<cwd>/.pi/pi-mycelium/journal/` |

Project scope requires `pi-mycelium` to be registered in the current working
directory's `.pi/settings.json`, as `pi install ... -l` does. A package loaded
only with `pi -e` has no project registration and defaults to the global mount.
Legacy/manual copies already under `~/.pi/agent/extensions/` are also global.

## Identity

`MYCELIUM_AGENT_ID` defaults to `pi-agent`. Agent IDs are 1–128 filename-safe
ASCII characters using letters, digits, `.`, `_`, or `-`; `.` and `..` are
invalid. Set it explicitly when running multiple concurrent agents against the
same journal:

```bash
MYCELIUM_AGENT_ID=researcher pi
```

`MYCELIUM_SESSION_ID` uses `ctx.sessionManager.getLeafId()` when pi provides one. If not, the extension generates a `pi-auto-*` id; the CLI still has an `auto-*` fallback for diagnostics. Session IDs are activity metadata, not filenames, and the CLI does not apply the agent-ID filename grammar to them.

## Binary resolution and diagnostics

The extension prefers the bundled `@fuentesjr/mycelium-cli-<platform>` package. If optional dependencies were skipped or the platform is unsupported, it falls back to `which mycelium` on PATH. That fallback is for development and diagnostics; it does not add a platform to the support matrix. If neither exists, the injected prompt reports Mycelium as unavailable and the pi session continues without persistent memory.

After install, verify all three continuity signals:

```bash
pi list
test -f .pi/pi-mycelium/journal/MYCELIUM_MEMORY.md   # project-local install
find .pi/pi-mycelium/journal/_activity -type f -print
```

Use the global path from the mount table for a global install. If the prompt
reports `UNAVAILABLE`, check that optional dependencies were installed and that
a supported platform package is present. If the memory file or boundary entry
is missing, check journal permissions and run `mycelium ls` from pi's shell;
best-effort bootstrap/lifecycle failures do not abort the session.

For troubleshooting: exit 64 is a CAS/destination conflict with a JSON envelope; exit 65 is a protocol violation such as reserved `_` path mutation or oversize rationale.

## Development

```bash
git clone https://github.com/fuentesjr/mycelium
cd mycelium
npm install --prefix extensions/pi-mycelium
npm test --prefix extensions/pi-mycelium
pi install ./extensions/pi-mycelium -l --approve
pi
```

The repository-only `stub/mycelium` is an optional manual wiring helper. It is
excluded from the npm package, returns canned success, and does not validate
persistence, locking, activity logging, or error handling. Automated extension
tests build their own instrumented stubs instead. From the repository root, a
minimal shell-wiring check is:

```bash
PATH="$PWD/extensions/pi-mycelium/stub:$PATH" \
  mycelium write smoke.md </dev/null
```
