# Mycelium Simplification Epic

**Status:** Implemented through Stage 7; release prep in progress for v0.4.0 (2026-06-26)
**Theme:** Collapse the system onto its own public model — "a folder + safe mutations + a searchable activity log." Everything below either makes that sentence truer or deletes something that is not in it.

## Summary

The review of design docs and implementation (2026-06-11) found that the load-bearing core — CAS, atomic mutations, the JSONL activity log, reserved-prefix protection, the no-tools adapter stance — is sound and well tested, while three subsystems violate the project's first principle, and the 24k-word doc surface has begun drifting from the ~3k-line implementation (the documented ripgrep fallback chain does not exist; the T3 writes-without-reads detector cannot work on real traces).

Three decisions were made:

1. **Conventions-as-files (ADR-0004, supersedes ADR-0001).** Remove `mycelium evolve`. "Current rules in effect" is mutable state; storing it in the append-only log forced supersession chains, ULIDs, a kind registry, and four query projections. The conventions file (`MYCELIUM_MEMORY.md`) becomes the single source of truth: editing it is supersession, reading it is `--active`, and the activity log plus `--rationale` (ADR-0003) already record the history and the why. ADR-0001's archaeology justification was retired by ADR-0003 eleven days after acceptance; the T2 rubric already accepts convention edits as a pass signal. Kinds move from code registry to documented vocabulary — the same move ADR-0002 made for adapter events, and the maximally open taxonomy.
2. **Activity log is durable history, not a transactional ledger (ADR-0005).** Remove the `_tx/` journal. It protects only log completeness in the microsecond window between content rename and log append, at the cost of 21% of the binary, five fsyncs per write, and the only failure mode that can freeze a mount. Nothing derives state from the log once `evolve` is gone. New contract: lock, CAS check, content commit, log append, success — append failure is loud (non-zero exit), and the power-loss gap is documented honestly.
3. **Reference adapter records memory-relevant events only (ADR-0006).** pi-mycelium keeps session boundaries, `compaction`, and deduped `context_checkpoint`; it stops recording `turn_start/end`, `tool_start/end`, and the usage/cost/assistant payload introspection (48% of `activity-log.ts`). The portable-events vocabulary (ADR-0002) is unchanged — other adapters may still emit the full set; the reference adapter just emits what the memory loop uses.

Net effect: Go ~3,050 → ~1,450 production lines and ~6,300 → ~3,900 test lines; extension ~1,060 → ~650 lines; functional subcommands 10 → 8 (plus a hidden transitional `evolve` diagnostic stub until 1.0); per-session prompt ~1,000 → ~400 tokens; docs ~24k → ~15k words; external Go dependencies 1 → 0. Unchanged: CAS semantics, atomic mutations, conflict envelopes, the reserved `_` prefix, JSONL log format and `tx_id` on mutation entries, `--rationale` capture, existing mounts and logs.

## Why now

Phase 2 (Claude Code skill, Hermes plugin — see `mycelium-skill-adapter-plan.md`) multiplies every concept across new adapters and guidance surfaces. Simplifying first means distribution lands on eight verbs and one guidance source instead of ten verbs and five documents explaining the same machinery. Benchmarks T1/T2 have not run yet, so rubric edits are still free.

## Stages

Each stage is one reviewable jj change, independently shippable, in this order. Stages 1–2 are pure subtraction; 3–5 implement the ADRs; 6–7 land distribution and consolidate docs.

### Stage 0 — ADRs

- `docs/adr/0004-conventions-as-files.md` — records decision 1; sets ADR-0001 status to "Superseded by ADR-0004".
- `docs/adr/0005-activity-log-as-durable-history.md` — records decision 2; rewrites the guarantee language ("authoritative" → "durable, append-only, fail-loud; bounded gap under power loss").
- `docs/adr/0006-reference-adapter-memory-relevant-events.md` — records decision 3; notes ADR-0002's vocabulary is unchanged.

### Stage 1 — Truth and dead weight (no behavior change)

- Fix doc drift: remove the ripgrep/grep fallback claim (`docs/mycelium-design.md` sections 4 and 5, `docs/mycelium-phases.md` Phase 1 scope) — grep is and stays pure Go (one search implementation, one regex dialect, deterministic across machines). Add exit code 2 (`ExitUsage`) to the design doc's failure modes.
- Move `internal/mycelium/detect.go`, `detect_test.go`, and `testdata/trajectories/` out of the product package into benchmark tooling (e.g. `docs/benchmarks/tasks/T3-failure-detectors/tool/` as a standalone test package). Drop detector 1 (writes-without-reads): its denominator, `op:"read_signal"`, is emitted by nothing and is absent from the portable vocabulary — it can only classify its own fixtures. T3 proceeds on detectors 2 and 3 (near-duplicate paths, thrashing); update `docs/benchmarks/phase-1.md` and the T3 harness doc accordingly.
- Extension: delete the legacy exports `recordContextSignal`, standalone `recordContextCheckpoint`, and standalone `recordSessionBoundary` (`activity-log.ts:197-255`, none called by `index.ts`); plan to move `resolveBinary`/`setupEnv` off `env.ts` first (it is currently imported by `index.ts`) before deleting that file.

