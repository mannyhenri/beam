// Command beam securely connects the machines you control through a relay
// that only ever sees sealed data.
package main

import (
	"fmt"
	"os"
)

const usage = "usage: beam <command> [arguments]\n" +
	"commands: relay, open, join, peers, send, recv, pipe, allow, forget, sysinfo, tunnel, ui"

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		return
	}
	fmt.Fprintf(os.Stderr, "beam: unknown command %q\n%s\n", os.Args[1], usage)
	os.Exit(2)
}
