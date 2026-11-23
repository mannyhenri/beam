// AC-8: no non-test Go file implements HKDF or PBKDF2; the only *hkdf*/*pbkdf2* Go files
// are the two vector tests.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import { join, relative, sep } from "node:path";
import { HKDF_FILE, PBKDF2_FILE, ROOT } from "./_helpers.mjs";

function goFiles(dir, out = []) {
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    if (e.name === ".git" || e.name === "node_modules") continue;
    const p = join(dir, e.name);
    if (e.isDirectory()) goFiles(p, out);
    else if (e.isFile() && e.name.endsWith(".go")) out.push(relative(ROOT, p).split(sep).join("/"));
  }
  return out;
}

test("AC-8: no non-test .go file declares a func mentioning hkdf or pbkdf2", () => {
  const hits = goFiles(ROOT)
    .filter((f) => !f.endsWith("_test.go"))
    .filter((f) => /func .*(hkdf|pbkdf2)/i.test(readFileSync(join(ROOT, f), "utf8")));
  assert.deepEqual(hits, [], `non-test files implementing HKDF/PBKDF2: ${hits.join(", ")}`);
});

test("AC-8: the only *hkdf*.go / *pbkdf2*.go files are the two vector tests", () => {
  const named = goFiles(ROOT).filter((f) => /hkdf|pbkdf2/i.test(f.split("/").pop())).sort();
  assert.deepEqual(named, [HKDF_FILE, PBKDF2_FILE].sort());
});
