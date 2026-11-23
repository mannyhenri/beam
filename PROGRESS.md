# Beam progress

## Current phase

Phase 0: Setup. The repository skeleton is in place (feature: Zero-dependency Go project skeleton), and `scripts/build_all.sh` builds all six release binaries (feature: Six-target build script with checksums). `internal/simnet` starts and stops three fake machines (feature: Simnet fake-machine harness).

## Time spent

About 0.5 hours (Phase 0 skeleton); about 0.25 hours (six-target build script); about 0.5 hours (simnet harness).

## Go environment

`go version` output:

```
go version go1.27.4 darwin/amd64
```

Package availability, from `go list <pkg>` against the installed toolchain:

- crypto/ecdh: available
- crypto/hkdf: available
- crypto/pbkdf2: available

## Test results

Phase 0 skeleton (2026-11-22):

- `CGO_ENABLED=0 go build ./...`: ok
- `go vet ./...`: ok, no output
- Acceptance tests AC-1..AC-10 (`node --test acceptance/go-project-skeleton/ac-N-*.test.mjs`): 10/10 pass

Six-target build script (2026-11-23):

- `bash scripts/build_all.sh`: ok, six binaries of 1.5-1.7 MB each, `dist/checksums.txt` verifies with `sha256sum -c` and `shasum -a 256 -c`
- Acceptance tests AC-1..AC-3 (`node --test acceptance/six-target-build-script/ac-N-*.test.mjs`): 3/3 pass

Simnet harness (2026-11-23):

- `go test -race ./internal/simnet`: ok (8 pass, 1 skipped: address order needs the 127.0.0.2-.4 aliases)
- Acceptance AC-3 (missing alias error): pass. AC-1 and AC-2: fail on this Mac until the loopback aliases are added; the failure is the harness's own message with the `sudo ifconfig lo0 alias 127.0.0.2 up` fix

## Decisions that differ from the spec

- Module path is `beam`: the spec names none and the repository has no remote.
- The layout sits at the repository root rather than in a nested `beam/` folder.
- The `go` directive in go.mod is the installed toolchain's exact version (`1.27.4`).
- Placeholder `scripts/run_gates.sh` and `scripts/build_all.sh` exit 1, so a gate run cannot falsely pass before the real scripts exist.
- `beam` with an unrecognised argument prints the usage to stderr and exits 2. The spec only defines the no-argument case; exiting 0 for an unknown command would hide mistakes.
- The toolchain was not present in the sandbox; Go 1.27.4 was installed with Homebrew, and go.mod pins `go 1.27.4`, so older toolchains will refuse to build (or try to download 1.27.4 when `GOTOOLCHAIN=auto`).
- The `internal/*/doc.go` package comments describe each package's intended role; the packages themselves are empty until their features land.
- `scripts/build_all.sh` writes to `dist/` at the repository root (git-ignored): `beam-<goos>-<goarch>` for linux and darwin, `beam-<goos>-<goarch>.exe` for windows, on amd64 and arm64, plus `dist/checksums.txt` in `sha256sum` format with bare filenames sorted by name. The spec names no output location or file names.
- Binaries are built with `-trimpath -ldflags='-s -w'` (stripped symbols and paths) to stay well under the 15 MB target; the script fails if any binary reaches 15,728,640 bytes.
- `sha256sum` is used when present, otherwise `shasum -a 256` (stock macOS has no `sha256sum`); the script stops before building if neither exists.
- `scripts/build_all.sh` runs `go build` with `GOTOOLCHAIN=local GOFLAGS=''` so the caller's environment cannot fetch a toolchain or change the release flags, and removes `dist/beam-*` and `dist/checksums.txt` if any step fails, so a failed run leaves no partial release.
- The placeholder note above no longer applies to `scripts/build_all.sh`; `scripts/run_gates.sh` is still a placeholder.
- Simnet on macOS needs loopback aliases for 127.0.0.2-.4, added once by the person with `sudo ifconfig lo0 alias 127.0.0.N up` and again after every reboot (Linux binds all of 127.0.0.0/8). `Harness.Start` and `Harness.Preflight` try to bind each address first and fail with that command; the harness never runs sudo or ifconfig.
- Simnet layout: `<dir>/<name>/home` is the machine's HOME and `<dir>/<name>/machine.log` collects its stdout and stderr (appended). The child gets `HOME`, `XDG_CONFIG_HOME=<home>/.config`, `USERPROFILE=<home>`, `APPDATA=<home>/AppData/Roaming`, `LOCALAPPDATA=<home>/AppData/Local`, `BEAM_SIMNET_NAME` and `BEAM_SIMNET_ADDR`. At most three machines, named `[A-Za-z0-9-]+`, get 127.0.0.2, .3, .4 in start order.
- Until beam has a long-running command, each machine runs a stub (`simnet.MaybeRunStub`, re-executing the current binary with `BEAM_SIMNET_STUB=1`). It listens on its address with a free port, prints `simnet: ready <addr:port> name=… pid=… home=…`, answers each connection with one JSON line `{"name","pid","home"}`, and exits 0 on SIGTERM. `Config.Command` runs another program instead.
- Stopping sends SIGTERM, waits up to 5 s per machine, then kills; a machine counts as clean only if it exited 0 without the kill. On Windows there is no SIGTERM, so machines are killed and reported as not clean; the harness compiles there but its tests run on macOS and Linux only.
- Test driver `go run ./internal/simnet/cmd/simnet --dir <D> --names A,B,C [--addrs …]` (not part of the release build): prints one line per machine `{"name","pid","addr","home","log"}` where `addr` is `ip:port`, waits for stdin to close or SIGTERM/interrupt, stops all, prints `{"stopped":[{"name","pid","clean"}]}` and exits 0 only if all were clean. It exits 1 before starting anything if an address cannot be bound, and 2 on missing flags.
- The driver adds a 13th package, `beam/internal/simnet/cmd/simnet`. The person unlocked go-project-skeleton AC-4 (2026-11-23) to add it to the expected `go list ./...` set, which was exactly 12 packages before; the locked simnet driver builds that path, so the two locked tests could not both pass otherwise.

## Cuts taken

None.
