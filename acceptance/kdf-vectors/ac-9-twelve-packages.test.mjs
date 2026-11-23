// AC-9: `go list ./...` still lists exactly the 12 skeleton packages, with the vector tests
// as in-package test files of session and pairing.
import { test } from "node:test";
import assert from "node:assert/strict";
import { SKELETON_PACKAGES, go, requireGoMod } from "./_helpers.mjs";

test("AC-9: go list ./... returns exactly the 12 skeleton packages", () => {
  requireGoMod();
  const r = go(["list", "./..."]);
  assert.equal(r.code, 0, r.out);
  const got = r.stdout.split(/\r?\n/).filter(Boolean).sort();
  assert.deepEqual(got, SKELETON_PACKAGES);
});

test("AC-9: the vector tests live in package session and package pairing", () => {
  requireGoMod();
  const r = go(["list", "-f", "{{.ImportPath}}|{{join .TestGoFiles \",\"}}|{{join .XTestGoFiles \",\"}}", "beam/internal/session", "beam/internal/pairing"]);
  assert.equal(r.code, 0, r.out);
  const rows = Object.fromEntries(r.stdout.split(/\r?\n/).filter(Boolean).map((l) => {
    const [pkg, t, x] = l.split("|");
    return [pkg, { t: t.split(",").filter(Boolean), x: x.split(",").filter(Boolean) }];
  }));
  assert.ok(rows["beam/internal/session"].t.includes("hkdf_vectors_test.go"), "hkdf_vectors_test.go is not an in-package test of beam/internal/session");
  assert.ok(rows["beam/internal/pairing"].t.includes("pbkdf2_vectors_test.go"), "pbkdf2_vectors_test.go is not an in-package test of beam/internal/pairing");
  assert.ok(!rows["beam/internal/session"].x.includes("hkdf_vectors_test.go"), "hkdf_vectors_test.go must be package session, not session_test");
  assert.ok(!rows["beam/internal/pairing"].x.includes("pbkdf2_vectors_test.go"), "pbkdf2_vectors_test.go must be package pairing, not pairing_test");
});
