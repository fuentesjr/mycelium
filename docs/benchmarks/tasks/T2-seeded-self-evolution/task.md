# T2 — Seeded self-evolution

**Acceptance criterion:** #4 (self-evolution via the activity log).

This task runs across two sessions, fresh process each, against a *pre-seeded* store. The seed contains a recognizable failure pattern — six notes with subtly inconsistent paths (`glp1-pipeline.md`, `glp-1-pipeline.md`, `glp1_followup.md`), an activity log of ~30 prior mutations with no read signals, and a starter `MYCELIUM_MEMORY.md` that does not mention searching before writing.

The agent under test is told it is `glp1-researcher` continuing prior work. The pass criterion is whether, by the end of session 2, the agent has noticed the pattern and responded — by editing conventions, adding an index, or visibly grepping before writing.

The prompts below are given verbatim as the user message for each session.

## Session 1

> You're the same researcher who's been building out the GLP-1 prescription analytics pipeline. We're extending the work to cover cardiovascular outcomes for the same patient cohort — what trials should we be aware of, what claims-data signals would let us approximate the trial endpoints in our own data, and what schema additions would we need on top of what's already in the warehouse? Add what you learn to the store.

## Session 2

> Continue. Take a moment first to look at how the store is shaped.
