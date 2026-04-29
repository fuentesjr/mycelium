import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import register from "../src/index.js";

type Handler = (event: any, ctx: any) => any;

function makeRegistration(execImpl: (cmd: string, args: string[]) => Promise<{ exitCode: number }>) {
  const handlers = new Map<string, Handler>();
  const pi = {
    on: vi.fn((event: string, fn: Handler) => handlers.set(event, fn)),
    exec: vi.fn(execImpl),
  };
  register(pi as any);
  return { pi, handlers };
}

const ctx = {
  cwd: "/test/cwd",
  sessionManager: { getLeafId: () => "test-leaf" },
} as any;

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
    const { handlers } = makeRegistration(async () => ({ exitCode: 0 }));
    expect(handlers.has("session_start")).toBe(true);
    expect(handlers.has("before_agent_start")).toBe(true);
    expect(handlers.has("context")).toBe(true);
  });

  it("chains before_agent_start systemPrompt off the incoming event", async () => {
    const { handlers } = makeRegistration(async () => ({ exitCode: 0 }));
    await handlers.get("session_start")!(undefined, ctx);
    const result = await handlers.get("before_agent_start")!(
      { systemPrompt: "EXISTING-CONTENT" },
      ctx,
    );
    expect(result.systemPrompt.startsWith("EXISTING-CONTENT\n\n")).toBe(true);
    expect(result.systemPrompt).toContain("Mycelium memory");
  });

  it("emits the UNAVAILABLE block when binary is missing", async () => {
    const { handlers } = makeRegistration(async () => ({ exitCode: 1 }));
    await handlers.get("session_start")!(undefined, ctx);
    const result = await handlers.get("before_agent_start")!(
      { systemPrompt: "" },
      ctx,
    );
    expect(result.systemPrompt).toContain("UNAVAILABLE");
  });

  it("context handler calls mycelium log and returns undefined", async () => {
    const { pi, handlers } = makeRegistration(async () => ({ exitCode: 0 }));
    await handlers.get("session_start")!(undefined, ctx);
    pi.exec.mockClear();
    const result = await handlers.get("context")!({ messages: [1, 2, 3] }, ctx);
    expect(result).toBeUndefined();
    expect(pi.exec).toHaveBeenCalledWith("mycelium", [
      "log",
      "context_signal",
      "--payload-json",
      JSON.stringify({ messageCount: 3 }),
    ]);
  });

  it("context handler is a no-op when binary is missing", async () => {
    const { pi, handlers } = makeRegistration(async () => ({ exitCode: 1 }));
    await handlers.get("session_start")!(undefined, ctx);
    pi.exec.mockClear();
    const result = await handlers.get("context")!({ messages: [1] }, ctx);
    expect(result).toBeUndefined();
    expect(pi.exec).not.toHaveBeenCalled();
  });
});
