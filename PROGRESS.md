# Beam progress

## Current phase

Phase 0: Setup. The repository skeleton is in place (feature: Zero-dependency Go project skeleton).

## Time spent

About 0.5 hours (Phase 0 skeleton), plus about 0.5 hours (HKDF and PBKDF2 vector tests).

## Go environment

`go version` output:

```
go version go1.27.4 darwin/amd64
```

Package availability, from `go list <pkg>` against the installed toolchain:

- crypto/ecdh: available
- crypto/hkdf: available
- crypto/pbkdf2: available

## Test results

Phase 0 skeleton (2026-11-22):

- `CGO_ENABLED=0 go build ./...`: ok
- `go vet ./...`: ok, no output
- Acceptance tests AC-1..AC-10 (`node --test acceptance/go-project-skeleton/ac-N-*.test.mjs`): 10/10 pass

HKDF and PBKDF2 vector tests (2026-11-23):

- `go test ./internal/session -run HKDF -v`: ok, `TestHKDFSHA256Vectors` passes RFC 5869 A.1, A.2 and A.3 (PRK from `hkdf.Extract`, OKM from `hkdf.Key`, `hkdf.Expand(PRK)` equals OKM)
- `go test ./internal/pairing -run PBKDF2 -v`: ok, `TestPBKDF2SHA256Vectors` passes both RFC 7914 section 11 vectors (c=1 and c=80000)

## Decisions that differ from the spec

- Module path is `beam`: the spec names none and the repository has no remote.
- The layout sits at the repository root rather than in a nested `beam/` folder.
- The `go` directive in go.mod is the installed toolchain's exact version (`1.27.4`).
- Placeholder `scripts/run_gates.sh` and `scripts/build_all.sh` exit 1, so a gate run cannot falsely pass before the real scripts exist.
- `beam` with an unrecognised argument prints the usage to stderr and exits 2. The spec only defines the no-argument case; exiting 0 for an unknown command would hide mistakes.
- The toolchain was not present in the sandbox; Go 1.27.4 was installed with Homebrew, and go.mod pins `go 1.27.4`, so older toolchains will refuse to build (or try to download 1.27.4 when `GOTOOLCHAIN=auto`).
- The `internal/*/doc.go` package comments describe each package's intended role; the packages themselves are empty until their features land.
- The HKDF and PBKDF2 fallbacks were not written: go1.27.4 provides crypto/hkdf and crypto/pbkdf2, and both pass the RFC 5869 and RFC 7914 vectors in `internal/session/hkdf_vectors_test.go` and `internal/pairing/pbkdf2_vectors_test.go`.

## Cuts taken

None.
