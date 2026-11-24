// AC-4: the HKDF vector test also passes RFC 5869 A.2 and A.3 (PRK and OKM byte for byte).
import { test } from "node:test";
import assert from "node:assert/strict";
import {
  HKDF_FILE, HKDF_VECTORS, assertHasHexLiteral, assertValueIsChecked, assertVerbosePass,
  go, readRepoFile, requireGoMod,
} from "./_helpers.mjs";

test("AC-4: hkdf_vectors_test.go carries RFC 5869 A.2 and A.3 with citations", () => {
  const src = readRepoFile(HKDF_FILE);
  assert.match(src, /RFC\s*5869/, "must cite RFC 5869");
  for (const name of ["A.2", "A.3"]) {
    assert.match(src, new RegExp(`\\b${name.replace(".", "\\.")}\\b`), `must cite test case ${name}`);
    assertHasHexLiteral(src, HKDF_VECTORS[name].prk, `${name} PRK`);
    assertHasHexLiteral(src, HKDF_VECTORS[name].okm, `${name} OKM`);
  }
  assertHasHexLiteral(src, HKDF_VECTORS["A.2"].ikm.toString("hex"), "A.2 IKM");
  assertHasHexLiteral(src, HKDF_VECTORS["A.2"].salt.toString("hex"), "A.2 salt");
  assertHasHexLiteral(src, HKDF_VECTORS["A.2"].info.toString("hex"), "A.2 info");
});

test("AC-4: the HKDF vector test passes", () => {
  requireGoMod();
  assertVerbosePass(go(["test", "./internal/session", "-run", "HKDF", "-v"]), /HKDF/, "go test ./internal/session -run HKDF -v");
});

test("AC-4: the test checks the A.2 and A.3 PRK and OKM byte for byte", () => {
  requireGoMod();
  for (const name of ["A.2", "A.3"]) {
    assertValueIsChecked(HKDF_FILE, HKDF_VECTORS[name].prk, "HKDF", `the ${name} PRK`);
    assertValueIsChecked(HKDF_FILE, HKDF_VECTORS[name].okm, "HKDF", `the ${name} OKM`);
  }
});
