// AC-8: PROGRESS.md's go version and package lines match what the installed toolchain reports.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";
const PKGS = ["crypto/ecdh", "crypto/hkdf", "crypto/pbkdf2"];

function go(args) {
  const r = spawnSync(GO, args, {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run \`${GO} ${args.join(" ")}\`: ${r.error.message}`);
  return { code: r.status, stdout: r.stdout, stderr: r.stderr };
}

function readProgress() {
  const p = join(ROOT, "PROGRESS.md");
  assert.ok(existsSync(p), "PROGRESS.md missing at repo root");
  return readFileSync(p, "utf8");
}

// Body of the `Go environment` section, where the package lines live.
function goEnvSection(md) {
  const lines = md.split(/\r?\n/);
  const start = lines.findIndex((l) => /^#{1,6}\s+go environment\b/i.test(l));
  assert.ok(start >= 0, "PROGRESS.md needs a `Go environment` section");
  const rest = lines.slice(start + 1);
  const end = rest.findIndex((l) => /^#{1,6}\s/.test(l));
  return (end < 0 ? rest : rest.slice(0, end)).join("\n");
}

test("AC-8: recorded go version equals `go version` output", () => {
  const md = readProgress();
  const r = go(["version"]);
  assert.equal(r.code, 0, r.stderr);
  const actual = r.stdout.trim();
  assert.ok(goEnvSection(md).includes(actual), `Go environment section must contain the verbatim \`go version\` output: ${actual}`);
});

for (const pkg of PKGS) {
  test(`AC-8: ${pkg} line agrees with \`go list ${pkg}\``, () => {
    const env = goEnvSection(readProgress());
    const lines = env.split(/\r?\n/).filter((l) => l.includes(pkg));
    assert.equal(lines.length, 1, `expected exactly one line mentioning ${pkg} in Go environment, got ${lines.length}`);
    const saysAvailable = /\bavailable\b/i.test(lines[0]);
    const saysMissing = /\bmissing\b/i.test(lines[0]);
    assert.ok(saysAvailable !== saysMissing, `line for ${pkg} must say exactly one of 'available' or 'missing': ${lines[0]}`);
    const listed = go(["list", pkg]).code === 0;
    assert.equal(saysAvailable, listed, `${pkg}: go list exit ${listed ? 0 : "non-zero"} but PROGRESS.md says: ${lines[0]}`);
  });
}
