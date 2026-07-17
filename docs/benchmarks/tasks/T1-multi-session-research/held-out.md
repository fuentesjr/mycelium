# T1 — Held-out grading

The grader is a Frontier model from the opposite provider. Judge correctness
against the operator-supplied, checksum-pinned answer key; do not rely on the
grader's parametric knowledge for the truth set.

## Mycelium-store grading

The grader reads the store contents and answers each question below. For each,
the grader returns:

- **Answer:** the agent's answer per the store, in plain prose.
- **Traceable:** yes/no — is the answer supported by specific notes in the store (cite path)?
- **Verdict:** correct / incorrect / unsupported.

A question scores **pass** only when verdict is `correct` AND traceable is `yes`. "Mentioned in passing in session-3 prose without backing notes elsewhere" does not pass.

**Pass threshold:** ≥4 of 5 questions pass.

## Baseline-transcript grading

For the no-memory baseline, the grader reads only the concatenated transcript
and the same answer key. For every question below, return:

- **Answer:** the answer supported by the baseline transcript.
- **Transcript-supported:** yes/no — cite the specific prompt/response segment
  containing the supporting facts.
- **Verdict:** correct / incorrect / unsupported.

A baseline question scores pass only when the verdict is `correct` and
transcript-supported is `yes`. Report the baseline score out of five for the
paired comparison; it is not an independent acceptance gate. Do not require or
invent store paths for a baseline run.

For the comparison verdict, use `more grounded` only when the Mycelium answer
set is at least as accurate and contains material source, version, or
operational details absent from the baseline. More citations or more prose
alone do not qualify. Otherwise return `not more grounded` and explain the
decisive difference.

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
- For Mycelium-store grading, the agent's answer is what's in its notes, not
  what the model can synthesize on the spot at grading time. If the notes are
  silent, mark `unsupported` even if the session-3 prose is right — the test is
  whether memory persisted, not whether the model knew.
- For question 5, prepared-statement support in transaction pooling has shifted
  across pooler versions. In Mycelium-store grading, accept a different
  version-specific answer only when the store cites a supporting primary
  source; internal consistency alone is not enough.
