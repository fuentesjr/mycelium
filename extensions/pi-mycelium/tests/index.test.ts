import { createRequire } from "node:module";
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
} from "@earendil-works/pi-coding-agent";

// Mock bootstrap so tests don't touch the real filesystem mount or spawn the
// (non-existent) mycelium binary. bootstrap.ts has its own dedicated test file.
vi.mock("../bootstrap.js", () => ({
  bootstrapMemoryFile: vi.fn(async () => {}),
  MEMORY_FILE: "MYCELIUM_MEMORY.md",
}));

vi.mock("../binary-resolver.js", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../binary-resolver.js")>();
  return {
    ...actual,
    resolveBundledBinary: vi.fn(() => null),
    resolveMyceliumBinary: vi.fn(async (pi) => {
    const r = await pi.exec("which", ["mycelium"]);
    if (r.code === 0) {
      const out = r.stdout.trim();
      if (out) return out;
    }
    return null;
    }),
  };
});

import register from "../index.js";
import { execResult } from "./helpers.js";
import { bootstrapMemoryFile } from "../bootstrap.js";

type AnyHandler = (event: unknown, ctx: ExtensionContext) => unknown;

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const require = createRequire(import.meta.url);
const RESOLVED_BINARY = "/usr/local/bin/mycelium";
const ADAPTER_VERSION = (require("../package.json") as { version: string }).version;

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
 * - returns success for everything else
 *
 * The first arg will be "which" for the PATH lookup, and RESOLVED_BINARY
 * (or any non-"which" string) for actual mycelium invocations.
 */
function defaultExec(cmd: string, args: string[]): Promise<ExecResult> {
  if (cmd === "which") return Promise.resolve(execResult(0, RESOLVED_BINARY));
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

function argsFromExecCall(exec: ReturnType<typeof vi.fn>, callIndex = 0): string[] {
  return (exec.mock.calls[callIndex] as unknown as [string, string[]])[1];
}

function payloadFromExecCall(exec: ReturnType<typeof vi.fn>, callIndex = 0): Record<string, unknown> {
  const args = argsFromExecCall(exec, callIndex);
  expect(args[2]).toBe("--payload-json");
  return JSON.parse(args[3]) as Record<string, unknown>;
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

  it("registers lifecycle, context, turn, tool, and compaction handlers", () => {
    const { handlers } = makeRegistration(async () => execResult(0));
    for (const eventName of [
      "session_start",
      "session_shutdown",
      "before_agent_start",
      "turn_start",
      "turn_end",
      "tool_execution_start",
      "tool_execution_end",
      "session_compact",
      "context",
    ]) {
      expect(handlers.has(eventName)).toBe(true);
    }
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
    expect(exec).toHaveBeenCalledTimes(1);
    const args = argsFromExecCall(exec);
    expect(args[0]).toBe("log");
    expect(args[1]).toBe("context_checkpoint");
    const payload = payloadFromExecCall(exec);
    expect(payload).toMatchObject({
      harness: "pi.dev",
      adapter_version: ADAPTER_VERSION,
      seq: 2,
      message_count: 3,
      last_role: "user",
      role_counts: { user: 2, assistant: 1 },
      provider: "anthropic",
      model: "m",
      stop_reason: "stop",
    });
    expect(payload.fingerprint).toMatch(/^sha256:/);
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
    expect(exec).toHaveBeenCalledWith(
      RESOLVED_BINARY,
      expect.arrayContaining(["log", "session_new", "--payload-json"]),
    );
    expect(payloadFromExecCall(exec, 1)).toMatchObject({
      session_reason: "new",
      harness: "pi.dev",
    });
  });

  it("logs session_startup for reason=startup", async () => {
    const { exec, handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent("startup"), ctx);
    expect(exec).toHaveBeenCalledWith(
      RESOLVED_BINARY,
      expect.arrayContaining(["log", "session_startup", "--payload-json"]),
    );
    expect(payloadFromExecCall(exec, 1)).toMatchObject({
      session_reason: "startup",
      harness: "pi.dev",
    });
  });

  it("invokes bootstrapMemoryFile with the resolved binary and the mount path", async () => {
    const mocked = vi.mocked(bootstrapMemoryFile);
    mocked.mockClear();
    const { handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    expect(mocked).toHaveBeenCalledTimes(1);
    const [binary, mount] = mocked.mock.calls[0];
    expect(binary).toBe(RESOLVED_BINARY);
    expect(mount).toContain("pi-mycelium");
    expect(mount).toContain("journal");
  });

  it("does not invoke bootstrapMemoryFile when binary is missing", async () => {
    const mocked = vi.mocked(bootstrapMemoryFile);
    mocked.mockClear();
    const { handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    expect(mocked).not.toHaveBeenCalled();
  });

  it("does not log a boundary when binary is missing", async () => {
    const { exec, handlers } = makeRegistration(async () => execResult(1));
    await handlers.get("session_start")!(makeSessionStartEvent("fork"), ctx);
    const boundaryCalls = exec.mock.calls.filter(
      ([, args]) => Array.isArray(args) && args[0] === "log",
    );
    expect(boundaryCalls).toHaveLength(0);
  });

  it("does not query the binary during before_agent_start", async () => {
    const { exec, handlers } = makeRegistration(defaultExec);
    await handlers.get("session_start")!(makeSessionStartEvent(), ctx);
    exec.mockClear();
    const result = (await handlers.get("before_agent_start")!(
      makeBeforeAgentStartEvent(""),
      ctx,
    )) as BeforeAgentStartEventResult;
    expect(result.systemPrompt).toContain("### Conventions file");
    expect(exec).not.toHaveBeenCalled();
  });
});
