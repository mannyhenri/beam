// AC-1: when a fake machine connects to another fake machine's address, the harness check
// fails and names every direct connection: A→C and B→A are both reported, C (which only
// dials the relay) is not, the driver exits 3 and no machine process is left.
// Needs the loopback aliases 127.0.0.2-.4 (on macOS: sudo ifconfig lo0 alias 127.0.0.N up).
import { test } from "node:test";
import assert from "node:assert/strict";
import { EXPECTED, buildDriver, isAlive, parseMachine, processesMatching, startDriver, tempDir } from "../simnet-harness/driver.mjs";
import { connectionsOf, parseCheck, parseJSON, parseRelay } from "./sockets.mjs";

test("AC-1: direct machine-to-machine connections fail the check and are all named", async (t) => {
  const bin = buildDriver(t);
  const dir = tempDir(t);
  const d = startDriver(t, bin, ["--dir", dir, "--names", "A,B,C", "--relay", "--dial", "A=C,B=A,C=relay"]);

  const relay = parseRelay(await d.nextLine(30000));
  const byName = {};
  for (const [name, host] of EXPECTED) {
    const m = parseMachine(await d.nextLine(30000));
    d.track(m.pid);
    assert.equal(m.name, name);
    assert.equal(m.host, host, `${name} must be on ${host}, got ${m.addr}`);
    byName[name] = m;
  }
  const { A, B, C } = byName;

  const check = parseCheck(await d.nextLine(30000));
  const line = JSON.stringify(check);
  assert.equal(check.ok, false, `check must fail when machines connect directly: ${line}`);
  assert.equal(check.direct.length, 2, `direct must list exactly A→C and B→A: ${line}`);

  const want = [
    { from: A, to: C },
    { from: B, to: A },
  ];
  for (const { from, to } of want) {
    const hits = check.direct.filter((x) => x.from === from.name && x.to === to.name);
    assert.equal(hits.length, 1, `exactly one ${from.name}→${to.name} entry expected: ${line}`);
    const hit = hits[0];
    assert.equal(hit.remote, to.addr, `${from.name}→${to.name} remote must be ${to.name}'s address ${to.addr}: ${line}`);
    const lm = /^(\d+\.\d+\.\d+\.\d+):(\d+)$/.exec(hit.local);
    assert.ok(lm, `${from.name}→${to.name} local is not ip:port: ${line}`);
    assert.equal(lm[1], from.host, `${from.name} must dial from its own address ${from.host}: ${line}`);

    // The connection must really exist, owned by the dialing machine's process.
    const real = connectionsOf(from.pid);
    if (real !== null) {
      assert.ok(real.includes(`${hit.local}->${hit.remote}`), `${from.name} (pid ${from.pid}) has no connection ${hit.local}->${hit.remote}; lsof shows ${JSON.stringify(real)}`);
    }
  }
  assert.ok(!check.direct.some((x) => x.from === "C"), `C only dials the relay and must not be reported: ${line}`);

  // C's relay connection is real too, and is not a violation.
  const cReal = connectionsOf(C.pid);
  if (cReal !== null) {
    assert.ok(cReal.some((c) => c.endsWith(`->${relay.addr}`)), `C (pid ${C.pid}) must hold a connection to the relay ${relay.addr}; lsof shows ${JSON.stringify(cReal)}`);
  }

  d.child.stdin.end();
  const { code, signal } = await d.waitExit(15000);
  const { stdout, stderr } = d.output();
  assert.equal(signal, null, `driver was killed by ${signal}\n${stderr}`);
  assert.equal(code, 3, `driver must exit 3 after a direct connection, got ${code}\nstdout:\n${stdout}\nstderr:\n${stderr}`);

  const lines = d.allLines();
  const report = parseJSON(lines[lines.length - 1], "last driver line");
  assert.ok(Array.isArray(report.stopped), `last line must be {"stopped":[...]}: ${lines[lines.length - 1]}`);
  assert.deepEqual(report.stopped.map((s) => s.name).sort(), ["A", "B", "C"]);

  for (const m of [A, B, C]) {
    assert.equal(isAlive(m.pid), false, `${m.name} (pid ${m.pid}) is still running after stop`);
  }
  assert.deepEqual(processesMatching(bin), [], `processes from ${bin} are still running`);
});
