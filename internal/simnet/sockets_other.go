//go:build !darwin && !linux

package simnet

import (
	"fmt"
	"runtime"
)

// socketToolReady is nil so machines still start; the check itself reports
// that it is not supported here.
func socketToolReady() error { return nil }

func listSockets([]int) ([]Socket, error) {
	return nil, fmt.Errorf("simnet: the no-direct-connection check is not supported on %s", runtime.GOOS)
}
