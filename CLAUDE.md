# Beam: notes for agents

Beam is a single Go binary, standard library only. PROGRESS.md records every decision that differs from the spec; docs/solutions/ holds per-feature learnings.

## Patterns

- Acceptance tests live in `acceptance/<slug>/ac-N-*.test.mjs` (Node `node:test`, no dependencies). Run them one file at a time: `node --test acceptance/<slug>/ac-1-*.test.mjs`.
- Locked acceptance tests are committed byte-identical as the first commit on the feature branch ("Add locked acceptance tests for <slug>"); compare SHA-256 before and after.
- Write acceptance tests that check the whole set (exact list of names, loop every member) so a partial fix fails, and give them a hostile setup (conflicting env vars, stale files) to force the behaviour the scope asks for.
- Before locking tests, prove they can pass against a throwaway implementation in a scratch copy, not in the repo.
- Check failure paths in a throwaway `git worktree add --detach <scratch path> HEAD`, then `git worktree remove --force`.
- Feature work happens in `../<slug>-work` on `feature/<slug>`; build summaries go to `docs/<slug>/` and `todos/<slug>/`.
- Before reporting any stage done, run the locked tests in the main checkout (`/Users/mannyhenri/code/beam`) as well as the worktree. NeonCore runs them in the main checkout.

## Anti-patterns

- Calling `go` from a script without `GOTOOLCHAIN=local GOFLAGS=''`. Signal: the acceptance tests pin these but the script does not. With an older Go and `GOTOOLCHAIN=auto`, go downloads the toolchain pinned in go.mod, breaking "no outside dependencies". The six-target build script shipped without this and review had to add it.
- Leaving partial output after a failed run. Signal: a script writes several artifacts in a loop with no cleanup on error. Use an `EXIT` trap that removes them on non-zero status (see `scripts/build_all.sh`). Also caught only in review.
- Checking for a required tool after the expensive work. Check tools first.
- Running several acceptance test files in one `node --test` call when they share an output directory (`dist/`): they run in parallel and can collide.
- Planning a pipeline stage whose focus asks for a step the chosen workflow does not have (CE: Phase 4 — Review has no "Open PR" step, so the PR was never opened). Check the workflow's steps against the focus.
- Reporting "3/3 pass" from the worktree while `main` still holds the old code. Signal: `git log main..feature/<slug>` is non-empty when a stage reports success. On six-target-build-script every stage passed on the branch, yet the goal check failed 0/3 and needed an extra retest stage to fast-forward `main`.

## Taste decisions

- Release output: `dist/` at the repo root, git-ignored; binaries `beam-<goos>-<goarch>`, `.exe` only for windows; `dist/checksums.txt` in `sha256sum` format with bare filenames, sorted in C-locale order. Reuse this for any later release work.
- Release builds: `CGO_ENABLED=0 -trimpath -ldflags='-s -w'`; binaries are about 1.5-1.7 MB against a 15 MB limit.
- Shell scripts: `#!/usr/bin/env bash` + `set -euo pipefail`, flat, no functions unless needed, shellcheck-clean.
- Getting the feature onto `main`: fast-forward local `main` to the feature branch (no merge commit). Pushing to origin or opening a PR is outward-facing, so ask the person first; a local fast-forward is enough for NeonCore's check.

## Caveats

- Stock macOS has no `sha256sum`; fall back to `shasum -a 256`. Use `wc -c <file` for sizes (BSD and GNU `stat` differ); wrap it as `$(($(wc -c <f)))` to strip BSD padding.
- shellcheck flags `VAR= cmd` (SC1007); write `VAR='' cmd`.
- The agent's interactive shell is zsh: `${PIPESTATUS[0]}` is empty there. Avoid pipes when you need an exit code, or run under `bash -c`.
- `node --test <directory>` does not work on Node 23; pass files.
- go.mod pins `go 1.27.4`; older toolchains refuse to build.
- The main checkout keeps untracked copies of the locked tests (`acceptance/<slug>/`), so `git merge` into `main` fails with "untracked working tree files would be overwritten". Do not delete or move them. Confirm their SHA-256 matches the branch's copies, then `git update-ref refs/heads/main <branch-sha> <old-main-sha> && git reset --mixed && git checkout -- .`.
