// AC-4: `go list ./...` lists cmd/beam and all eleven internal packages.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

const EXPECTED = [
  "beam/cmd/beam",
  ...["ws", "relay", "identity", "pairing", "session", "mux", "transfer", "sysinfo", "tunnel", "ui", "simnet"].map((n) => `beam/internal/${n}`),
].sort();

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

test("AC-4: go list ./... returns exactly the 12 skeleton packages", () => {
  const r = go(["list", "./..."]);
  assert.equal(r.code, 0, r.stderr);
  const got = r.stdout.split(/\r?\n/).filter(Boolean).sort();
  assert.equal(got.length, 12, `expected 12 packages, got ${got.length}:\n${got.join("\n")}`);
  assert.deepEqual(got, EXPECTED);
});
