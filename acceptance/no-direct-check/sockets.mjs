// Helpers for the no-direct-check acceptance tests. The driver plumbing comes from the
// locked simnet-harness helper; this file adds JSON parsing and an independent look at
// each machine's real TCP sockets, so a driver that only prints the right JSON cannot pass.
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";

export function parseJSON(line, what) {
  try {
    return JSON.parse(line);
  } catch {
    assert.fail(`${what} is not JSON: ${line}`);
  }
}

// Parses the {"relay":"127.0.0.1:P"} line.
export function parseRelay(line) {
  const r = parseJSON(line, "relay line");
  assert.equal(typeof r.relay, "string", `first line must be {"relay":"127.0.0.1:P"}: ${line}`);
  const m = /^127\.0\.0\.1:(\d+)$/.exec(r.relay);
  assert.ok(m, `relay stand-in must listen on 127.0.0.1:<port>, got ${r.relay}`);
  const port = Number(m[1]);
  assert.ok(port > 0 && port < 65536, `bad relay port: ${line}`);
  return { host: "127.0.0.1", port, addr: r.relay };
}

// Parses the {"check":{"ok":…,"direct":[…]}} line.
export function parseCheck(line) {
  const c = parseJSON(line, "check line");
  assert.ok(c.check && typeof c.check === "object", `line after the machines must be {"check":{…}}: ${line}`);
  assert.equal(typeof c.check.ok, "boolean", `check.ok must be true or false: ${line}`);
  assert.ok(Array.isArray(c.check.direct), `check.direct must be a list: ${line}`);
  for (const d of c.check.direct) {
    for (const k of ["from", "to", "local", "remote"]) {
      assert.equal(typeof d[k], "string", `direct entry lacks "${k}": ${JSON.stringify(d)}`);
    }
  }
  return c.check;
}

// The TCP connections (any state but LISTEN) of one process as "local->remote" strings,
// read with lsof on macOS. Returns null on other systems (the check is not cross-checked
// there). Not only ESTABLISHED: the stub answers a connection and closes its end, so the
// dialer's socket sits in CLOSE_WAIT, which still counts as a direct connection.
export function connectionsOf(pid) {
  if (process.platform !== "darwin") return null;
  const r = spawnSync("/usr/sbin/lsof", ["-nP", "-a", "-p", String(pid), "-iTCP", "-Fn"], {
    encoding: "utf8",
    timeout: 30000,
  });
  if (r.error) assert.fail(`could not run lsof: ${r.error.message}`);
  // lsof exits 1 when nothing matches.
  return r.stdout
    .split("\n")
    .filter((l) => l.startsWith("n") && l.includes("->"))
    .map((l) => l.slice(1));
}
