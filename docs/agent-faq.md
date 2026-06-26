# Mycelium Agent FAQ

> **For AI agents, not humans.** This is operational reference for agents
> running against a mycelium mount. If you're a person evaluating or operating
> mycelium, read [faq.md](faq.md) instead.

---

## Session lifecycle

### What do I do at session start?

Read `MYCELIUM_MEMORY.md` at the mount root. It is the current conventions
file: naming rules, durable user preferences, index locations, archive policy,
lessons, and open questions live there.

```bash
mycelium read MYCELIUM_MEMORY.md --format json
```

Consult task-relevant notes by path. Do not prefetch broadly.

### What do I do before finishing?

If you noticed a repeated pattern, mistake, durable user preference, naming
rule, useful index, stale region, or open question, update
`MYCELIUM_MEMORY.md` with `--rationale`. Do not leave durable lessons implicit.

Routine reads and writes are already in the activity log automatically. Use
`mycelium log decision|agent_note --rationale "..."` only for point-in-time
signals that should remain history but should not become standing conventions.

---

## Choosing where and how to record

### The user asked me to remember something — where does it go?

One-shot context for this task: write a note at a meaningful path. Durable
behavioral guidance ("always do X here"): update `MYCELIUM_MEMORY.md`.

### Should I update MYCELIUM_MEMORY.md or rely on the activity log?

Update `MYCELIUM_MEMORY.md` for current rules. The activity log is durable
history: it records that the file changed and why, but it is not the active rule
projection.

### Where should new notes live and how should I name them?

Follow `MYCELIUM_MEMORY.md` when it has guidance. Otherwise choose paths that
reflect content, for example `auth/session-token-rotation.md`, not
`2024-05-10-note.md`. Avoid any path starting with `_`.

### When do I use log vs just writing a note?

Write a **note** for content. Edit **`MYCELIUM_MEMORY.md`** for durable
conventions, lessons, index locations, archive policy, or open questions. Use
**`log`** for point-in-time signals with arbitrary op names, usually
`decision` or `agent_note`.

---

## Writes and conflicts

### When do I need --expected-version?

Always for `edit`: the substring being replaced is itself a claim about prior
state. Always for `rm` when you're removing because of content you observed.
Strongly recommended for `write` when revising a file you've read this session.
Can be omitted for the first write to a new path or when intentionally
clobbering. See [conflict-resolution.md](conflict-resolution.md).

### I got exit 64 — what now?

CAS conflict. Read the JSON envelope on stderr; it carries `current_version`.
Re-read the path with `mycelium read <path> --format json`, merge your intended
change with the current state, then retry with the new version token.

```bash
echo "merged content" | mycelium write notes/foo.md --expected-version sha256:<current>
```

### I don't know a file's current version — how do I get it cheaply?

```bash
mycelium read --format json notes/foo.md
# {"path":"notes/foo.md","version":"sha256:...","content":"..."}
```

### Why does writing under _activity/ fail?

All root paths beginning with `_` are reserved for system writes. `_activity/`
is read-only history for you; other `_` paths are internal implementation
details. To record a non-mutation event, use `mycelium log`.

---

## Self-evolution

### How do I query the rules currently in effect?

Read `MYCELIUM_MEMORY.md`. The current file is the active rule set.

### How do I retire or revise a convention?

Edit the relevant prose entry in `MYCELIUM_MEMORY.md`, or add a dated
replacement that explicitly says what it replaces. Include `--rationale` so the
activity log records why the standing guidance changed.

### How do I track a question that later becomes a lesson?

Keep an `Open Questions` section in `MYCELIUM_MEMORY.md` or a linked file. When
the question resolves, edit it into a lesson with rationale. The file contains
the current state; the activity log contains the transition.

---

## Recall

### Can I read _activity/ to recall what I did earlier?

Yes. It is plain JSONL. `cat`, `tail -f`, and `grep` all work directly.

```bash
cat $MYCELIUM_MOUNT/_activity/2026/05/10/*.jsonl
mycelium ls '_activity/2026/05/*/coder.jsonl' --recursive
mycelium grep --pattern 'notes/incidents' --path _activity --format json --limit 200
```

Reads of `_activity/` are not themselves logged. See
[portable-activity-events.md](portable-activity-events.md) for the event schema
and [mycelium-design.md](mycelium-design.md) for the path layout.
