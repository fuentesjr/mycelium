import type { ExecResult } from "@earendil-works/pi-coding-agent";

export function execResult(code: number, stdout = "", stderr = ""): ExecResult {
  return { stdout, stderr, code, killed: false };
}
