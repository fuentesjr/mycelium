# Mycelium

**Persistent memory for AI coding agents.** A CLI and on-disk format
that lets agents keep notes across sessions, processes, and concurrent
runs — using plain files plus a JSONL activity log. No daemon, no
network, no database.

```
   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
   │ Claude Code  │  │      π       │  │   scripts    │
   └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
          │                 │                 │
          └─────────────────┼─────────────────┘
                            │  mycelium <subcommand>
                    ┌───────▼───────┐
                    │ mycelium CLI  │
                    └───────┬───────┘
                            │  atomic writes, CAS, _tx journal
                    ┌───────▼───────┐
                    │    mount/     │  ◀── git, grep, tar, cat
                    │  plain files  │      read this directly
                    └───────────────┘
```

## Why

AI coding agents lose context the moment a session ends. The usual
workarounds — system-prompt stuffing, vector stores, ad-hoc scratch
files — either don't survive across processes or hide context behind
opaque retrieval.

Mycelium gives agents a durable, inspectable filesystem they own:
`cat` reads it, `grep` searches it, `git` versions it, and multiple
agents can write to it concurrently without stepping on each other.

## Features

- **Atomic writes with optimistic concurrency.** Every write returns a
  SHA-256 version token. Pass it back as `--expected-version` for
  compare-and-swap; on conflict, Mycelium returns the current version
  (and optionally the current content) so the caller can re-merge
  without re-reading.
- **Crash-safe.** Content mutations and activity-log entries recover
  together via `_tx/pending/` journal records.
- **Multi-agent safe.** Mount-level `flock` plus CAS lets sibling
  processes share a mount without corruption.
- **Append-only activity log per agent.** Plain JSONL at
  `<mount>/_activity/YYYY/MM/DD/<agent>.jsonl` — `tail -f` works.
- **Self-evolution.** Agents record conventions and rationale with
  `mycelium evolve`, then query the active rules at any time.
- **Boring on disk.** Plain files in a directory you own. Inspect with
  `cat`, search with `grep`, version with `git`, back up with `tar`.

## How it fits together

A *mount* is a directory that holds an agent's notes plus its
`_activity/` log and `_tx/` journal. Agents — running in pi.dev,
Claude Code, a script, whatever — invoke `mycelium <subcommand>` to
read and write inside the mount. The reserved `_` path prefix keeps
agent writes from clobbering system metadata.

```
.mycelium-store/
├── notes/                                        ← agent-owned content
│   ├── incidents/
│   │   ├── 2026-04-30-query-latency-spike.md     ← mycelium evolve convention: <date>-<slug>.md
│   │   └── 2026-05-02-checkout-503s.md
│   ├── services/
│   │   ├── _index.md                             ← mycelium evolve index: services by team
│   │   ├── checkout-api.md
│   │   └── payments-worker.md
│   ├── reviews/
│   │   └── 2026-05-08-pr-1247.md
│   └── spikes/
│       └── 2026-Q1/                              ← mycelium evolve archive (no file changes)
│           └── caching-prototype.md
├── _lock                                         ← mount-level flock target
├── _activity/                                    ← append-only JSONL log per agent
│   └── 2026/05/09/
│       ├── coder.jsonl                           ← writes + evolve events
│       └── reviewer.jsonl
└── _tx/
    └── pending/                                  ← crash-recovery journal
```

## Subcommands

| Command | Group | Purpose |
|---|---|---|
| `read` | content | Read a note (optionally with version metadata) |
| `write` | content | Atomic write; returns version, supports CAS via `--expected-version` and optional `--rationale` |
| `edit` | content | In-place edit with the same CAS semantics as `write`; accepts `--rationale` |
| `rm` | content | Remove a note; accepts `--rationale` |
| `mv` | content | Move/rename a note; accepts `--rationale` |
| `ls` | discovery | List entries under a path |
| `glob` | discovery | Match notes by glob pattern |
| `grep` | discovery | Content search across the mount |
| `log` | meta | Append a signal entry to the activity log; accepts `--rationale` |
| `evolve` | meta | Record or query self-evolution events (conventions, indices, archives); `--rationale` is required |

## Concurrent writes

Two agents racing on the same file resolve via compare-and-swap. Each
write returns a SHA-256 version; pass it back as `--expected-version`
on the next write. On conflict, mycelium emits a JSON envelope with
`current_version` (and `current_content` if requested) so the caller
can re-merge in memory without a second read:

```
coder         mycelium         reviewer
  │              │                │
  │──write v1───▶│                │
  │◀───ok, v2────│                │
  │              │◀───write v1────│
  │              │─CONFLICT(64)──▶│
  │              │   current=v2   │   ← conflict envelope
  │              │  content="..." │     (caller has both fields)
  │              │                │
  │              │                │     re-merge in memory, no re-read
  │              │                │
  │              │◀───write v2────│
  │              │──ok, v3───────▶│
```

## Status

