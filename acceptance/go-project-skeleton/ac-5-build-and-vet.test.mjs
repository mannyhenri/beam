// AC-5: `CGO_ENABLED=0 go build ./...` and `go vet ./...` succeed; vet prints nothing.
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
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "", CGO_ENABLED: "0" },
  });
  if (r.error) assert.fail(`could not run \`${GO} ${args.join(" ")}\`: ${r.error.message}`);
  return { code: r.status, stdout: r.stdout, stderr: r.stderr };
}

test("AC-5: CGO_ENABLED=0 go build ./... exits 0", () => {
  assert.ok(existsSync(join(ROOT, "go.mod")), "go.mod missing at repo root");
  const r = go(["build", "./..."]);
  assert.equal(r.code, 0, `go build failed:\n${r.stdout}${r.stderr}`);
});

test("AC-5: go vet ./... exits 0 with no output", () => {
  assert.ok(existsSync(join(ROOT, "go.mod")), "go.mod missing at repo root");
  const r = go(["vet", "./..."]);
  assert.equal(r.code, 0, `go vet failed:\n${r.stdout}${r.stderr}`);
  assert.equal((r.stdout + r.stderr).trim(), "", "go vet must print nothing");
});
