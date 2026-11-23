// Package simnet simulates several machines inside one sandbox for testing.
//
// A Harness starts up to three named fake machines as child processes. In
// start order they get the loopback addresses 127.0.0.2, 127.0.0.3 and
// 127.0.0.4 (DefaultAddrs). Each machine lives under the harness directory:
//
//	<dir>/<name>/home         the child's HOME (also XDG_CONFIG_HOME=<home>/.config,
//	                          USERPROFILE and APPDATA for Windows)
//	<dir>/<name>/machine.log  the child's stdout and stderr, appended
//
// The child also gets BEAM_SIMNET_NAME and BEAM_SIMNET_ADDR. By default it
// runs the stub machine (see MaybeRunStub) from the current executable;
// Config.Command runs something else.
//
// macOS only configures 127.0.0.1 on lo0. Start and Preflight check that each
// address can be bound and, if not, fail with the fix to run once (and after
// every reboot): sudo ifconfig lo0 alias 127.0.0.N up. The harness never runs
// sudo or ifconfig itself.
//
// StopAll sends SIGTERM (Kill on Windows), waits up to Config.StopTimeout for
// each machine and force-kills any that remain; a forced kill is not clean.
//
// Machines must never connect to each other directly. CheckNoDirect lists
// each machine's TCP sockets (lsof on macOS, /proc/net/tcp and tcp6 on Linux;
// not supported on Windows) and returns a *DirectError naming every connection
// whose remote address is another machine's. While machines run, a poller
// repeats the check every Config.Poll (200 ms) and Seen keeps what it found,
// even after StopAll; a connection that opens and closes between two polls is
// missed. The harness itself never dials a machine: only a stub told to with
// Machine.Dial does. StartRelay gives a 127.0.0.1 stand-in for the relay.
//
// The driver in cmd/simnet wraps a Harness for the acceptance tests.
package simnet
