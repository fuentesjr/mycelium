# Conflict Resolution Conventions

**Status:** Documentation only — not enforced, not auto-injected into agent context.
**Audience:** Operators integrating Mycelium and contributors reading the codebase. The pi.dev system-prompt block already names the recovery convention; this doc records the rationale and the corner cases.
**Source:** Concurrency contract from `mycelium-design.md` § 6. Conflict-envelope shape from § 8 / Phase 1 acceptance criterion 3.

---

## The contract

`mycelium write`, `edit`, `rm`, and `mv` accept an optional `--expected-version <sha256>` flag. When the flag is present and the stored version doesn't match (or, for `mv`, the destination already exists), the binary exits **64** and prints one line of JSON to stderr:

```json
{"error":"conflict","op":"write","path":"foo.md","current_version":"sha256:...","expected_version":"sha256:..."}
{"error":"destination_exists","op":"mv","path":"dst.md","current_version":"sha256:..."}
```

Adding `--include-current-content` extends the envelope with `current_content` (the file's bytes inline, UTF-8 only — omitted for non-UTF-8 files):

```json
{"error":"conflict","op":"write","path":"foo.md","current_version":"sha256:...","expected_version":"sha256:...","current_content":"hello, world\n"}
```

The envelope is **a single line** terminated with `\n`. It fits in one PIPE_BUF, so even concurrent writers to stderr don't interleave it with other output. Parse the first line of stderr as JSON; ignore subsequent lines (none today, but reserved for future diagnostics).

---

## The recovery pattern

> **Re-read, merge, retry.**

Three steps, in order:

1. **Re-read** the path. Either parse the `current_content` field from the envelope (cheapest), or call `mycelium read <path>` (one extra invocation, but always works regardless of envelope contents). The result is the version the agent's write is now competing with.

2. **Merge** the agent's intended change with the current state. What "merge" means depends on the operation:
   - For `write`: combine the agent's new content with the current content. Frequently the agent's write was a small append or section addition — apply that intent against the current bytes rather than the bytes the agent expected to overwrite.
   - For `edit`: re-locate the substring the agent wanted to replace. If it still exists in the current content, retry with the same `--old`/`--new`. If the substring is gone — another agent already changed it — decide whether the change still makes sense.
   - For `mv` `destination_exists`: read the destination, decide whether to overwrite (issue a `mycelium rm <dst>` then `mv` again, or `write` directly), or pick a new destination path.
   - For `rm`: a CAS conflict on `rm` means the file changed since the agent observed it. Re-read; if the new content still merits deletion, retry; if not, abort the deletion.

3. **Retry** with a fresh `--expected-version` token taken from the re-read, or omit the flag entirely if the agent has decided to stop being pessimistic.

---

## When to use `--expected-version`

Use it whenever the agent's write logically depends on the prior state:

- **Always** for `edit` — the substring being replaced *is* a claim about prior state.
- **Always** for `rm` when the agent is removing because of *content* it observed.
- **Strongly recommended** for `write` when appending to or revising a file the agent has read in the same session. `mycelium read` prints raw content only — to obtain a version token the agent either consults the most recent `_activity/` entry for the path (`mycelium grep --pattern '"path":"foo.md"' --path _activity --format json --limit 1`) or remembers the token returned in stdout from its own prior `write`/`edit` of that file.
- **Often unnecessary** for the first write to a path the agent owns exclusively (e.g. `tasks/T-current/notes.md` in a single-agent session). Skip the flag and accept last-writer-wins; the activity log still records both writes if a race somehow occurred.

The flag is the agent's coordination knob. It's never set by the binary on the agent's behalf, and never required.

---

## When recovery doesn't apply

Some conflicts can't be auto-recovered, and the agent should surface them to the user instead of looping:

- **Repeated conflict after merge.** If the same write conflicts a second time after a merge attempt, another agent is actively writing the same path. Stop, log a `mycelium log race_observed --path <path>`, and let the user decide whether to coordinate manually or accept the other agent's writes.
- **Semantic conflict.** The current content tells a different story than the agent expected. The agent's intended write may no longer make sense — e.g. it was about to add a "TODO: investigate X" line, and the current content already says "DONE: investigated X." A merge that ignores the semantic contradiction is silently wrong; surface to the user instead.
- **`mv destination_exists` with valuable content.** Don't `rm` and retry blindly. Read the destination first; treat the resolution as a content decision, not a path decision.

---

## Why CAS, not locks

`flock`-based locking was considered and rejected for the agent-facing surface. Locks introduce timeouts, deadlocks, and lifecycle questions (what if the agent crashes holding the lock?). CAS via versioned writes degrades cleanly: a conflict is an error message the agent reads, reasons about, and handles with the same primitives it uses for everything else. The binary uses `flock` internally to serialize the read-check-write sequence on a single host, but that's an implementation detail; the agent's contract is the version token plus the typed conflict error.

---

## Identity and the activity log

Every conflict is traceable. The activity-log entry that "won" — the write that produced the `current_version` the agent's stale token didn't match — is in `_activity/YYYY/MM/DD/<winning_agent>.jsonl`. To find it:

```
mycelium grep --pattern '"path":"foo.md"' --path _activity --format json --limit 50
```

The matching entries include `agent_id`, `session_id`, and `ts`. If the agent recovered without surfacing the conflict, the trail is preserved for an operator who later wonders "who wrote this?"

---

## Quick reference

| Situation | Exit code | Stderr |
|---|---|---|
| Stale `--expected-version` on write/edit/rm/mv-src | 64 | JSON envelope, `error: "conflict"` |
| `mv` destination already exists | 64 | JSON envelope, `error: "destination_exists"` |
| Agent path under reserved `_` prefix | 65 | Plain text, contains `reserved` |
| Other failures (path escape, missing file, bad args, unknown subcommand) | 1 or 2 | Plain text |

The **conflict (64)** and **reservation (65)** exit codes are stable contracts; the JSON envelope shape on 64 is documented above. Other non-zero exits are generic failures whose specific code and stderr text may evolve.
