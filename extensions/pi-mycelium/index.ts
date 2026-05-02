import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { resolveConfig } from "./config.js";
import { isBinaryAvailable, setupEnv } from "./env.js";
import { systemPromptAvailable, systemPromptUnavailable } from "./system-prompt.js";
import type { EvolutionKindRow, ActiveEvolutionEvent } from "./system-prompt.js";
import { recordContextSignal, recordSessionBoundary } from "./activity-log.js";
import { runMyceliumJSON, runMyceliumNDJSON } from "./mycelium.js";

export default function (pi: ExtensionAPI) {
  let binaryAvailable = false;
  let mountPath = "";

  pi.on("session_start", async (event, ctx) => {
    const cfg = resolveConfig(ctx.cwd);
    mountPath = cfg.mountPath;
    binaryAvailable = await isBinaryAvailable(pi);
    if (binaryAvailable) {
      setupEnv(cfg, ctx.sessionManager.getLeafId());
      await recordSessionBoundary(pi, event.reason);
    }
  });

  pi.on("before_agent_start", async (event, _ctx) => {
    if (!binaryAvailable) {
      return { systemPrompt: event.systemPrompt + "\n\n" + systemPromptUnavailable({ mountPath }) };
    }

    const [kinds, activeEvolution] = await Promise.all([
      runMyceliumJSON<EvolutionKindRow[]>(pi, ["evolution", "--kinds", "--format", "json"]).then(
        (r) => r ?? [],
      ),
      runMyceliumNDJSON<ActiveEvolutionEvent>(pi, ["evolution", "--active", "--format", "json"]),
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
    if (binaryAvailable) await recordContextSignal(pi, event);
    return undefined;
  });
}
