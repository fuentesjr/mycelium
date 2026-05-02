/** Shape of one row returned by `mycelium evolution --kinds --format json`. */
export interface EvolutionKindRow {
  name: string;
  definition: string;
  defined_at_version?: string;
  source: "builtin" | "agent";
  event_count: number;
}

/** Shape of one entry returned by `mycelium evolution --active --format json`. */
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

Evolution surface unavailable — \`mycelium evolution --kinds\` did not return data.`;
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
      ? `\n\n...and ${overflow} more — run \`mycelium evolution --active\` for the full list.`
      : "";

  return `### Active evolution

The conventions, lessons, and other evolution currently in force on this mount:

${lines.join("\n")}${footer}`;
}

function renderRecordingEvolutionSection(): string {
  return `### Recording evolution

Use \`mycelium evolve\` to record structured evolution decisions. This is
metadata only — it never mutates the store.

When to call it:

- Adopting or retiring a convention:
  \`mycelium evolve convention --target <path-or-glob> --rationale "..."\`
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
via \`(kind, target)\` matching — no need to pass \`--supersedes\` manually.`;
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
