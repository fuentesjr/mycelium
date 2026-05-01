# T1 — Multi-session research synthesis

**Acceptance criterion:** #1 (single-agent multi-session).

This task runs across three sessions, fresh process each, same mounted store. The prompts below are given verbatim as the user message for each session — no additional system prompt or framing beyond what the harness sets up (mount, identity env vars, the standard system-prompt block from the pi.dev extension).

The topic is intentionally narrow and stable: connection-pooler selection for PostgreSQL. Stable enough that training-data drift between Opus 4.7 and GPT-5.5 doesn't dominate the result; narrow enough that one focused engineer could finish the writeup in a few hours.

## Session 1

> You're helping me pick a connection pooler for a small SaaS running on PostgreSQL 16 — a few thousand active users, a single primary database with one read replica, ~200 application server processes opening connections from across a Kubernetes cluster. The candidates are PgBouncer, Pgpool-II, and pgcat.
>
> Investigate the three options. Build whatever notes you'll want later — we'll come back to this across multiple sessions, and a different process each time.

## Session 2

> Continuing from the prior session. Extend the analysis to cover failover and high-availability behavior in each option, with attention to what an unattended pool restart looks like under load (rolling deploys, Kubernetes pod evictions, primary failover events). Update or add notes as you go.

## Session 3

> Produce a final recommendation drawing on what you've gathered across the prior sessions. Include the reasoning, not just the verdict — what trade-offs are you weighing, and what would change your answer.
