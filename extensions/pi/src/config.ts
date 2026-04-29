import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

export type Scope = "project" | "global";

export interface MyceliumConfig {
  scope: Scope;
  mountPath: string;
}

export const GLOBAL_EXT_ROOT = path.join(os.homedir(), ".pi", "agent", "extensions");

export function detectScopeFromPath(filePath: string): Scope {
  return filePath.startsWith(GLOBAL_EXT_ROOT + path.sep) ? "global" : "project";
}

export function mountPathFor(scope: Scope, cwd: string): string {
  return scope === "global"
    ? path.join(os.homedir(), ".pi", "mycelium", "store")
    : path.join(cwd, ".pi", "mycelium", "store");
}

export function resolveConfig(cwd: string): MyceliumConfig {
  const scope = detectScopeFromPath(fileURLToPath(import.meta.url));
  return { scope, mountPath: mountPathFor(scope, cwd) };
}
