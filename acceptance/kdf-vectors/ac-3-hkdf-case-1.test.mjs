// AC-3: `go test ./internal/session -run HKDF -v` passes RFC 5869 A.1 using crypto/hkdf with SHA-256.
import { test } from "node:test";
import assert from "node:assert/strict";
import {
  HKDF_FILE, HKDF_VECTORS, assertHasHexLiteral, assertHkdfOutputIsChecked, assertValueIsChecked, assertVerbosePass,
  go, readRepoFile, requireGoMod,
} from "./_helpers.mjs";

const V = HKDF_VECTORS["A.1"];

test("AC-3: hkdf_vectors_test.go is a package session test of crypto/hkdf with SHA-256 citing RFC 5869 A.1", () => {
  const src = readRepoFile(HKDF_FILE);
  assert.match(src, /^package session\s*$/m, "must be in package session");
  assert.match(src, /"crypto\/hkdf"/, "must import crypto/hkdf");
  assert.match(src, /"crypto\/sha256"/, "must import crypto/sha256");
  for (const fn of ["Extract", "Expand", "Key"]) {
    assert.match(src, new RegExp(`\\bhkdf\\.${fn}\\s*\\(`), `must call hkdf.${fn}`);
  }
  assert.match(src, /sha256\.New\b/, "must use sha256.New as the hash");
  assert.match(src, /RFC\s*5869/, "must cite RFC 5869");
  assert.match(src, /\bA\.1\b/, "must cite test case A.1");
  assertHasHexLiteral(src, "000102030405060708090a0b0c", "A.1 salt");
  assertHasHexLiteral(src, "f0f1f2f3f4f5f6f7f8f9", "A.1 info");
  assertHasHexLiteral(src, V.prk, "A.1 PRK");
  assertHasHexLiteral(src, V.okm, "A.1 OKM");
});

test("AC-3: go test ./internal/session -run HKDF -v exits 0", () => {
  requireGoMod();
  assertVerbosePass(go(["test", "./internal/session", "-run", "HKDF", "-v"]), /HKDF/, "go test ./internal/session -run HKDF -v");
});

test("AC-3: the test checks the A.1 PRK and OKM byte for byte", () => {
  requireGoMod();
  assertValueIsChecked(HKDF_FILE, V.prk, "HKDF", "the A.1 PRK");
  assertValueIsChecked(HKDF_FILE, V.okm, "HKDF", "the A.1 OKM");
});

test("AC-3: the PRK from hkdf.Extract, the OKM from hkdf.Key and hkdf.Expand(PRK) are each compared in full", () => {
  requireGoMod();
  for (const fn of ["Extract", "Key", "Expand"]) assertHkdfOutputIsChecked(fn);
});
