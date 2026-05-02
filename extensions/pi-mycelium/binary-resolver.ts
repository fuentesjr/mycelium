import { createRequire } from "node:module";
import path from "node:path";
import fs from "node:fs";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

const require = createRequire(import.meta.url);

/**
 * Resolve the bundled mycelium binary shipped via the matching
 * `@mycelium/cli-<platform>-<arch>` optional dependency. Returns the
 * absolute path, or null if the package wasn't installed (e.g. the
 * user is on an unsupported platform, or the optional dep was skipped).
 */
export function resolveBundledBinary(): string | null {
  const pkgName = `@mycelium/cli-${process.platform}-${process.arch}`;
  try {
    const pkgJson = require.resolve(`${pkgName}/package.json`);
    const bin = path.join(path.dirname(pkgJson), "mycelium");
    return fs.existsSync(bin) ? bin : null;
  } catch {
    return null;
  }
}

/**
 * Resolve a usable mycelium binary path. Prefer the bundled one
 * (predictable version, no PATH races). Fall back to PATH for
 * dev workflows where someone has built from source.
 *
 * Returns null if neither is available.
 */
export async function resolveMyceliumBinary(pi: ExtensionAPI): Promise<string | null> {
  const bundled = resolveBundledBinary();
  if (bundled) return bundled;
  const r = await pi.exec("which", ["mycelium"]);
  if (r.code === 0) {
    const out = r.stdout.trim();
    if (out) return out;
  }
  return null;
}
