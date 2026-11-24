# Build summary: ws-client

Branch `feature/ws-client`, worktree `../ws-client-work`, from main 45d4466. The scope brief served as the plan (no plan.md).

## What changed

- `internal/ws/client.go` (new): `Dial(ctx, url, *DialConfig)` for ws:// and wss:// only; the upgrade goes through an `http.Client` whose Transport uses `http.ProxyFromEnvironment`; 16-byte `crypto/rand` key; the 101 answer must carry `Upgrade: websocket`, `Connection: Upgrade` and the matching `Sec-WebSocket-Accept`, else Dial closes and returns an error naming the status or header. A dial hook records the TCP connection so `NetConn()` works on client Conns too.
- `internal/ws/frame.go` (new): `OpText`, `OpBinary`, `Conn.WriteFrame`: FIN + opcode, 7/16/64-bit length forms, client frames masked with a fresh key per frame (on a copy), server frames unmasked, one Write under a mutex.
- `internal/ws/upgrade.go`: `Conn` now holds a reader, writer, closer and a client flag so one type serves both sides; server behaviour unchanged.
- `internal/ws/client_test.go` (new): 6 test functions, 11 cases, written first (2ff834c).
- `internal/ws/doc.go`, `PROGRESS.md`: client side, proxy variables, the ws:// via HTTP_PROXY limit, masking rule.
- `acceptance/ws-client/*`: the locked tests, committed byte-identical (fca4e99).

## Verification

- Worktree and main checkout: `go vet ./...` clean, `gofmt -l .` empty, `go list ./...` 13 packages, `CGO_ENABLED=0 GOOS=windows/linux go build ./...` ok, `bash scripts/build_all.sh` exit 0, `go test -race ./internal/...` ok.
- Acceptance, one file per `node --test`, main checkout (main 5312c14): 26/28 pass. ws-client 3/3, ws-server 2/2, no-direct-check 3/3, simnet-harness 3/3, go-project-skeleton 10/10, six-target-build-script 3/3, gate-runner 2/4.
- gate-runner AC-2 and AC-3 fail exactly as on main 45d4466 before this change (they assume no real `Gate: Phase 0 passed` commit exists). This change touches neither `scripts/` nor `acceptance/gate-runner/`. Locked; the person must unlock them.

## Deviations

- None in behaviour. Additions beyond the brief: Dial refuses non-ws/wss schemes, and `NetConn()` on a client Conn is the TCP connection Dial opened (to the relay, or to the proxy).
