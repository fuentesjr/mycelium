import { describe, it, expect, vi } from "vitest";
import { recordContextSignal } from "../src/activity-log.js";

describe("recordContextSignal", () => {
  it("invokes mycelium log with op=context_signal and a JSON payload", async () => {
    const pi = { exec: vi.fn(async () => ({ exitCode: 0 })) } as any;
    await recordContextSignal(pi, { messages: [{}, {}, {}] });

    expect(pi.exec).toHaveBeenCalledTimes(1);
    const [cmd, args] = pi.exec.mock.calls[0];
    expect(cmd).toBe("mycelium");
    expect(args).toEqual([
      "log",
      "context_signal",
      "--payload-json",
      JSON.stringify({ messageCount: 3 }),
    ]);
  });
});
