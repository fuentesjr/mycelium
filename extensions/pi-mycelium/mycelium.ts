import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

/**
 * Run a mycelium subcommand and parse its stdout as JSON.
 *
 * Returns null on any failure: binary missing, non-zero exit, or JSON parse
 * error. Callers should treat null as "unavailable" and fall through to their
 * empty-state rendering path.
 */
export async function runMyceliumJSON<T>(
  pi: ExtensionAPI,
  binaryPath: string,
  args: string[],
): Promise<T | null> {
  try {
    const r = await pi.exec(binaryPath, args);
    if (r.code !== 0) return null;
    const text = r.stdout.trim();
    if (!text) return null;
    return JSON.parse(text) as T;
  } catch {
    return null;
  }
}

/**
 * Run a mycelium subcommand whose stdout is newline-delimited JSON objects.
 *
 * Returns an empty array on any failure: binary missing, non-zero exit, or
 * any line failing to parse. Lines that are empty or whitespace-only are
 * skipped silently.
 */
export async function runMyceliumNDJSON<T>(
  pi: ExtensionAPI,
  binaryPath: string,
  args: string[],
): Promise<T[]> {
  try {
    const r = await pi.exec(binaryPath, args);
    if (r.code !== 0) return [];
    const lines = r.stdout.split("\n");
    const results: T[] = [];
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      try {
        results.push(JSON.parse(trimmed) as T);
      } catch {
        // skip unparseable lines; log is best-effort
      }
    }
    return results;
  } catch {
    return [];
  }
}
