package simnet

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const lsofPath = "/usr/sbin/lsof"

// socketToolReady reports whether the socket listing can run on this host.
func socketToolReady() error {
	if _, err := os.Stat(lsofPath); err != nil {
		return fmt.Errorf("simnet: the no-direct-connection check needs %s: %w", lsofPath, err)
	}
	return nil
}

// listSockets lists the TCP sockets of the given processes with one lsof call.
func listSockets(pids []int) ([]Socket, error) {
	if len(pids) == 0 {
		return nil, nil
	}
	list := make([]string, len(pids))
	for i, p := range pids {
		list[i] = strconv.Itoa(p)
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(lsofPath, "-nP", "-a", "-p", strings.Join(list, ","), "-iTCP", "-FpTn")
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		// lsof exits 1 when nothing matches, which here means no sockets.
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 1 || stdout.Len() > 0 {
			return nil, fmt.Errorf("simnet: lsof: %v: %s", err, bytes.TrimSpace(stderr.Bytes()))
		}
	}
	return ParseLsof(&stdout)
}
