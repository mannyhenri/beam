// AC-2: PROGRESS.md records the go version and availability of crypto/ecdh, crypto/hkdf, crypto/pbkdf2.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const PKGS = ["crypto/ecdh", "crypto/hkdf", "crypto/pbkdf2"];

function sections(md) {
  const out = [];
  let cur = null;
  for (const line of md.split(/\r?\n/)) {
    const m = /^#{1,6}\s+(.+?)\s*#*\s*$/.exec(line);
    if (m) { cur = { heading: m[1].trim(), body: "" }; out.push(cur); }
    else if (cur) cur.body += line + "\n";
  }
  return out;
}

test("AC-2: Go environment section has `go version` output and one availability line per package", () => {
  const p = join(ROOT, "PROGRESS.md");
  assert.ok(existsSync(p), "PROGRESS.md missing at repo root");
  const env = sections(readFileSync(p, "utf8")).find((s) => /go environment/i.test(s.heading));
  assert.ok(env, "PROGRESS.md needs a `Go environment` section");

  assert.match(env.body, /go version go\d+\.\d+(\.\d+)?\S* \S+\/\S+/, "Go environment must contain `go version` output, e.g. `go version go1.24.1 darwin/arm64`");

  for (const pkg of PKGS) {
    const lines = env.body.split(/\r?\n/).filter((l) => l.includes(pkg));
    assert.equal(lines.length, 1, `expected exactly one line mentioning ${pkg} in Go environment, got ${lines.length}`);
    const avail = /\bavailable\b/i.test(lines[0]);
    const missing = /\bmissing\b/i.test(lines[0]);
    assert.ok(avail !== missing, `line for ${pkg} must say exactly one of 'available' or 'missing': ${lines[0]}`);
  }
});
