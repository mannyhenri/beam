# Build summary: go-project-skeleton

Branch `feature/go-project-skeleton`. Checked 2026-11-22 with go1.27.4 darwin/amd64 and Node v23.11.0.

- **Tests:** 19 passing / 0 failing. The acceptance tests AC-1..AC-10 (`node --test acceptance/go-project-skeleton/ac-N-*.test.mjs`) contain 19 test cases. `go test ./...` exits 0; no package has Go test files yet.
- **Type check:** clean. `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go vet ./...` both exit 0 with no output.
- **Lint:** clean. `gofmt -l .` is empty; `shellcheck scripts/*.sh` reports 0 issues. No other linter is configured, and staticcheck/golangci-lint are not installed because the project avoids extra tools.
- **Coverage:** 0.0% of Go statements from unit tests (`beam/cmd/beam` is the only package with statements; the rest are doc-only). AC-7 covers the no-argument usage path in `cmd/beam/main.go:14-16` end to end. The unknown-command path (`cmd/beam/main.go:18-19`) has no test. The project sets no coverage threshold.
- **Files changed (vs. 630d46b):**
  - `go.mod`
  - `cmd/beam/main.go`
  - `internal/{identity,mux,pairing,relay,session,simnet,sysinfo,transfer,tunnel,ui,ws}/doc.go`
  - `scripts/run_gates.sh`, `scripts/build_all.sh`
  - `PROGRESS.md`, `REPORT.md`, `OUTSIDE_TESTS.md`, `SECURITY.md`
  - `acceptance/go-project-skeleton/ac-{1..10}-*.test.mjs` (locked, unchanged since lock)
  - `todos/go-project-skeleton/build-summary.md` (this file)
- **Deviations from plan:**
  - `docs/go-project-skeleton/plan.md` does not exist; the work followed the scope brief instead.
  - The implementation was committed in one stage-2 commit (`b22a286`) rather than as per-step WIP commits.
  - No Go toolchain was present, so Go 1.27.4 was installed with Homebrew; `go.mod` uses `go 1.27.4`.
  - `node --test <directory>` does not work on Node 23; run the locked commands one file at a time.
