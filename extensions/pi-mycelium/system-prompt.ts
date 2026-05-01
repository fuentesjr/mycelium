export interface AvailableContext {
  mountPath: string;
  agentId: string;
  sessionId: string;
}

export interface UnavailableContext {
  mountPath: string;
}

export function systemPromptAvailable(c: AvailableContext): string {
  return `## Mycelium memory

You have a persistent file-based memory store mounted at ${c.mountPath}.
It survives sessions and may be shared with other agents mounted concurrently.

Use these subcommands via the \`bash\` tool:
- \`mycelium read <path>\` — read a file
- \`mycelium write <path>\` — write or overwrite (content via stdin)
- \`mycelium edit <path> --old <str> --new <str>\` — find/replace a unique substring
- \`mycelium ls <path> [--recursive]\` — list entries
- \`mycelium glob <pattern>\` — paths matching a glob (e.g. \`learnings/*.md\`)
- \`mycelium grep --pattern <str> [--path P] [--format json] [--limit N]\` — search content
- \`mycelium rm <path>\` — delete
- \`mycelium mv <src> <dst>\` — atomic rename (fails if \`<dst>\` exists)
- \`mycelium log <op> [--path PATH] [--payload-json STR | --stdin]\` — append a non-mutation signal

Conventions for organizing this store live in \`MYCELIUM_MEMORY.md\` at the root.
Read it once at session start; revise it whenever you find a better way to
organize what you're working with — it's yours.

Concurrency: \`write\`, \`edit\`, \`rm\`, and \`mv\` accept an optional
\`--expected-version <sha>\` flag for optimistic concurrency. On a stale token
or an \`mv\` destination collision, the command exits 64 and prints one line of
JSON to stderr:

\`\`\`
{"error":"conflict","op":"write","path":"foo.md","current_version":"sha256:...","expected_version":"sha256:..."}
{"error":"destination_exists","op":"mv","path":"dst.md","current_version":"sha256:..."}
\`\`\`

Add \`--include-current-content\` to also retrieve the current bytes inline
(\`current_content\` field, UTF-8 only). Standard recovery: re-read, merge, retry.

Reserved paths: \`mycelium\` rejects writes to any first-segment path beginning
with \`_\`. Today this means:

- \`_activity/YYYY/MM/DD/${c.agentId}.jsonl\` — auto-generated metadata for every
  mutation you perform. Read-only to you, but greppable: try
  \`mycelium grep --path _activity --format json --pattern <str>\` to look back across
  sessions. Payloads from \`mycelium log\` are inlined on each entry as a \`payload\`
  field — no separate file to look up.

When to log explicitly: this extension already records context boundaries
automatically, so use \`mycelium log <op> --stdin\` only for signals you'd want
to grep later — e.g. a \`decision\` with rationale, or a \`compaction\` marker
before truncating a working note.

This session's identity: MYCELIUM_AGENT_ID=${c.agentId}, MYCELIUM_SESSION_ID=${c.sessionId}.`;
}

export function systemPromptUnavailable(c: UnavailableContext): string {
  return `## Mycelium memory (UNAVAILABLE)

Memory store is configured at ${c.mountPath} but the \`mycelium\` binary
is not on PATH. Install it and ensure it's visible to this session to enable
persistent memory. Until then, this conversation is the only memory available.`;
}
