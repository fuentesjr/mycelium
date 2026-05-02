# Conflict Resolution Conventions

**Status:** Documentation only. Recovery guidance for the conflict envelope contract in section 6 and CLI section 4 of `mycelium-design.md`. The envelope shape and exit codes are mycelium's contract; this doc records what to do with them.

---

## The recovery pattern

> **Re-read, merge, retry.**

1. **Re-read** the path. Either parse `current_content` from the envelope (cheapest — requires `--include-current-content` on the original call), or call `mycelium read <path>` (one extra invocation, always works).

2. **Merge** the agent's intended change with the current state. Semantics depend on the operation:
   - **`write`:** combine the new content with the current. Most "writes" are appends or section additions — apply that intent against the current bytes rather than the bytes the agent expected to overwrite.
   - **`edit`:** re-locate the substring. If still present, retry with the same `--old`/`--new`. If gone, another agent already changed it — decide whether the change still makes sense.
   - **`mv` `destination_exists`:** read the destination, decide whether to overwrite (`mycelium rm <dst>` then `mv`) or pick a new path.
   - **`rm`:** the file changed since the agent observed it. Re-read; if it still merits deletion, retry; if not, abort.

3. **Retry** with a fresh `--expected-version` token from the re-read, or omit the flag entirely if the agent has decided to stop being pessimistic.

---

## When to use `--expected-version`

- **Always** for `edit` — the substring being replaced is itself a claim about prior state.
- **Always** for `rm` when the agent is removing because of *content* it observed.
- **Strongly recommended** for `write` when revising a file the agent has read this session. The version token comes from the most recent `_activity/` entry for the path, or from the stdout of the agent's own prior `write`/`edit`.
- **Often unnecessary** for the first write to a path the agent owns exclusively (e.g., `tasks/T-current/notes.md` in a single-agent session).

The flag is the agent's coordination knob. Never set by mycelium, never required.

---

## When recovery doesn't apply

Some conflicts can't be auto-recovered, and the agent should surface them rather than loop:

- **Repeated conflict after merge.** Another agent is actively writing the same path. Stop, log a `mycelium log race_observed --path <path>`, and let the user coordinate.
- **Semantic conflict.** The current content tells a different story than the agent expected — e.g., agent was about to add "TODO: investigate X" and current content says "DONE: investigated X." A merge that ignores the semantic contradiction is silently wrong; surface to the user.
- **`mv destination_exists` with valuable content.** Don't `rm` and retry blindly. Read the destination first; treat the resolution as a content decision, not a path decision.

---

## Quick reference

| Situation | Exit code | Stderr |
|---|---|---|
| Stale `--expected-version` on write/edit/rm/mv-src | 64 | JSON envelope, `error: "conflict"` |
| `mv` destination already exists | 64 | JSON envelope, `error: "destination_exists"` |
| Agent path under reserved `_` prefix | 65 | Plain text, contains `reserved` |
| Other failures (path escape, missing file, bad args) | 1 or 2 | Plain text |

The **conflict (64)** and **reservation (65)** exit codes are stable contracts; the JSON envelope shape on 64 is documented in section 4 of `mycelium-design.md`. Other non-zero exits are generic failures whose specific code and stderr text may evolve.
