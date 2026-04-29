import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { isBinaryAvailable, setupEnv } from "../src/env.js";
import type { MyceliumConfig } from "../src/config.js";
import { execResult } from "./helpers.js";

const cfg: MyceliumConfig = { scope: "project", mountPath: "/tmp/store" };

describe("setupEnv", () => {
  const original = { ...process.env };

  beforeEach(() => {
    delete process.env.MYCELIUM_AGENT_ID;
    delete process.env.MYCELIUM_SESSION_ID;
    delete process.env.MYCELIUM_MOUNT;
  });

  afterEach(() => {
    process.env = { ...original };
  });

  it("defaults MYCELIUM_AGENT_ID to 'pi-agent' when unset", () => {
    setupEnv(cfg, "session-leaf-1");
    expect(process.env.MYCELIUM_AGENT_ID).toBe("pi-agent");
  });

  it("preserves MYCELIUM_AGENT_ID when already set (operator override)", () => {
    process.env.MYCELIUM_AGENT_ID = "researcher-7";
    setupEnv(cfg, "session-leaf-1");
    expect(process.env.MYCELIUM_AGENT_ID).toBe("researcher-7");
  });

  it("always sets MYCELIUM_SESSION_ID from the provided leaf id", () => {
    process.env.MYCELIUM_SESSION_ID = "stale";
    setupEnv(cfg, "fresh-leaf");
    expect(process.env.MYCELIUM_SESSION_ID).toBe("fresh-leaf");
  });

  it("always sets MYCELIUM_MOUNT from the config", () => {
    setupEnv(cfg, "leaf");
    expect(process.env.MYCELIUM_MOUNT).toBe("/tmp/store");
  });

  it("clears MYCELIUM_SESSION_ID when leaf id is null", () => {
    process.env.MYCELIUM_SESSION_ID = "stale";
    setupEnv(cfg, null);
    expect(process.env.MYCELIUM_SESSION_ID).toBeUndefined();
  });
});

describe("isBinaryAvailable", () => {
  it("returns true when `which mycelium` exits 0", async () => {
    const pi = { exec: vi.fn(async () => execResult(0)) } as unknown as ExtensionAPI;
    expect(await isBinaryAvailable(pi)).toBe(true);
    expect(pi.exec).toHaveBeenCalledWith("which", ["mycelium"]);
  });

  it("returns false when `which mycelium` exits non-zero", async () => {
    const pi = { exec: vi.fn(async () => execResult(1)) } as unknown as ExtensionAPI;
    expect(await isBinaryAvailable(pi)).toBe(false);
  });
});
