# Build summary: ws-server

Branch `feature/ws-server`, worktree `../ws-server-work`, from main 6b8c0fa (`Gate: Phase 0 passed`). The scope brief served as the plan (no plan.md).

## What changed

- `internal/ws/upgrade.go`: `Upgrade(w, r) (*Conn, error)` with the checks in the brief's order (405 + `Allow: GET`; 400 for Upgrade/Connection tokens; 426 + `Sec-WebSocket-Version: 13`; 400 for a key that is not base64 of 16 bytes; 500 without `http.Hijacker`), the 101 written on the hijacked connection, `acceptKey` (crypto/sha1 + encoding/base64), and `Conn` (`io.ReadWriteCloser` reading through the hijacked buffered reader, plus `NetConn()`).
- `internal/ws/upgrade_test.go`: 4 test functions, 19 cases, written first (9ffe288).
- `internal/ws/doc.go`, `PROGRESS.md`: API, checks and status codes; current phase is now Phase 1.
- `acceptance/ws-server/*`: the locked tests, committed byte-identical (59b2e7b).

## Verification

- Worktree and main checkout: `go vet ./...` clean, `gofmt -l .` empty, `go list ./...` 13 packages, `CGO_ENABLED=0 GOOS=windows/linux go build ./...` ok, `bash scripts/build_all.sh` exit 0, `go test -race ./internal/ws` ok.
- Acceptance, one file per `node --test`, main checkout (main e5716c7): 23/25 pass. ws-server 2/2, no-direct-check 3/3, simnet-harness 3/3, go-project-skeleton 10/10, six-target-build-script 3/3, gate-runner 2/4.
- gate-runner AC-2 and AC-3 fail, and already failed on main 6b8c0fa before any ws-server code: they clone the repo and assume its history has no `Gate: Phase 0 passed` commit. Since the person recorded the real phase 0 gate, every clone inherits it, so AC-2's "no marker commit may exist" and AC-3's "phase 1 is refused" no longer hold. `scripts/run_gates.sh` behaves correctly (it allows phase 1 when the marker exists). These are locked tests; only the person can unlock them.

## Deviations

- None in ws-server. One invariant ("existing acceptance tests still pass") is not met, for the gate-runner reason above, which this feature neither caused nor can fix.