### Stage 2 — CLI surface trim (10 → 9 verbs)

- Fold `glob` into `ls`: `mycelium ls [pattern] [--recursive]` — no pattern lists as today; with a pattern, matches it (`**/*.md` etc.). One WalkDir loop instead of two near-identical ones in `listing.go`.
- Drop `--file-type` from grep (`--path` covers the real cases).
- Drop `--include-current-content` from write/edit/rm/mv. By the project's own razor it should not have shipped, but this is explicitly a weaker conflict-recovery loop: `read --format json` one command later can observe a newer version under concurrency, not the same-conflict bytes. Stage 2 updates conflict docs/tests around re-read/merge/retry semantics; if same-conflict content remains required, defer removal by one release instead.
- CHANGELOG entries marking the breaking changes for the simplification release (pre-1.0; README already warns the surface may shift).

### Stage 3 — Conventions-as-files (ADR-0004; 9 → 8 functional verbs)

Binary:

- Delete `evolve.go`, `evolution.go`, `kinds.go`, their tests (~770 production, ~1,600 test lines), and the `evolve` handler in `cli.go`. Keep `appendActivityLineDurable` through Stage 3 because `_tx` recovery still needs `appendActivityEntryDurable`; final deletion/cleanup moves to Stage 4 if unused.
- Add a hidden transitional `evolve` diagnostic stub: exit 1 with one line — "evolve was removed in 0.4.0; record conventions in your conventions file (see MYCELIUM_MEMORY.md)". It is not a functional verb and should not appear in the normal help surface; remove it at 1.0. This prevents confusing failures in mounts whose seeded templates still mention evolve.
- Move shared constants (for example `maxRationaleSize`) that evolve callers still need before deleting the evolve files.
- Restate `activity_scan.go` ownership so Stage 3 can ship: keep it while `_tx` recovery still needs it, and perform the final delete sweep after Stage 4 when tx recovery scan is removed.

Extension:

- `index.ts` `before_agent_start` stops invoking `evolve --kinds` / `evolve --active`; `mycelium.ts` NDJSON helper goes if unused after this.
- `system-prompt.ts`: the three evolve sections (kinds table, active evolution, recording guide — roughly 40% of the block) are replaced by one short conventions-file section: read `MYCELIUM_MEMORY.md` at session start; record durable rules by editing it with `--rationale`; `mycelium log decision|agent_note --rationale` remains for point-in-time signals.
- `extensions/pi-mycelium/templates/MYCELIUM_MEMORY.md` rewrite: conventions live in this file (dated prose entries); revise by editing; the activity log records every change and its rationale; no divergence policy (there is nothing to diverge from); keep the starter layout and activity-log reading recipes.

Docs:

- `docs/mycelium-design.md` section 7 rewritten around files-as-conventions (shorter); CLI section 4 and architecture diagrams drop evolve.
- `docs/self-evolution.md` rewritten at roughly a third of its size: the six patterns survive with file-based mechanics (bootstrap = read the file; revise = edit with rationale; index/archive = build/move files and note them in the conventions file or a `log decision` entry; lessons/questions = sections or files such as `learnings/`).
- `docs/faq.md`, `docs/agent-faq.md`, README: evolve references replaced; the log-vs-evolve-vs-note decision tree disappears.
- Benchmarks: T2 `task.md`/`grading.md` reworded (pass signals: convention-file edit, index file, grep-before-write — already three of the four listed; the evolve event option is dropped).

