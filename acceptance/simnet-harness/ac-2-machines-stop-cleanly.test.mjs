// AC-2: closing the driver's stdin stops machines A, B and C cleanly: the driver reports
// each as clean, exits 0 within 15 s, and no machine process or listener is left.
// Needs the loopback aliases 127.0.0.2-.4 (on macOS: sudo ifconfig lo0 alias 127.0.0.N up).
import { test } from "node:test";
import assert from "node:assert/strict";
import { EXPECTED, buildDriver, isAlive, parseMachine, processesMatching, startDriver, tempDir, tryConnect } from "./driver.mjs";

test("AC-2: stopping the machines leaves no process or listener and reports all clean", async (t) => {
  const bin = buildDriver(t);
  const dir = tempDir(t);
  const d = startDriver(t, bin, ["--dir", dir, "--names", "A,B,C"]);

  const machines = [];
  for (const [name, host] of EXPECTED) {
    const m = parseMachine(await d.nextLine(30000));
    d.track(m.pid);
    assert.equal(m.name, name);
    assert.equal(m.host, host, `${name} must be on ${host}, got ${m.addr}`);
    assert.ok(isAlive(m.pid), `${name} (pid ${m.pid}) is not running before stop`);
    assert.equal(await tryConnect(m.host, m.port), "connected", `${name} must listen on ${m.addr} before stop`);
    machines.push(m);
  }

  d.child.stdin.end();
  const started = Date.now();
  const { code, signal } = await d.waitExit(15000);
  const { stdout, stderr } = d.output();
  assert.ok(Date.now() - started <= 15000, "driver must exit within 15 s");
  assert.equal(signal, null, `driver was killed by ${signal}\n${stderr}`);
  assert.equal(code, 0, `driver exited ${code}\nstdout:\n${stdout}\nstderr:\n${stderr}`);

  const lines = d.allLines();
  const last = lines[lines.length - 1];
  let report;
  try {
    report = JSON.parse(last);
  } catch {
    assert.fail(`last driver line is not JSON: ${last}`);
  }
  assert.ok(Array.isArray(report.stopped), `last line must be {"stopped":[...]}: ${last}`);
  assert.equal(report.stopped.length, 3, `stopped must list 3 machines: ${last}`);
  for (const m of machines) {
    const s = report.stopped.find((x) => x.name === m.name);
    assert.ok(s, `${m.name} missing from stopped: ${last}`);
    assert.equal(s.pid, m.pid, `${m.name} stopped pid ${s.pid}, started pid ${m.pid}`);
    assert.equal(s.clean, true, `${m.name} must stop cleanly (not force-killed): ${last}`);
  }

  for (const m of machines) {
    assert.equal(isAlive(m.pid), false, `${m.name} (pid ${m.pid}) is still running after stop`);
    assert.equal(await tryConnect(m.host, m.port), "ECONNREFUSED", `${m.addr} must refuse connections after stop`);
  }
  assert.deepEqual(processesMatching(bin), [], `processes from ${bin} are still running`);
});
