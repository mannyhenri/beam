# Build summary: ws-server

Local `main` c33595f (fast-forwarded from `feature/ws-server`, since deleted). Checked 2026-11-23 with go1.27.4 darwin/amd64 and Node v23.11.0, loopback aliases 127.0.0.2-.4 present, phase 0 gate recorded (6b8c0fa).

- **Tests:** `go test -race ./internal/ws`: 4 functions, 19 cases, all pass. Acceptance, one file per `node --test`: 25 files, 23 pass, 2 fail. ws-server AC-1 and AC-2 pass. The 2 failures are gate-runner AC-2 and AC-3, which already failed on main 6b8c0fa before any ws-server code: they assume the cloned history has no real `Gate: Phase 0 passed` commit. Locked; the person must unlock them.
- **Type check:** clean. `go vet ./...` exits 0; `CGO_ENABLED=0 GOOS=windows` and `GOOS=linux go build ./...` ok.
- **Lint:** clean. `gofmt -l .` is empty; no staticcheck or golangci-lint installed (standard library only project).
- **Coverage:** 91.9% of statements in `internal/ws`. Every function is at 100% except `Upgrade` (87.5%): the two untested lines are the `Hijack` error return (`internal/ws/upgrade.go:73`) and the failed-flush close (`internal/ws/upgrade.go:81-82`), both I/O failures after the checks pass. No project threshold.
- **Files changed (vs. main 6b8c0fa):**
  - `internal/ws/upgrade.go` (new)
  - `internal/ws/upgrade_test.go` (new)
  - `internal/ws/doc.go`
  - `PROGRESS.md`
  - `docs/ws-server/build-summary.md`
  - `acceptance/ws-server/*` (locked, unchanged since lock)
  - `todos/ws-server/build-summary.md` (this file)
- **Deviations from plan:**
  - `docs/ws-server/plan.md` does not exist; the scope brief served as the plan.
  - A missing `Sec-WebSocket-Version` gets 426 like a wrong one (the brief says "is 13, else 426").
  - The invariant "existing acceptance tests still pass" is not met for gate-runner AC-2/AC-3, for the reason above; this feature neither caused nor can fix it.
