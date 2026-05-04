import { describe, it, expect } from "vitest";
import { systemPromptAvailable, systemPromptUnavailable } from "../system-prompt.js";
import type { EvolutionKindRow, ActiveEvolutionEvent } from "../system-prompt.js";

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const builtinKinds: EvolutionKindRow[] = [
  {
    name: "convention",
    definition: "A naming, layout, structural, or behavioral pattern for organizing or operating on the store.",
    defined_at_version: "0.1.0",
    source: "builtin",
    event_count: 0,
  },
  {
    name: "lesson",
    definition: "A distilled insight from past work, intended to inform future behavior.",
    defined_at_version: "0.1.0",
    source: "builtin",
    event_count: 2,
  },
];

const agentKinds: EvolutionKindRow[] = [
  ...builtinKinds,
  {
    name: "experiment",
    definition: "An in-progress hypothesis I'm actively testing.",
    source: "agent",
    event_count: 3,
  },
];

const sampleActiveEvents: ActiveEvolutionEvent[] = [
  {
    ts: "2026-05-02T14:00:00Z",
    agent_id: "researcher-7",
    session_id: "abc123",
    op: "evolve",
    id: "01HXKP4Z9M8YV1W6E2RTSA9KFG",
    kind: "convention",
    target: "notes/incidents/",
    rationale: "Adopting <date>-<slug>.md filenames so incidents sort chronologically.",
  },
  {
    ts: "2026-05-03T09:00:00Z",
    agent_id: "researcher-7",
    session_id: "def456",
    op: "evolve",
    id: "01HXKP4Z9M8YV1W6E2RTSA9KFH",
    kind: "lesson",
    rationale: "Queries that mention `latency` correlate with deploys 80% of the time.",
  },
];

function makeBlock(overrides: Partial<Parameters<typeof systemPromptAvailable>[0]> = {}) {
  return systemPromptAvailable({
    mountPath: "/test/store",
    agentId: "test-agent",
    sessionId: "session-xyz",
    kinds: builtinKinds,
    activeEvolution: sampleActiveEvents,
    ...overrides,
  });
}

// ---------------------------------------------------------------------------
// Core system-prompt tests (existing coverage)
// ---------------------------------------------------------------------------

describe("systemPromptAvailable", () => {
  const block = makeBlock();

  it("interpolates the mount path", () => {
    expect(block).toContain("/test/store");
  });

  it("interpolates the agent and session ids", () => {
    expect(block).toContain("MYCELIUM_AGENT_ID=test-agent");
    expect(block).toContain("MYCELIUM_SESSION_ID=session-xyz");
  });

  it("documents all nine subcommands", () => {
    for (const sub of ["read", "write", "edit", "ls", "glob", "grep", "rm", "mv", "log"]) {
      expect(block).toContain(`mycelium ${sub}`);
    }
  });

  it("describes the conflict-recovery contract", () => {
    expect(block).toContain("--expected-version");
    expect(block).toContain("exits 64");
    expect(block).toContain("re-read");
  });

  it("documents both conflict envelope variants", () => {
    expect(block).toContain('"error":"conflict"');
    expect(block).toContain('"error":"destination_exists"');
    expect(block).toContain("--include-current-content");
  });

  it("describes the reserved _ prefix and inline payload layout", () => {
    expect(block).toContain("Reserved paths");
    expect(block).toContain("_activity/YYYY/MM/DD/test-agent.jsonl");
    expect(block).not.toContain("logs/YYYY/MM/DD");
    expect(block).toContain("payload");
  });

  it("guides explicit log usage", () => {
    expect(block).toContain("When to log explicitly");
  });
});

// ---------------------------------------------------------------------------
// Evolution kinds section
// ---------------------------------------------------------------------------

describe("systemPromptAvailable — kinds section", () => {
  it("renders a kinds table from a non-empty payload", () => {
    const block = makeBlock({ kinds: builtinKinds });
    expect(block).toContain("### Evolution kinds");
    expect(block).toContain("`convention`");
    expect(block).toContain("`lesson`");
    expect(block).toContain("builtin");
  });

  it("shows agent kinds and their source column", () => {
    const block = makeBlock({ kinds: agentKinds });
    expect(block).toContain("`experiment`");
    expect(block).toContain("agent");
  });

  it("renders the empty-state message when kinds array is empty (binary unavailable)", () => {
    const block = makeBlock({ kinds: [] });
    expect(block).toContain("Evolution surface unavailable");
    expect(block).toContain("mycelium evolution --kinds");
    // Should not render a table header
    expect(block).not.toContain("| Kind |");
  });
});

// ---------------------------------------------------------------------------
// Active evolution section
// ---------------------------------------------------------------------------

