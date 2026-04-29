import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type {
  BeforeAgentStartEvent,
  BeforeAgentStartEventResult,
  BuildSystemPromptOptions,
  ContextEvent,
  ExecResult,
  ExtensionAPI,
  ExtensionContext,
} from "@mariozechner/pi-coding-agent";
import register from "../src/index.js";
import { execResult } from "./helpers.js";

type AnyHandler = (event: unknown, ctx: ExtensionContext) => unknown;

function makeRegistration(execImpl: (cmd: string, args: string[]) => Promise<ExecResult>) {
  const handlers = new Map<string, AnyHandler>();
  const exec = vi.fn(execImpl);
  const pi = {
    on: vi.fn((event: string, fn: AnyHandler) => handlers.set(event, fn)),
    exec,
  } as unknown as ExtensionAPI;
  register(pi);
  return { exec, handlers };
}

const ctx = {
  cwd: "/test/cwd",
  sessionManager: { getLeafId: () => "test-leaf" },
} as unknown as ExtensionContext;

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
    const { handlers } = makeRegistration(async () => execResult(0));
    await handlers.get("session_start")!(undefined, ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent("EXISTING-CONTENT"),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt?.startsWith("EXISTING-CONTENT\n\n")).toBe(true);
    expect(result.systemPrompt).toContain("Mycelium memory");
  });

  it("emits the UNAVAILABLE block when binary is missing", async () => {
    const { handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(undefined, ctx);
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt).toContain("UNAVAILABLE");
  });

  it("context handler calls mycelium log and returns undefined", async () => {
    const { exec, handlers } = makeRegistration(async () => execResult(0));
    await handlers.get("session_start")!(undefined, ctx);
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
    expect(exec).toHaveBeenCalledWith("mycelium", [
      "log",
      "context_signal",
      "--payload-json",
      JSON.stringify({ messageCount: 3, lastRole: "user" }),
    ]);
  });

  it("context handler is a no-op when binary is missing", async () => {
    const { exec, handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(undefined, ctx);
    exec.mockClear();
    const result = await handlers.get("context")!(
      makeContextEvent([{ role: "user", content: "", timestamp: 0 }]),
      ctx,
    );
    expect(result).toBeUndefined();
    expect(exec).not.toHaveBeenCalled();
  });
});
