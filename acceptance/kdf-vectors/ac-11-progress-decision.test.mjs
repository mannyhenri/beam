// AC-11: PROGRESS.md's Decisions section records that no fallback was written because the
// standard crypto/hkdf and crypto/pbkdf2 packages are available.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readRepoFile, sections } from "./_helpers.mjs";

const DECISIONS = /^decisions that differ from (the )?spec$/i;

test("AC-11: a Decisions line names crypto/hkdf and crypto/pbkdf2 and says no fallback was written", () => {
  const dec = sections(readRepoFile("PROGRESS.md")).find((s) => DECISIONS.test(s.heading));
  assert.ok(dec, "PROGRESS.md has no `Decisions that differ from the spec` section");
  const line = dec.body.split(/\r?\n/).find((l) =>
    l.includes("crypto/hkdf") && l.includes("crypto/pbkdf2") && /fallback/i.test(l) &&
    /\b(no|not|never|without|skipped)\b|n't\b/i.test(l));
  assert.ok(line, `Decisions section needs one line naming crypto/hkdf and crypto/pbkdf2 and saying no fallback was written, got:\n${dec.body}`);
});

test("AC-11: Test results records the HKDF and PBKDF2 vector test runs", () => {
  const res = sections(readRepoFile("PROGRESS.md")).find((s) => /^test results$/i.test(s.heading));
  assert.ok(res, "PROGRESS.md has no `Test results` section");
  assert.match(res.body, /hkdf/i, `Test results must record the HKDF vector test result, got:\n${res.body}`);
  assert.match(res.body, /pbkdf2/i, `Test results must record the PBKDF2 vector test result, got:\n${res.body}`);
});
