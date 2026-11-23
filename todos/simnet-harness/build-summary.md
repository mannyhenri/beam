# Build summary: simnet-harness

Local `main` 463c49b (fast-forwarded from `feature/simnet-harness`). Checked 2026-11-23 with go1.27.4 darwin/amd64 and Node v23.11.0, loopback aliases 127.0.0.2-.4 present.

- **Tests:** 34 passing / 0 failing. `go test ./internal/simnet`: 9/9 (also under `-race`). Acceptance, one file per `node --test`: 25 cases in 16 files, all pass (simnet AC-1..AC-3, skeleton 10 files, six-target build script 3 files).
- **Type check:** clean. `go build ./...` and `go vet ./...` exit 0; `CGO_ENABLED=0 GOOS=windows go build ./internal/simnet/...` ok.
- **Lint:** clean. `gofmt -l .` is empty; `go vet` clean. No staticcheck or golangci-lint installed (standard library only project).
- **Coverage:** 85.0% of statements in `internal/simnet` (`go test -cover`). Uncovered lines are mostly error returns for I/O failures (mkdir, log open, pipe, exec start) in `simnet.go:142-231` and the Windows-only path. `internal/simnet/cmd/simnet` has no unit tests; the locked acceptance tests drive it end to end. The project sets no coverage threshold.
- **Files changed (vs. main 8c0e9af):**
  - `internal/simnet/{doc.go,simnet.go,stub.go,simnet_test.go,terminate_unix.go,terminate_windows.go}`
  - `internal/simnet/cmd/simnet/main.go`
  - `PROGRESS.md`
  - `docs/simnet-harness/build-summary.md`
  - `acceptance/simnet-harness/{ac-1,ac-2,ac-3,driver}` (locked, unchanged since lock)
  - `acceptance/go-project-skeleton/ac-4-go-list-packages.test.mjs` (unlocked by the person)
  - `todos/simnet-harness/build-summary.md` (this file)
- **Deviations from plan:**
  - `docs/simnet-harness/plan.md` does not exist; the scope brief served as the plan.
  - Skeleton AC-4 now expects 13 packages including `beam/internal/simnet/cmd/simnet`; the locked simnet driver needs that package.
  - On macOS the aliases must be re-added after every reboot before running simnet AC-1/AC-2.
