// AC-1: the section 3 layout exists with a go.mod that declares no dependencies.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

const INTERNAL = ["ws", "relay", "identity", "pairing", "session", "mux", "transfer", "sysinfo", "tunnel", "ui", "simnet"];

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

function stripComments(src) {
  return src.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
}

test("AC-1: go.mod declares module beam, matches the toolchain, and has no require/replace", () => {
  const modPath = join(ROOT, "go.mod");
  assert.ok(existsSync(modPath), "go.mod missing at repo root");
  const mod = stripComments(readFileSync(modPath, "utf8"));
  const lines = mod.split(/\r?\n/).map((l) => l.trim()).filter(Boolean);

  assert.ok(lines.includes("module beam"), `go.mod must declare \`module beam\`, got:\n${mod}`);
  for (const l of lines) {
    assert.ok(!/^(require|replace|exclude|retract)\b/.test(l), `go.mod must not contain dependency directives: ${l}`);
  }
  assert.ok(!existsSync(join(ROOT, "go.sum")), "go.sum must not exist");

  const goLine = lines.find((l) => /^go\s+/.test(l));
  assert.ok(goLine, "go.mod must contain a `go` directive");
  const directive = goLine.split(/\s+/)[1];
  const v = go(["env", "GOVERSION"]);
  assert.equal(v.code, 0, v.stderr);
  const installed = v.stdout.trim().replace(/^go/, ""); // e.g. 1.24.1
  const majorMinor = installed.split(".").slice(0, 2).join(".");
  assert.ok(
    directive === installed || directive === majorMinor,
    `go directive ${directive} must match installed toolchain ${installed} (or ${majorMinor})`,
  );
});

test("AC-1: cmd/beam/main.go is package main", () => {
  const p = join(ROOT, "cmd", "beam", "main.go");
  assert.ok(existsSync(p), "cmd/beam/main.go missing");
  assert.match(stripComments(readFileSync(p, "utf8")), /^\s*package\s+main\s*$/m);
});

test("AC-1: every internal/* package has a doc.go with a package comment and matching clause", () => {
  for (const name of INTERNAL) {
    const p = join(ROOT, "internal", name, "doc.go");
    assert.ok(existsSync(p), `internal/${name}/doc.go missing`);
    const src = readFileSync(p, "utf8");
    assert.match(stripComments(src), new RegExp(`^\\s*package\\s+${name}\\s*$`, "m"), `internal/${name}/doc.go must say \`package ${name}\``);
    // A package comment is a comment block immediately preceding the package clause.
    assert.match(src, new RegExp(`(//[^\\n]*\\n|\\*/\\s*\\n)package\\s+${name}\\b`), `internal/${name}/doc.go needs a package comment directly above the clause`);
  }
});
