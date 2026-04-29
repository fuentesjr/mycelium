# Phase 1 Benchmark Rubric

**Status:** Draft
**Project:** Mycelium
**Owns:** Acceptance criteria 1, 4, and 5 from `mycelium-phases.md`.

This document specifies the task suite, detectors, scoring, and target models that operationalize Phase 1's acceptance criteria. Per `mycelium-phases.md`, criteria 1, 4, and 5 are unfalsifiable without it; this rubric is what makes them testable. Treat it as design, not test infrastructure.

---

## What this rubric is for

Phase 1 validates one bet: a real filesystem driven by general file tools, on a Frontier model, produces useful agent memory **without** specialized memory infrastructure. The test is not whether `mycelium` works as a CLI — that's covered by property tests on the binary. The test is whether a Frontier model, given the binary plus a starter `MYCELIUM_MEMORY.md`, produces a store an experienced engineer would judge as useful, and revises its own conventions in response to its own observed behavior.

Three of the eight Phase 1 acceptance criteria depend on this rubric:

- **#1 Single-agent multi-session.** Defined task spanning multiple sessions; "more useful than no memory" requires a comparable run.
- **#4 Self-evolution via the activity log.** "Revises its own conventions across sessions in response to patterns observed in the log" requires a seeded scenario and a defined detector.
- **#5 Failure-mode observability.** Distinguishing the "31 transcript files" failure mode from healthy use requires programmatic detectors agreed against a human-judgment standard.

Criteria 2, 3, 6, 7, 8 are validated by property tests, concurrent-process tests, and tarball inspection — not by this rubric. Criterion #3 (conflict recovery) appears here as T4 only to verify model-side recovery; the binary's exit-code behavior is property-tested separately.

---

## Target models

The "model-agnostic" claim is testable only if the rubric runs across providers. Phase 1 targets:

| Class | Concrete models (at run time) | Provider |
|---|---|---|
| Anthropic Frontier | Claude Opus 4.x, Claude Sonnet 4.x | Anthropic |
| OpenAI Frontier | GPT-5 Opus tier, GPT-5 Sonnet tier | OpenAI |
| Google Frontier | Gemini Ultra (latest) | Google |
| Open-weights stretch | Llama-4 70B+ Instruct or equivalent | self-hosted / Together / Fireworks |

The first three are the hard requirement for "model-agnostic." The fourth is a stretch to verify the bet doesn't covertly depend on closed-model behaviors. **If only one provider passes, "model-agnostic" was unmeasured** — the criterion fails regardless of how good the passing run was.

**Expected pass rates per criterion** (calibration target, not gate):

| Criterion | Anthropic Frontier | OpenAI Frontier | Google Frontier | Open-weights |
|---|---|---|---|---|
| #1 (multi-session) | ≥90% | ≥90% | ≥70% | ≥50% |
| #4 (self-evolution) | ≥70% | ≥70% | ≥50% | ≥30% |
| #5 (detectors) | ≥90% detector-vs-human agreement on every provider's runs (property of detectors, not models) |

If actual pass rates diverge sharply from these expectations, the rubric is wrong and gets revised before any release decision. The expectations exist to make "we underestimated the floor" or "the rubric is too lax" visible.

---

## Task suite

Each task ships as a directory under `docs/benchmarks/tasks/T<n>-<slug>/`, containing:

- `task.md` — agent-facing brief, given verbatim as the user prompt for that session.
- `harness.md` — operator-facing run protocol (which session uses which prompt; setup steps; pass/fail check).
- `held-out.md` — questions answered only by reading the post-task store, used by the rubric grader.
- `seed/` (T3 only) — pre-populated store contents and a synthetic activity log.

### T1 — Multi-session research synthesis (acceptance #1)

**Brief.** Three sessions over a research topic with a built-in continuation gate. Session 1: "Investigate X. Build whatever notes you'll want later." Session 2 (fresh process, same store, different prompt): "Continuing on X. Extend the analysis to a new sub-question Y." Session 3: "Produce a final report drawing on what you've gathered."

**Topic candidates** (rotate to limit training-data contamination):

