# Phase 1 Benchmark Rubric

**Status:** Active. Release-decoupled — `v0.1.0` ships on binary-side criteria; T1/T2 model runs land here as they complete.
**Owns:** Acceptance criteria 1, 4, and 5 from `mycelium-phases.md`.

This rubric operationalizes the three Phase 1 criteria that depend on real model runs:

- **#1 Single-agent multi-session.** Validated by T1.
- **#4 Self-evolution via the activity log.** Validated by T2.
- **#5 Failure-mode observability.** Validated by T3.

Criteria 2, 3, 6, 7, 8 are validated by the binary's property tests, concurrent-process tests, and tarball inspection — not here.

---

## Target models

Phase 1 targets one Anthropic model and one OpenAI model:

- **Anthropic:** Claude Opus 4.7
- **OpenAI:** GPT-5.5

Both must pass the model-dependent criteria (#1 and #4) for the "model-agnostic" claim to hold. Single-provider passes don't count.

Google Frontier and open-weights are out of scope for Phase 1; revisit in a later phase if the MVP holds up.

---

## Task suite

Each task ships as a directory under `docs/benchmarks/tasks/T<n>-<slug>/`, containing:

- `task.md` — agent-facing brief, given verbatim as the user prompt for that session.
- `harness.md` — operator-facing run protocol.
- `held-out.md` — questions used by the rubric grader.
- `seed/` (T2 only) — pre-populated store contents and synthetic activity log.

### T1 — Multi-session research synthesis (acceptance #1)

**Topic.** Connection-pooler selection for PostgreSQL: PgBouncer vs. Pgpool-II vs. pgcat for a small SaaS. Stable enough that training-data drift between Opus 4.7 and GPT-5.5 doesn't dominate; narrow enough that one focused engineer could finish the writeup in a few hours.

**Three sessions, fresh process each, same mounted store.** Per-session prompts live verbatim in `docs/benchmarks/tasks/T1-multi-session-research/task.md`:

1. Session 1 frames the SaaS scenario and asks the agent to investigate the three options.
2. Session 2 extends the analysis to failover and HA behavior under load.
3. Session 3 asks for a final recommendation with reasoning.

Three sessions, not two: two sessions test "did the agent re-read its notes"; three tests "did the agent build something compositional across resumes." The latter is the bet.

**Held-out questions.** Five questions in `held-out.md` probe specific differentiators (pool modes, read/write splitting, implementation language, failover, prepared statements in transaction pooling). A grader (a Frontier model from the other provider than the one under test) reads the final store and answers each from the agent's notes. **Pass: ≥4/5 answered correctly *and* traceable to specific notes.**

**Comparison run.** Same task, same model, no Mycelium mount, single session with the three prompts concatenated. The grader reads the transcript only. **Pass: the Mycelium run's output is judged more substantively grounded than the no-memory run.** Per `harness.md`: run 5 instances per model; both criteria require ≥3/5 to pass.

### T2 — Seeded self-evolution scenario (acceptance #4)

**Seed.** A pre-populated store containing a recognizable failure pattern:

- `notes/` with 6 files on related-but-overlapping topics with subtly inconsistent paths (e.g. `notes/glp1-pipeline.md`, `notes/glp-1-pipeline.md`, `notes/glp1_followup.md`).
- Activity log preloaded with ~30 entries showing prior writes-without-reads and one near-duplicate the seed-agent created.
- Starter `MYCELIUM_MEMORY.md` *without* a search-before-writing rule.

**Two sessions.**

1. "Investigate [topic adjacent to seeded notes]. Add what you learn."
2. "Continue. Take a moment first to look at how the store is shaped."

**Pass.** After session 2, the rubric grader judges that the agent recognized the seeded pattern and responded with at least one convention edit, index file, or grep-before-write behavior. Single human/grader judgment; no signal-counting threshold. **Run 5 instances per model; pass requires majority.**

Why seeded: self-evolution requires something to evolve in response to. A clean store has nothing for the agent to notice.

### T3 — Failure-mode detectors (acceptance #5)

**Detectors** operate on activity-log content alone:

1. **Writes-without-reads ratio.** `op=write` + `op=edit` counts divided by explicit `op=read_signal` counts (since reads aren't auto-logged). Threshold: ratio >0.7 for ≥3 consecutive sessions = unhealthy. Sessions with mutations and zero read signals count as +∞ ratio.
2. **Near-duplicate path count.** Levenshtein-1 path collisions per session across `op=write` entries. Threshold: ≥3 in a single session = unhealthy.
3. **Thrashing.** Activity-log entries per session. Threshold: ≥50 in a single session = unhealthy. (The "too few entries" tail is deferred — hard to disambiguate from a quick-lookup session in practice.)

**Validation.** Hand-craft 4 trajectories — 1 healthy, 3 unhealthy (one per detector). The detectors must classify all 4 correctly. The 30-trajectory human-judgment validation is deferred to Phase 2 once we have real run data to calibrate against.

**Pass.** Detectors classify the 4 hand-crafted trajectories correctly. Model-independent.

---

## Scoring

A *run* executes T1–T3 against one model. Per-task scoring is binary (pass/fail).

- **Acceptance #1** passes for a model if T1 passes.
- **Acceptance #4** passes for a model if T2 passes.
- **Acceptance #5** passes when T3's detectors classify the 4 hand-crafted trajectories correctly. Model-independent.

The **model-agnostic claim passes** when both Claude Opus 4.7 and GPT-5.5 clear #1 and #4. Acceptance #3 (conflict recovery) is verified by the binary's property tests.

---

## Out of scope

Performance, long-running stores, cost ceilings, and non-pi.dev harnesses (Hermes, Claude Code) are all Phase 2+.

---

## Release decoupling

Binary-side acceptance criteria (#2 multi-agent concurrency, #5 failure-mode observability, #6 activity-log integrity, #7 backend correctness, #8 store readability) are met by tests in `cmd/mycelium/` and ship with `v0.1.0`. Model-run criteria (#1 multi-session synthesis, #3 conflict recovery on real models, #4 self-evolution) are open and validate against the released artifact rather than gating release. Results land here as runs complete.

Rationale: model-run validation does not change the artifact, only the public claim about it. Shipping early-access lets real users exercise the multi-session and concurrent-agent paths that the synthetic suite covers; waiting for full validation costs months of zero-feedback delay. Release framing is "early access, validation in progress" until both target models pass #1 and #4.

## Open issues

T3 is executable: detectors implemented in `cmd/mycelium/detect.go`, fixtures under `cmd/mycelium/testdata/trajectories/`, harness at `docs/benchmarks/tasks/T3-failure-detectors/harness.md`. Run via `go test -run TestDetectors`.

T1 is drafted: `task.md`, `harness.md`, `held-out.md` under `docs/benchmarks/tasks/T1-multi-session-research/`. Awaiting first runs against Opus 4.7 and GPT-5.5.

T2 still needs content under `docs/benchmarks/tasks/T2-<slug>/` (`task.md`, `harness.md`, `held-out.md`, plus `seed/`).
