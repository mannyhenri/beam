// AC-7: `go list -test -deps` of the vector tests includes crypto/hkdf and crypto/pbkdf2,
// and the tests depend on nothing outside the standard library and beam.
import { test } from "node:test";
import assert from "node:assert/strict";
import { go, requireGoMod } from "./_helpers.mjs";

function deps(...pkgs) {
  const r = go(["list", "-test", "-deps", "-f", "{{.ImportPath}} {{.Standard}}", ...pkgs]);
  assert.equal(r.code, 0, r.out);
  return r.stdout.split(/\r?\n/).filter(Boolean).map((l) => {
    const i = l.lastIndexOf(" ");
    return { path: l.slice(0, i), standard: l.slice(i + 1) === "true" };
  });
}

test("AC-7: go list -test -deps ./internal/session ./internal/pairing includes crypto/hkdf and crypto/pbkdf2", () => {
  requireGoMod();
  const paths = deps("./internal/session", "./internal/pairing").map((d) => d.path);
  assert.ok(paths.includes("crypto/hkdf"), "crypto/hkdf missing from the test deps");
  assert.ok(paths.includes("crypto/pbkdf2"), "crypto/pbkdf2 missing from the test deps");
});

test("AC-7: crypto/hkdf comes from the session tests and crypto/pbkdf2 from the pairing tests", () => {
  requireGoMod();
  assert.ok(deps("./internal/session").some((d) => d.path === "crypto/hkdf"), "session tests do not depend on crypto/hkdf");
  assert.ok(deps("./internal/pairing").some((d) => d.path === "crypto/pbkdf2"), "pairing tests do not depend on crypto/pbkdf2");
});

test("AC-7: the vector tests use only the standard library (testing, no third-party framework)", () => {
  requireGoMod();
  const foreign = deps("./internal/session", "./internal/pairing")
    .filter((d) => !d.standard && !/^beam(\/|$)/.test(d.path))
    .map((d) => d.path);
  assert.deepEqual(foreign, [], `non-standard test dependencies: ${foreign.join(", ")}`);
});
