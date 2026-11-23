// AC-1: a valid upgrade gets 101 with the RFC 6455 accept key and raw bytes flow both ways.
// The exact checks are in testdata/ac1 (a Go test against ws.Upgrade over a raw TCP client);
// it sits under testdata so go vet ./..., go list ./... and go test ./... skip it.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

test("AC-1: a valid upgrade gets 101 with the RFC 6455 accept key and raw bytes flow both ways", () => {
  const r = spawnSync(GO, ["test", "-count=1", "-v", "./acceptance/ws-server/testdata/ac1"], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run go test: ${r.error.message}`);
  const out = `${r.stdout}${r.stderr}`;
  assert.equal(r.status, 0, `go test ./acceptance/ws-server/testdata/ac1 failed:\n${out}`);
  assert.match(out, /^--- PASS: TestAC1UpgradeAndRawBytes /m, `TestAC1UpgradeAndRawBytes did not pass:\n${out}`);
});
