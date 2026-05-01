import type { ExecResult } from "@mariozechner/pi-coding-agent";

export function execResult(code: number): ExecResult {
  return { stdout: "", stderr: "", code, killed: false };
}
