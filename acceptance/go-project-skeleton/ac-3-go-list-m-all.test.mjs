// AC-3: `go list -m all` lists only the main module; no go.sum exists.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

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

test("AC-3: go list -m all prints exactly `beam`", () => {
  const r = go(["list", "-m", "all"]);
  assert.equal(r.code, 0, r.stderr);
  assert.deepEqual(r.stdout.split(/\r?\n/).filter(Boolean), ["beam"]);
});

test("AC-3: no go.sum file exists", () => {
  assert.ok(existsSync(join(ROOT, "go.mod")), "go.mod missing at repo root");
  assert.ok(!existsSync(join(ROOT, "go.sum")), "go.sum must not exist");
});
