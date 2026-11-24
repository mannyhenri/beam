// AC-6: the PBKDF2 vector test also passes the RFC 7914 §11 c=80000 vector.
import { test } from "node:test";
import assert from "node:assert/strict";
import {
  PBKDF2_FILE, PBKDF2_VECTORS, assertHasHexLiteral, assertValueIsChecked, assertVerbosePass,
  go, readRepoFile, requireGoMod,
} from "./_helpers.mjs";

const V = PBKDF2_VECTORS["c=80000"];

test("AC-6: pbkdf2_vectors_test.go carries the c=80000 vector", () => {
  const src = readRepoFile(PBKDF2_FILE);
  assert.match(src, /RFC\s*7914/, "must cite RFC 7914");
  assert.match(src, /"Password"/, 'must use P="Password"');
  assert.match(src, /"NaCl"/, 'must use S="NaCl"');
  assert.match(src, /\b80_?000\b/, "must use c=80000");
  assertHasHexLiteral(src, V.dk, "RFC 7914 §11 c=80000 DK");
});

test("AC-6: the PBKDF2 vector test passes", () => {
  requireGoMod();
  assertVerbosePass(go(["test", "./internal/pairing", "-run", "PBKDF2", "-v"]), /PBKDF2/, "go test ./internal/pairing -run PBKDF2 -v");
});

test("AC-6: the test checks the c=80000 DK byte for byte", () => {
  requireGoMod();
  assertValueIsChecked(PBKDF2_FILE, V.dk, "PBKDF2", "the c=80000 DK");
});
