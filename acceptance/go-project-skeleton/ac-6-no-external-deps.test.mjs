// AC-6: every non-standard package in the dependency graph belongs to the beam module.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

function go(args) {
  const r = spawnSync(GO, args, {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "" },
  });
  if (r.error) assert.fail(`could not run \`${GO} ${args.join(" ")}\`: ${r.error.message}`);
  return { code: r.status, stdout: r.stdout, stderr: r.stderr };
}

test("AC-6: go list -deps shows only beam/* outside the standard library", () => {
  const r = go(["list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./..."]);
  assert.equal(r.code, 0, r.stderr);
  const nonStd = r.stdout.split(/\r?\n/).map((l) => l.trim()).filter(Boolean);
  // Guard against a vacuous pass: the module's own packages must be in the graph.
  assert.ok(nonStd.includes("beam/cmd/beam"), `expected beam/cmd/beam in the dependency graph, got:\n${nonStd.join("\n")}`);
  const foreign = nonStd.filter((p) => p !== "beam" && !p.startsWith("beam/"));
  assert.deepEqual(foreign, [], `non-standard packages outside the beam module: ${foreign.join(", ")}`);
});
