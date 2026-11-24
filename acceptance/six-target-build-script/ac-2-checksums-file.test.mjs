// AC-2: after `bash scripts/build_all.sh`, dist/checksums.txt holds exactly six correct
// SHA-256 lines ('<64 lowercase hex>  <bare filename>'), one per binary, and verifies
// with sha256sum -c / shasum -a 256 -c run inside dist/.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const DIST = join(ROOT, "dist");

const BINARIES = [
  "beam-darwin-amd64",
  "beam-darwin-arm64",
  "beam-linux-amd64",
  "beam-linux-arm64",
  "beam-windows-amd64.exe",
  "beam-windows-arm64.exe",
];

function has(cmd) {
  return spawnSync("bash", ["-c", `command -v ${cmd}`], { encoding: "utf8" }).status === 0;
}

test("AC-2: dist/checksums.txt lists a correct SHA-256 for each of the six binaries", () => {
  const r = spawnSync("bash", ["scripts/build_all.sh"], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 600000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run build_all.sh: ${r.error.message}`);
  assert.equal(r.status, 0, `build_all.sh exited ${r.status}:\n${r.stdout}${r.stderr}`);

  const sumsPath = join(DIST, "checksums.txt");
  assert.ok(existsSync(sumsPath), "dist/checksums.txt missing");
  const text = readFileSync(sumsPath, "utf8");
  const lines = text.split("\n").filter((l) => l !== "");
  assert.equal(lines.length, 6, `expected exactly 6 lines, got ${lines.length}:\n${text}`);

  const names = [];
  for (const line of lines) {
    const m = /^([0-9a-f]{64})  (\S+)$/.exec(line);
    assert.ok(m, `line not in '<64 lowercase hex>  <filename>' format: ${JSON.stringify(line)}`);
    const [, hash, name] = m;
    assert.ok(BINARIES.includes(name), `unexpected or path-qualified filename: ${name}`);
    const actual = createHash("sha256").update(readFileSync(join(DIST, name))).digest("hex");
    assert.equal(hash, actual, `checksum mismatch for ${name}`);
    names.push(name);
  }
  assert.deepEqual([...names].sort(), [...BINARIES].sort(), "each binary must appear exactly once");
  assert.deepEqual(names, [...names].sort(), "lines must be sorted by filename");

  const tool = has("sha256sum") ? ["sha256sum", "-c", "checksums.txt"]
    : has("shasum") ? ["shasum", "-a", "256", "-c", "checksums.txt"]
    : null;
  assert.ok(tool, "neither sha256sum nor shasum is available to verify checksums.txt");
  const c = spawnSync(tool[0], tool.slice(1), { cwd: DIST, encoding: "utf8", timeout: 60000 });
  assert.equal(c.status, 0, `${tool.join(" ")} failed:\n${c.stdout}${c.stderr}`);
  const ok = c.stdout.split("\n").filter((l) => /: OK$/.test(l));
  assert.equal(ok.length, 6, `expected 6 OK lines from ${tool[0]}:\n${c.stdout}`);
});
