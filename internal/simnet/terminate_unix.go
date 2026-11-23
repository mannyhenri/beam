//go:build !windows

package simnet

import (
	"os"
	"syscall"
)

// terminate asks the process to stop.
func terminate(p *os.Process) { p.Signal(syscall.SIGTERM) }
