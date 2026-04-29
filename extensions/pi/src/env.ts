import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import type { MyceliumConfig } from "./config.js";

export async function isBinaryAvailable(pi: ExtensionAPI): Promise<boolean> {
  const r = await pi.exec("which", ["mycelium"]);
  return r.code === 0;
}

export function setupEnv(config: MyceliumConfig, sessionLeafId: string | null): void {
  process.env.MYCELIUM_AGENT_ID ??= "pi-agent";
  if (sessionLeafId !== null) {
    process.env.MYCELIUM_SESSION_ID = sessionLeafId;
  } else {
    delete process.env.MYCELIUM_SESSION_ID;
  }
  process.env.MYCELIUM_MOUNT = config.mountPath;
}
