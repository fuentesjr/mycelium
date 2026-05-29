import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { resolveConfig } from "./config.js";
import { resolveMyceliumBinary, setupEnv } from "./binary-resolver.js";
import {
	systemPromptAvailable,
	systemPromptUnavailable,
} from "./system-prompt.js";
import type {
	EvolutionKindRow,
	ActiveEvolutionEvent,
} from "./system-prompt.js";
import { createActivityLogRecorder } from "./activity-log.js";
import { bootstrapMemoryFile } from "./bootstrap.js";
import { runMyceliumJSON, runMyceliumNDJSON } from "./mycelium.js";

export default function (pi: ExtensionAPI) {
	const activity = createActivityLogRecorder();
	let binaryPath: string | null = null;
	let mountPath = "";
	let currentTurnIndex: number | undefined;

	pi.on("session_start", async (event, ctx) => {
		const cfg = resolveConfig(ctx.cwd);
		mountPath = cfg.mountPath;
		binaryPath = await resolveMyceliumBinary(pi);
		if (binaryPath) {
			setupEnv(cfg, ctx.sessionManager.getLeafId(), binaryPath);
			await activity.recordSessionBoundary(pi, binaryPath, event.reason);
			await bootstrapMemoryFile(binaryPath, mountPath);
		}
	});

	pi.on("session_shutdown", async (event, _ctx) => {
		if (binaryPath) await activity.recordSessionShutdown(pi, binaryPath, event);
	});

	pi.on("before_agent_start", async (event, _ctx) => {
		if (!binaryPath) {
			return {
				systemPrompt:
					event.systemPrompt + "\n\n" + systemPromptUnavailable({ mountPath }),
			};
		}

		const [kinds, activeEvolution] = await Promise.all([
			runMyceliumJSON<EvolutionKindRow[]>(pi, binaryPath, [
				"evolve",
				"--kinds",
				"--format",
				"json",
			]).then((r) => r ?? []),
			runMyceliumNDJSON<ActiveEvolutionEvent>(pi, binaryPath, [
				"evolve",
				"--active",
				"--format",
				"json",
			]),
		]);

		const block = systemPromptAvailable({
			mountPath,
			agentId: process.env.MYCELIUM_AGENT_ID!,
			sessionId: process.env.MYCELIUM_SESSION_ID!,
			kinds,
			activeEvolution,
		});

		return { systemPrompt: event.systemPrompt + "\n\n" + block };
	});

	pi.on("turn_start", async (event, _ctx) => {
		currentTurnIndex = event.turnIndex;
		if (binaryPath) await activity.recordTurnStart(pi, binaryPath, event);
	});

	pi.on("turn_end", async (event, _ctx) => {
		if (binaryPath) await activity.recordTurnEnd(pi, binaryPath, event);
	});

	pi.on("tool_execution_start", async (event, _ctx) => {
		if (binaryPath) await activity.recordToolStart(pi, binaryPath, event);
	});

	pi.on("tool_execution_end", async (event, _ctx) => {
		if (binaryPath) await activity.recordToolEnd(pi, binaryPath, event);
	});

	pi.on("session_compact", async (event, _ctx) => {
		if (binaryPath) await activity.recordCompaction(pi, binaryPath, event);
	});

	pi.on("context", async (event, _ctx) => {
		if (binaryPath) {
			await activity.recordContextCheckpoint(pi, binaryPath, event, {
				turnIndex: currentTurnIndex,
			});
		}
		return undefined;
	});
}