- "Trade-offs of three popular Rust async runtimes for low-latency networking. Recommend one for a video-streaming startup in 2026."
- "State of post-quantum cryptography deployment in major cloud KMS offerings. Recommend a path for a healthcare API gateway."
- "Container image build cache strategies for monorepos. Compare Bazel remote cache, Earthly, BuildKit, and Nix."

**Held-out questions.** A grader (a different Frontier model from the one under test) reads the final state of the store and answers 5 questions whose answers are only in session-1 or session-2 content but should be referenced in session-3 output. Pass if 4/5 are answered correctly from the store contents alone.

**Comparison run.** Same task, same model, same prompts, **no Mycelium mount**. The grader reads only the session-3 transcript. **Pass for the criterion:** the rubric grader judges the Mycelium run's session-3 output as more substantively grounded than the no-memory run, with cited reasoning, on at least 7 of 10 task instances.

**Why three sessions, not two.** Two sessions tests "did the agent re-read its notes." Three sessions tests "did the agent build something compositional across resumes." The latter is the bet.

### T2 — Multi-session coding task (acceptance #1)

**Brief.** Build a small CLI tool across three sessions. Session 1: "Design and partially implement [tool]. You'll resume later." Session 2: "Continue on [tool] — finish the core feature." Session 3: "Add tests and document edge cases."

**Tool candidates** (small enough that 3 × ~30-min sessions suffice):

- A `git diff` filter that pretty-prints semantic-version bumps in `package.json` files.
- A wrapper around `rg` that ranks results by recency-of-edit and produces a markdown summary.
- A `find`-replacement that maintains a search index in a small SQLite file.

**Held-out questions.** 5 questions about design decisions or edge-case rationale that the model would only answer correctly if it consulted prior-session notes. Pass if 4/5 are correctly grounded in the store. **Plus:** the resulting tool must compile and pass its own tests.

**Comparison run.** Same task, no Mycelium. Pass if the Mycelium run's design coherence and test coverage are judged superior on at least 7 of 10 task instances.

### T3 — Seeded duplicate-creation scenario (acceptance #4)

**Brief.** Configure the store with a pre-existing structure that is *almost* what the agent would naturally create, but contains a recognizable failure pattern. Specifically:

- A `notes/` directory containing 6 files on related-but-overlapping topics with subtly inconsistent paths (e.g. `notes/glp1-pipeline.md`, `notes/glp-1-pipeline.md`, `notes/glp1_followup.md`).
- The activity log preloaded with 30 entries showing prior writes-without-reads and one near-duplicate the seed-agent had previously created.
- A starter `MYCELIUM_MEMORY.md` *without* a search-before-writing rule.

**Sessions.**

- Session 1: "Investigate [topic adjacent to the seeded notes]. Add what you learn."
- Session 2: "Continue. Take a moment first to look at how the store is shaped."

**Detector.** After session 2, the rubric grader checks whether the agent edited a convention file in a way that addresses the seeded duplicate pattern. Specific positive signals:

1. `MYCELIUM_MEMORY.md` (or any convention file the agent has chosen) gained a "search before writing" or naming-convention rule.
2. An `INDEX.md` or equivalent exists at the root or in `notes/` after session 2.
3. The session-2 activity log shows `mycelium grep` or `mycelium glob` calls preceding writes.

**Pass.** Two of three positive signals **and** the rubric grader judges the change as substantively responsive to the seeded pattern (not coincidental). Target rates: ≥70% on Anthropic and OpenAI Frontier, ≥50% on Google.

**Why seeded.** Self-evolution requires something to evolve in response to. Asking "did the model evolve?" of a clean store is malformed — there's nothing for the agent to notice. Seeding a recognizable failure pattern lets us measure whether the agent (a) notices it via the log and (b) does something about it via convention edits. Both are required.

### T4 — Conflict recovery (acceptance #3, model-side coverage)

**Brief.** Two agents on the same store, instructed to update overlapping content concurrently. Each is given a script that issues a `mycelium write --expected-version=...` against a known-stale token at a controlled point.