Early access (pre-1.0). Phase 1 is feature-complete and tested: atomic
CAS, transaction-journal recovery, property-checked activity log, and
the on-disk format contract. Benchmark validation against frontier
models runs against the released artifact rather than gating release;
see [`docs/benchmarks/phase-1.md`](docs/benchmarks/phase-1.md).

## Install

### Binary (from source)

Requires Go 1.26+.

```
git clone https://github.com/fuentesjr/mycelium.git
cd mycelium
make build
sudo install cmd/mycelium/mycelium /usr/local/bin/
mycelium    # prints subcommand list
```

Or via `go install`:

```
go install ./cmd/mycelium
```

### Pi extension

The extension ships on npm and bundles the platform-matching
`mycelium` binary as an optional dependency — no separate binary
install or PATH setup needed.

```
# Global — available in every pi session, mounts at ~/.pi/agent/extensions/pi-mycelium/journal/
pi install npm:pi-mycelium

# Or project-local — mounts at <cwd>/.pi/pi-mycelium/journal/
pi install npm:pi-mycelium -l
```

Verify with `pi list`. Updates: `pi update npm:pi-mycelium`.

The bundled binary takes precedence; if the optional dependency was
skipped (unsupported platform, `--omit=optional`), the extension falls
back to `which mycelium` on PATH and contributes an `UNAVAILABLE`
system-prompt notice if neither is found. See
[`extensions/pi-mycelium/README.md`](extensions/pi-mycelium/README.md)
for the full install / scope-detection / identity story.

## Quick example

```
export MYCELIUM_MOUNT=$(pwd)/.mycelium-store
export MYCELIUM_AGENT_ID=coder

# Write a note (atomic, returns version)
echo "incident: query latency spike correlates with deploys at 14:30" \
  | mycelium write notes/incident-2026-04-30.md
# {"version":"sha256:..."}

# Read it back
mycelium read notes/incident-2026-04-30.md

# Read content plus version for a future CAS update
mycelium read notes/incident-2026-04-30.md --format json
# {"path":"notes/incident-2026-04-30.md","version":"sha256:...","content":"..."}

# Search
mycelium grep --pattern latency --format json

# Record a self-evolution event — the agent's reasoning at the moment of decision
mycelium evolve convention \
  --target notes/incidents/ \
  --rationale "Adopting <date>-<slug>.md filenames so incidents sort chronologically without a separate index. Tried index.md first; it drifted from reality within a week."
# {"id":"01HXKP4Z9M8YV1W6E2RTSA9KFG"}

# View the current rules in effect across all kinds (NDJSON; one event per line)
mycelium evolve --active --format json
# {"ts":"2026-04-28T09:14:32Z","agent_id":"coder","session_id":"01HXJX2K7N9R5T2YQ8M3D1B6V4","op":"evolve","id":"01HXKP4Z9M8YV1W6E2RTSA9KFG","kind":"convention","target":"notes/incidents/","supersedes":"","kind_definition":"","rationale":"Adopting <date>-<slug>.md filenames so incidents sort chronologically without a separate index."}
# {"ts":"2026-05-01T14:22:09Z","agent_id":"coder","session_id":"01HXKM5R8P2Q6V3Z9N4S1T0Y7K","op":"evolve","id":"01HXKP6F3J8C2YV1W6E2RTSA9K","kind":"index","target":"notes/services/","supersedes":"","kind_definition":"","rationale":"Built _index.md grouped by team owner; lookups were dominated by 'whose service is this?'"}
# {"ts":"2026-05-05T16:08:51Z","agent_id":"coder","session_id":"01HXKQ8T9V3R5W4Y2N7Z1B6P0M","op":"evolve","id":"01HXKP9YQ7M2K8V1W6E2RTSA9F","kind":"archive","target":"notes/spikes/2026-Q1/","supersedes":"","kind_definition":"","rationale":"Archiving Q1 spikes; none referenced in 30+ days and they were drowning grep results for active work."}

# Concurrent-safe update via CAS — pass the prior version, retry on conflict (exit 64).
# On conflict, mycelium emits a JSON envelope with current_version (and current_content
# when --include-current-content is set) so the caller can re-merge without re-reading.
echo "updated content" | mycelium write notes/incident-2026-04-30.md \
  --expected-version sha256:abc123... --include-current-content

# Inspect activity log directly — plain JSONL, no tooling required
cat $MYCELIUM_MOUNT/_activity/*/*/*/coder.jsonl
```

A log entry — the keys are self-describing; the annotations explain the
value formats:

```
{"id":"01HXKP4Z9M","ts":"2026-05-09T15:32Z","kind":"write","path":"notes/inc.md","version":"sha256:abc...","rationale":"Capturing initial symptoms before mitigation closes the window."}
       │                 │                          │              │                        │                        │
       └─ ULID           └─ ISO timestamp           └─ event kind  └─ mount-relative        └─ post-write version    └─ optional; omitted when not supplied
```

