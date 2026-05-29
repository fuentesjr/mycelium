import path from "node:path";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import type { MyceliumConfig } from "../config.js";
import { setupEnv } from "../binary-resolver.js";

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
		setupEnv(cfg, "session-leaf-1", null);
		expect(process.env.MYCELIUM_AGENT_ID).toBe("pi-agent");
	});

	it("preserves MYCELIUM_AGENT_ID when already set (operator override)", () => {
		process.env.MYCELIUM_AGENT_ID = "researcher-7";
		setupEnv(cfg, "session-leaf-1", null);
		expect(process.env.MYCELIUM_AGENT_ID).toBe("researcher-7");
	});

	it("always sets MYCELIUM_SESSION_ID from the provided leaf id", () => {
		process.env.MYCELIUM_SESSION_ID = "stale";
		setupEnv(cfg, "fresh-leaf", null);
		expect(process.env.MYCELIUM_SESSION_ID).toBe("fresh-leaf");
	});

	it("always sets MYCELIUM_MOUNT from the config", () => {
		setupEnv(cfg, "leaf", null);
		expect(process.env.MYCELIUM_MOUNT).toBe("/tmp/store");
	});

	it("auto-generates MYCELIUM_SESSION_ID when leaf id is null", () => {
		process.env.MYCELIUM_SESSION_ID = "stale";
		setupEnv(cfg, null, null);
		expect(process.env.MYCELIUM_SESSION_ID).toMatch(/^pi-auto-/);
		expect(process.env.MYCELIUM_SESSION_ID).not.toBe("stale");
	});

	it("prepends the bundled binary's directory to PATH so agent shells can find it", () => {
		process.env.PATH = "/usr/bin:/bin";
		setupEnv(
			cfg,
			"leaf",
			"/some/install/node_modules/@fuentesjr/mycelium-cli-darwin-amd64/mycelium",
		);
		expect(process.env.PATH).toBe(
			`/some/install/node_modules/@fuentesjr/mycelium-cli-darwin-amd64${path.delimiter}/usr/bin:/bin`,
		);
	});

	it("does not duplicate PATH entries when called twice", () => {
		process.env.PATH = "/usr/bin";
		setupEnv(cfg, "leaf", "/bundled/mycelium");
		setupEnv(cfg, "leaf", "/bundled/mycelium");
		expect(process.env.PATH).toBe(`/bundled${path.delimiter}/usr/bin`);
	});

	it("leaves PATH alone when binaryPath is null", () => {
		process.env.PATH = "/usr/bin";
		setupEnv(cfg, "leaf", null);
		expect(process.env.PATH).toBe("/usr/bin");
	});
});
