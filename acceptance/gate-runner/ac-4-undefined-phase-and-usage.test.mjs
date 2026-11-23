// AC-4: with the phase 0 gate passed, phase 1 has no gate yet and exits 2 with
// "gate for phase 1 is not defined yet"; no argument, "x", "8" and "-1" each exit 2 with a
// usage message. No case may commit, build or test (fake go and node on PATH record any use).
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { emptyCommit, git, head, makeClone, runGates, trapTools, worktreeCount } from "./clone.mjs";

test("AC-4: undefined phases and bad arguments exit 2 without committing", (t) => {
  const { base, dir } = makeClone(t);
  emptyCommit(dir, "Gate: Phase 0 passed");
  const before = head(dir);
  const { env, marker } = trapTools(base);

  const undef = runGates(dir, ["1"], { env });
  assert.equal(undef.code, 2, `run_gates.sh 1 must exit 2, got ${undef.code}:\n${undef.out}`);
  assert.match(undef.out, /gate for phase 1 is not defined yet/, `missing 'gate for phase 1 is not defined yet':\n${undef.out}`);
  assert.equal(head(dir), before, "run_gates.sh 1 must not commit");

  for (const args of [[], ["x"], ["8"], ["-1"]]) {
    const r = runGates(dir, args, { env });
    const shown = args.length ? args.join(" ") : "(no argument)";
    assert.equal(r.code, 2, `run_gates.sh ${shown} must exit 2, got ${r.code}:\n${r.out}`);
    assert.match(r.out, /usage/i, `run_gates.sh ${shown} must print a usage message:\n${r.out}`);
    assert.equal(head(dir), before, `run_gates.sh ${shown} must not commit`);
  }

  assert.ok(!existsSync(marker), `go or node was run:\n${existsSync(marker) ? readFileSync(marker, "utf8") : ""}`);
  assert.equal(worktreeCount(dir), 1, `a worktree was created or left behind:\n${git(dir, "worktree", "list")}`);
});
