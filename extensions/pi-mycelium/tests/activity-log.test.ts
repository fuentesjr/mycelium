import { createRequire } from "node:module";
import { describe, it, expect, vi } from "vitest";
import type {
	ExtensionAPI,
	SessionStartEvent,
} from "@earendil-works/pi-coding-agent";
import { createActivityLogRecorder } from "../activity-log.js";
import { execResult } from "./helpers.js";

const require = createRequire(import.meta.url);
const BINARY_PATH = "/resolved/mycelium";
const ADAPTER_VERSION = (require("../package.json") as { version: string })
	.version;

function argsFromCall(exec: ReturnType<typeof vi.fn>, callIndex = 0): string[] {
	return (exec.mock.calls[callIndex] as unknown as [string, string[]])[1];
}

function payloadFromCall(
	exec: ReturnType<typeof vi.fn>,
	callIndex = 0,
): Record<string, unknown> {
	const args = argsFromCall(exec, callIndex);
	expect(args[2]).toBe("--payload-json");
	try {
		return JSON.parse(args[3]) as Record<string, unknown>;
	} catch (error) {
		throw new Error(`invalid payload JSON: ${String(error)}`);
	}
}

describe("ActivityLogRecorder.recordSessionBoundary", () => {
	const cases: ReadonlyArray<[SessionStartEvent["reason"], string]> = [
		["new", "session_new"],
		["resume", "session_resume"],
		["fork", "session_fork"],
		["startup", "session_startup"],
		["reload", "session_reload"],
	];

	for (const [reason, expectedOp] of cases) {
		it(`logs ${expectedOp} for reason=${reason}`, async () => {
			const exec = vi.fn(async () => execResult(0));
			const pi = { exec } as unknown as ExtensionAPI;
			const recorder = createActivityLogRecorder();
			await recorder.recordSessionBoundary(pi, BINARY_PATH, reason);

			expect(exec).toHaveBeenCalledTimes(1);
			const args = argsFromCall(exec);
			expect(args[0]).toBe("log");
			expect(args[1]).toBe(expectedOp);
			expect(payloadFromCall(exec)).toEqual({
				harness: "pi.dev",
				adapter_version: ADAPTER_VERSION,
				seq: 1,
				session_reason: reason,
			});
		});
	}
});
