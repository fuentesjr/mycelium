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

Read \`${memoryPath}\` once at session start. It is the active rule set for
naming, user preferences, index locations, archive policy, lessons, and open
questions. Do not broad-search for a substitute.

When a durable pattern, mistake, preference, index, or open question emerges,
edit that file in the same session with \`--rationale\`. For point-in-time
signals that should remain history but not become standing guidance, use
\`mycelium log decision --rationale "..."\` or
\`mycelium log agent_note --rationale "..."\`.`;
}

function renderActivityEventsSection(): string {
	return `### Activity events

The pi-mycelium extension automatically records pi lifecycle events under
\`_activity/\`: session boundaries and \`compaction\`. Older logs may contain
turn/tool events, \`context_checkpoint\`, or legacy \`context_signal\` entries.

Payloads are metadata-only by default; put larger details in normal files and
reference them with \`--path\`. Read events with normal file/search tools,
for example \`mycelium grep --path _activity --pattern session_ --format json\`.`;
}

export function systemPromptAvailable(c: AvailableContext): string {
	const memoryPath = `${c.mountPath}/MYCELIUM_MEMORY.md`;

	return `## Mycelium memory

You have a persistent file-based memory store mounted at ${c.mountPath}.
It survives sessions and may be shared with other agents mounted concurrently.

Mental model: **a folder + safe mutations + a searchable activity log**.
Most work uses the everyday commands; the rest are occasional or metadata-only.

Everyday commands via the \`bash\` tool:
- \`mycelium read <path> [--format text|json]\`
- \`mycelium write <path> [--expected-version SHA] [--rationale STR]\`
- \`mycelium edit <path> --old STR --new STR [--expected-version SHA] [--rationale STR]\`
- \`mycelium ls [pattern] [--recursive]\`
- \`mycelium grep --pattern STR [--path P] [--format json] [--limit N]\`

Occasional commands:
- \`mycelium rm <path> [--expected-version SHA] [--rationale STR]\`
- \`mycelium mv <src> <dst> [--expected-version SHA] [--rationale STR]\`

Metadata commands:
- \`mycelium log <op> [--path PATH] [--payload-json STR | --stdin] [--rationale STR]\`

${renderConventionsSection(memoryPath)}

Use \`--rationale\` when the operation carries reasoning a future reviewer
would need. Use \`--expected-version\` for edits, content-based deletes, and
revisions to files read this session. On a stale token or an \`mv\` destination
collision, the command exits 64 and prints one JSON line:

\`\`\`
{"error":"conflict","op":"write","path":"foo.md","current_version":"sha256:...","expected_version":"sha256:..."}
{"error":"destination_exists","op":"mv","path":"dst.md","current_version":"sha256:..."}
\`\`\`

Recovery: re-read with \`mycelium read <path> --format json\`, merge, and retry
with the fresh version token.

Reserved paths: \`mycelium\` rejects writes to any first-segment path beginning
with \`_\`. \`_activity/YYYY/MM/DD/${c.agentId}.jsonl\` is auto-generated,
read-only history. Other \`_\` paths are internal; do not edit them.

${renderActivityEventsSection()}

Use explicit \`mycelium log\` only for signals you'd want to grep later, such as
\`decision\` or \`agent_note\` with rationale.

This session's identity: MYCELIUM_AGENT_ID=${c.agentId}, MYCELIUM_SESSION_ID=${c.sessionId}.
`;
}

export function systemPromptUnavailable(c: UnavailableContext): string {
	return `## Mycelium memory (UNAVAILABLE)

Memory store is configured at ${c.mountPath} but the \`mycelium\` binary
is not on PATH. Install it and ensure it's visible to this session to enable
persistent memory. Until then, this conversation is the only memory available.`;
}
