import os from "node:os";
import path from "node:path";
import { describe, it, expect } from "vitest";
import { detectScopeFromPath, mountPathFor, GLOBAL_EXT_ROOT } from "../config.js";

describe("detectScopeFromPath", () => {
  it("returns 'global' for paths under ~/.pi/agent/extensions/", () => {
    const filePath = path.join(GLOBAL_EXT_ROOT, "mycelium", "index.ts");
    expect(detectScopeFromPath(filePath)).toBe("global");
  });

  it("returns 'project' for project-local paths under <repo>/.pi/extensions/", () => {
    const filePath = path.join("/home/dev/proj", ".pi", "extensions", "mycelium", "index.ts");
    expect(detectScopeFromPath(filePath)).toBe("project");
  });

  it("returns 'project' for arbitrary paths (e.g. quick-test via pi -e ./path)", () => {
    expect(detectScopeFromPath("/tmp/scratch/index.ts")).toBe("project");
  });

  it("does not match prefixes that are not directory boundaries", () => {
    const fake = path.join(os.homedir(), ".pi", "agent", "extensionsFoo", "x.ts");
    expect(detectScopeFromPath(fake)).toBe("project");
  });
});

describe("mountPathFor", () => {
  it("uses ~/.pi/mycelium/store for global scope", () => {
    expect(mountPathFor("global", "/some/cwd")).toBe(
      path.join(os.homedir(), ".pi", "mycelium", "store"),
    );
  });

  it("uses <cwd>/.pi/mycelium/store for project scope", () => {
    expect(mountPathFor("project", "/some/cwd")).toBe(
      path.join("/some/cwd", ".pi", "mycelium", "store"),
    );
  });
});
