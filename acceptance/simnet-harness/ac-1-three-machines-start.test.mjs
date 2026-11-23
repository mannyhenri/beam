// AC-1: the simnet driver starts machines A, B and C as three separate processes on
// 127.0.0.2, 127.0.0.3 and 127.0.0.4, each with its own HOME and log file.
// Needs the loopback aliases 127.0.0.2-.4 (on macOS: sudo ifconfig lo0 alias 127.0.0.N up).
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, realpathSync, statSync } from "node:fs";
import { join } from "node:path";
import { EXPECTED, buildDriver, fetchLine, isAlive, parseMachine, startDriver, tempDir } from "./driver.mjs";

test("AC-1: three named machines run as separate processes with own HOME, address and log", async (t) => {
  const bin = buildDriver(t);
  const dir = tempDir(t);
  const d = startDriver(t, bin, ["--dir", dir, "--names", "A,B,C"]);

  const machines = [];
  for (const [name, host] of EXPECTED) {
    const m = parseMachine(await d.nextLine(30000));
    d.track(m.pid);
    assert.equal(m.name, name, `machines must be reported in start order; expected ${name}, got ${m.name}`);
    assert.equal(m.host, host, `${name} must be on ${host}, got ${m.addr}`);
    machines.push(m);
  }

  const pids = new Set(machines.map((m) => m.pid));
  assert.equal(pids.size, 3, `pids must be distinct: ${[...pids]}`);
  for (const m of machines) {
    assert.notEqual(m.pid, d.child.pid, `${m.name} must be a child process, not the driver itself`);
    assert.ok(isAlive(m.pid), `${m.name} (pid ${m.pid}) is not running`);

    const wantHome = join(dir, m.name, "home");
    assert.ok(existsSync(m.home) && statSync(m.home).isDirectory(), `${m.name} home ${m.home} is not a directory`);
    assert.equal(realpathSync(m.home), wantHome, `${m.name} home must be ${wantHome}`);

    const wantLog = join(dir, m.name, "machine.log");
    assert.ok(existsSync(m.log) && statSync(m.log).isFile(), `${m.name} log ${m.log} is not a file`);
    assert.equal(realpathSync(m.log), wantLog, `${m.name} log must be ${wantLog}`);
  }
  assert.equal(new Set(machines.map((m) => m.home)).size, 3, "homes must differ");
  assert.equal(new Set(machines.map((m) => m.log)).size, 3, "logs must differ");

  // The listener on each address belongs to that machine's process and HOME.
  for (const m of machines) {
    const raw = await fetchLine(m.host, m.port);
    let got;
    try {
      got = JSON.parse(raw.trim());
    } catch {
      assert.fail(`${m.addr} did not answer one JSON line: ${JSON.stringify(raw)}`);
    }
    assert.equal(got.name, m.name, `${m.addr} answered name ${got.name}`);
    assert.equal(got.pid, m.pid, `${m.addr} answered pid ${got.pid}, driver reported ${m.pid}`);
    assert.equal(realpathSync(got.home), join(dir, m.name, "home"), `${m.addr} answered home ${got.home}`);
  }

  d.child.stdin.end();
  await d.waitExit(30000);
});
