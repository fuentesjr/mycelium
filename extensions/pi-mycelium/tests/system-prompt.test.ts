import { describe, it, expect } from "vitest";
import {
	systemPromptAvailable,
	systemPromptUnavailable,
} from "../system-prompt.js";

function makeBlock(
	overrides: Partial<Parameters<typeof systemPromptAvailable>[0]> = {},
) {
	return systemPromptAvailable({
		mountPath: "/test/store",
		agentId: "test-agent",
		sessionId: "session-xyz",
		...overrides,
	});
}

describe("systemPromptAvailable", () => {
	const block = makeBlock();

	it("interpolates mount, memory file, agent, and session", () => {
		expect(block).toContain("/test/store");
		expect(block).toContain("/test/store/MYCELIUM_MEMORY.md");
		expect(block).toContain("MYCELIUM_AGENT_ID=test-agent");
		expect(block).toContain("MYCELIUM_SESSION_ID=session-xyz");
	});

	it("documents the current command tiers without evolve", () => {
		expect(block).toContain("Everyday commands");
		expect(block).toContain("Occasional commands");
		expect(block).toContain("Metadata commands");
		for (const sub of [
			"read",
			"write",
			"edit",
			"ls",
			"grep",
			"rm",
			"mv",
			"log",
		]) {
			expect(block).toContain(`mycelium ${sub}`);
		}
		expect(block).not.toContain("mycelium evolve");
	});

	it("points agents at the conventions file instead of broad rediscovery", () => {
		expect(block).toContain("### Conventions file");
		expect(block).toContain(
			"Read `/test/store/MYCELIUM_MEMORY.md` once at session start",
		);
		expect(block).toContain("Do not broad-search for a substitute");
		expect(block).toContain("edit that file in the same session");
		expect(block).toContain('mycelium log decision --rationale "..."');
		expect(block).toContain('mycelium log agent_note --rationale "..."');
		expect(block).not.toContain("decision|agent_note");
	});

	it("describes the conflict-recovery contract", () => {
		expect(block).toContain("--expected-version");
		expect(block).toContain("exits 64");
		expect(block).toContain('"error":"conflict"');
		expect(block).toContain('"error":"destination_exists"');
		expect(block).toContain(
			"re-read with `mycelium read <path> --format json`",
		);
		expect(block).not.toContain("--include-current-content");
	});

	it("describes reserved paths and pi lifecycle events", () => {
		expect(block).toContain("Reserved paths");
		expect(block).toContain("_activity/YYYY/MM/DD/test-agent.jsonl");
		expect(block).not.toContain("_tx/pending");
		expect(block).toContain("### Activity events");
		expect(block).toContain("compaction");
		expect(block).toContain("context_checkpoint");
		expect(block).not.toContain("tool_start");
		expect(block).toContain("context_signal");
		expect(block).not.toContain("deduped");
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
