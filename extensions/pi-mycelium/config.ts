import os from "node:os";
import path from "node:path";
import fs from "node:fs";
import { fileURLToPath } from "node:url";

export type Scope = "project" | "global";

export interface MyceliumConfig {
  scope: Scope;
  mountPath: string;
}

export const GLOBAL_EXT_ROOT = path.join(os.homedir(), ".pi", "agent", "extensions");
const PKG_NAME = "pi-mycelium";

// True if `pi-mycelium` is registered in pi's project-local settings file at
// <cwd>/.pi/settings.json. `pi install -l` writes here; `pi install` writes to
// the global settings file under ~/.pi/agent/.
function isRegisteredProjectLocal(cwd: string): boolean {
  const projectSettings = path.join(cwd, ".pi", "settings.json");
  try {
    const raw = fs.readFileSync(projectSettings, "utf8");
    const parsed = JSON.parse(raw) as { packages?: string[] };
    return (parsed.packages ?? []).some((p) => p === `npm:${PKG_NAME}` || p.endsWith(`/${PKG_NAME}`));
  } catch {
    return false;
  }
}

// Detection order:
//   1. If the extension file lives under ~/.pi/agent/extensions/, it was placed
//      there directly (legacy / manual install) — global scope.
//   2. If pi-mycelium is listed in <cwd>/.pi/settings.json, the user ran
//      `pi install -l` from this project — project scope.
//   3. Otherwise default to global. `pi install npm:pi-mycelium` installs into
//      a node_modules tree outside both .pi roots, so we can't detect it from
//      the file path; the user explicitly chose global by omitting `-l`.
export function detectScope(filePath: string, cwd: string): Scope {
  if (filePath.startsWith(GLOBAL_EXT_ROOT + path.sep)) return "global";
  if (isRegisteredProjectLocal(cwd)) return "project";
  return "global";
}

export function mountPathFor(scope: Scope, cwd: string): string {
  return scope === "global"
    ? path.join(os.homedir(), ".pi", "mycelium", "store")
    : path.join(cwd, ".pi", "mycelium", "store");
}

export function resolveConfig(cwd: string): MyceliumConfig {
  const scope = detectScope(fileURLToPath(import.meta.url), cwd);
  return { scope, mountPath: mountPathFor(scope, cwd) };
}
