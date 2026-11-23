// AC-2: with crypto/hkdf and crypto/pbkdf2 in the standard library, they are used directly
// and no fallback code is compiled in.
import { test } from "node:test";
import assert from "node:assert/strict";
import { go, requireGoMod } from "./_helpers.mjs";

test("AC-2: the installed Go provides crypto/hkdf and crypto/pbkdf2", () => {
  const r = go(["list", "crypto/hkdf", "crypto/pbkdf2"]);
  assert.equal(r.code, 0, r.out);
  assert.deepEqual(r.stdout.split(/\r?\n/).filter(Boolean), ["crypto/hkdf", "crypto/pbkdf2"]);
});

test("AC-2: the vector tests import the standard packages directly, with no HMAC fallback", () => {
  requireGoMod();
  for (const [pkg, want] of [["./internal/session", "crypto/hkdf"], ["./internal/pairing", "crypto/pbkdf2"]]) {
    const r = go(["list", "-f", "{{join .TestImports \"\\n\"}}", pkg]);
    assert.equal(r.code, 0, r.out);
    const imports = r.stdout.split(/\r?\n/).filter(Boolean);
    assert.ok(imports.includes(want), `${pkg} tests must import ${want}; imports: ${imports.join(", ")}`);
    assert.ok(imports.includes("crypto/sha256"), `${pkg} tests must import crypto/sha256; imports: ${imports.join(", ")}`);
    assert.ok(!imports.includes("crypto/hmac"), `${pkg} tests import crypto/hmac (a hand-written KDF); use ${want} directly`);
    assert.deepEqual(imports.filter((i) => i.startsWith("beam/")), [], `${pkg} tests must not route through another beam package`);
  }
});

test("AC-2: no beam package compiles in, or holds behind build tags, an HMAC-based KDF", () => {
  requireGoMod();
  const r = go(["list", "-f", "{{.ImportPath}}|{{join .Imports \",\"}}|{{join .GoFiles \",\"}}|{{join .IgnoredGoFiles \",\"}}", "./..."]);
  assert.equal(r.code, 0, r.out);
  for (const line of r.stdout.split(/\r?\n/).filter(Boolean)) {
    const [pkg, imports, files, ignored] = line.split("|");
    assert.ok(!imports.split(",").includes("crypto/hmac"), `${pkg} imports crypto/hmac in non-test code`);
    for (const f of [...files.split(","), ...ignored.split(",")].filter(Boolean)) {
      assert.doesNotMatch(f, /hkdf|pbkdf2/i, `${pkg} has fallback file ${f}`);
    }
  }
});
