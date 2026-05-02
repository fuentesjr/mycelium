import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type {
  BeforeAgentStartEvent,
  BeforeAgentStartEventResult,
  BuildSystemPromptOptions,
  ContextEvent,
  ExecResult,
  ExtensionAPI,
  ExtensionContext,
  SessionStartEvent,
} from "@mariozechner/pi-coding-agent";
import register from "../index.js";
import { execResult } from "./helpers.js";
import type { EvolutionKindRow, ActiveEvolutionEvent } from "../system-prompt.js";

type AnyHandler = (event: unknown, ctx: ExtensionContext) => unknown;

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const RESOLVED_BINARY = "/usr/local/bin/mycelium";

const sampleKinds: EvolutionKindRow[] = [
  {
    name: "convention",
    definition: "A naming, layout, or structural pattern.",
    defined_at_version: "0.1.0",
    source: "builtin",
    event_count: 1,
  },
];

const sampleActiveEvent: ActiveEvolutionEvent = {
  ts: "2026-05-02T14:00:00Z",
  agent_id: "pi-agent",
  session_id: "s1",
  op: "evolve",
  id: "01HXKP4Z9M8YV1W6E2RTSA9KFG",
  kind: "convention",
  target: "notes/",
  rationale: "Use date-slug filenames.",
};

// ---------------------------------------------------------------------------
// Factory helpers
// ---------------------------------------------------------------------------

/**
 * Build a pi registration whose exec mock dispatches based on the args passed.
 * `routedExec` receives (cmd, args) and returns the ExecResult to use for that
 * call. Defaults to execResult(0) for unrecognised calls.
 */
function makeRegistration(
  routedExec: (cmd: string, args: string[]) => Promise<ExecResult> = async () => execResult(0),
) {
  const handlers = new Map<string, AnyHandler>();
  const exec = vi.fn(routedExec);
  const pi = {
    on: vi.fn((event: string, fn: AnyHandler) => handlers.set(event, fn)),
    exec,
  } as unknown as ExtensionAPI;
  register(pi);
  return { exec, handlers };
}

/**
 * Default exec that:
 * - passes `which mycelium` → returns the resolved binary path
 * - returns sampleKinds JSON for `--kinds`
 * - returns one active event NDJSON for `--active`
 * - returns success for everything else
 *
 * The first arg will be "which" for the PATH lookup, and RESOLVED_BINARY
 * (or any non-"which" string) for actual mycelium invocations.
 */
function defaultExec(cmd: string, args: string[]): Promise<ExecResult> {
  if (cmd === "which") return Promise.resolve(execResult(0, RESOLVED_BINARY));
  // Any other first arg is the resolved binary path
  if (args.includes("--kinds")) {
    return Promise.resolve(execResult(0, JSON.stringify(sampleKinds)));
  }
  if (args.includes("--active")) {
    return Promise.resolve(execResult(0, JSON.stringify(sampleActiveEvent)));
  }
  return Promise.resolve(execResult(0));
}

// ---------------------------------------------------------------------------
// Context / event builders
// ---------------------------------------------------------------------------

const ctx = {
  cwd: "/test/cwd",
  sessionManager: { getLeafId: () => "test-leaf" },
} as unknown as ExtensionContext;

function makeSessionStartEvent(
  reason: SessionStartEvent["reason"] = "startup",
): SessionStartEvent {
  return { type: "session_start", reason };
}

function makeBeforeAgentStartEvent(systemPrompt: string): BeforeAgentStartEvent {
  return {
    type: "before_agent_start",
    prompt: "",
    systemPrompt,
    systemPromptOptions: {} as BuildSystemPromptOptions,
  };
}

