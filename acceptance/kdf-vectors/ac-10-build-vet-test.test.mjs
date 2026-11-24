// AC-10: `CGO_ENABLED=0 go build ./...`, `go vet ./...` and `go test ./...` succeed;
// go.mod still has no require lines.
import { test } from "node:test";
import assert from "node:assert/strict";
import { go, readRepoFile, requireGoMod } from "./_helpers.mjs";

test("AC-10: CGO_ENABLED=0 go build ./... exits 0", () => {
  requireGoMod();
  const r = go(["build", "./..."], { env: { CGO_ENABLED: "0" } });
  assert.equal(r.code, 0, `go build failed:\n${r.out}`);
});

test("AC-10: go vet ./... exits 0", () => {
  requireGoMod();
  const r = go(["vet", "./..."]);
  assert.equal(r.code, 0, `go vet failed:\n${r.out}`);
});

test("AC-10: go test ./... exits 0 and runs the session and pairing tests", () => {
  requireGoMod();
  const r = go(["test", "./..."]);
  assert.equal(r.code, 0, `go test failed:\n${r.out}`);
  assert.match(r.stdout, /^ok\s+beam\/internal\/session\s/m, `beam/internal/session tests did not run:\n${r.out}`);
  assert.match(r.stdout, /^ok\s+beam\/internal\/pairing\s/m, `beam/internal/pairing tests did not run:\n${r.out}`);
});

test("AC-10: go.mod has no require lines", () => {
  const mod = readRepoFile("go.mod");
  assert.doesNotMatch(mod, /^\s*require\b/m, `go.mod has a require:\n${mod}`);
});
