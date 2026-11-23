---
title: Six-target build script with checksums
slug: six-target-build-script
categories: [devops, testing]
tags: [bash, go, cross-compile, cgo-disabled, sha256, macos, linux, release, scope:system]
date: 2026-11-23
---

## Problem

Beam's definition of done needs "six binaries with checksums built from the final commit", and the skeleton left `scripts/build_all.sh` as a placeholder that exits 1. The stated requirement was "build six targets and write checksums"; the real need is one command that anyone (or a later gate) can run on macOS or Linux and get a release-ready, verifiable `dist/`, without extra tools or network.

## Solution

`scripts/build_all.sh` (46 lines, bash + Go toolchain + coreutils/shasum only). Conventions the later "Release binaries with checksums and sizes" feature should reuse as-is:

- **Layout:** `dist/` at the repo root, git-ignored. Each run clears `dist/beam-*` and `dist/checksums.txt` first.
- **Naming:** `beam-<goos>-<goarch>`, `.exe` added only for windows. Targets: linux, darwin, windows × amd64, arm64.
- **Checksums:** `dist/checksums.txt` in `sha256sum` format, `<64 lowercase hex>  <bare filename>` (two spaces, no path), sorted in C-locale order. Verify with `cd dist && sha256sum -c checksums.txt` or `shasum -a 256 -c checksums.txt`.
- **Tool fallback:** `sha256sum` if on PATH, else `shasum -a 256` (stock macOS has no `sha256sum`); neither → exit 1 before any build.
- **Build env:** `GOTOOLCHAIN=local GOFLAGS='' CGO_ENABLED=0 GOOS=… GOARCH=…` set per build, with `-trimpath -ldflags='-s -w'`. Binaries come out at 1.5-1.7 MB against a 15 MB limit.
- **Failure:** any build error or a binary ≥ 15,728,640 bytes exits 1 naming the file; an `EXIT` trap then clears `dist/` so no partial release is left.

## What Worked

- Writing the locked acceptance tests first with a hostile environment (caller sets `CGO_ENABLED=1 GOOS=plan9 GOARCH=386`, a stale `beam-freebsd-amd64` is planted) forced explicit env and cleanup in the script, rather than leaving them as review comments.
- Set-based tests: each asserts the exact six names, so a partial fix cannot pass.
- Proving the tests could pass against a throwaway script in a scratch copy before locking them.
- Checking failure paths in throwaway `git worktree add --detach` copies (lowered size limit, syntax error in `cmd/beam/main.go`, PATH without `sha256sum`), which kept the real branch clean.

## What Did Not Work

- The checksum-tool check first sat after the builds, so a missing tool failed only after six builds; moved to the top.
- `GOFLAGS=` as an env prefix triggers shellcheck SC1007; write `GOFLAGS=''`.
- Checking `${PIPESTATUS[0]}` from the agent's shell returned nothing because that shell is zsh; run the command without a pipe and read `$?`.
- The Phase 4 workflow had no "Open PR" step, so the stage ended with the branch unpushed even though its focus asked for a PR.

## Prevention

- Pin `GOTOOLCHAIN=local` and `GOFLAGS=''` in every script that calls `go`, as the acceptance tests already do; otherwise an older toolchain with `GOTOOLCHAIN=auto` silently downloads the pinned one (breaks "no outside dependencies").
- Check required tools before doing expensive work.
- Tests that share an output directory (`dist/`) must run one file at a time; NeonCore does, but `node --test` with several files runs them in parallel.
- When planning a pipeline, check that the chosen workflow actually contains the step the focus asks for (here: opening the PR).
- Error paths of the script are still only checked by hand; a future gate-runner feature could add them as tests.

## Related Docs

- No `docs/solutions/` existed before this; this is the first entry.
- `docs/six-target-build-script/build-summary.md`: what was built and how it was verified.
- `docs/six-target-build-script/review.md`: review findings and resolutions.
- `todos/go-project-skeleton/build-summary.md`: earlier note that `node --test <directory>` does not work on Node 23 (run files one by one), which applied here too.
- `PROGRESS.md`, "Decisions that differ from the spec": the recorded layout and env decisions.

## Key Code References

- `scripts/build_all.sh:9-16`: `sha256sum` / `shasum -a 256` selection.
- `scripts/build_all.sh:19-21`: clear `dist/`, `EXIT` trap for failed runs.
- `scripts/build_all.sh:23-24`: 15 MB limit and the six targets.
- `scripts/build_all.sh:35-36`: per-target build env and flags.
- `scripts/build_all.sh:38`: `$(($(wc -c <"$out")))` strips BSD `wc` padding.
- `scripts/build_all.sh:46-48`: `LC_ALL=C` glob order and `checksums.txt`.
- `acceptance/six-target-build-script/ac-{1,2,3}-*.test.mjs`: locked set-based tests.
