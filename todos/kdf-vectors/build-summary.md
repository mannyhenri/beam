# Build summary: kdf-vectors

Branch `feature/kdf-vectors` at `010f3ac`, based on `main` at `10e7152` (PR #1, go-project-skeleton, was merged before the build stage, so `origin/main` was used rather than `origin/feature/go-project-skeleton`). Checked 2026-11-23 with go1.27.4 darwin/amd64 and Node v23.11.0.

- **Tests:** 50 passing / 0 failing across the 21 locked acceptance files: kdf-vectors AC-1..AC-11 (`node --test acceptance/kdf-vectors/ac-N-*.test.mjs`) and go-project-skeleton AC-1..AC-10. `go test ./... -v` exits 0: `TestHKDFSHA256Vectors` (subtests A.1, A.2, A.3) in `beam/internal/session` and `TestPBKDF2SHA256Vectors` (subtests `passwd` for c=1, `Password` for c=80000) in `beam/internal/pairing` pass; the other 10 packages have no test files.
- **Type check:** clean. `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go vet ./...` exit 0 with no output. `go list ./...` lists the 12 skeleton packages.
- **Lint:** clean. `gofmt -l .` is empty; `shellcheck scripts/*.sh` reports 0 issues. staticcheck and golangci-lint are not installed (the project adds no extra tools).
- **Coverage:** not applicable to this change. It adds only test files, and `go test -cover` reports `[no statements]` for `internal/session` and `internal/pairing`; `beam/cmd/beam` stays at 0.0%, unchanged from the skeleton. The project sets no coverage threshold.
- **Vector provenance:** the expected bytes were copied from the RFC texts fetched from rfc-editor.org (`rfc5869.txt` Appendix A, `rfc7914.txt` section 11). Each test keeps the RFC's line breaks (PBKDF2 also keeps its space-separated octets), with a comment citing the section and test case. A script confirmed that all 45 hex lines in the two test files appear exactly in the RFC text.
- **Files changed (vs. 10e7152):**
  - `internal/session/hkdf_vectors_test.go` (new, package session)
  - `internal/pairing/pbkdf2_vectors_test.go` (new, package pairing)
  - `PROGRESS.md`: Decisions line (no fallback; crypto/hkdf and crypto/pbkdf2 are provided by go1.27.4), Test results entry, Time spent
  - `acceptance/kdf-vectors/_helpers.mjs`, `acceptance/kdf-vectors/ac-{1..11}-*.test.mjs` (locked; unchanged since `4374c32`)
  - `todos/kdf-vectors/build-summary.md` (this file)
- **Commits:** `4374c32` locked acceptance tests; `e1e8e3a` vector tests and PROGRESS.md; `eaa114f` build summary; `010f3ac` PBKDF2 test simplified to one subtest per vector (no behavior change).
- **Deviations from plan:**
  - `docs/kdf-vectors/plan.md` does not exist; the work followed the stage's scope brief.
  - The stage text says to add a separate `../kdf-vectors-work` worktree. The work stayed in the existing `feature/kdf-vectors` worktree the pipeline created, as the "Your branch" section asks.
  - The vector tests and PROGRESS.md were committed in one commit (`e1e8e3a`) rather than as per-step WIP commits; only the later simplification uses the WIP format.
  - No fallback code was written (by the spec's rule, since both packages exist), and no non-test `.go` file was added.
- **For the review stage:** the acceptance tests copy the module to a temp folder and corrupt the expected values and the `hkdf.Extract`/`Key`/`Expand` outputs, so a passing run means every vector is compared in full. The RFC texts used are not committed; re-fetch them from rfc-editor.org to re-check the literals.
