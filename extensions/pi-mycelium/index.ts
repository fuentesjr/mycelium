import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { resolveConfig } from "./config.js";
import { resolveBinary, setupEnv } from "./env.js";
import { systemPromptAvailable, systemPromptUnavailable } from "./system-prompt.js";
import type { EvolutionKindRow, ActiveEvolutionEvent } from "./system-prompt.js";
import { recordContextSignal, recordSessionBoundary } from "./activity-log.js";
import { runMyceliumJSON, runMyceliumNDJSON } from "./mycelium.js";

export default function (pi: ExtensionAPI) {
  let binaryPath: string | null = null;
  let mountPath = "";

  pi.on("session_start", async (event, ctx) => {
    const cfg = resolveConfig(ctx.cwd);
    mountPath = cfg.mountPath;
    binaryPath = await resolveBinary(pi);
    if (binaryPath) {
      setupEnv(cfg, ctx.sessionManager.getLeafId());
      await recordSessionBoundary(pi, binaryPath, event.reason);
    }
  });

  pi.on("before_agent_start", async (event, _ctx) => {
    if (!binaryPath) {
      return { systemPrompt: event.systemPrompt + "\n\n" + systemPromptUnavailable({ mountPath }) };
    }

    const [kinds, activeEvolution] = await Promise.all([
      runMyceliumJSON<EvolutionKindRow[]>(pi, binaryPath, ["evolution", "--kinds", "--format", "json"]).then(
        (r) => r ?? [],
      ),
      runMyceliumNDJSON<ActiveEvolutionEvent>(pi, binaryPath, ["evolution", "--active", "--format", "json"]),
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

  pi.on("context", async (event, _ctx) => {
    if (binaryPath) await recordContextSignal(pi, binaryPath, event);
    return undefined;
  });
}
