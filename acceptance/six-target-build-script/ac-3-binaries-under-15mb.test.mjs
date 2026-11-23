// AC-3: after `bash scripts/build_all.sh`, each of the six binaries in dist/ is
// smaller than 15 MB (15,728,640 bytes).
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const DIST = join(ROOT, "dist");
const LIMIT = 15 * 1024 * 1024;

const BINARIES = [
  "beam-darwin-amd64",
  "beam-darwin-arm64",
  "beam-linux-amd64",
  "beam-linux-arm64",
  "beam-windows-amd64.exe",
  "beam-windows-arm64.exe",
];

test("AC-3: every binary built by build_all.sh is under 15,728,640 bytes", () => {
  const r = spawnSync("bash", ["scripts/build_all.sh"], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 600000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run build_all.sh: ${r.error.message}`);
  assert.equal(r.status, 0, `build_all.sh exited ${r.status}:\n${r.stdout}${r.stderr}`);

  for (const name of BINARIES) {
    const p = join(DIST, name);
    assert.ok(existsSync(p), `dist/${name} missing`);
    const size = statSync(p).size;
    assert.ok(size > 0, `dist/${name} is empty`);
    assert.ok(size < LIMIT, `dist/${name} is ${size} bytes; must be < ${LIMIT}`);
  }
});
