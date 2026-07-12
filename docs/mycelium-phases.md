# Mycelium: Pi-Focused Roadmap

**Status:** Roadmap, companion to `mycelium-design.md`.

Mycelium is now scoped to pi coding agents. The roadmap keeps the LocalFS/POSIX guarantee set, the Go CLI engine, and journal compatibility while improving the supported pi product path.

## Phase 1 — Core and pi extension (current)

**Goal.** Prove durable agent memory under realistic pi conditions: multi-session use, concurrent agents sharing a local journal, and self-revision through `MYCELIUM_MEMORY.md`.

**Shipped / in scope.**

- Go CLI engine: `read`, `write`, `edit`, `ls`, `grep`, `rm`, `mv`, `log`.
- Local POSIX journal with atomic single-file mutations, CAS, typed conflicts, reserved `_` paths, and durable JSONL activity entries.
- pi extension install through `pi install npm:pi-mycelium`, with bundled platform CLI packages, scope-aware journal paths, identity env vars, prompt guidance, and starter template bootstrap.
- Pi activity contract: session boundaries, `session_shutdown`, `compaction`, core mutation entries, and agent-authored `decision` / `agent_note` signals.
- Benchmarks run through pi against multiple frontier models. Model diversity evaluates model behavior; it is not a multi-harness support claim.

**Acceptance criteria.**

1. A pi agent can resume a multi-session research task using only the journal and shipped prompt/template guidance.
2. Concurrent pi agents on one LocalFS journal can resolve overlapping edits through CAS without silent loss.
3. Self-evolution happens through conventions-file edits with rationale and activity-log evidence.
4. Failure-mode detectors can distinguish healthy and dysfunctional traces from journal contents.
5. `go test ./...`, extension tests, race checks, and npm package inspection pass.

## Phase 2 — Pi distribution and operational polish

**Goal.** Make the supported pi path easy to install, diagnose, and trust.

**In scope.**

- Clear pi install/update/troubleshooting docs.
- Better diagnostics for missing platform optional dependencies, missing binary fallback, unavailable memory, and legacy `_tx/pending/*.json` records.
- Optional read-byte caps or warnings only if pi benchmark runs show practical failures.
- Continued platform CLI npm packages and binary archives because they ship and debug the engine used by pi.
- Concise activity documentation for the current pi lifecycle contract and historical-event tolerance.

**Out of scope.**

- New non-pi harness integrations.
- Cross-harness adapter APIs or portable event vocabularies.
- Rewriting the Go engine in TypeScript absent evidence that the process boundary is the bottleneck.

## Phase 3 — Workflow integration

**Goal.** Fit pi journals into normal engineering workflows without weakening the LocalFS contract.

**In scope.**

- Opt-in git/jj integration for journal snapshots or per-operation commits.
- Historical reads for git/jj-backed journals, if the workflow earns the complexity.
- Activity-log retention policy surfaced as plain text.
- Curated pi journal templates and an optional `mycelium init` for development/diagnostic use.

**Acceptance criteria.**

1. A journal can be inspected and transferred with standard filesystem and VCS tools.
2. A different pi agent can resume from the transferred journal without migration.
3. Retention and template behavior stays visible in normal files.

## Always absent

Mycelium still does not provide automatic memory extraction, vector retrieval as the primary access path to the agent's own memory, opaque databases, automatic summarization, schema enforcement on agent-authored notes, or system-driven reflection. The agent owns those decisions; Mycelium provides the safe file substrate.
