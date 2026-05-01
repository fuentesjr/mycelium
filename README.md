# Mycelium

Persistent memory for AI coding agents. A small CLI plus a daily activity log on disk: agents read and write notes via `mycelium <subcommand>`, and the log preserves context across sessions, processes, and concurrent agents.

**Status:** early access (pre-1.0). The binary is correct and complete per the Phase 1 design — 338 tests including sibling-process CAS validation, property tests on the activity log, T3 failure-mode detectors, and a tarball-roundtrip test that pins the "plain files plus JSONL" contract. Benchmark validation against Frontier models runs against the released artifact rather than gating release; see `docs/benchmarks/phase-1.md`.

## What's here

- **`cmd/mycelium/`** — Go binary. Nine subcommands (`read`, `write`, `edit`, `ls`, `glob`, `grep`, `rm`, `mv`, `log`). Mount-level `flock`-guarded CAS, SHA-256 version tokens, JSONL activity log at `<mount>/_activity/YYYY/MM/DD/<agent>.jsonl`. Reserved `_`-prefix protects backend metadata from agent writes.
- **`extensions/pi/`** — pi.dev extension. Sets up env vars on `session_start`, contributes a system-prompt block on `before_agent_start`, records `context_signal` entries on `context`. Registers no tools — agents invoke `mycelium` through pi's built-in `bash`.
- **`docs/`** — design (`mycelium-design.md`), phasing (`mycelium-phases.md`), conflict-resolution conventions, self-evolution patterns, benchmark rubric.

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

```
cd extensions/pi
npm install
npm test    # 37 tests, ~1s

# Project install
ln -s "$(pwd)" <your-repo>/.pi/extensions/mycelium

# Global install
ln -s "$(pwd)" ~/.pi/agent/extensions/mycelium
```

Mount path is auto-detected from install location: project install mounts at `<cwd>/.pi/mycelium/store/`, global install at `~/.pi/mycelium/store/`. See `extensions/pi/README.md`.

## Quick example

```
export MYCELIUM_MOUNT=$(pwd)/.mycelium-store
export MYCELIUM_AGENT_ID=alice

# Write a note (atomic, returns version)
echo "incident: query latency spike correlates with deploys at 14:30" \
  | mycelium write notes/incident-2026-04-30.md
# {"version":"sha256:..."}

# Read it back
mycelium read notes/incident-2026-04-30.md

# Search
mycelium grep --pattern latency --format json

# Concurrent-safe update via CAS
VERSION=$(mycelium read notes/incident-2026-04-30.md | sha256sum | cut -d' ' -f1)
echo "updated content" | mycelium write notes/incident-2026-04-30.md \
  --expected-version sha256:$VERSION

# Inspect activity log directly — plain JSONL, no tooling required
cat $MYCELIUM_MOUNT/_activity/*/*/*/alice.jsonl
```

## Documentation

- [`mycelium-design.md`](mycelium-design.md) — design rationale, architecture, principles.
- [`mycelium-phases.md`](mycelium-phases.md) — phasing roadmap; what's in scope when, and why.
- [`docs/conflict-resolution.md`](docs/conflict-resolution.md) — multi-agent conflict-resolution conventions.
- [`docs/self-evolution.md`](docs/self-evolution.md) — convention bootstrap, self-built indices, archiving patterns.
- [`docs/benchmarks/phase-1.md`](docs/benchmarks/phase-1.md) — validation rubric, target models, scoring.

## Development

```
make test     # run the Go test suite (338 tests)
make build    # build the host-platform binary
make dist     # cross-compile release tarballs for darwin+linux x amd64+arm64
make clean    # remove build artifacts
```

The repository uses [Jujutsu](https://docs.jj-vcs.dev/) (`jj`) co-located with git. Either toolchain works against the same history. See `AGENTS.md` for repository conventions.

## License

MIT. See [`LICENSE`](LICENSE).
