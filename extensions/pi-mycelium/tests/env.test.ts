import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { execResult } from "./helpers.js";
import type { MyceliumConfig } from "../config.js";

// Module-level mock must come before importing the module under test.
vi.mock("../binary-resolver.js", () => ({
  resolveBundledBinary: vi.fn(() => null),
  resolveMyceliumBinary: vi.fn(async () => null),
}));

// Import after the mock declaration so the mock is in place.
import { resolveBinary, setupEnv } from "../env.js";
import { resolveMyceliumBinary } from "../binary-resolver.js";

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

describe("resolveBinary", () => {
  const mockResolve = vi.mocked(resolveMyceliumBinary);

  it("returns the bundled binary path when the optional package is installed", async () => {
    mockResolve.mockResolvedValueOnce("/bundled/mycelium");
    const pi = { exec: vi.fn() } as unknown as ExtensionAPI;
    const result = await resolveBinary(pi);
    expect(result).toBe("/bundled/mycelium");
  });

  it("falls back to PATH when bundled binary is absent but `which mycelium` succeeds", async () => {
    mockResolve.mockResolvedValueOnce("/usr/local/bin/mycelium");
    const pi = {
      exec: vi.fn(async () => execResult(0, "/usr/local/bin/mycelium")),
    } as unknown as ExtensionAPI;
    const result = await resolveBinary(pi);
    expect(result).toBe("/usr/local/bin/mycelium");
  });

  it("returns null when both bundled and PATH lookups fail", async () => {
    mockResolve.mockResolvedValueOnce(null);
    const pi = {
      exec: vi.fn(async () => execResult(1)),
    } as unknown as ExtensionAPI;
    const result = await resolveBinary(pi);
    expect(result).toBeNull();
  });
});
