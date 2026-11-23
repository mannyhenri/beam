# Review: six-target-build-script

Branch `feature/six-target-build-script`, reviewed 2026-11-23. Reviews: security, architecture/patterns, code quality/simplicity (Performance Oracle and Codex left out by plan).

## Verdicts

- Security: clean. No injection surface, quoted paths, `rm` scoped to `dist/beam-*` and `dist/checksums.txt`, `set -e` stops before `rm` if root resolution fails.
- Architecture: 1 finding (P3).
- Quality: 2 findings (P3).

## Findings

No P1 or P2 findings. Three unique P3 findings, all resolved in this stage:

| # | Where | Finding | Found by | Resolution |
|---|---|---|---|---|
| 1 | `scripts/build_all.sh:35` | Build inherited `GOTOOLCHAIN`/`GOFLAGS` from the caller: an older Go with `GOTOOLCHAIN=auto` would download the pinned toolchain (breaks "fetches no tools"), and a stray `GOFLAGS` would alter release binaries. | architecture (security noted it below threshold) | Build now runs with `GOTOOLCHAIN=local GOFLAGS=''`, matching the acceptance tests. Verified: `GOFLAGS=-race GOTOOLCHAIN=go1.99 bash scripts/build_all.sh` exits 0. |
| 2 | `scripts/build_all.sh:20-21` | A failed build or size check left partial binaries in `dist/` that could pass for a release. | quality | `EXIT` trap removes `dist/beam-*` and `checksums.txt` when the script exits non-zero. Verified: size-limit failure and a syntax error in `cmd/beam/main.go` each exit 1 with `dist/` empty. |
| 3 | `scripts/build_all.sh:2-3` | Two concurrent runs share `dist/` and can break each other. | quality | Documented in the script header ("Run one at a time"); no locking, which would be over-engineering for a release script. |

## Checks after fixes

- `shellcheck scripts/*.sh`: clean.
- Locked tests AC-1..AC-3: 3/3 pass, file hashes unchanged.
- Skeleton acceptance tests: 10/10 pass.
- `scripts/run_gates.sh`, `acceptance/go-project-skeleton`, `go.mod`: unchanged from main.
