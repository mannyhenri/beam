# Build summary: no-direct-check

Local `main` 152a7b1 (fast-forwarded from `feature/no-direct-check`, since deleted). Checked 2026-11-23 with go1.27.4 darwin/amd64 and Node v23.11.0, loopback aliases 127.0.0.2-.4 present.

- **Tests:** 52 passing / 0 failing. `go test -race ./internal/simnet`: 23/23 (14 new). Acceptance, one file per `node --test`: 29 cases in 19 files (no-direct-check 4 cases in 3 files, simnet-harness 3, go-project-skeleton 10 files, six-target build script 3 files).
- **Type check:** clean. `go vet ./...` exits 0; `CGO_ENABLED=0 GOOS=windows` and `GOOS=linux go build ./internal/simnet/...` ok.
- **Lint:** clean. `gofmt -l .` is empty (including `acceptance/no-direct-check/testdata`); no staticcheck or golangci-lint installed (standard library only project).
- **Coverage:** 86.6% of statements in `internal/simnet` on macOS (was 85.0%). New code: parsers 83-92%, `FindDirect` 87%, `scan` 88%, `poll` and `Seen` 100%, `Machine.Dial` 67% (untested: dial on a custom-command machine, the machine exiting mid-dial, the 10 s timeout). `sockets_linux.go` is not compiled on macOS, so only its parser (`ParseProcNetTCP`, shared) is covered here; the `/proc/<pid>/fd` inode matching runs only on Linux. No project threshold.
- **Files changed (vs. main 4dc3f78):**
  - `internal/simnet/{sockets.go,sockets_darwin.go,sockets_linux.go,sockets_other.go,relay.go}` (new)
  - `internal/simnet/{simnet.go,stub.go,doc.go}`
  - `internal/simnet/cmd/simnet/main.go`
  - `internal/simnet/{sockets_parse_test.go,nodirect_test.go}` (new)
  - `PROGRESS.md`, `docs/no-direct-check/build-summary.md`
  - `acceptance/no-direct-check/*` (locked, unchanged since lock)
  - `todos/no-direct-check/build-summary.md` (this file)
- **Deviations from plan:**
  - `docs/no-direct-check/plan.md` does not exist; the scope brief served as the plan.
  - `StopAll` runs one last scan before stopping (about 0.2 s on macOS), so a direct connection open at stop time is recorded even if no poll saw it.
  - `Socket` has an extra `Pid` field (0 from `/proc/net/tcp`).
