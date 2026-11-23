# Build summary: six-target-build-script

Branch `feature/six-target-build-script`, worktree `../six-target-build-script-work`. The scope brief served as the plan (no plan.md; Phases 1-2 skipped).

## What changed

- `scripts/build_all.sh`: replaces the placeholder. Finds the repo root from its own location, checks for `sha256sum` (else `shasum -a 256`) before building, clears `dist/beam-*` and `dist/checksums.txt`, builds `./cmd/beam` for linux/darwin/windows × amd64/arm64 with `CGO_ENABLED`, `GOOS`, `GOARCH` set explicitly and `-trimpath -ldflags='-s -w'`, fails naming the file if any binary is ≥ 15,728,640 bytes (`wc -c`), then writes `dist/checksums.txt` with bare filenames in C-locale order.
- `.gitignore`: adds `dist/`.
- `PROGRESS.md`: current phase, time, test results, and decisions (output layout, build flags, checksum tool fallback).
- `acceptance/six-target-build-script/*.test.mjs`: the locked tests, committed byte-identical (3712173).

## Verification

- `shellcheck scripts/build_all.sh`: clean.
- Run from repo root and from `/tmp`: exit 0, six binaries of 1.5-1.7 MB.
- Two consecutive runs: same file list, identical `checksums.txt`.
- `cd dist && sha256sum -c checksums.txt` and `shasum -a 256 -c checksums.txt`: 6/6 OK.
- Locked tests AC-1, AC-2, AC-3 (`node --test acceptance/six-target-build-script/ac-N-*.test.mjs`): 3/3 pass. File hashes unchanged.
- Skeleton acceptance tests (`acceptance/go-project-skeleton`): 10/10 still pass.
- Negative paths, checked manually: limit lowered to 1000 bytes → exit 1, `build_all.sh: beam-linux-amd64 is 1519776 bytes; limit is under 1000`; PATH with only `shasum` → exit 0, checksums verify; PATH with neither tool → exit 1 before any build, `build_all.sh: need sha256sum or shasum to write checksums.txt`.
- `scripts/run_gates.sh`, `acceptance/go-project-skeleton`, `go.mod`: no diff from main.

## Deviations

- The checksum-tool check runs before the builds rather than after, so a missing tool fails immediately instead of after six builds.

## For review

Points worth a look: `export LC_ALL=C` makes the `beam-*` glob sort by byte; `$(($(wc -c <file)))` strips BSD `wc` padding; when the size check fails it leaves the binaries built so far in dist/ and writes no checksums.txt.
