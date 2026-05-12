# Mycelium Agent FAQ

> **For AI agents, not humans.** This is operational reference for agents running against a mycelium mount. If you're a person evaluating or operating mycelium, read [faq.md](faq.md) instead. The pi-mycelium system prompt is your primary guidance channel; this FAQ goes deeper for when you need it.

---

## Contents

- [Session lifecycle](#session-lifecycle)
  - [What do I do at session start?](#what-do-i-do-at-session-start)
  - [What do I do at session end?](#what-do-i-do-at-session-end)
- [Choosing where and how to record](#choosing-where-and-how-to-record)
  - [The user asked me to remember something — where does it go?](#the-user-asked-me-to-remember-something--where-does-it-go)
  - [Should I update MYCELIUM_MEMORY.md or rely on the activity log?](#should-i-update-mycelium_memorymd-or-rely-on-the-activity-log)
  - [Where should new notes live and how should I name them?](#where-should-new-notes-live-and-how-should-i-name-them)
  - [When do I use log vs evolve vs just writing a note?](#when-do-i-use-log-vs-evolve-vs-just-writing-a-note)
- [Writes and conflicts](#writes-and-conflicts)
  - [When do I need --expected-version?](#when-do-i-need---expected-version)
  - [I got exit 64 — what now?](#i-got-exit-64--what-now)
  - [I don't know a file's current version — how do I get it cheaply?](#i-dont-know-a-files-current-version--how-do-i-get-it-cheaply)
  - [Why does writing under _activity/ fail?](#why-does-writing-under-_activity-fail)
- [Self-evolution](#self-evolution)
  - [--active vs --list — which do I want?](#--active-vs---list--which-do-i-want)
  - [Does my new evolve event automatically supersede a previous one?](#does-my-new-evolve-event-automatically-supersede-a-previous-one)
  - [I want to retire a question that's now a lesson — how?](#i-want-to-retire-a-question-thats-now-a-lesson--how)
- [Recall](#recall)
  - [Can I read _activity/ to recall what I did earlier?](#can-i-read-_activity-to-recall-what-i-did-earlier)
  - [How do I query the rules currently in effect?](#how-do-i-query-the-rules-currently-in-effect)

---

## Session lifecycle

### What do I do at session start?

Read `MYCELIUM_MEMORY.md` if it exists at the mount root — it's a prose summary of prior conventions. Then query the authoritative active rules:

```bash
mycelium evolve --active
```

Consult any notes your task references by path. That's it — don't prefetch broadly.

### What do I do at session end?

Record `evolve` events for any structural decisions you made this session: conventions adopted, lessons learned, indexes built, regions archived, questions opened or resolved. Routine reads and writes are already in the activity log automatically — you don't need to log them again.

---

## Choosing where and how to record

### The user asked me to remember something — where does it go?

Decide by durability. One-shot context for this task only: write a note at a meaningful path. Durable behavioral guidance ("always do X here"): emit an `evolve` event of the appropriate kind (`convention`, `lesson`, `index`, `archive`, or `question`).

### Should I update MYCELIUM_MEMORY.md or rely on the activity log?

The activity log is authoritative. `MYCELIUM_MEMORY.md` is a prose mirror — useful for human readability and agent onboarding, but when it disagrees with the log, the log wins. Update `MYCELIUM_MEMORY.md` when it serves clarity; don't treat it as primary state.

### Where should new notes live and how should I name them?

Your call. Choose paths that reflect content — `auth/session-token-rotation.md`, not `2024-05-10-note.md`. Group by topic, not date. Avoid any path starting with `_` (reserved for system directories).

### When do I use log vs evolve vs just writing a note?

Write a **note** for content. Use **`evolve`** for structural decisions you'll want to query later by kind (`--active`, `--list`). Use **`log`** to emit observability signals with arbitrary op names — it's typically called by adapters, not directly by agents, though `mycelium log decision --rationale "..."` is a legitimate direct use for a point-in-time operational decision that doesn't merit a full `evolve` record. If you're unsure between `log` and `evolve`, prefer `evolve` for anything you'll want to recall by kind.

See [portable-activity-events.md](portable-activity-events.md) for the `log` event vocabulary and [self-evolution.md](self-evolution.md) for `evolve` patterns.

### When should I supply --rationale on write/edit/rm/mv/log?

Whenever the operation has separable operational reasoning — why this mutation, why now. Routine appends, status updates, artifact saves, and cleanups need no rationale; omit the flag. Mutations that close a decision window (diagnosing an incident, removing a file because its successor was confirmed, renaming on hypothesis confirmation) benefit from it. The pi-mycelium system prompt will nudge you when it's appropriate.

When supplied, rationale appears as a top-level field on the activity log entry. On a CAS conflict it also surfaces in the conflict envelope on stderr. Maximum 64 KiB; oversize input is rejected with exit 65 before any mutation runs. `evolve` always requires `--rationale`; no change there.

---

## Writes and conflicts

### When do I need --expected-version?

Always for `edit` — the substring being replaced is itself a claim about prior state. Always for `rm` when you're removing because of content you observed. Strongly recommended for `write` when revising a file you've read this session. Can be omitted for the first write to a new path or when intentionally clobbering. See [conflict-resolution.md](conflict-resolution.md).

### I got exit 64 — what now?

CAS conflict. Read the JSON envelope on stderr — it carries `current_version` and, if you passed `--include-current-content`, `current_content`. Merge your intended change with the current state in memory, then retry with the new version token. See [conflict-resolution.md](conflict-resolution.md) for the full re-read/merge/retry pattern and cases where recovery doesn't apply.

```bash
# Retry after a conflict — use the current_version from the envelope
echo "merged content" | mycelium write notes/foo.md --expected-version sha256:<current>
```

### I don't know a file's current version — how do I get it cheaply?

```bash
mycelium read --format json notes/foo.md
# {"path":"notes/foo.md","version":"sha256:...","content":"..."}
```

One call returns content and version together. You never need a separate stat round-trip.

### Why does writing under _activity/ fail?

`_activity/` and `_tx/` are reserved for system writes. Agent writes to any `_`-prefixed path return exit 65. To record an event, use `mycelium log` or `mycelium evolve` — the binary writes to `_activity/` on your behalf.

---

## Self-evolution

### --active vs --list — which do I want?

`--active` shows the rules currently in effect — superseded entries are dropped. `--list` shows the full timeline including superseded entries. Default to `--active` when deciding how to behave; use `--list` when investigating history.

```bash
mycelium evolve --active            # what rules apply right now
mycelium evolve --list              # full timeline including retired entries
mycelium evolve --active --kind convention   # narrow to one kind
```

See [self-evolution.md](self-evolution.md) for query patterns.

### Does my new evolve event automatically supersede a previous one?

Only when `--target` is non-empty and the new event has the same `(kind, target)` as an active prior event — then the prior is superseded automatically and the response includes `"supersedes":"..."`. Targetless events are additive and never implicitly supersede each other. Cross-kind retirement (e.g., a `question` resolved by a `lesson`) requires `--supersedes` explicitly. See [self-evolution.md](self-evolution.md).

### I want to retire a question that's now a lesson — how?

Emit a new `lesson` event with `--supersedes` pointing at the question's ID. Cross-kind retirement is never implicit — you must be explicit:

```bash
mycelium evolve lesson \
  --target hypotheses/glp1-cardio.md \
  --supersedes 01HXKP4Z9M8YV1W6E2RTSA9KFG \
  --rationale "GLP-1 cardio protection confirmed for non-diabetic populations across 4 independent studies."
```

The question's ID comes from the `id` field in the original `evolve` response or from `mycelium evolve --list`. See [self-evolution.md](self-evolution.md) for the full pattern.

---

## Recall

### Can I read _activity/ to recall what I did earlier?

Yes — it's plain JSONL. `cat`, `tail -f`, and `grep` all work directly. Filter by date using the path structure:

```bash
# All entries today, all agents
cat $MYCELIUM_MOUNT/_activity/2026/05/10/*.jsonl

# Your agent's entries this month
mycelium glob '_activity/2026/05/*/coder.jsonl'

# Search for a specific path across the log
mycelium grep --pattern 'notes/incidents' --path _activity --format json --limit 200
```

Reads of `_activity/` are not themselves logged. See [portable-activity-events.md](portable-activity-events.md) for the event schema and [mycelium-design.md](mycelium-design.md) for the path layout.

### How do I query the rules currently in effect?

```bash
mycelium evolve --active
mycelium evolve --active --format json
mycelium evolve --active --kind convention
```

`--active` returns the latest non-superseded targeted entry per `(kind, target)` pair, plus all targetless entries that haven't been explicitly superseded. Use `--kind` to narrow to one kind. See [self-evolution.md](self-evolution.md).
