import os from "node:os";
import path from "node:path";
import fs from "node:fs";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { detectScope, mountPathFor, GLOBAL_EXT_ROOT } from "../config.js";

describe("detectScope", () => {
  let tmp: string;

  beforeEach(() => {
    tmp = fs.mkdtempSync(path.join(os.tmpdir(), "pi-mycelium-test-"));
  });

  afterEach(() => {
    fs.rmSync(tmp, { recursive: true, force: true });
  });

  it("returns 'global' for paths under ~/.pi/agent/extensions/", () => {
    const filePath = path.join(GLOBAL_EXT_ROOT, "mycelium", "index.ts");
    expect(detectScope(filePath, tmp)).toBe("global");
  });

  it("returns 'project' when pi-mycelium is registered in <cwd>/.pi/settings.json", () => {
    fs.mkdirSync(path.join(tmp, ".pi"), { recursive: true });
    fs.writeFileSync(
      path.join(tmp, ".pi", "settings.json"),
      JSON.stringify({ packages: ["npm:pi-mycelium"] }),
    );
    expect(detectScope("/opt/somewhere/node_modules/pi-mycelium/index.ts", tmp)).toBe("project");
  });

  it("defaults to 'global' for npm-installed paths with no project-local registration", () => {
    expect(detectScope("/opt/somewhere/node_modules/pi-mycelium/index.ts", tmp)).toBe("global");
  });

  it("does not match prefixes that are not directory boundaries", () => {
    const fake = path.join(os.homedir(), ".pi", "agent", "extensionsFoo", "x.ts");
    expect(detectScope(fake, tmp)).toBe("global");
  });

  it("ignores project settings without a matching package entry", () => {
    fs.mkdirSync(path.join(tmp, ".pi"), { recursive: true });
    fs.writeFileSync(
      path.join(tmp, ".pi", "settings.json"),
      JSON.stringify({ packages: ["npm:pi-something-else"] }),
    );
    expect(detectScope("/opt/somewhere/node_modules/pi-mycelium/index.ts", tmp)).toBe("global");
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
