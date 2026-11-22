// AC-7: `beam` with no arguments prints a usage line naming its commands and exits 0.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

const COMMANDS = ["relay", "open", "join", "peers", "send", "recv", "pipe", "allow", "forget", "sysinfo", "tunnel", "ui"];

function go(args) {
  const r = spawnSync(GO, args, {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    input: "",
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "", CGO_ENABLED: "0" },
  });
  if (r.error) assert.fail(`could not run \`${GO} ${args.join(" ")}\`: ${r.error.message}`);
  return { code: r.status, stdout: r.stdout, stderr: r.stderr };
}

test("AC-7: go run ./cmd/beam prints usage naming every command on stdout and exits 0", () => {
  const r = go(["run", "./cmd/beam"]);
  assert.equal(r.code, 0, `exit ${r.code}\nstdout:\n${r.stdout}\nstderr:\n${r.stderr}`);
  assert.match(r.stdout, /usage/i, `stdout must contain a usage line, got:\n${r.stdout}`);
  for (const cmd of COMMANDS) {
    assert.match(r.stdout, new RegExp(`\\b${cmd}\\b`), `stdout must name the \`${cmd}\` command, got:\n${r.stdout}`);
  }
});
