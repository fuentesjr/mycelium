import { describe, it, expect, vi } from "vitest";
import type {
  ContextEvent,
  ExtensionAPI,
  SessionStartEvent,
} from "@mariozechner/pi-coding-agent";
import {
  recordContextSignal,
  recordSessionBoundary,
} from "../src/activity-log.js";
import { execResult } from "./helpers.js";

function makeContextEvent(messages: ContextEvent["messages"]): ContextEvent {
  return { type: "context", messages };
}

describe("recordContextSignal", () => {
  it("invokes mycelium log with op=context_signal and the messageCount + lastRole payload", async () => {
    const exec = vi.fn(async () => execResult(0));
    const pi = { exec } as unknown as ExtensionAPI;
    await recordContextSignal(
      pi,
      makeContextEvent([
        { role: "user", content: "hi", timestamp: 0 },
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
        {
          role: "toolResult",
          toolCallId: "t1",
          toolName: "read",
          content: [],
          isError: false,
          timestamp: 0,
        },
      ]),
    );

    expect(exec).toHaveBeenCalledTimes(1);
    expect(exec).toHaveBeenCalledWith("mycelium", [
      "log",
      "context_signal",
      "--payload-json",
      JSON.stringify({ messageCount: 3, lastRole: "toolResult" }),
    ]);
  });

  it("omits lastRole when the messages array is empty", async () => {
    const exec = vi.fn(async () => execResult(0));
    const pi = { exec } as unknown as ExtensionAPI;
    await recordContextSignal(pi, makeContextEvent([]));

    expect(exec).toHaveBeenCalledWith("mycelium", [
      "log",
      "context_signal",
      "--payload-json",
      JSON.stringify({ messageCount: 0 }),
    ]);
  });
});

describe("recordSessionBoundary", () => {
  const cases: ReadonlyArray<[SessionStartEvent["reason"], string | null]> = [
    ["new", "session_new"],
    ["resume", "session_resume"],
    ["fork", "session_fork"],
    ["startup", null],
    ["reload", null],
  ];

  for (const [reason, expectedOp] of cases) {
    it(`${expectedOp ? "logs" : "skips"} for reason=${reason}`, async () => {
      const exec = vi.fn(async () => execResult(0));
      const pi = { exec } as unknown as ExtensionAPI;
      await recordSessionBoundary(pi, reason);
      if (expectedOp) {
        expect(exec).toHaveBeenCalledWith("mycelium", ["log", expectedOp]);
      } else {
        expect(exec).not.toHaveBeenCalled();
      }
    });
  }
});