describe("systemPromptAvailable — active evolution section", () => {
  it("renders active events from a non-empty payload", () => {
    const block = makeBlock({ activeEvolution: sampleActiveEvents });
    expect(block).toContain("### Active evolution");
    expect(block).toContain("[convention]");
    expect(block).toContain("notes/incidents/");
    expect(block).toContain("Adopting <date>-<slug>.md filenames");
    expect(block).toContain("[lesson]");
    // lesson has no target — should not emit an empty path segment
    expect(block).toContain("[lesson] —");
  });

  it("omits target when the event has no target field", () => {
    const block = makeBlock({ activeEvolution: sampleActiveEvents });
    // lesson entry has no target; line should be "[lesson] — <rationale>"
    // not "[lesson]  — " (double-space) or "[lesson] undefined —"
    const lessonLine = block
      .split("\n")
      .find((l) => l.includes("[lesson]"));
    expect(lessonLine).toBeDefined();
    // Should match the exact no-target pattern: `[lesson] —` with a single space
    expect(lessonLine).toMatch(/\[lesson\] —/);
    // Should NOT have extra tokens between [lesson] and —
    expect(lessonLine).not.toMatch(/\[lesson\] \S+ —/);
  });

  it("truncates long rationale to 200 chars with ellipsis", () => {
    const longRationale = "x".repeat(300);
    const block = makeBlock({
      activeEvolution: [
        {
          ts: "2026-05-01T00:00:00Z",
          agent_id: "a",
          session_id: "s",
          op: "evolve",
          id: "01AAAAAAAAAAAAAAAAAAAAAAAA",
          kind: "lesson",
          rationale: longRationale,
        },
      ],
    });
    expect(block).toContain("x".repeat(200) + "…");
    expect(block).not.toContain("x".repeat(201));
  });

  it("renders the empty-state message when active payload is empty", () => {
    const block = makeBlock({ activeEvolution: [] });
    expect(block).toContain("No active evolution recorded yet");
    expect(block).toContain("mycelium evolve");
  });

  it("truncates list at 10 entries and shows overflow footer", () => {
    const manyEvents: ActiveEvolutionEvent[] = Array.from({ length: 13 }, (_, i) => ({
      ts: "2026-05-01T00:00:00Z",
      agent_id: "a",
      session_id: "s",
      op: "evolve",
      id: `01AAAAAAAAAAAAAAAAAAAA${String(i).padStart(4, "0")}`,
      kind: "lesson",
      rationale: `Lesson ${i}`,
    }));

    const block = makeBlock({ activeEvolution: manyEvents });
    expect(block).toContain("Lesson 0");
    expect(block).toContain("Lesson 9");
    expect(block).not.toContain("Lesson 10");
    expect(block).toContain("...and 3 more");
    expect(block).toContain("mycelium evolution --active");
  });

  it("does not show overflow footer when events are exactly at the limit", () => {
    const tenEvents: ActiveEvolutionEvent[] = Array.from({ length: 10 }, (_, i) => ({
      ts: "2026-05-01T00:00:00Z",
      agent_id: "a",
      session_id: "s",
      op: "evolve",
      id: `01AAAAAAAAAAAAAAAAAAAA${String(i).padStart(4, "0")}`,
      kind: "lesson",
      rationale: `Lesson ${i}`,
    }));

    const block = makeBlock({ activeEvolution: tenEvents });
    expect(block).not.toContain("...and");
  });
});

// ---------------------------------------------------------------------------
// Recording evolution section (always-present static instructions)
// ---------------------------------------------------------------------------

describe("systemPromptAvailable — recording evolution section", () => {
  it("always renders the section regardless of payload state", () => {
    const blockFull = makeBlock({ kinds: builtinKinds, activeEvolution: sampleActiveEvents });
    const blockEmpty = makeBlock({ kinds: [], activeEvolution: [] });

    for (const block of [blockFull, blockEmpty]) {
      expect(block).toContain("### Recording evolution");
      expect(block).toContain("mycelium evolve convention");
      expect(block).toContain("mycelium evolve index");
      expect(block).toContain("mycelium evolve archive");
      expect(block).toContain("mycelium evolve lesson");
      expect(block).toContain("mycelium evolve question");
      expect(block).toContain("--kind-definition");
    }
  });

  it("mentions that evolve is metadata-only and does not mutate the store", () => {
    const block = makeBlock();
    expect(block).toContain("never mutates the store");
  });
});

// ---------------------------------------------------------------------------
// systemPromptUnavailable
// ---------------------------------------------------------------------------

describe("systemPromptUnavailable", () => {
  const block = systemPromptUnavailable({ mountPath: "/missing/store" });

  it("flags the binary as unavailable", () => {
    expect(block).toContain("UNAVAILABLE");
    expect(block).toContain("not on PATH");
  });

  it("interpolates the configured mount path", () => {
    expect(block).toContain("/missing/store");
  });
});
