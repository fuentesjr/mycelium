/** Shape of one row returned by `mycelium evolve --kinds --format json`. */
export interface EvolutionKindRow {
	name: string;
	definition: string;
	defined_at_version?: string;
	source: "builtin" | "agent";
	event_count: number;
}

/** Shape of one entry returned by `mycelium evolve --active --format json`. */
export interface ActiveEvolutionEvent {
	ts: string;
	agent_id: string;
	session_id: string;
	op: string;
	id: string;
	kind: string;
	target?: string;
	supersedes?: string;
	rationale: string;
}

export interface AvailableContext {
	mountPath: string;
	agentId: string;
	sessionId: string;
	kinds: EvolutionKindRow[];
	activeEvolution: ActiveEvolutionEvent[];
}

export interface UnavailableContext {
	mountPath: string;
}

const ACTIVE_EVOLUTION_DISPLAY_LIMIT = 10;
const RATIONALE_TRUNCATE_LENGTH = 200;

function renderKindsSection(kinds: EvolutionKindRow[]): string {
	if (kinds.length === 0) {
		return `### Evolution kinds

Evolution surface unavailable — \`mycelium evolve --kinds\` did not return data.`;
	}

	const rows = kinds
		.map((k) => `| \`${k.name}\` | ${k.source} | ${k.definition} |`)
		.join("\n");

	return `### Evolution kinds

The following kinds are available in this mount. Built-in kinds ship with
mycelium; agent kinds were introduced by a prior session on this mount.

| Kind | Source | Definition |
|------|--------|------------|
${rows}

To introduce a new kind, pass \`--kind-definition "..."\` on first use.`;
}

function renderActiveEvolutionSection(active: ActiveEvolutionEvent[]): string {
	if (active.length === 0) {
		return `### Active evolution

No active evolution recorded yet. Use \`mycelium evolve\` to record conventions, lessons, indices, archives, or open questions.`;
	}

	const displayItems = active.slice(0, ACTIVE_EVOLUTION_DISPLAY_LIMIT);
	const overflow = active.length - displayItems.length;

	const lines = displayItems.map((e) => {
		const target = e.target ? ` ${e.target}` : "";
		const rationale =
			e.rationale.length > RATIONALE_TRUNCATE_LENGTH
				? e.rationale.slice(0, RATIONALE_TRUNCATE_LENGTH) + "…"
				: e.rationale;
		return `- [${e.kind}]${target} — ${rationale}`;
	});

	const footer =
		overflow > 0
			? `\n\n...and ${overflow} more — run \`mycelium evolve --active\` for the full list.`
			: "";

	return `### Active evolution

The conventions, lessons, and other evolution currently in force on this mount:

${lines.join("\n")}${footer}`;
}

function renderRecordingEvolutionSection(): string {
	return `### Recording evolution

Use \`mycelium evolve\` to record structured evolution decisions. This is
metadata only — it never mutates the store and is not a second memory system;
it is typed activity-log history for conventions, lessons, indices, archives,
and questions.

When to call it:

- Adopting or retiring a convention (covers structural *or* behavioral patterns):
  \`mycelium evolve convention --target <path-or-glob-or-scope> --rationale "..."\`
  Path-scoped example: \`--target notes/incidents/ --rationale "Use <date>-<slug>.md filenames."\`
  Behavior-scoped example: \`--target memory-discipline --rationale "Record durable preferences proactively."\`
- Building or regenerating a derived index:
  \`mycelium evolve index --target <path> --rationale "..."\`
- Archiving a region (run \`mycelium mv\` separately to move the files):
  \`mycelium evolve archive --target <path> --rationale "..."\`
- Distilling a lesson from past work:
  \`mycelium evolve lesson --rationale "..."\`
- Opening a tracked unknown:
  \`mycelium evolve question --target <path-or-topic> --rationale "..."\`
- When built-in kinds don't fit, introduce a new kind on first use:
  \`mycelium evolve experiment --target ... --kind-definition "An in-progress hypothesis I'm actively testing." --rationale "..."\`

The \`--target\` flag is optional for kinds that aren't path-scoped (e.g.
\`lesson\`). When superseding a prior entry, the binary detects it automatically
via non-empty \`(kind, target)\` matching — no need to pass \`--supersedes\` manually for targeted entries. Targetless entries are additive unless you pass \`--supersedes\` explicitly.`;
}

function renderActivityEventsSection(): string {
	return `### Activity events

The pi-mycelium adapter automatically records portable activity events under
\`_activity/\`: session boundaries, \`turn_start\` / \`turn_end\`,
\`tool_start\` / \`tool_end\`, \`compaction\`, and deduped
\`context_checkpoint\` entries. Older logs may contain legacy
\`context_signal\` entries; treat them as low-detail context checkpoints.

These event names are adapter conventions, not binary-enforced schema. Payloads
are metadata-only by default (counts, roles, tool names/ids, timings, usage,
error flags, fingerprints), not full prompt/tool/file contents. Read them with
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
- \`mycelium evolve ...\` — record/query typed activity-log entries for durable conventions, lessons, indices, archives, and questions
- \`mycelium log <op> [--path PATH] [--payload-json STR | --stdin] [--rationale STR]\` — append an arbitrary signal; mostly adapter-facing

Conventions for organizing this store live in \`${memoryPath}\`.
Read that exact file once at session start. Do not broad-search to rediscover
required files; if the path is missing, report it instead of guessing.
Revise it whenever you find a better way to organize what you're working with
— it's yours.

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

---

## Self-evolution

${renderKindsSection(c.kinds)}

${renderActiveEvolutionSection(c.activeEvolution)}

${renderRecordingEvolutionSection()}`;
}

export function systemPromptUnavailable(c: UnavailableContext): string {
	return `## Mycelium memory (UNAVAILABLE)

Memory store is configured at ${c.mountPath} but the \`mycelium\` binary
is not on PATH. Install it and ensure it's visible to this session to enable
persistent memory. Until then, this conversation is the only memory available.`;
}
