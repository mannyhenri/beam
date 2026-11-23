// AC-3: phase N ≥ 1 is refused until a commit whose subject is exactly
// "Gate: Phase <N-1> passed" exists. The history is hostile: subjects mention "Phase 0"
// and "Phase 3" in other wording, one quotes the marker inside a longer subject, and one
// has the exact marker text only in its body. None of these count. Fake go and node on
// PATH prove that nothing was built or tested.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { emptyCommit, git, head, makeClone, runGates, trapTools, worktreeCount } from "./clone.mjs";

test("AC-3: phase 1 is refused without an exact 'Gate: Phase 0 passed' commit", (t) => {
  const { base, dir } = makeClone(t);
  emptyCommit(dir, "Add summary for Phase 0");
  emptyCommit(dir, "test: something — write failing TDD tests for Phase 3");
  emptyCommit(dir, 'Revert "Gate: Phase 0 passed"');
  emptyCommit(dir, "Notes from the gate review", "Gate: Phase 0 passed");
  const before = head(dir);
  const { env, marker } = trapTools(base);

  const r = runGates(dir, ["1"], { env });
  assert.equal(r.code, 2, `run_gates.sh 1 must exit 2 without the phase 0 gate, got ${r.code}:\n${r.out}`);
  assert.match(r.out, /phase 0/i, `the message must name the missing phase 0 gate:\n${r.out}`);

  assert.ok(!existsSync(marker), `go or node was run although the gate was refused:\n${existsSync(marker) ? readFileSync(marker, "utf8") : ""}`);
  assert.equal(head(dir), before, "no commit may be made");
  assert.equal(worktreeCount(dir), 1, `a worktree was created or left behind:\n${git(dir, "worktree", "list")}`);
});
