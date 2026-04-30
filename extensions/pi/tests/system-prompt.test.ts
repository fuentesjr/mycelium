import { describe, it, expect } from "vitest";
import { systemPromptAvailable, systemPromptUnavailable } from "../src/system-prompt.js";

describe("systemPromptAvailable", () => {
  const block = systemPromptAvailable({
    mountPath: "/test/store",
    agentId: "test-agent",
    sessionId: "session-xyz",
  });

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