Compatibility: existing logs keep their `op:"evolve"` lines as valid, tolerated history (readers are liberal per ADR-0002's stance). No store migration.

### Stage 4 — Remove the transaction journal (ADR-0005)

- Delete `tx.go` (354 lines) and `tx_test.go`; slim `mutate_tx.go` (288 → ~120 lines, rename to `mutate.go`): each mutation becomes resolve → lock → CAS check → atomic content op → durable log append → success. No pending records, no recovery scan, no refuse-mutations mode. Append failure exits non-zero with content committed — loud, documented.
- The reserved `_` prefix stays (still protects `_activity/` and `_lock`).
- Preserve JSONL schema shape while removing `_tx`: `tx_id` remains on mutation entries, and generation switches from ULID to stdlib time/randomness (`tx-<zero-padded-unix-nano>-<rand>`), preserving timestamp sortability.
- Replace the remaining default-session ULID generator in `identity.go` with stdlib time/randomness (`auto-<zero-padded-unix-nano>-<rand>`) before deleting `ulid.go` and the `oklog/ulid` entries from `go.mod`/`go.sum`.
- Simplification-release legacy compatibility for `_tx/pending/*.json`: preflight reject with loud instructions to run documented recovery before normal operations. This requirement has explicit docs and tests.
- Characterization tests added before deletion: mutation ordering (no success without a durable append attempt), append-failure behavior, CAS semantics unchanged, and `_tx/pending/*.json` compatibility behavior. Crash-recovery tests are deleted with the feature.
- Docs: design doc sections 3, 5, and 8 drop `_tx/`; the FAQ crash answer and `docs/mycelium-phases.md` acceptance criterion 6 are rewritten to the new contract; README's "Crash-safe" bullet becomes: atomic content mutations, durable append-only log, no silent loss — at worst, a power loss in a microsecond window leaves the final mutation unlogged.

### Stage 5 — Slim the reference adapter (ADR-0006)

- `activity-log.ts` (496 → ~250 lines): keep `recordSessionBoundary`, `recordSessionShutdown`, `recordCompaction`, `recordContextCheckpoint` with fingerprint dedupe; delete turn/tool methods, the tool-timing map (and its leak), and the usage/cost/assistant/tool payload helpers.
- `index.ts`: drop turn/tool hook registrations (9 hooks → ~6).
- `system-prompt.ts` final size ~400 tokens: public model + command tiers, conventions file, `--rationale` nudge, CAS recovery, reserved paths, recorded-events note, identity.
- `tests/portable-events-fixture.test.ts`: the fixture stays as full-vocabulary documentation for all adapters; the test's "exactly 8 op types" assertion is retargeted to "fixture ops are within the documented vocabulary".
- `docs/portable-activity-events.md`: vocabulary unchanged; the L3 example paragraph updated to match what pi-mycelium now emits (session boundaries, compaction, deduped context checkpoints).
- `extensions/pi-mycelium` remains responsible for those events only after Stage 5; turn and tool events are intentionally not emitted by this adapter.
- Original plan: release v0.3.0 after this stage (binary + extension + platform packages), per `docs/release-checklist.md`. Actual release prep folded Stages 6-7 into the same cut and ships the combined tree as v0.4.0.

### Stage 6 — Skill + adapter split on the slim core

Land `mycelium-skill-adapter-plan.md` as written, with one strengthening: `skills/mycelium/SKILL.md` + `references/` become the canonical agent-facing guidance, single-sourced. The references absorb what is today spread across `docs/agent-faq.md`, `docs/conflict-resolution.md`, and the self-evolution recipes; the pi system prompt stays a thin block consistent with the skill. Ships as v0.4.0 — the Phase 2 entry point (Claude Code skill distribution, Hermes plugin next).

### Stage 7 — Docs consolidation

With the skill as the agent-facing home: delete `docs/agent-faq.md` (folded into skill references), fold `docs/conflict-resolution.md` into the design doc's concurrency section plus a skill reference, trim README and `docs/faq.md` to explain each concept once and link elsewhere. Target: every concept has exactly one explanatory home; everything else points at it.

## Out of scope (unchanged by this epic)

CAS semantics and version tokens; atomic write/edit/rm/mv; the conflict envelope contract (exit 64/65); the reserved `_` prefix; JSONL log format and `_activity/YYYY/MM/DD/<agent>.jsonl` layout; `--rationale` (ADR-0003); `mycelium log` and `--payload-json`; the no-tools adapter principle; binary distribution via npm optional dependencies; the release pipeline; Phase 3 roadmap items (git/jj integration, historical reads, retention).

## Test and verification plan

- Per stage: `make test` and extension `npm test` green; removals are preceded by characterization tests of the surviving contracts (Stage 4 especially).
- Property and concurrent-process tests for CAS, reserved prefix, and atomicity run unchanged throughout — they guard the invariants this epic must not move.
- Manual pi.dev session smoke test after Stages 3 and 5: fresh mount bootstraps the new template, prompt renders, mutations and conflicts behave.
- T3 detectors 2 and 3 still classify their fixtures from the new location.

## Risks

- **Published extension users (pre-1.0):** v0.4.0 is breaking (evolve gone, two flags gone, glob folded). Mitigations: CHANGELOG, the transitional evolve stub, early-access framing already in README.
- **Seeded templates in existing mounts reference evolve:** the stub's pointer message covers the transition; templates self-heal as agents edit them.
- **T2 grading drift:** rubric edit happens in Stage 3 alongside the code so the benchmark never references a removed feature.

## Assumptions

- Pre-1.0 breaking changes are acceptable with CHANGELOG documentation (per README's early-access framing).
- No external consumers of `evolve` or the journal beyond this repo's extension and docs.
- Benchmark T1/T2 model runs have not been published, so rubric rewording has no comparability cost.
- Phase 2 distribution work begins after Stage 7, on the simplified surface.
