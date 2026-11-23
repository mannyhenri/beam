package simnet

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// stubEnv marks a child started by the harness as the stub machine.
const stubEnv = "BEAM_SIMNET_STUB"

// MaybeRunStub runs the stub machine and exits if this process was started by
// a Harness as one; otherwise it returns at once. Call it first in main (and in
// TestMain) of any program that uses the default machine command.
//
// The stub listens on BEAM_SIMNET_ADDR (any free port), prints its ready line,
// answers each connection with one JSON line {"name","pid","home"} and exits 0
// on SIGTERM or interrupt. A "dial host:port" line on its stdin makes it
// connect there from its own address and hold the connection (see Machine.Dial).
func MaybeRunStub() {
	if os.Getenv(stubEnv) != "1" {
		return
	}
	os.Exit(runStub())
}

func runStub() int {
	name := os.Getenv("BEAM_SIMNET_NAME")
	home := os.Getenv("HOME")
	ln, err := net.Listen("tcp", net.JoinHostPort(os.Getenv("BEAM_SIMNET_ADDR"), "0"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "simnet stub %s: %v\n", name, err)
		return 1
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)

	answer, _ := json.Marshal(map[string]any{"name": name, "pid": os.Getpid(), "home": home})
	answer = append(answer, '\n')
	fmt.Printf("%s%s name=%s pid=%d home=%s\n", readyPrefix, ln.Addr(), name, os.Getpid(), home)

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Write(answer)
			c.Close()
		}
	}()
	go dialCommands(name, os.Getenv("BEAM_SIMNET_ADDR"))
	sig := <-stop
	ln.Close()
	fmt.Printf("simnet stub %s: stopping on %v\n", name, sig)
	return 0
}

// dialCommands reads "dial host:port" lines from stdin, dials each from the
// machine's own address, holds the connection until the stub exits and
// answers with a dialed or dial failed line on stdout.
func dialCommands(name, addr string) {
	d := net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP(addr)}, Timeout: 5 * time.Second}
	var held []net.Conn
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		target, ok := strings.CutPrefix(sc.Text(), "dial ")
		if !ok {
			continue
		}
		c, err := d.Dial("tcp", target)
		if err != nil {
			fmt.Printf("%s%s: %v\n", dialFailedPrefix, target, err)
			continue
		}
		held = append(held, c)
		fmt.Printf("%s%s->%s (%s)\n", dialedPrefix, c.LocalAddr(), c.RemoteAddr(), name)
	}
	// Keep held reachable, so no connection is closed before the process exits.
	select {}
}
