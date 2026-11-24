// AC-1: with HTTPS_PROXY the client reaches a wss:// relay through the proxy, and NO_PROXY bypasses it.
// The exact checks are in testdata/ac1 (a Go test against ws.Dial and ws.Conn);
// it sits under testdata so go vet ./..., go list ./... and go test ./... skip it.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

test("AC-1: with HTTPS_PROXY the client reaches a wss:// relay through the proxy, and NO_PROXY bypasses it", () => {
  const r = spawnSync(GO, ["test", "-count=1", "-v", "./acceptance/ws-client/testdata/ac1"], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run go test: ${r.error.message}`);
  const out = `${r.stdout}${r.stderr}`;
  assert.equal(r.status, 0, `go test ./acceptance/ws-client/testdata/ac1 failed:\n${out}`);
  assert.match(out, /^--- PASS: TestAC1ProxyAndNoProxy /m, `TestAC1ProxyAndNoProxy did not pass:\n${out}`);
});
