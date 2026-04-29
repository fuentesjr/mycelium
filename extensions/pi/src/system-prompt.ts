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
- \`mycelium write <path> --stdin\` — write or overwrite (content via stdin)
- \`mycelium edit <path> --old <str> --new <str>\` — find/replace a unique substring
- \`mycelium ls <path> [--recursive]\` — list entries
- \`mycelium glob <pattern>\` — paths matching a glob (e.g. \`learnings/*.md\`)
- \`mycelium grep <pattern> [--path P] [--format json] [--limit N]\` — search content
- \`mycelium rm <path>\` — delete
- \`mycelium mv <src> <dst>\` — atomic rename
- \`mycelium log <op> [--path PATH] [--stdin]\` — append a signal entry to your activity log

Conventions for organizing this store live in \`MYCELIUM_MEMORY.md\` at the root.
Read it once at session start; revise it whenever you find a better way to
organize what you're working with — it's yours.

Concurrency: every content-mutating subcommand accepts an optional \`--expected-version <sha>\` flag.
On version mismatch the command exits 64 with structured JSON on stderr
(\`{"error":"conflict","current_version":"sha256:..."}\`); add \`--include-current-content\`
to also retrieve current content. Re-read, merge, retry.

Reserved paths: \`mycelium\` rejects writes to any path beginning with \`_\`. The
activity log lives at \`_activity/YYYY/MM/DD/{agent_id}.jsonl\` — read it via
\`mycelium read\` or \`mycelium grep --format json\` when you want to look back
at what you've done across sessions.

This session's identity: MYCELIUM_AGENT_ID=${c.agentId}, MYCELIUM_SESSION_ID=${c.sessionId}.`;
}

export function systemPromptUnavailable(c: UnavailableContext): string {
  return `## Mycelium memory (UNAVAILABLE)

Memory store is configured at ${c.mountPath} but the \`mycelium\` binary
is not on PATH. Install it and ensure it's visible to this session to enable
persistent memory. Until then, this conversation is the only memory available.`;
}
