// Helpers for the gate-runner acceptance tests. Every test works in a throwaway clone of
// this repo, so run_gates.sh never commits to the real repository. The clone gets the
// working tree's scripts/run_gates.sh (committed into the clone), so the script under
// test is the one on disk, committed or not.
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { chmodSync, copyFileSync, mkdirSync, mkdtempSync, realpathSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");

export const PHASE0_DIRS = [
  "acceptance/go-project-skeleton",
  "acceptance/six-target-build-script",
  "acceptance/simnet-harness",
  "acceptance/no-direct-check",
];

export function sh(cwd, cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { cwd, encoding: "utf8", timeout: opts.timeout ?? 120000, env: opts.env ?? process.env });
  if (r.error) assert.fail(`could not run ${cmd} ${args.join(" ")}: ${r.error.message}`);
  return { code: r.status, signal: r.signal, out: `${r.stdout}${r.stderr}`, stdout: r.stdout };
}

export function git(cwd, ...args) {
  const r = sh(cwd, "git", args);
  assert.equal(r.code, 0, `git ${args.join(" ")} failed in ${cwd}:\n${r.out}`);
  return r.stdout.trim();
}

// A throwaway clone of the repo with the working tree's run_gates.sh committed into it.
export function makeClone(t) {
  const base = realpathSync(mkdtempSync(join(tmpdir(), "gate-ac-")));
  t.after(() => rmSync(base, { recursive: true, force: true }));
  const dir = join(base, "repo");
  const r = sh(ROOT, "git", ["clone", "--quiet", "--no-hardlinks", ROOT, dir]);
  assert.equal(r.code, 0, `git clone failed:\n${r.out}`);
  git(dir, "config", "user.name", "Gate Test");
  git(dir, "config", "user.email", "gate-test@example.invalid");
  git(dir, "config", "commit.gpgsign", "false");
  copyFileSync(join(ROOT, "scripts/run_gates.sh"), join(dir, "scripts/run_gates.sh"));
  chmodSync(join(dir, "scripts/run_gates.sh"), 0o755);
  if (git(dir, "status", "--porcelain", "--", "scripts/run_gates.sh") !== "") {
    git(dir, "add", "scripts/run_gates.sh");
    git(dir, "commit", "--quiet", "-m", "test: use the run_gates.sh under test");
  }
  return { base, dir };
}

export function emptyCommit(dir, subject, body) {
  const args = ["commit", "--quiet", "--allow-empty", "-m", subject];
  if (body) args.push("-m", body);
  git(dir, ...args);
}

export function head(dir) {
  return git(dir, "rev-parse", "HEAD");
}

export function subjects(dir) {
  return git(dir, "log", "--format=%s").split("\n");
}

// Number of worktrees git knows for the clone (1 = only the clone itself).
export function worktreeCount(dir) {
  return git(dir, "worktree", "list", "--porcelain")
    .split("\n")
    .filter((l) => l.startsWith("worktree ")).length;
}

export function runGates(dir, args, opts = {}) {
  return sh(dir, "bash", ["scripts/run_gates.sh", ...args], { timeout: opts.timeout ?? 120000, env: opts.env });
}

// A PATH whose go and node answer version queries but record any other use in a marker
// file, so a test can prove the script ran no build and no test.
export function trapTools(base) {
  const bin = join(base, "trap-bin");
  const marker = join(base, "tool-was-run");
  mkdirSync(bin);
  for (const tool of ["go", "node"]) {
    writeFileSync(
      join(bin, tool),
      `#!/bin/sh\ncase "$1" in version|--version|-v|env) echo "${tool} fake"; exit 0;; esac\necho "${tool} $*" >> "${marker}"\nexit 0\n`,
    );
    chmodSync(join(bin, tool), 0o755);
  }
  return { env: { ...process.env, PATH: `${bin}:${process.env.PATH}` }, marker };
}
