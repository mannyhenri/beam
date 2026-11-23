# Beam progress

## Current phase

Phase 1: Transport and relay. The phase 0 gate passed (`Gate: Phase 0 passed`, 6b8c0fa). `internal/ws` upgrades WebSocket requests on the server side (feature: WebSocket server).

Phase 0: Setup. The repository skeleton is in place (feature: Zero-dependency Go project skeleton), and `scripts/build_all.sh` builds all six release binaries (feature: Six-target build script with checksums). `internal/simnet` starts and stops three fake machines (feature: Simnet fake-machine harness) and fails when one connects directly to another (feature: No-direct-connection socket check). `scripts/run_gates.sh 0` runs the phase 0 gate (feature: Per-phase gate runner); the first real phase 0 gate has not been run yet.

## Time spent

About 0.5 hours (Phase 0 skeleton); about 0.25 hours (six-target build script); about 0.5 hours (simnet harness); about 0.75 hours (no-direct-connection check); about 0.5 hours (gate runner); about 0.25 hours (WebSocket server).

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

- `go test -race ./internal/simnet`: ok, 9/9 pass with the 127.0.0.2-.4 aliases in place (without them the address-order test skips with the sudo fix)
- Acceptance AC-1..AC-3 (`node --test acceptance/simnet-harness/ac-N-*.test.mjs`): 3/3 pass once the aliases are added. Without them AC-1 and AC-2 fail with the harness's own `sudo ifconfig lo0 alias 127.0.0.N up` message

No-direct-connection check (2026-11-23):

- `go test -race ./internal/simnet`: ok, 23/23 pass (14 new: parsers, FindDirect, live check, relay stand-in, dial, poller)
- Acceptance AC-1..AC-3 (`node --test acceptance/no-direct-check/ac-N-*.test.mjs`): 3/3 pass with the aliases in place; simnet-harness 3/3 still pass

Gate runner (2026-11-23):

- `shellcheck scripts/run_gates.sh`: clean
- Acceptance AC-1..AC-4 (`node --test acceptance/gate-runner/ac-N-*.test.mjs`): 4/4 pass; they run the script only in throwaway clones. In a clone, a full phase 0 gate takes about 30 s; SIGTERM mid-gate exits 143 and leaves no worktree

WebSocket server (2026-11-23):

