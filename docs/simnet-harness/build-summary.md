# Build summary: simnet-harness

Branch `feature/simnet-harness`, worktree `../simnet-harness-work`. The scope brief served as the plan (no plan.md; there is no planning stage).

## What changed

- `internal/simnet/simnet.go`: `Config{Dir, Addrs, Command, StopTimeout}`, `New`, `Harness.Start`, `Preflight`, `Machines`, `StopAll` → `[]StopResult{Name, Pid, Clean}`, `DefaultAddrs` (127.0.0.2, .3, .4). At most three machines, names `[A-Za-z0-9-]+` and unique, addresses in start order. Each machine gets `<dir>/<name>/home` as HOME and appends stdout and stderr to `<dir>/<name>/machine.log`. The child env sets `HOME`, `XDG_CONFIG_HOME`, `USERPROFILE`, `APPDATA`, `LOCALAPPDATA`, `BEAM_SIMNET_NAME`, `BEAM_SIMNET_ADDR`. Start binds each needed address first; if one fails it starts nothing and returns the `sudo ifconfig lo0 alias 127.0.0.N up` fix. It never runs sudo or ifconfig.
- `internal/simnet/stub.go`: `MaybeRunStub`, the default machine program (re-exec with `BEAM_SIMNET_STUB=1`). It listens on its own address only, logs a ready line, answers each connection with `{"name","pid","home"}`, and exits 0 on SIGTERM.
- `internal/simnet/terminate_{unix,windows}.go`: SIGTERM, then kill after 5 s; on Windows, kill only (reported as not clean).
- `internal/simnet/cmd/simnet/main.go`: the driver used by the locked tests. `--dir --names [--addrs]`; one JSON line per machine `{"name","pid","addr","home","log"}`, waits for stdin EOF or SIGTERM, prints `{"stopped":[…]}`, exits 0 only if all stopped cleanly, 1 on an unbindable address, 2 on missing flags. Not built by `scripts/build_all.sh`.
- `internal/simnet/simnet_test.go`: 9 unit tests, written first (87476a2).
- `acceptance/go-project-skeleton/ac-4-go-list-packages.test.mjs`: expected set now includes `beam/internal/simnet/cmd/simnet` (13 packages). Unlocked by the person (a562d24).
- `PROGRESS.md`: alias requirement, dir layout, env vars, stub, stop rules, driver JSON, AC-4 change.
- `acceptance/simnet-harness/*`: the locked tests, committed byte-identical (a8c587b).

## Verification (worktree, `GOTOOLCHAIN=local GOFLAGS=''`)

- `go vet ./...`: clean. `GOOS=linux go vet ./internal/simnet/...`: clean. `CGO_ENABLED=0 GOOS=windows go build ./internal/simnet/...`: ok.
- `go test -race ./internal/simnet -count=1`: ok, 9/9 pass (with the aliases; without them `TestStartAssignsAddressesInOrder` skips with the sudo fix).
- `bash scripts/build_all.sh`: exit 0.
- Skeleton acceptance tests: 10/10 pass. Six-target build script acceptance tests: 3/3 pass.
- Locked simnet tests, one file per `node --test`: 3/3 pass after the person added the 127.0.0.2-.4 aliases. Before that, AC-1 and AC-2 failed only with the harness's own message, `cannot bind loopback address 127.0.0.N … sudo ifconfig lo0 alias 127.0.0.N up`.
- Main checkout: local `main` fast-forwarded to this branch; locked tests rerun there (see below).

## Deviations

- Skeleton AC-4 changed (by the person): the locked simnet driver builds `./internal/simnet/cmd/simnet`, a 13th package, so the old set of exactly 12 could never pass alongside it.
