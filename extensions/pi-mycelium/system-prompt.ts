export interface AvailableContext {
	mountPath: string;
	agentId: string;
	sessionId: string;
}

export interface UnavailableContext {
	mountPath: string;
}

function renderConventionsSection(memoryPath: string): string {
	return `### Conventions file

Conventions for organizing and operating this store live in \`${memoryPath}\`.
Read that exact file once at session start. Do not broad-search to rediscover
required files; if the path is missing, report it instead of guessing.

Record durable rules by editing that file with \`--rationale\`. Use dated prose
entries for conventions, lessons, index locations, archive policy, and open
questions. Be proactive: when a repeated pattern, mistake, durable user
preference, naming rule, or useful index emerges, update the conventions file in
the same session instead of leaving the lesson implicit. For point-in-time
signals that do not belong in the conventions file, use
\`mycelium log decision|agent_note --rationale "..."\`.`;
}

function renderActivityEventsSection(): string {
	return `### Activity events

The pi-mycelium adapter automatically records portable activity events under
\`_activity/\`: session boundaries, \`compaction\`, and deduped
\`context_checkpoint\` entries. Older logs may contain turn/tool events or
legacy \`context_signal\` entries; treat \`context_signal\` as a low-detail
context checkpoint.

These event names are adapter conventions, not binary-enforced schema. Payloads
are metadata-only by default (counts, roles, fingerprints, duplicate counts),
not full prompt/tool/file contents. Read them with
\`mycelium grep --path _activity --pattern context_checkpoint --format json\` or
other normal file/search tools.`;
}

export function systemPromptAvailable(c: AvailableContext): string {
	const memoryPath = `${c.mountPath}/MYCELIUM_MEMORY.md`;

	return `## Mycelium memory

You have a persistent file-based memory store mounted at ${c.mountPath}.
It survives sessions and may be shared with other agents mounted concurrently.

Mental model: **a folder + safe mutations + a searchable activity log**.
Most work uses the everyday commands; the rest are occasional or metadata-only.

Everyday commands via the \`bash\` tool:
- \`mycelium read <path> [--format text|json]\` — read a file; JSON includes UTF-8 content plus version for CAS
- \`mycelium write <path> [--rationale STR]\` — write or overwrite (content via stdin)
- \`mycelium edit <path> --old <str> --new <str> [--rationale STR]\` — find/replace a unique substring
- \`mycelium ls [pattern] [--recursive]\` — list entries, optionally filtered by glob pattern
- \`mycelium grep --pattern <str> [--path P] [--format json] [--limit N]\` — search content

Occasional commands:
- \`mycelium rm <path> [--rationale STR]\` — delete
- \`mycelium mv <src> <dst> [--rationale STR]\` — atomic rename (fails if \`<dst>\` exists)

Metadata commands:
- \`mycelium log <op> [--path PATH] [--payload-json STR | --stdin] [--rationale STR]\` — append an arbitrary signal; mostly adapter-facing

${renderConventionsSection(memoryPath)}

Operational rationale: \`write\`, \`edit\`, \`rm\`, \`mv\`, and \`log\` accept an
optional \`--rationale "..."\` flag (≤64 KiB). Supply it when the operation
carries reasoning a future reviewer would need — what triggered the change,
what alternative you rejected, why this and not that. Captured into the
activity log line as a top-level \`rationale\` field and into the CAS
conflict envelope on conflicts. Skip it for routine appends, status
updates, and saved artifacts where no separable reasoning exists; a
forced placeholder is worse than no field.

Concurrency: \`write\`, \`edit\`, \`rm\`, and \`mv\` accept an optional
\`--expected-version <sha>\` flag for optimistic concurrency. On a stale token
or an \`mv\` destination collision, the command exits 64 and prints one line of
JSON to stderr:

\`\`\`
{"error":"conflict","op":"write","path":"foo.md","current_version":"sha256:...","expected_version":"sha256:..."}
{"error":"destination_exists","op":"mv","path":"dst.md","current_version":"sha256:..."}
\`\`\`

Standard recovery: re-read with \`mycelium read <path> --format json\`,
merge with the current content, and retry with the fresh version token.

Reserved paths: \`mycelium\` rejects writes to any first-segment path beginning
with \`_\`. \`_activity/YYYY/MM/DD/${c.agentId}.jsonl\` is auto-generated metadata for every
mutation you perform. It is read-only to you, but greppable: try
\`mycelium grep --path _activity --format json --pattern <str>\` to look back across
sessions. Payloads from \`mycelium log\` are inlined on each entry as a \`payload\`
field — no separate file to look up. Other \`_\` paths are internal; do not edit them.

${renderActivityEventsSection()}

When to log explicitly: this extension already records portable activity events
automatically, so use \`mycelium log <op> --stdin\` only for signals you'd want
to grep later — e.g. a \`decision\` or \`agent_note\` with rationale.

This session's identity: MYCELIUM_AGENT_ID=${c.agentId}, MYCELIUM_SESSION_ID=${c.sessionId}.
`;
}

export function systemPromptUnavailable(c: UnavailableContext): string {
	return `## Mycelium memory (UNAVAILABLE)

Memory store is configured at ${c.mountPath} but the \`mycelium\` binary
is not on PATH. Install it and ensure it's visible to this session to enable
persistent memory. Until then, this conversation is the only memory available.`;
}
