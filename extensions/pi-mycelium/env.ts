import path from "node:path";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import type { MyceliumConfig } from "./config.js";
import { resolveMyceliumBinary } from "./binary-resolver.js";

export async function resolveBinary(pi: ExtensionAPI): Promise<string | null> {
  return resolveMyceliumBinary(pi);
}

export function setupEnv(
  config: MyceliumConfig,
  sessionLeafId: string | null,
  binaryPath: string | null,
): void {
  process.env.MYCELIUM_AGENT_ID ??= "pi-agent";
  if (sessionLeafId !== null) {
    process.env.MYCELIUM_SESSION_ID = sessionLeafId;
  } else {
    delete process.env.MYCELIUM_SESSION_ID;
  }
  process.env.MYCELIUM_MOUNT = config.mountPath;

  // Make the bundled binary visible to the agent's bash invocations.
  // The system prompt instructs the agent to call `mycelium <sub>`; if the
  // binary lives only inside our optional-dep node_modules, that call would
  // 127 without this prepend.
  if (binaryPath) {
    const binDir = path.dirname(binaryPath);
    const current = process.env.PATH ?? "";
    if (!current.split(path.delimiter).includes(binDir)) {
      process.env.PATH = current ? `${binDir}${path.delimiter}${current}` : binDir;
    }
  }
}
