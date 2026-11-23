# Build summary: gate-runner

Branch `feature/gate-runner`, worktree `../gate-runner-work`, from main 8ea8ba8. The scope brief served as the plan (no plan.md; the locked acceptance tests are the TDD tests, so no separate test suite was written).

## What changed

- `scripts/run_gates.sh`: replaces the placeholder. Validates the argument (0-7, else usage and exit 2), checks git/go/node/bash, refuses phase N ≥ 1 without an exact `Gate: Phase <N-1> passed` subject (exit 2), exits 2 for phases 1-7 ("gate for phase N is not defined yet"), and for phase 0 runs in a temporary `git worktree` of HEAD: `bash scripts/build_all.sh`, then every committed `*.test.mjs` under the four phase 0 directories, one per `node --test`, C-locale order, stopping at the first failure with that check's output. On pass it makes one empty commit `Gate: Phase 0 passed` (body `Validated <sha>.`) if HEAD did not move. An EXIT trap (with INT/TERM traps so it also runs on signals) removes the worktree and temp dir.
- `PROGRESS.md`: gate usage, exit codes, marker format, phase 0 contents, test results; the placeholder note updated.
- `acceptance/gate-runner/*`: the locked tests, committed byte-identical (cf10e0a).

## Verification (worktree)

- `shellcheck scripts/*.sh`: clean.
- Acceptance, one file per `node --test`: 23/23 pass (gate-runner 4, no-direct-check 3, simnet-harness 3, go-project-skeleton 10, six-target-build-script 3). The gate-runner tests run the script only in throwaway clones.
- By hand in a throwaway clone: SIGTERM mid-gate → exit 143, `git worktree list` back to one entry, no marker commit.
- `run_gates.sh` was never run against this repo or the feature worktree.
- Main checkout (`/Users/mannyhenri/code/beam`, local `main` fast-forwarded to ace3e1b, no merge commit): shellcheck clean, acceptance 23/23 pass, and `git log` has no `Gate: Phase` commit (the tests committed only in their clones). Not pushed.

## Deviations

- None in behaviour. Two hardening details beyond the brief: each check runs with stdin from /dev/null (so a check cannot swallow the file list), and an empty phase 0 file list fails the gate instead of passing with nothing run.

## First real gate

Run it yourself from the main checkout, with 127.0.0.2-.4 up: `bash scripts/run_gates.sh 0`. It adds the commit `Gate: Phase 0 passed` to local main; nothing is pushed.
