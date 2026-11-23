// AC-5: `go test ./internal/pairing -run PBKDF2 -v` passes the RFC 7914 §11 c=1 PBKDF2-HMAC-SHA256 vector.
import { test } from "node:test";
import assert from "node:assert/strict";
import {
  PBKDF2_FILE, PBKDF2_VECTORS, assertHasHexLiteral, assertValueIsChecked, assertVerbosePass,
  go, readRepoFile, requireGoMod,
} from "./_helpers.mjs";

const V = PBKDF2_VECTORS["c=1"];

test("AC-5: pbkdf2_vectors_test.go is a package pairing test of crypto/pbkdf2 with SHA-256 citing RFC 7914 §11", () => {
  const src = readRepoFile(PBKDF2_FILE);
  assert.match(src, /^package pairing\s*$/m, "must be in package pairing");
  assert.match(src, /"crypto\/pbkdf2"/, "must import crypto/pbkdf2");
  assert.match(src, /"crypto\/sha256"/, "must import crypto/sha256");
  assert.match(src, /\bpbkdf2\.Key\s*\(/, "must call pbkdf2.Key");
  assert.match(src, /sha256\.New\b/, "must use sha256.New as the hash");
  assert.match(src, /RFC\s*7914/, "must cite RFC 7914");
  assert.match(src, /(§|section|sec\.?)\s*11\b/i, "must cite section 11");
  assert.match(src, /"passwd"/, 'must use P="passwd"');
  assert.match(src, /"salt"/, 'must use S="salt"');
  assertHasHexLiteral(src, V.dk, "RFC 7914 §11 c=1 DK");
});

test("AC-5: go test ./internal/pairing -run PBKDF2 -v exits 0", () => {
  requireGoMod();
  assertVerbosePass(go(["test", "./internal/pairing", "-run", "PBKDF2", "-v"]), /PBKDF2/, "go test ./internal/pairing -run PBKDF2 -v");
});

test("AC-5: the test checks the c=1 DK byte for byte", () => {
  requireGoMod();
  assertValueIsChecked(PBKDF2_FILE, V.dk, "PBKDF2", "the c=1 DK");
});
