import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

export type Scope = "project" | "global";

export interface MyceliumConfig {
  scope: Scope;
  mountPath: string;
}

const GLOBAL_EXT_ROOT = path.join(os.homedir(), ".pi", "agent", "extensions");

function detectScope(): Scope {
  const here = fileURLToPath(import.meta.url);
  return here.startsWith(GLOBAL_EXT_ROOT + path.sep) ? "global" : "project";
}

export function resolveConfig(cwd: string): MyceliumConfig {
  const scope = detectScope();
  const mountPath =
    scope === "global"
      ? path.join(os.homedir(), ".pi", "mycelium", "store")
      : path.join(cwd, ".pi", "mycelium", "store");
  return { scope, mountPath };
}
