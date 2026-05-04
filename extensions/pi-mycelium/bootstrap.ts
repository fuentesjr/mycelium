import fs from "node:fs";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

export const MEMORY_FILE = "MYCELIUM_MEMORY.md";
const TEMPLATE_PATH = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  "templates",
  MEMORY_FILE,
);

// Seed MYCELIUM_MEMORY.md from the bundled template if the mount doesn't have
// one yet. The system prompt tells the agent "Read it once at session start" —
// this makes that promise true on first use without any manual install step.
// Routes through `mycelium write` (not direct fs) so the bootstrap shows up in
// the activity log as a normal `op:write` entry and respects the mount's lock.
// pi.exec doesn't expose stdin, so we spawn the binary directly here.
export async function bootstrapMemoryFile(
  binaryPath: string,
  mountPath: string,
): Promise<void> {
  const target = path.join(mountPath, MEMORY_FILE);
  if (fs.existsSync(target)) return;

  let template: string;
  try {
    template = fs.readFileSync(TEMPLATE_PATH, "utf8");
  } catch {
    return;
  }

  await new Promise<void>((resolve) => {
    const child = spawn(binaryPath, ["write", MEMORY_FILE], {
      env: process.env,
      stdio: ["pipe", "ignore", "ignore"],
    });
    child.on("error", () => resolve());
    child.on("close", () => resolve());
    child.stdin.end(template);
  });
}
