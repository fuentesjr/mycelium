import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import type { MyceliumConfig } from "./config.js";

export async function isBinaryAvailable(pi: ExtensionAPI): Promise<boolean> {
  const r = await pi.exec("which", ["mycelium"]);
  return r.exitCode === 0;
}

export function setupEnv(config: MyceliumConfig, sessionLeafId: string): void {
  process.env.MYCELIUM_AGENT_ID ??= "pi-agent";
  process.env.MYCELIUM_SESSION_ID = sessionLeafId;
  process.env.MYCELIUM_MOUNT = config.mountPath;
}
