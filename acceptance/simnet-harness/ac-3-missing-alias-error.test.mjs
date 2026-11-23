// AC-3: when a needed loopback address cannot be bound, the driver starts no machine,
// exits non-zero and prints the exact macOS fix. Skipped off macOS: Linux binds all of
// 127.0.0.0/8, so there is no missing alias to report.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { ROOT, buildDriver, processesMatching, tempDir } from "./driver.mjs";

test(
  "AC-3: an unbindable address starts nothing and names the sudo fix",
  { skip: process.platform !== "darwin" && "only macOS needs loopback aliases" },
  (t) => {
    const bin = buildDriver(t);
    const dir = tempDir(t);
    const r = spawnSync(bin, ["--dir", dir, "--names", "A", "--addrs", "127.0.0.250"], {
      cwd: ROOT,
      encoding: "utf8",
      input: "",
      timeout: 30000,
    });
    if (r.error) assert.fail(`driver did not finish: ${r.error.message}`);
    assert.equal(r.signal, null, `driver was killed by ${r.signal}`);
    assert.notEqual(r.status, 0, `driver must fail when 127.0.0.250 cannot be bound\nstdout:\n${r.stdout}`);
    assert.ok(r.stderr.includes("127.0.0.250"), `stderr must name 127.0.0.250:\n${r.stderr}`);
    assert.ok(
      r.stderr.includes("sudo ifconfig lo0 alias 127.0.0.250 up"),
      `stderr must give the fix 'sudo ifconfig lo0 alias 127.0.0.250 up':\n${r.stderr}`,
    );

    for (const line of r.stdout.split("\n").filter((l) => l.trim() !== "")) {
      let v = null;
      try {
        v = JSON.parse(line);
      } catch {}
      assert.ok(!(v && typeof v === "object" && "pid" in v && "name" in v), `no machine may start, got: ${line}`);
    }
    assert.deepEqual(processesMatching(bin), [], `processes from ${bin} are still running`);
  },
);