function makeContextEvent(messages: ContextEvent["messages"]): ContextEvent {
  return { type: "context", messages };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("pi extension factory", () => {
  const original = { ...process.env };

  beforeEach(() => {
    delete process.env.MYCELIUM_AGENT_ID;
    delete process.env.MYCELIUM_SESSION_ID;
    delete process.env.MYCELIUM_MOUNT;
  });

  afterEach(() => {
    process.env = { ...original };
  });

  it("registers session_start, before_agent_start, and context handlers", () => {
    const { handlers } = makeRegistration(async () => execResult(0));
    expect(handlers.has("session_start")).toBe(true);
    expect(handlers.has("before_agent_start")).toBe(true);
    expect(handlers.has("context")).toBe(true);
  });

  it("chains before_agent_start systemPrompt off the incoming event", async () => {
    const { handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent("EXISTING-CONTENT"),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt?.startsWith("EXISTING-CONTENT\n\n")).toBe(true);
    expect(result.systemPrompt).toContain("Mycelium memory");
  });

  it("emits the UNAVAILABLE block when binary is missing", async () => {
    const { handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt).toContain("UNAVAILABLE");
  });

  it("emits the UNAVAILABLE block when resolveMyceliumBinary returns null (no bundled, which fails)", async () => {
    // All exec calls fail — which fails and there's no bundled binary
    const { handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent("PREFIX"),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt).toContain("UNAVAILABLE");
    expect(result.systemPrompt?.startsWith("PREFIX\n\n")).toBe(true);
  });

  it("context handler calls mycelium log and returns undefined", async () => {
    const { exec, handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    exec.mockClear();
    const result = await handlers.get("context")!(
      makeContextEvent([
        { role: "user", content: "", timestamp: 0 },
        {
          role: "assistant",
          content: [],
          api: "anthropic-messages",
          provider: "anthropic",
          model: "m",
          usage: {
            input: 0,
            output: 0,
            cacheRead: 0,
            cacheWrite: 0,
            totalTokens: 0,
            cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0 },
          },
          stopReason: "stop",
          timestamp: 0,
        },
        { role: "user", content: "", timestamp: 0 },
      ]),
      ctx,
    );
    expect(result).toBeUndefined();
    expect(exec).toHaveBeenCalledWith(RESOLVED_BINARY, [
      "log",
      "context_signal",
      "--payload-json",
      JSON.stringify({ messageCount: 3, lastRole: "user" }),
    ]);
  });

  it("context handler is a no-op when binary is missing", async () => {
    const { exec, handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    exec.mockClear();
    const result = await handlers.get("context")!(
      makeContextEvent([{ role: "user", content: "", timestamp: 0 }]),
      ctx,
    );
    expect(result).toBeUndefined();
    expect(exec).not.toHaveBeenCalled();
  });

  it("logs session_new for reason=new when binary is available", async () => {
    const { exec, handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent("new"), ctx);
    expect(exec).toHaveBeenCalledWith(RESOLVED_BINARY, ["log", "session_new"]);
  });

  it("does not log a boundary for reason=startup", async () => {
    const { exec, handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent("startup"), ctx);
    const boundaryCalls = exec.mock.calls.filter(
      ([, args]) =>
        Array.isArray(args) && args[0] === "log" && typeof args[1] === "string" && args[1].startsWith("session_"),
    );
    expect(boundaryCalls).toHaveLength(0);
  });

  it("does not log a boundary when binary is missing", async () => {
    const { exec, handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(makeSessionStartEvent("fork"), ctx);
    const boundaryCalls = exec.mock.calls.filter(
      ([, args]) => Array.isArray(args) && args[0] === "log",
    );
    expect(boundaryCalls).toHaveLength(0);
  });

  // -------------------------------------------------------------------------
  // Evolution binary calls wiring
  // -------------------------------------------------------------------------

  it("invokes mycelium evolution --kinds --format json in before_agent_start", async () => {
    const { exec, handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    exec.mockClear();
    await handlers.get("before_agent_start")!(makeBeforeAgentStartEvent(""), ctx);
    expect(exec).toHaveBeenCalledWith(RESOLVED_BINARY, ["evolution", "--kinds", "--format", "json"]);
  });

  it("invokes mycelium evolution --active --format json in before_agent_start", async () => {
    const { exec, handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    exec.mockClear();
    await handlers.get("before_agent_start")!(makeBeforeAgentStartEvent(""), ctx);
    expect(exec).toHaveBeenCalledWith(RESOLVED_BINARY, ["evolution", "--active", "--format", "json"]);
  });

  it("passes kinds payload into the system prompt when binary returns data", async () => {
    const { handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    // sampleKinds contains "convention" — should appear in the kinds table
    expect(result.systemPrompt).toContain("`convention`");
    expect(result.systemPrompt).toContain("builtin");
  });

  it("passes active evolution payload into the system prompt when binary returns data", async () => {
    const { handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt).toContain("[convention]");
    expect(result.systemPrompt).toContain("notes/");
    expect(result.systemPrompt).toContain("Use date-slug filenames.");
  });

  it("falls through with empty arrays when --kinds call fails (non-zero exit)", async () => {
    const { handlers } = makeRegistration(async (cmd, args) => {
      if (cmd === "which") return execResult(0, RESOLVED_BINARY);
      if (args.includes("--kinds")) return execResult(1);
      if (args.includes("--active"))
        return execResult(0, JSON.stringify(sampleActiveEvent));
      return execResult(0);
    });
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    // Should not crash; should show the empty-state kinds message
    expect(result.systemPrompt).toContain("Evolution surface unavailable");
    // Active data should still render
    expect(result.systemPrompt).toContain("[convention]");
  });

  it("falls through with empty arrays when --active call fails (non-zero exit)", async () => {
    const { handlers } = makeRegistration(async (cmd, args) => {
      if (cmd === "which") return execResult(0, RESOLVED_BINARY);
      if (args.includes("--kinds"))
        return execResult(0, JSON.stringify(sampleKinds));
      if (args.includes("--active")) return execResult(1);
      return execResult(0);
    });
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    // Should not crash; kinds render normally
    expect(result.systemPrompt).toContain("`convention`");
    // Active section shows empty-state message
    expect(result.systemPrompt).toContain("No active evolution recorded yet");
  });

  it("falls through with empty arrays when both evolution calls fail", async () => {
    const { handlers } = makeRegistration(async (cmd) => {
      if (cmd === "which") return execResult(0, RESOLVED_BINARY);
      // All mycelium calls fail
      return execResult(1);
    });
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    // Should not throw
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt).toBeDefined();
    expect(result.systemPrompt).toContain("Evolution surface unavailable");
    expect(result.systemPrompt).toContain("No active evolution recorded yet");
  });

  it("does not invoke evolution calls when binary is missing", async () => {
    const { exec, handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    exec.mockClear();
    await handlers.get("before_agent_start")!(makeBeforeAgentStartEvent(""), ctx);
    // No calls should have been made with evolution args
    const evolutionCalls = exec.mock.calls.filter(
      ([, args]) => Array.isArray(args) && args[0] === "evolution",
    );
    expect(evolutionCalls).toHaveLength(0);
  });
});
