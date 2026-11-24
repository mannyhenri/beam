// AC-3: the Linux /proc/net/tcp parser maps each line to its local and remote address,
// state and inode. Two checks, both required:
//  1. the criterion's own command, `go test -run ProcNetTCP ./internal/simnet`, exits 0 and
//     runs at least one test;
//  2. testdata/ac3 pins the exact values: line
//     "1: 0200007F:1F90 0300007F:2000 01 … 4242 …" → local 127.0.0.2:8080, remote
//     127.0.0.3:8192, ESTABLISHED, inode 4242 (plus a LISTEN line and a header line), through
//     simnet.ParseProcNetTCP(io.Reader) ([]simnet.Socket, error) with
//     Socket{Local, Remote netip.AddrPort; State string; Inode uint64}.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { ROOT } from "../simnet-harness/driver.mjs";

const GO = process.env.GO || "go";

function goTest(args) {
  const r = spawnSync(GO, ["test", "-count=1", "-v", ...args], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run go test: ${r.error.message}`);
  return { code: r.status, out: `${r.stdout}${r.stderr}` };
}

test("AC-3: go test -run ProcNetTCP ./internal/simnet runs and passes at least one test", () => {
  const r = goTest(["-run", "ProcNetTCP", "./internal/simnet"]);
  assert.equal(r.code, 0, `go test -run ProcNetTCP ./internal/simnet failed:\n${r.out}`);
  const passed = r.out.split("\n").filter((l) => /^\s*--- PASS: \S*ProcNetTCP/.test(l));
  assert.ok(passed.length >= 1, `no test matching ProcNetTCP ran in ./internal/simnet:\n${r.out}`);
});

test("AC-3: the /proc/net/tcp parser returns the exact local, remote, state and inode", () => {
  const r = goTest(["./acceptance/no-direct-check/testdata/ac3"]);
  assert.equal(r.code, 0, `the pinned parser test failed:\n${r.out}`);
  assert.match(r.out, /--- PASS: TestAC3ProcNetTCP/, `TestAC3ProcNetTCP did not run:\n${r.out}`);
});
