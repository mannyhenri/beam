package simnet

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// socketToolReady reports whether the socket listing can run on this host.
func socketToolReady() error {
	if _, err := os.Stat("/proc/net/tcp"); err != nil {
		return fmt.Errorf("simnet: the no-direct-connection check needs /proc/net/tcp: %w", err)
	}
	return nil
}

// listSockets reads /proc/net/tcp and tcp6 and keeps the sockets whose inode
// one of the given processes holds open.
func listSockets(pids []int) ([]Socket, error) {
	owner := map[uint64]int{}
	for _, pid := range pids {
		fds, err := filepath.Glob(filepath.Join("/proc", strconv.Itoa(pid), "fd", "*"))
		if err != nil {
			return nil, fmt.Errorf("simnet: %w", err)
		}
		for _, fd := range fds {
			link, err := os.Readlink(fd)
			if err != nil {
				continue // the descriptor closed meanwhile
			}
			if s, ok := strings.CutPrefix(link, "socket:["); ok {
				if inode, err := strconv.ParseUint(strings.TrimSuffix(s, "]"), 10, 64); err == nil {
					owner[inode] = pid
				}
			}
		}
	}
	var out []Socket
	for _, name := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		f, err := os.Open(name)
		if os.IsNotExist(err) && name == "/proc/net/tcp6" {
			continue // IPv6 disabled
		}
		if err != nil {
			return nil, fmt.Errorf("simnet: %w", err)
		}
		socks, err := ParseProcNetTCP(f)
		f.Close()
		if err != nil {
			return nil, err
		}
		for _, s := range socks {
			if pid, ok := owner[s.Inode]; ok {
				s.Pid = pid
				out = append(out, s)
			}
		}
	}
	return out, nil
}
