// AC-2: when a phase 0 check fails, the gate exits 1, makes no commit and leaves no
// worktree behind. A committed file makes cmd/beam fail to build, so build_all.sh fails
// first and no acceptance test may report a pass after it.
import { test } from "node:test";
import assert from "node:assert/strict";
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import { git, head, makeClone, runGates, subjects, worktreeCount } from "./clone.mjs";

test("AC-2: a failing phase 0 check exits 1 with no commit and no leftover worktree", (t) => {
  const { dir } = makeClone(t);
  writeFileSync(join(dir, "cmd/beam/zz_broken.go"), "package main\n\nfunc broken() { undefinedFunction() }\n");
  git(dir, "add", "cmd/beam/zz_broken.go");
  git(dir, "commit", "--quiet", "-m", "Break cmd/beam on purpose");
  const before = head(dir);

  const r = runGates(dir, ["0"], { timeout: 900000 });
  assert.equal(r.code, 1, `run_gates.sh 0 must exit 1 when a check fails, got ${r.code}:\n${r.out}`);

  const lines = r.out.split("\n");
  assert.ok(lines.some((l) => l.includes("build_all.sh") && /\bFAIL\b/.test(l)), `no FAIL line for build_all.sh:\n${r.out}`);
  assert.ok(!lines.some((l) => l.includes(".test.mjs") && /\bPASS\b/.test(l)), `acceptance tests ran after build_all.sh failed:\n${r.out}`);

  assert.equal(head(dir), before, "HEAD must not move when the gate fails");
  assert.ok(!subjects(dir).includes("Gate: Phase 0 passed"), "no 'Gate: Phase 0 passed' commit may exist");
  assert.equal(worktreeCount(dir), 1, `a temporary worktree was left behind:\n${git(dir, "worktree", "list")}`);
});
