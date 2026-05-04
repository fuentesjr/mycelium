import os from "node:os";
import path from "node:path";
import fs from "node:fs";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { bootstrapMemoryFile, MEMORY_FILE } from "../bootstrap.js";

// The real bundled template lives next to bootstrap.ts at templates/MYCELIUM_MEMORY.md.
// These tests use a shell stub for the mycelium binary so we can observe the
// args and stdin without needing a Go build.

let tmp: string;
let stubBin: string;
let stubLog: string;

beforeEach(() => {
  tmp = fs.mkdtempSync(path.join(os.tmpdir(), "pi-mycelium-bootstrap-"));
  stubLog = path.join(tmp, "stub.log");
  stubBin = path.join(tmp, "mycelium-stub.sh");
  // Stub: append "<arg1> <arg2>\n<stdin>\n---\n" to stubLog so tests can verify both.
  fs.writeFileSync(
    stubBin,
    `#!/bin/sh
echo "$1 $2" >> "${stubLog}"
cat >> "${stubLog}"
echo "---" >> "${stubLog}"
`,
    { mode: 0o755 },
  );
});

afterEach(() => {
  fs.rmSync(tmp, { recursive: true, force: true });
});

describe("bootstrapMemoryFile", () => {
  it("does nothing when MYCELIUM_MEMORY.md already exists in the mount", async () => {
    fs.writeFileSync(path.join(tmp, MEMORY_FILE), "existing content");
    await bootstrapMemoryFile(stubBin, tmp);
    expect(fs.existsSync(stubLog)).toBe(false);
  });

  it("invokes mycelium write with the template piped on stdin when missing", async () => {
    await bootstrapMemoryFile(stubBin, tmp);
    expect(fs.existsSync(stubLog)).toBe(true);
    const log = fs.readFileSync(stubLog, "utf8");
    expect(log).toMatch(/^write MYCELIUM_MEMORY\.md\n/);
    // Sanity-check the template content reached stdin
    expect(log).toContain("# MYCELIUM_MEMORY.md");
    expect(log).toContain("## Conventions");
    expect(log.endsWith("---\n")).toBe(true);
  });

  it("does not throw when the binary path is invalid", async () => {
    await expect(
      bootstrapMemoryFile("/nonexistent/binary", tmp),
    ).resolves.toBeUndefined();
  });
});
