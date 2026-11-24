// AC-1: run_gates.sh 0 runs build_all.sh and every phase 0 acceptance file from a clean
// checkout, exits 0 and adds exactly one empty commit "Gate: Phase 0 passed". The working
// tree is hostile: an uncommitted edit breaks cmd/beam/main.go and junk.txt is untracked,
// so a gate that runs in place instead of in a clean checkout fails the build.
// Takes several minutes. Needs the loopback aliases 127.0.0.2-.4 (simnet tests).
import { test } from "node:test";
import assert from "node:assert/strict";
import { appendFileSync, existsSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { PHASE0_DIRS, git, head, makeClone, runGates, sh, worktreeCount } from "./clone.mjs";

test("AC-1: phase 0 gate passes from a clean checkout and records one empty marker commit", (t) => {
  const { dir } = makeClone(t);
  const before = head(dir);
  const files = git(dir, "ls-files", ...PHASE0_DIRS.map((d) => `${d}/*.test.mjs`))
    .split("\n")
    .filter(Boolean);
  assert.ok(files.length >= 19, `expected the phase 0 acceptance files in the clone, found ${files.length}`);

  const mainGo = join(dir, "cmd/beam/main.go");
  appendFileSync(mainGo, "\nthis line is not Go and breaks the build\n");
  const brokenMain = readFileSync(mainGo, "utf8");
  writeFileSync(join(dir, "junk.txt"), "untracked junk\n");

  const r = runGates(dir, ["0"], { timeout: 1800000 });
  assert.equal(r.code, 0, `run_gates.sh 0 must exit 0, got ${r.code}:\n${r.out}`);

  const lines = r.out.split("\n");
  const passLine = (name) => lines.some((l) => l.includes(name) && /\bPASS\b/.test(l));
  assert.ok(passLine("build_all.sh"), `no PASS line for build_all.sh:\n${r.out}`);
  for (const f of files) {
    assert.ok(passLine(f), `no PASS line for ${f}:\n${r.out}`);
  }
  assert.ok(!lines.some((l) => /\bFAIL\b/.test(l)), `a check reported FAIL:\n${r.out}`);

  assert.equal(git(dir, "rev-list", "--count", `${before}..HEAD`), "1", "exactly one new commit expected");
  assert.equal(git(dir, "rev-parse", "HEAD~1"), before, "the new commit must sit directly on the old HEAD");
  assert.equal(git(dir, "log", "-1", "--format=%s"), "Gate: Phase 0 passed");
  assert.equal(git(dir, "diff", "--name-only", "HEAD~1", "HEAD"), "", "the marker commit must change no files");

  assert.equal(readFileSync(mainGo, "utf8"), brokenMain, "the uncommitted main.go edit must be left untouched");
  assert.ok(existsSync(join(dir, "junk.txt")), "junk.txt must still exist");
  assert.equal(readFileSync(join(dir, "junk.txt"), "utf8"), "untracked junk\n");
  const status = sh(dir, "git", ["status", "--porcelain"]).stdout; // untrimmed: the leading space matters
  assert.match(status, /^ M cmd\/beam\/main\.go$/m, `main.go must still be modified, not staged or committed:\n${status}`);
  assert.match(status, /^\?\? junk\.txt$/m, `junk.txt must still be untracked:\n${status}`);

  assert.equal(worktreeCount(dir), 1, `a temporary worktree was left behind:\n${git(dir, "worktree", "list")}`);
});