- `go test -race ./internal/ws`: ok (accept key, 13 refusal cases, 4 handover variants, close)
- Acceptance AC-1, AC-2 (`node --test acceptance/ws-server/ac-N-*.test.mjs`): 2/2 pass

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
- The placeholder note above no longer applies: both `scripts/build_all.sh` and `scripts/run_gates.sh` are real.
- Simnet on macOS needs loopback aliases for 127.0.0.2-.4, added once by the person with `sudo ifconfig lo0 alias 127.0.0.N up` and again after every reboot (Linux binds all of 127.0.0.0/8). `Harness.Start` and `Harness.Preflight` try to bind each address first and fail with that command; the harness never runs sudo or ifconfig.
- Simnet layout: `<dir>/<name>/home` is the machine's HOME and `<dir>/<name>/machine.log` collects its stdout and stderr (appended). The child gets `HOME`, `XDG_CONFIG_HOME=<home>/.config`, `USERPROFILE=<home>`, `APPDATA=<home>/AppData/Roaming`, `LOCALAPPDATA=<home>/AppData/Local`, `BEAM_SIMNET_NAME` and `BEAM_SIMNET_ADDR`. At most three machines, named `[A-Za-z0-9-]+`, get 127.0.0.2, .3, .4 in start order.
- Until beam has a long-running command, each machine runs a stub (`simnet.MaybeRunStub`, re-executing the current binary with `BEAM_SIMNET_STUB=1`). It listens on its address with a free port, prints `simnet: ready <addr:port> name=… pid=… home=…`, answers each connection with one JSON line `{"name","pid","home"}`, and exits 0 on SIGTERM. `Config.Command` runs another program instead.
- Stopping sends SIGTERM, waits up to 5 s per machine, then kills; a machine counts as clean only if it exited 0 without the kill. On Windows there is no SIGTERM, so machines are killed and reported as not clean; the harness compiles there but its tests run on macOS and Linux only.
- Test driver `go run ./internal/simnet/cmd/simnet --dir <D> --names A,B,C [--addrs …]` (not part of the release build): prints one line per machine `{"name","pid","addr","home","log"}` where `addr` is `ip:port`, waits for stdin to close or SIGTERM/interrupt, stops all, prints `{"stopped":[{"name","pid","clean"}]}` and exits 0 only if all were clean. It exits 1 before starting anything if an address cannot be bound, and 2 on missing flags.
- The driver adds a 13th package, `beam/internal/simnet/cmd/simnet`. The person unlocked go-project-skeleton AC-4 (2026-11-23) to add it to the expected `go list ./...` set, which was exactly 12 packages before; the locked simnet driver builds that path, so the two locked tests could not both pass otherwise.
- No-direct check: a violation is any TCP socket owned by a machine's process (not its children) whose remote IP is another machine's address, on any port and in any state but LISTEN (SYN_SENT and CLOSE_WAIT count). A machine's connection to itself, the relay, and sockets of other processes (such as the test reading a stub) are not violations.
- Socket listing: macOS runs `/usr/sbin/lsof -nP -a -p <all machine pids> -iTCP -FpTn` once per check (about 0.1-0.2 s); Linux reads `/proc/net/tcp` and `/proc/net/tcp6` and matches inodes from `/proc/<pid>/fd`; other systems return "not supported". The first `Start` fails if lsof (macOS) or `/proc/net/tcp` (Linux) is missing.
- The poller runs every 200 ms (`Config.Poll`; negative turns it off) and `StopAll` scans once more before stopping, which adds about 0.2 s per `StopAll` on macOS. A connection that opens and closes between two scans is not caught.
- Stub dial command: the harness writes `dial host:port` to the stub's stdin; the stub dials from its own address, holds the connection until it exits and answers `simnet: dialed …` or `simnet: dial failed …` on stdout, which `Machine.Dial` waits for (10 s limit).
- Driver flags `--relay` (prints `{"relay":"127.0.0.1:P"}` before the machine lines) and `--dial FROM=TO,…` (TO a machine or `relay`). With either flag it prints `{"check":{"ok":…,"direct":[{"from","to","local","remote"}]}}` after the dials and exits 3 if any direct connection was seen during the run. A bad `--dial` exits 2 before anything starts. Without the flags, output and exit codes are unchanged.
- Gates: `scripts/run_gates.sh <phase 0-7>`. Exit 0 = gate passed and marked; 1 = a check failed (no commit); 2 = usage error, missing tool (git, go, node, bash), missing previous gate, or phase without a gate yet. Phase N ≥ 1 needs a commit whose subject is exactly `Gate: Phase <N-1> passed` (exact match, because other commits mention "Phase 3" etc.).
- A gate runs in a temporary `git worktree` of HEAD, so only committed files count and the working tree (including untracked locked-test copies) is never touched; an EXIT trap removes it, also on Ctrl-C or SIGTERM.
- On pass the script makes one empty commit `Gate: Phase <N> passed` (body: `Validated <sha>.`) on the current branch, only if HEAD did not move during the run. It never pushes.
- Phase 0 gate: `bash scripts/build_all.sh`, then every committed `*.test.mjs` under acceptance/go-project-skeleton, six-target-build-script, simnet-harness and no-direct-check, one file per `node --test`, in C-locale order; the first failure stops it. The gate runner's own tests are not part of it. It needs the 127.0.0.2-.4 aliases. Phases 1-7 exit 2 with "gate for phase N is not defined yet" until their features add a section.

- `ws.Upgrade(w, r) (*ws.Conn, error)` checks, in order: method GET (else 405 with `Allow: GET`); token `websocket` in Upgrade and `Upgrade` in Connection, comma lists, any case (else 400); `Sec-WebSocket-Version: 13` (else 426 with `Sec-WebSocket-Version: 13`, also when missing); `Sec-WebSocket-Key` base64 of exactly 16 bytes (else 400); a writer without `http.Hijacker` gets 500. Refusals go through the ResponseWriter and nothing is hijacked. On success it writes the 101 (Upgrade, Connection, Sec-WebSocket-Accept only) on the hijacked connection.
- `*ws.Conn` is an `io.ReadWriteCloser` over the hijacked connection; reads go through net/http's buffered reader so bytes sent with the request are kept; `NetConn()` exposes the `net.Conn`. The caller closes it. No origin check, subprotocols or extensions; frames are the framing feature's job.

## Cuts taken

None.
