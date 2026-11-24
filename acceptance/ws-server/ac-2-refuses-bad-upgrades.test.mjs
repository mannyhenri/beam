// AC-2: missing or wrong upgrade headers are refused with an HTTP error and no takeover.
// The exact checks are in testdata/ac2 (a Go test against ws.Upgrade over a raw TCP client);
// it sits under testdata so go vet ./..., go list ./... and go test ./... skip it.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

test("AC-2: missing or wrong upgrade headers are refused with an HTTP error and no takeover", () => {
  const r = spawnSync(GO, ["test", "-count=1", "-v", "./acceptance/ws-server/testdata/ac2"], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run go test: ${r.error.message}`);
  const out = `${r.stdout}${r.stderr}`;
  assert.equal(r.status, 0, `go test ./acceptance/ws-server/testdata/ac2 failed:\n${out}`);
  assert.match(out, /^--- PASS: TestAC2RefusesBadUpgrades /m, `TestAC2RefusesBadUpgrades did not pass:\n${out}`);
});
