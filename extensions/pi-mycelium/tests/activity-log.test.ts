import { createRequire } from "node:module";
import { describe, it, expect, vi } from "vitest";
import type {
	ContextEvent,
	ExtensionAPI,
	SessionStartEvent,
} from "@earendil-works/pi-coding-agent";
import { createActivityLogRecorder } from "../activity-log.js";
import { execResult } from "./helpers.js";

const require = createRequire(import.meta.url);
const BINARY_PATH = "/resolved/mycelium";
const ADAPTER_VERSION = (require("../package.json") as { version: string })
	.version;

function makeContextEvent(messages: ContextEvent["messages"]): ContextEvent {
	return { type: "context", messages };
}

function argsFromCall(exec: ReturnType<typeof vi.fn>, callIndex = 0): string[] {
	return (exec.mock.calls[callIndex] as unknown as [string, string[]])[1];
}

function payloadFromCall(
	exec: ReturnType<typeof vi.fn>,
	callIndex = 0,
): Record<string, unknown> {
	const args = argsFromCall(exec, callIndex);
	expect(args[2]).toBe("--payload-json");
	return JSON.parse(args[3]) as Record<string, unknown>;
}

const sampleMessages: ContextEvent["messages"] = [
	{ role: "user", content: "hi", timestamp: 0 },
	{
		role: "assistant",
		content: [],
		api: "anthropic-messages",
		provider: "anthropic",
		model: "m",
		usage: {
			input: 1,
			output: 2,
			cacheRead: 3,
			cacheWrite: 4,
			totalTokens: 10,
			cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0.5 },
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
];

describe("ActivityLogRecorder.recordContextCheckpoint", () => {
	it("emits the portable context_checkpoint payload", async () => {
		const exec = vi.fn(async () => execResult(0));
		const pi = { exec } as unknown as ExtensionAPI;
		const recorder = createActivityLogRecorder();

		await recorder.recordContextCheckpoint(
			pi,
			BINARY_PATH,
			makeContextEvent(sampleMessages),
		);

		expect(exec).toHaveBeenCalledTimes(1);
		const args = argsFromCall(exec);
		expect(args[0]).toBe("log");
		expect(args[1]).toBe("context_checkpoint");

		const payload = payloadFromCall(exec);
		expect(payload).toMatchObject({
			harness: "pi.dev",
			adapter_version: ADAPTER_VERSION,
			seq: 1,
			message_count: 3,
			last_role: "toolResult",
			role_counts: { user: 1, assistant: 1, toolResult: 1 },
		});
		expect(payload.fingerprint).toMatch(/^sha256:/);
		expect(payload).not.toHaveProperty("message_delta");
	});

	it("omits last_role when the messages array is empty", async () => {
		const exec = vi.fn(async () => execResult(0));
		const pi = { exec } as unknown as ExtensionAPI;
		const recorder = createActivityLogRecorder();

		await recorder.recordContextCheckpoint(
			pi,
			BINARY_PATH,
			makeContextEvent([]),
		);

		const payload = payloadFromCall(exec);
		expect(payload.message_count).toBe(0);
		expect(payload.role_counts).toEqual({});
		expect(payload).not.toHaveProperty("last_role");
	});

	it("suppresses duplicate checkpoints and reports the count on the next distinct checkpoint", async () => {
		const exec = vi.fn(async () => execResult(0));
		const pi = { exec } as unknown as ExtensionAPI;
		const recorder = createActivityLogRecorder();
		const first = makeContextEvent(sampleMessages);
		const second = makeContextEvent([
			...sampleMessages,
			{ role: "user", content: "next", timestamp: 1 },
		]);

		await recorder.recordContextCheckpoint(pi, BINARY_PATH, first);
		await recorder.recordContextCheckpoint(pi, BINARY_PATH, first);
		await recorder.recordContextCheckpoint(pi, BINARY_PATH, second);

		expect(exec).toHaveBeenCalledTimes(2);
		const payload = payloadFromCall(exec, 1);
		expect(payload.seq).toBe(2);
		expect(payload.message_count).toBe(4);
		expect(payload.message_delta).toBe(1);
		expect(payload.suppressed_duplicates).toBe(1);
	});
});

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
