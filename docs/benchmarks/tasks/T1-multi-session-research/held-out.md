# T1 — Held-out grading questions

The grader (Frontier model from the opposite provider) reads the store contents and answers each question below. For each, the grader returns:

- **Answer:** the agent's answer per the store, in plain prose.
- **Traceable:** yes/no — is the answer supported by specific notes in the store (cite path)?
- **Verdict:** correct / incorrect / unsupported.

A question scores **pass** only when verdict is `correct` AND traceable is `yes`. "Mentioned in passing in session-3 prose without backing notes elsewhere" does not pass.

**Pass threshold:** ≥4 of 5 questions pass.

## Why these five

Each question targets a fact that should surface during a serious investigation of the three poolers and that distinguishes them from one another. They probe whether the agent (a) actually researched rather than relying on training-data summarization, (b) recorded findings durably enough to persist across sessions, and (c) organized them so the synthesis in session 3 has something to draw from.

## Questions

**1. Pool modes.** What pool modes does PgBouncer support, and which is the default? How do `transaction` mode and `session` mode differ in their handling of session-local state (e.g., temporary tables, prepared statements, `SET LOCAL`)?

**2. Read/write splitting.** Which of the three poolers offers built-in routing of reads to replicas vs. writes to the primary, and how is the routing decision made (statement parsing, connection-string suffix, application hint)? For poolers that don't offer it built-in, what's the typical workaround?

**3. Implementation language and concurrency model.** What language is each pooler implemented in, and what concurrency model does each use (single-threaded event loop, process-per-connection, async runtime, etc.)? The agent's notes should distinguish the operational implications, not just the labels.

**4. Failover and HA.** For each pooler, describe what an unattended failover of the primary database looks like from the pooler's perspective — does the pooler detect it, retry, drop in-flight transactions, require external orchestration? Pgpool-II in particular has an opinionated story; what is it?

**5. Prepared statements in transaction pooling.** Which pooler(s) support PostgreSQL prepared statements (`PREPARE` / `EXECUTE` or extended-query-protocol prepares) when running in transaction-pooling mode? For those that gained support recently, when (which version)? For those that don't or didn't, what's the typical client-side workaround?

## Notes for the grader

- These are not trick questions, and there is room for the agent to be partially right. A two-out-of-three answer (correct on PgBouncer and pgcat, missing or wrong on Pgpool-II) is a partial; the grader should use judgment but lean toward `incorrect` rather than splitting hairs.
- The agent's answer is what's in its notes, not what the model can synthesize on the spot at grading time. If the notes are silent, mark `unsupported` even if the agent's session-3 prose contains the right answer — the test is whether memory persisted, not whether the model knew.
- For question 5, prepared-statement support in transaction pooling has shifted across pooler versions; accept any internally-consistent answer that names a specific version or behavior, even if it's slightly out of date.
