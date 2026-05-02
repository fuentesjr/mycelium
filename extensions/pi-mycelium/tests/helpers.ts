import type { ExecResult } from "@mariozechner/pi-coding-agent";

export function execResult(code: number, stdout = "", stderr = ""): ExecResult {
  return { stdout, stderr, code, killed: false };
}
