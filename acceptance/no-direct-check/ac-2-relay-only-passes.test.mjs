// AC-2: when machines connect only to the relay, the check passes: {"check":{"ok":true,
// "direct":[]}}, then {"stopped":[…]} with A and B clean, and the driver exits 0.
// Needs the loopback aliases 127.0.0.2-.3 (on macOS: sudo ifconfig lo0 alias 127.0.0.N up).
import { test } from "node:test";
import assert from "node:assert/strict";
import { EXPECTED, buildDriver, isAlive, parseMachine, processesMatching, startDriver, tempDir } from "../simnet-harness/driver.mjs";
import { connectionsOf, parseCheck, parseJSON, parseRelay } from "./sockets.mjs";

test("AC-2: machines that only connect to the relay pass the check", async (t) => {
  const bin = buildDriver(t);
  const dir = tempDir(t);
  const d = startDriver(t, bin, ["--dir", dir, "--names", "A,B", "--relay", "--dial", "A=relay,B=relay"]);

  const relay = parseRelay(await d.nextLine(30000));
  const machines = [];
  for (const [name, host] of EXPECTED.slice(0, 2)) {
    const m = parseMachine(await d.nextLine(30000));
    d.track(m.pid);
    assert.equal(m.name, name);
    assert.equal(m.host, host, `${name} must be on ${host}, got ${m.addr}`);
    machines.push(m);
  }

  const check = parseCheck(await d.nextLine(30000));
  assert.deepEqual(check, { ok: true, direct: [] }, `relay-only traffic must pass the check: ${JSON.stringify(check)}`);

  // Both machines really are connected to the relay, so the check had something to look at.
  for (const m of machines) {
    const real = connectionsOf(m.pid);
    if (real !== null) {
      assert.ok(real.some((c) => c.endsWith(`->${relay.addr}`)), `${m.name} (pid ${m.pid}) must hold a connection to the relay ${relay.addr}; lsof shows ${JSON.stringify(real)}`);
    }
  }

  d.child.stdin.end();
  const { code, signal } = await d.waitExit(15000);
  const { stdout, stderr } = d.output();
  assert.equal(signal, null, `driver was killed by ${signal}\n${stderr}`);
  assert.equal(code, 0, `driver must exit 0, got ${code}\nstdout:\n${stdout}\nstderr:\n${stderr}`);

  const lines = d.allLines();
  const last = lines[lines.length - 1];
  const report = parseJSON(last, "last driver line");
  assert.ok(Array.isArray(report.stopped), `last line must be {"stopped":[...]}: ${last}`);
  assert.equal(report.stopped.length, 2, `stopped must list A and B: ${last}`);
  for (const m of machines) {
    const s = report.stopped.find((x) => x.name === m.name);
    assert.ok(s, `${m.name} missing from stopped: ${last}`);
    assert.equal(s.pid, m.pid, `${m.name} stopped pid ${s.pid}, started pid ${m.pid}`);
    assert.equal(s.clean, true, `${m.name} must stop cleanly: ${last}`);
    assert.equal(isAlive(m.pid), false, `${m.name} (pid ${m.pid}) is still running after stop`);
  }
  assert.deepEqual(processesMatching(bin), [], `processes from ${bin} are still running`);
});
