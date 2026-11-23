// AC-1: HKDF-SHA256 and PBKDF2-HMAC-SHA256 are available and pass the published vectors
// (on go1.27.4 via the standard packages; a fallback would have to pass the same tests).
import { test } from "node:test";
import assert from "node:assert/strict";
import { createHmac, hkdfSync, pbkdf2Sync } from "node:crypto";
import {
  HKDF_FILE, HKDF_VECTORS, PBKDF2_FILE, PBKDF2_VECTORS,
  assertHasHexLiteral, assertVerbosePass, go, readRepoFile, requireGoMod,
} from "./_helpers.mjs";

test("AC-1: the reference vectors match an independent HMAC-based implementation", () => {
  for (const [name, v] of Object.entries(HKDF_VECTORS)) {
    assert.equal(createHmac("sha256", v.salt).update(v.ikm).digest("hex"), v.prk, `RFC 5869 ${name} PRK`);
    assert.equal(Buffer.from(hkdfSync("sha256", v.ikm, v.salt, v.info, v.L)).toString("hex"), v.okm, `RFC 5869 ${name} OKM`);
  }
  for (const [name, v] of Object.entries(PBKDF2_VECTORS)) {
    assert.equal(pbkdf2Sync(v.P, v.S, v.c, v.dkLen, "sha256").toString("hex"), v.dk, `RFC 7914 §11 ${name}`);
  }
});

test("AC-1: the vector tests contain every published output and pass", () => {
  requireGoMod();
  const hkdfSrc = readRepoFile(HKDF_FILE);
  for (const [name, v] of Object.entries(HKDF_VECTORS)) {
    assertHasHexLiteral(hkdfSrc, v.prk, `RFC 5869 ${name} PRK`);
    assertHasHexLiteral(hkdfSrc, v.okm, `RFC 5869 ${name} OKM`);
  }
  const pbkdf2Src = readRepoFile(PBKDF2_FILE);
  for (const [name, v] of Object.entries(PBKDF2_VECTORS)) assertHasHexLiteral(pbkdf2Src, v.dk, `RFC 7914 §11 ${name} DK`);

  assertVerbosePass(go(["test", "./internal/session", "-run", "HKDF", "-v"]), /HKDF/, "go test ./internal/session -run HKDF -v");
  assertVerbosePass(go(["test", "./internal/pairing", "-run", "PBKDF2", "-v"]), /PBKDF2/, "go test ./internal/pairing -run PBKDF2 -v");
});
