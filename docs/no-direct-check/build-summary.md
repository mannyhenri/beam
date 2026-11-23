# Build summary: no-direct-check

Branch `feature/no-direct-check`, worktree `../no-direct-check-work`, from main 4dc3f78. The scope brief served as the plan (no plan.md; there is no planning stage).

## What changed

- `internal/simnet/sockets.go`: `Socket`, `Direct`, `DirectError`, `FindDirect` (machine-owned, non-LISTEN sockets whose remote IP is another machine's address; sorted), and the pure parsers `ParseProcNetTCP` (tcp and tcp6, little-endian hex, v4-mapped addresses unmapped) and `ParseLsof` (`-FpTn` fields in any order, `*:port` listeners).
- `internal/simnet/sockets_{darwin,linux,other}.go`: one lsof call for all machine pids on macOS; `/proc/net/tcp{,6}` plus `/proc/<pid>/fd` inodes on Linux; "not supported" elsewhere. `socketToolReady` makes the first `Start` fail if the tool is missing.
- `internal/simnet/simnet.go`: `Config.Poll` (200 ms default, negative off), poller started with the first machine, `CheckNoDirect`, `Seen`, `Machine.Dial`; `StopAll` stops the poller and scans once more before stopping.
- `internal/simnet/stub.go`: `dial host:port` on stdin → dial from the machine's own address, hold the connection, answer `simnet: dialed …` / `simnet: dial failed …`.
- `internal/simnet/relay.go`: `StartRelay` — 127.0.0.1 listener that accepts and holds connections.
- `internal/simnet/cmd/simnet/main.go`: `--relay`, `--dial FROM=TO,…` (validated before anything starts, exit 2), `{"relay":…}` line before the machines, `{"check":…}` line after the dials, exit 3 if any direct connection was seen. Unchanged without these flags.
- `internal/simnet/{sockets_parse_test.go,nodirect_test.go}`: 14 unit tests, written first (89b343e).
- `internal/simnet/doc.go`, `PROGRESS.md`: the check, per-OS method, poll interval and its limit, driver flags, JSON and exit 3.
- `acceptance/no-direct-check/*`: the locked tests, committed byte-identical (e6836b3).

## Verification (worktree, `GOTOOLCHAIN=local GOFLAGS=''`)

- `go vet ./...` clean; `gofmt -l .` empty; `CGO_ENABLED=0 GOOS=windows` and `GOOS=linux go build ./internal/simnet/...` ok.
- `go test -race ./internal/simnet -count=1`: ok, 23/23.
- `bash scripts/build_all.sh`: exit 0.
- Acceptance, one file per `node --test`: 19/19 pass (no-direct-check 3, simnet-harness 3, go-project-skeleton 10, six-target-build-script 3).
- Bad `--dial` values (`A=relay` without `--relay`, unknown FROM or TO, missing `=`) exit 2 before any machine starts.

## Deviations

- None from the brief's behaviour. API names were fixed by the tests: `ParseProcNetTCP`/`Socket` by the locked AC-3, the rest by the unit tests. `Socket` has an extra `Pid` field (0 from `/proc/net/tcp`).
- `StopAll` now runs one last scan, adding about 0.2 s per stop on macOS, so a direct connection still open at stop time is recorded even if no poll saw it.
