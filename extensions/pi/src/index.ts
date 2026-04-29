import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { resolveConfig } from "./config.js";
import { isBinaryAvailable, setupEnv } from "./env.js";
import { systemPromptAvailable, systemPromptUnavailable } from "./system-prompt.js";
import { recordContextSignal } from "./activity-log.js";

export default function (pi: ExtensionAPI) {
  let binaryAvailable = false;
  let mountPath = "";

  pi.on("session_start", async (_event, ctx) => {
    const cfg = resolveConfig(ctx.cwd);
    mountPath = cfg.mountPath;
    binaryAvailable = await isBinaryAvailable(pi);
    if (binaryAvailable) {
      setupEnv(cfg, ctx.sessionManager.getLeafId());
    }
  });

  pi.on("before_agent_start", async (event, _ctx) => {
    const block = binaryAvailable
      ? systemPromptAvailable({
          mountPath,
          agentId: process.env.MYCELIUM_AGENT_ID!,
          sessionId: process.env.MYCELIUM_SESSION_ID!,
        })
      : systemPromptUnavailable({ mountPath });
    return { systemPrompt: event.systemPrompt + "\n\n" + block };
  });

  pi.on("context", async (event, _ctx) => {
    if (binaryAvailable) await recordContextSignal(pi, event);
    return undefined;
  });
}