## What agents record

A note's *what* is the cheap part — the file content, the diff. The
*why* is what survives across sessions and travels between agents.
Mycelium has three recording surfaces, and the discipline for all of
them is: **capture the rationale at the moment of decision, and name
what was rejected, not just what was chosen.**

**File contents** carry the per-note reasoning. Incident notes name
the trigger. Investigation notes name the hypothesis being tested.
Plan files name the alternatives considered and rejected. The same
craft as a good commit message, applied to every note. This is a
convention; the binary does not enforce it.

**Operational rationale** can now be captured on the activity log line
itself, at the moment of the mutation or signal, via `--rationale`:

```bash
mycelium write notes/incident-2026-05-12.md \
  --rationale "API began returning 503 at 14:22; recording symptoms before mitigation closes the window."

mycelium rm notes/spikes/2026-Q1/deprecated.md \
  --rationale "Spike concluded; superseded by notes/decisions/2026-04-cache-layer.md."

mycelium log decision \
  --rationale "Choosing Redis over Memcached for the cache layer; cluster mode and persistence outweigh the marginal latency cost." \
  --payload-json '{"chosen":"redis","rejected":["memcached","dragonfly"]}'
```

When supplied, `rationale` appears as a top-level field on the
activity log entry (`omitempty` — absent when not supplied). On a CAS
conflict, it also appears in the conflict envelope on stderr so the
retrying agent sees both sides' intent. Maximum 64 KiB. The note-body
discipline and operational `--rationale` are complementary: note bodies
hold *why-this-thing*, the flag holds *why-this-operation*.

**Self-evolution events** carry structural decisions — conventions
adopted, indices built, patterns archived — each recorded as a
first-class entry in the activity log:

```bash
# A decision with its alternative
mycelium evolve convention \
  --target notes/incidents/ \
  --rationale "Using <date>-<slug>.md filenames instead of an index.md catalog; the catalog drifted from reality within a week. Filename sort gives chronology for free."

# An index the agent built for itself
mycelium evolve index \
  --target notes/services/ \
  --rationale "Built _index.md grouped by team owner; lookups were dominated by 'whose service is this?'"

# An archive event with its threshold
mycelium evolve archive \
  --target notes/spikes/2026-Q1/ \
  --rationale "Archiving Q1 spikes; none referenced in 30+ days and they were drowning grep results for active work."
```

A future agent asking "why are incidents named this way?" gets the
original reasoning, not a guess. The reasoning is queryable: `mycelium
evolve --active` shows the rules currently in effect; the dated
activity log preserves the full history of how those rules came to be.

See [`docs/self-evolution.md`](docs/self-evolution.md) for the full
event vocabulary and
[`docs/portable-activity-events.md`](docs/portable-activity-events.md)
for the schema.

## Documentation

**Start here:**

- [`docs/faq.md`](docs/faq.md) — common questions about adopting, integrating, and operating mycelium.
- [`docs/mycelium-design.md`](docs/mycelium-design.md) — design rationale, architecture, principles.
- [`docs/self-evolution.md`](docs/self-evolution.md) — convention bootstrap, self-built indices, archiving patterns.

**Reference:**

- [`docs/mycelium-phases.md`](docs/mycelium-phases.md) — phasing roadmap; what's in scope when, and why.
- [`docs/conflict-resolution.md`](docs/conflict-resolution.md) — multi-agent conflict-resolution conventions.
- [`docs/portable-activity-events.md`](docs/portable-activity-events.md) — adapter event vocabulary and payload conventions.
- [`docs/benchmarks/phase-1.md`](docs/benchmarks/phase-1.md) — validation rubric, target models, scoring.
- [`docs/adr/`](docs/adr/) — architecture decision records.
- [`CHANGELOG.md`](CHANGELOG.md) — release notes.
- [`docs/release-checklist.md`](docs/release-checklist.md) — step-by-step guide for cutting a release.

## Repository layout

- **`cmd/mycelium/`** — the Go binary. Ten subcommands: content
  (`read`, `write`, `edit`, `rm`, `mv`), discovery (`ls`, `glob`,
  `grep`), and meta (`log`, `evolve`).
- **`extensions/pi-mycelium/`** — pi.dev extension. Wires Mycelium
  into pi sessions: env vars, a system-prompt block, and portable
  activity events. Registers no tools; agents invoke `mycelium`
  through pi's built-in `bash`.
- **`docs/`** — design, phasing, conflict-resolution, self-evolution,
  benchmarks, ADRs.

## Development

```
make test     # run the Go test suite
make build    # build the host-platform binary
make dist     # cross-compile release tarballs for darwin+linux x amd64+arm64
make clean    # remove build artifacts
```

The repository uses [Jujutsu](https://docs.jj-vcs.dev/) (`jj`)
co-located with git. Either toolchain works against the same history.
See `AGENTS.md` for repository conventions.

## License

MIT. See [`LICENSE`](LICENSE).
