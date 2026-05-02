# Mycelium

Persistent memory for AI coding agents. A small CLI plus a daily activity log on disk: agents read and write notes via `mycelium <subcommand>`, and the log preserves context across sessions, processes, and concurrent agents.

**Status:** early access (pre-1.0). The binary is correct and complete per the Phase 1 design — 393 tests including sibling-process CAS validation, property tests on the activity log, T3 failure-mode detectors, and a tarball-roundtrip test that pins the "plain files plus JSONL" contract. Benchmark validation against Frontier models runs against the released artifact rather than gating release; see `docs/benchmarks/phase-1.md`.

## What's here

- **`cmd/mycelium/`** — Go binary. Eleven subcommands (`read`, `write`, `edit`, `ls`, `glob`, `grep`, `rm`, `mv`, `log`, `evolve`, `evolution`). Mount-level `flock`-guarded CAS, SHA-256 version tokens, JSONL activity log at `<mount>/_activity/YYYY/MM/DD/<agent>.jsonl`. Reserved `_`-prefix protects backend metadata from agent writes.
- **`extensions/pi-mycelium/`** — pi.dev extension. Sets up env vars on `session_start`, contributes a system-prompt block on `before_agent_start`, records `context_signal` entries on `context`. Registers no tools — agents invoke `mycelium` through pi's built-in `bash`.
- **`docs/`** — design (`docs/mycelium-design.md`), phasing (`docs/mycelium-phases.md`), conflict-resolution conventions, self-evolution patterns, benchmark rubric.

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

The extension ships on npm and bundles the platform-matching `mycelium` binary as an optional dependency — no separate binary install or PATH setup needed.

```
# Global — available in every pi session, mounts at ~/.pi/mycelium/store/
pi install npm:pi-mycelium

# Or project-local — mounts at <cwd>/.pi/mycelium/store/
pi install npm:pi-mycelium -l
```

Verify with `pi list`. Updates: `pi update npm:pi-mycelium`.

The bundled binary takes precedence; if the optional dependency was skipped (unsupported platform, `--omit=optional`), the extension falls back to `which mycelium` on PATH and contributes an `UNAVAILABLE` system-prompt notice if neither is found. See `extensions/pi-mycelium/README.md` for the full install / scope-detection / identity story.

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

# Record a self-evolution event — the agent's reasoning at the moment of decision
mycelium evolve convention \
  --target notes/incidents/ \
  --rationale "Adopting <date>-<slug>.md filenames so incidents sort chronologically without a separate index."
# {"id":"01HXKP4Z9M8YV1W6E2RTSA9KFG"}

# View the current rules in effect across all kinds
mycelium evolution --active --format json

# Concurrent-safe update via CAS — pass the prior version, retry on conflict (exit 64).
# On conflict, mycelium emits a JSON envelope with current_version (and current_content
# when --include-current-content is set) so the caller can re-merge without re-reading.
echo "updated content" | mycelium write notes/incident-2026-04-30.md \
  --expected-version sha256:abc123... --include-current-content

# Inspect activity log directly — plain JSONL, no tooling required
cat $MYCELIUM_MOUNT/_activity/*/*/*/alice.jsonl
```

## Documentation

- [`docs/mycelium-design.md`](docs/mycelium-design.md) — design rationale, architecture, principles.
- [`docs/mycelium-phases.md`](docs/mycelium-phases.md) — phasing roadmap; what's in scope when, and why.
- [`docs/conflict-resolution.md`](docs/conflict-resolution.md) — multi-agent conflict-resolution conventions.
- [`docs/self-evolution.md`](docs/self-evolution.md) — convention bootstrap, self-built indices, archiving patterns.
- [`docs/benchmarks/phase-1.md`](docs/benchmarks/phase-1.md) — validation rubric, target models, scoring.
- [`docs/adr/`](docs/adr/) — architecture decision records.
- [`CHANGELOG.md`](CHANGELOG.md) — release notes.
- [`docs/release-checklist.md`](docs/release-checklist.md) — step-by-step guide for cutting a release.

## Development

```
make test     # run the Go test suite (393 tests)
make build    # build the host-platform binary
make dist     # cross-compile release tarballs for darwin+linux x amd64+arm64
make clean    # remove build artifacts
```

The repository uses [Jujutsu](https://docs.jj-vcs.dev/) (`jj`) co-located with git. Either toolchain works against the same history. See `AGENTS.md` for repository conventions.

## License

MIT. See [`LICENSE`](LICENSE).
