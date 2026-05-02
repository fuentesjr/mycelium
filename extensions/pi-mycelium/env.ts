import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import type { MyceliumConfig } from "./config.js";
import { resolveMyceliumBinary } from "./binary-resolver.js";

export async function resolveBinary(pi: ExtensionAPI): Promise<string | null> {
  return resolveMyceliumBinary(pi);
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
