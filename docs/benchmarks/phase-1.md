# Phase 1 Benchmark Rubric

**Status:** Draft
**Owns:** Acceptance criteria 1, 4, and 5 from `mycelium-phases.md`.

This rubric operationalizes the three Phase 1 criteria that depend on real model runs:

- **#1 Single-agent multi-session.** Validated by T1.
- **#4 Self-evolution via the activity log.** Validated by T2.
- **#5 Failure-mode observability.** Validated by T3.

Criteria 2, 3, 6, 7, 8 are validated by the binary's property tests, concurrent-process tests, and tarball inspection — not here.

---

## Target models

Phase 1 targets one Anthropic model and one OpenAI model:

- **Anthropic:** Claude Opus 4.x
- **OpenAI:** GPT-5

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

**Topic.** "State of post-quantum cryptography deployment in major cloud KMS offerings. Recommend a path for a healthcare API gateway." Stable factual ground; less likely to drift between runs than a fast-moving topic.

**Three sessions, fresh process each, same mounted store.**

1. "Investigate the topic. Build whatever notes you'll want later."
2. "Continuing. Extend the analysis to [held-out sub-question]."
3. "Produce a final report drawing on what you've gathered."

Three sessions, not two: two sessions test "did the agent re-read its notes"; three tests "did the agent build something compositional across resumes." The latter is the bet.

**Held-out questions.** A grader (a Frontier model from the other provider than the one under test) reads the final store and answers 5 questions whose answers live only in session-1 or session-2 content but should be cited in session-3 output. **Pass: 4/5 answered correctly from store contents alone.**

**Comparison run.** Same task, same model, no Mycelium mount. The grader reads only the session-3 transcript. **Pass: the Mycelium run's session-3 output is judged more substantively grounded than the no-memory run.** Run 5 instances per model; pass requires majority.

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

1. **Writes-without-reads ratio.** `op=write` + `op=edit` counts divided by explicit `op=read_signal` counts (since reads aren't auto-logged). Threshold: ratio >0.7 for ≥3 consecutive sessions = unhealthy.
2. **Near-duplicate path count.** Levenshtein-1 path collisions per session across `op=write` entries. Threshold: ≥3 in a single session = unhealthy.
3. **Log entries per session.** Median across the run. <5 = agent isn't using the store; >50 = thrashing. Either tail is unhealthy.

**Validation.** Hand-craft 4 trajectories — 2 healthy, 2 unhealthy (one per failure mode). The detectors must classify all 4 correctly. The 30-trajectory human-judgment validation is deferred to Phase 2 once we have real run data to calibrate against.

**Pass.** Detectors classify the 4 hand-crafted trajectories correctly. Model-independent.

---

## Scoring

A *run* executes T1–T3 against one model. Per-task scoring is binary (pass/fail).

- **Acceptance #1** passes for a model if T1 passes.
- **Acceptance #4** passes for a model if T2 passes.
- **Acceptance #5** passes when T3's detectors classify the 4 hand-crafted trajectories correctly. Model-independent.

The **model-agnostic claim passes** when both Anthropic (Claude Opus 4.x) and OpenAI (GPT-5) clear #1 and #4. Acceptance #3 (conflict recovery) is verified by the binary's property tests.

---

## Out of scope

Performance, long-running stores, cost ceilings, and non-pi.dev harnesses (Hermes, Claude Code) are all Phase 2+.

---

## Open issues

The per-task files at `docs/benchmarks/tasks/T<n>-<slug>/` (`task.md`, `harness.md`, `held-out.md`, T2 seed) are not yet drafted. Until they exist this rubric is approved-in-principle but not executable.
