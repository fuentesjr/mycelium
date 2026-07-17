#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const packagePath = path.join(root, "extensions/pi-mycelium/package.json");
const lockPath = path.join(root, "extensions/pi-mycelium/package-lock.json");
const makefile = fs.readFileSync(path.join(root, "Makefile"), "utf8");
const pkg = JSON.parse(fs.readFileSync(packagePath, "utf8"));
const lock = JSON.parse(fs.readFileSync(lockPath, "utf8"));

const makefileMatch = makefile.match(/^VERSION\s*\?=\s*v?([^\s]+)$/m);
if (!makefileMatch) throw new Error("could not read VERSION from Makefile");

const normalize = (version) => version.replace(/^v/, "");
const expected = normalize(process.argv[2] ?? makefileMatch[1]);
const platformPackages = [
  "@fuentesjr/mycelium-cli-darwin-arm64",
  "@fuentesjr/mycelium-cli-darwin-amd64",
  "@fuentesjr/mycelium-cli-linux-arm64",
  "@fuentesjr/mycelium-cli-linux-amd64",
];
const checks = [
  ["Makefile VERSION", normalize(makefileMatch[1])],
  ["package.json version", pkg.version],
  ["package-lock.json version", lock.version],
  ["package-lock root version", lock.packages?.[""]?.version],
];

for (const name of platformPackages) {
  checks.push([`package.json optionalDependencies.${name}`, pkg.optionalDependencies?.[name]]);
  checks.push([
    `package-lock root optionalDependencies.${name}`,
    lock.packages?.[""]?.optionalDependencies?.[name],
  ]);
}

const failures = checks.filter(([, actual]) => actual !== expected);
if (failures.length > 0) {
  for (const [label, actual] of failures) {
    console.error(`${label}: expected ${expected}, got ${actual ?? "missing"}`);
  }
  process.exit(1);
}

console.log(`release versions agree at ${expected} (${checks.length} checks)`);
