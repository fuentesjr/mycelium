import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const fixturePath = fileURLToPath(
	new URL(
		"../../../docs/fixtures/portable-activity-events.jsonl",
		import.meta.url,
	),
);

const portableOps = new Set([
	"session_startup",
	"session_reload",
	"session_shutdown",
	"session_new",
	"session_resume",
	"session_fork",
	"turn_start",
	"turn_end",
	"tool_start",
	"tool_end",
	"context_checkpoint",
	"agent_note",
	"decision",
	"compaction",
]);

type FixtureEntry = {
	ts?: unknown;
	agent_id?: unknown;
	session_id?: unknown;
	op?: unknown;
	payload?: unknown;
};

describe("portable activity events fixture", () => {
	it("contains valid JSONL entries using documented portable payload fields", () => {
		const lines = readFileSync(fixturePath, "utf8").trim().split("\n");
		expect(lines.length).toBeGreaterThan(0);

		const entries = lines.map((line) => JSON.parse(line) as FixtureEntry);

		for (const entry of entries) {
			expect(typeof entry.ts).toBe("string");
			expect(Number.isNaN(Date.parse(entry.ts as string))).toBe(false);
			expect(typeof entry.agent_id).toBe("string");
			expect(typeof entry.session_id).toBe("string");
			expect(typeof entry.op).toBe("string");
			expect(portableOps.has(entry.op as string)).toBe(true);

			expect(isRecord(entry.payload)).toBe(true);
			const payload = entry.payload as Record<string, unknown>;
			expect(typeof payload.harness).toBe("string");
			expect(typeof payload.adapter_version).toBe("string");
			expect(typeof payload.seq).toBe("number");

			// Portable payloads should use the documented snake_case field names.
			expect(payload).not.toHaveProperty("messageCount");
			expect(payload).not.toHaveProperty("lastRole");
		}

		const checkpoint = entries.find((e) => e.op === "context_checkpoint")!
			.payload as Record<string, unknown>;
		expect(checkpoint.fingerprint).toMatch(/^sha256:/);
		expect(checkpoint.role_counts).toEqual({ user: 1, assistant: 1 });

		const toolEnd = entries.find((e) => e.op === "tool_end")!.payload as Record<
			string,
			unknown
		>;
		expect(toolEnd).toMatchObject({
			tool_call_id: "call_123",
			tool_name: "bash",
			is_error: false,
			duration_ms: 384,
			output_chars: 2048,
			exit_code: 0,
		});
	});
});

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}
