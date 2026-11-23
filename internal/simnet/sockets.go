package simnet

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/netip"
	"sort"
	"strconv"
	"strings"
)

// Socket is one TCP socket as the operating system reports it.
type Socket struct {
	Pid    int // owning process; 0 when the source does not say (/proc/net/tcp)
	Local  netip.AddrPort
	Remote netip.AddrPort // zero for a listener
	State  string         // ESTABLISHED, SYN_SENT, LISTEN, CLOSE_WAIT, …
	Inode  uint64         // Linux socket inode; 0 from lsof
}

// Direct is one connection from a machine to another machine's address.
type Direct struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Local  string `json:"local"`
	Remote string `json:"remote"`
}

// DirectError lists every direct machine-to-machine connection found.
type DirectError struct {
	Direct []Direct
}

func (e *DirectError) Error() string {
	parts := make([]string, len(e.Direct))
	for i, d := range e.Direct {
		parts[i] = fmt.Sprintf("%s -> %s (%s -> %s)", d.From, d.To, d.Local, d.Remote)
	}
	return "simnet: machines connected directly instead of through the relay: " + strings.Join(parts, "; ")
}

// FindDirect returns the sockets owned by a machine whose remote address is
// another machine's address, sorted by From, To and Local. Listeners, sockets
// of other processes and a machine's connections to itself are ignored.
func FindDirect(machines []*Machine, socks []Socket) []Direct {
	byPid := map[int]*Machine{}
	byAddr := map[netip.Addr]*Machine{}
	for _, m := range machines {
		byPid[m.Pid] = m
		if a, err := netip.ParseAddr(m.Addr); err == nil {
			byAddr[a] = m
		}
	}
	var out []Direct
	for _, s := range socks {
		from := byPid[s.Pid]
		if from == nil || s.State == "LISTEN" || !s.Remote.IsValid() {
			continue
		}
		to := byAddr[s.Remote.Addr().Unmap()]
		if to == nil || to == from {
			continue
		}
		out = append(out, Direct{From: from.Name, To: to.Name, Local: s.Local.String(), Remote: s.Remote.String()})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Local < b.Local
	})
	return out
}

// tcpStates maps the state codes of /proc/net/tcp to the names lsof uses.
var tcpStates = map[string]string{
	"01": "ESTABLISHED", "02": "SYN_SENT", "03": "SYN_RECV", "04": "FIN_WAIT1",
	"05": "FIN_WAIT2", "06": "TIME_WAIT", "07": "CLOSE", "08": "CLOSE_WAIT",
	"09": "LAST_ACK", "0A": "LISTEN", "0B": "CLOSING",
}

// ParseProcNetTCP parses Linux /proc/net/tcp or /proc/net/tcp6. Addresses are
// unmapped, so an IPv4 peer seen through tcp6 compares equal to its IPv4 form.
func ParseProcNetTCP(r io.Reader) ([]Socket, error) {
	var out []Socket
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 0 || f[0] == "sl" {
			continue
		}
		if len(f) < 10 {
			return nil, fmt.Errorf("simnet: short /proc/net/tcp line %q", sc.Text())
		}
		local, err := procAddr(f[1])
		if err != nil {
			return nil, err
		}
		remote, err := procAddr(f[2])
		if err != nil {
			return nil, err
		}
		inode, err := strconv.ParseUint(f[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("simnet: bad inode in /proc/net/tcp line %q", sc.Text())
		}
		state := tcpStates[strings.ToUpper(f[3])]
		if state == "" {
			state = "UNKNOWN"
		}
		out = append(out, Socket{Local: local, Remote: remote, State: state, Inode: inode})
	}
	return out, sc.Err()
}

// procAddr decodes "0200007F:1F90": an address stored as little-endian 32-bit
// words in hex, then the port in hex.
func procAddr(s string) (netip.AddrPort, error) {
	ip, port, ok := strings.Cut(s, ":")
	b, err := hex.DecodeString(ip)
	if !ok || err != nil || (len(b) != 4 && len(b) != 16) {
		return netip.AddrPort{}, fmt.Errorf("simnet: bad /proc/net/tcp address %q", s)
	}
	for i := 0; i < len(b); i += 4 {
		binary.BigEndian.PutUint32(b[i:], binary.LittleEndian.Uint32(b[i:]))
	}
	p, err := strconv.ParseUint(port, 16, 16)
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("simnet: bad /proc/net/tcp port %q", s)
	}
	a, _ := netip.AddrFromSlice(b)
	return netip.AddrPortFrom(a.Unmap(), uint16(p)), nil
}

// ParseLsof parses `lsof -nP -iTCP -FpTn` output: a p line per process, then
// per socket an f line followed in any order by its n (local[->remote]) and
// T (TST=state) lines.
func ParseLsof(r io.Reader) ([]Socket, error) {
	var out []Socket
	pid := 0
	var cur *Socket
	flush := func() {
		if cur != nil && cur.Local.IsValid() {
			out = append(out, *cur)
		}
		cur = nil
	}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		val := line[1:]
		switch line[0] {
		case 'p':
			flush()
			n, err := strconv.Atoi(val)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("simnet: bad lsof pid line %q", line)
			}
			pid = n
		case 'f':
			flush()
			if pid == 0 {
				return nil, fmt.Errorf("simnet: lsof socket before any process: %q", line)
			}
			cur = &Socket{Pid: pid}
		case 'n':
			if pid == 0 {
				return nil, fmt.Errorf("simnet: lsof socket before any process: %q", line)
			}
			if cur == nil {
				cur = &Socket{Pid: pid}
			}
			local, remote, hasRemote := strings.Cut(val, "->")
			var err error
			if cur.Local, err = lsofAddr(local); err != nil {
				return nil, err
			}
			if hasRemote {
				if cur.Remote, err = lsofAddr(remote); err != nil {
					return nil, err
				}
			}
		case 'T':
			if st, ok := strings.CutPrefix(val, "ST="); ok && cur != nil {
				cur.State = st
			}
		}
	}
	flush()
	return out, sc.Err()
}

// lsofAddr parses "127.0.0.2:5000", "[::1]:80" or "*:5000" (any address).
func lsofAddr(s string) (netip.AddrPort, error) {
	if rest, ok := strings.CutPrefix(s, "*:"); ok {
		s = "0.0.0.0:" + rest
	}
	a, err := netip.ParseAddrPort(s)
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("simnet: bad lsof address %q", s)
	}
	return netip.AddrPortFrom(a.Addr().Unmap(), a.Port()), nil
}
