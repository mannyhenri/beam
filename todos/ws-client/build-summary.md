# Build summary: ws-client

Local `main` 62b66d6 (fast-forwarded from `feature/ws-client`, since deleted). Checked 2026-11-24 with go1.27.4 darwin/amd64 and Node v23.11.0.

- **Tests:** `go test -race ./internal/ws`: 10 test functions pass (6 new client, 4 server). Acceptance, one file per `node --test`: 28 files, 26 pass, 2 fail. ws-client AC-1..AC-3 and ws-server AC-1..AC-2 pass. The 2 failures are gate-runner AC-2 and AC-3, which fail the same way on main 45d4466 before this change (they assume no real `Gate: Phase 0 passed` commit). Locked; the person must unlock them.
- **Type check:** clean. `go vet ./...` exits 0; `CGO_ENABLED=0 GOOS=windows` and `GOOS=linux go build ./...` ok.
- **Lint:** clean. `gofmt -l .` is empty; no staticcheck or golangci-lint installed (standard library only project).
- **Coverage:** 90.2% of statements in `internal/ws` (unit tests only). `Dial` 86.3%, `WriteFrame` 95.8%, `Upgrade` 87.5%, the rest 100%. Not reached by the unit tests: the `DialConfig.TLS` branch (`internal/ws/client.go:77`; covered by locked AC-1, which runs wss:// in a child process), a 101 lacking `Connection: Upgrade` (`client.go:95`), a 101 body that is not a ReadWriteCloser (`client.go:102`), and error returns for URL parsing, request building, the HTTP round trip and `crypto/rand` (`client.go:39, 45, 51, 81`; `frame.go:40`). No project threshold.
- **Files changed (vs. main 45d4466):**
  - `internal/ws/client.go` (new)
  - `internal/ws/frame.go` (new)
  - `internal/ws/client_test.go` (new)
  - `internal/ws/upgrade.go` (Conn generalised to reader/writer/closer + client flag)
  - `internal/ws/doc.go`
  - `PROGRESS.md`
  - `docs/ws-client/build-summary.md`
  - `acceptance/ws-client/*` (locked, unchanged since lock)
  - `todos/ws-client/build-summary.md` (this file)
- **Deviations from plan:**
  - `docs/ws-client/plan.md` does not exist; the scope brief served as the plan.
  - Additions: Dial refuses schemes other than ws:// and wss://; a client Conn's `NetConn()` is the TCP connection Dial opened (to the relay, or to the proxy).
  - The invariant "existing acceptance tests still pass" is not met for gate-runner AC-2/AC-3, for the pre-existing reason above.