**Pass.** Both agents (one human-impersonated harness, one Frontier model under test) parse the exit-64 conflict error from stderr without operator intervention. The Frontier-model agent re-reads, merges, and retries. Verified on all three closed-provider Frontier models per acceptance #3.

This task duplicates acceptance #3 for completeness. The binary's property test suite covers the exit-code and version-token side; this rubric covers the model-side recovery behavior.

### T5 — Failure-mode observability (acceptance #5)

**Brief.** Generate two trajectories of the same agent on the same task, where one is a healthy run and one exhibits the "31 transcript files" failure mode (writes without reads, near-duplicate paths, low log entries per session). Provide the activity logs alone — no store contents — to a programmatic detector.

**The "31 transcript files" mode.** Named after the canonical failure: an agent producing many session-scoped transcript files at near-identical paths (`notes/session-1.md`, `notes/session-2.md`, …) without ever reading them back, treating the store as write-only scratch space. The agent burns sessions producing artifacts it never consults; from outside, the store grows; from inside, the agent has no memory.

**Detectors.** All operate on activity-log content alone:

1. **Writes-without-reads ratio.** Per-session count of `op=write` and `op=edit` entries divided by the count of explicit-read signal entries logged via `mycelium log` (e.g. `op=read_signal` from the harness, since reads aren't auto-logged). Threshold: ratio >0.7 for ≥3 consecutive sessions = unhealthy.
2. **Near-duplicate path count.** Levenshtein-1 path collisions per session across `op=write` entries. Threshold: ≥3 in a single session = unhealthy.
3. **Log entries per session.** Median count across the run. Threshold: <5 = the agent isn't using the store; >50 = the agent is thrashing. Either tail is unhealthy.

**Human-judgment standard.** Three engineers familiar with agent workflows independently classify 30 trajectories (15 healthy, 15 unhealthy, mixed-quality, blind to detector output) as healthy/unhealthy. The detector passes if its classification agrees with the human majority on at least 27 of 30 trajectories (≥90%).

**Pass for acceptance #5.** Detectors meet the 90% agreement bar. This is a property of the detectors, not of any specific model run; the rubric ships when the detectors hold up.

---

## Scoring across the suite

A *run* consists of executing T1–T5 against one model under one provider. Per-task scoring above is binary (pass/fail). Acceptance criteria mapping:

- **Acceptance #1** passes for a model if T1 and T2 both pass.
- **Acceptance #4** passes for a model if T3 passes.
- **Acceptance #5** passes when the T5 detectors meet the 90% agreement bar against the human-judgment standard. Model-independent.

A model **passes Phase 1** if it clears acceptance criteria 1, 3, and 4 (the model-dependent ones supported by this rubric, plus #3 from the binary's property tests). The **model-agnostic claim passes** when this holds on Anthropic, OpenAI, *and* Google Frontier models. Open-weights is a stretch goal; failing it does not block the criterion.

---

## Out of scope for this rubric

- **Performance.** Latency and throughput aren't validated by Phase 1; that's Phase 2.
- **Long-running stores.** All tasks run on stores under 200 entries. Retention and pruning are Phase 3.
- **Token cost ceilings.** Frontier-model judgment quality is what's tested; cost limits aren't a Phase 1 gate.
- **Hermes harness or Claude Code skill.** Phase 1 runs through the pi.dev extension only. Other harnesses are Phase 2.

---

## Open issues for this rubric

- **Grader contamination.** The rubric grader is a Frontier model from a different provider than the model under test. Cross-provider grader bias (e.g., one grader being systematically more lenient) needs sanity-checking on the calibration set before scoring real runs.
- **Topic rotation.** The candidate topics in T1/T2 will leak into training data over time. The rubric should publish a fresh topic each release and retain prior topics as regression tests.
- **Comparison-run determinism.** The "no-memory" comparison runs need temperature-matching and prompt-matching protocols documented in `harness.md` to be defensible.
- **Concrete `harness.md` and `task.md` content.** This document specifies the rubric; the per-task files at `docs/benchmarks/tasks/T<n>-<slug>/` are not yet drafted. Until they exist, the rubric is approved-in-principle but not yet executable.

---

*End of draft. Open issues above are the gating items before the rubric can drive a real run.*
