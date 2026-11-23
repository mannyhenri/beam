package simnet_test

import (
	"errors"
	"net/netip"
	"strings"
	"testing"

	"beam/internal/simnet"
)

func ap(s string) netip.AddrPort { return netip.MustParseAddrPort(s) }

func TestParseProcNetTCP(t *testing.T) {
	const in = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0400007F:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 99 1 0 100 0 0 10 0
   1: 0200007F:1F90 0300007F:2000 01 00000000:00000000 00:00000000 00000000  1000        0 4242 1 0 20 4 30 10 -1
   2: 0300007F:C350 0200007F:1F90 02 00000000:00000000 01:00000064 00000002  1000        0 777 1 0 100 0 0 10 -1
   3: 0400007F:9C40 0100007F:1E61 08 00000000:00000000 00:00000000 00000000  1000        0 8 1 0 20 4 30 10 -1
`
	got, err := simnet.ParseProcNetTCP(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseProcNetTCP: %v", err)
	}
	want := []simnet.Socket{
		{Local: ap("127.0.0.4:80"), Remote: ap("0.0.0.0:0"), State: "LISTEN", Inode: 99},
		{Local: ap("127.0.0.2:8080"), Remote: ap("127.0.0.3:8192"), State: "ESTABLISHED", Inode: 4242},
		{Local: ap("127.0.0.3:50000"), Remote: ap("127.0.0.2:8080"), State: "SYN_SENT", Inode: 777},
		{Local: ap("127.0.0.4:40000"), Remote: ap("127.0.0.1:7777"), State: "CLOSE_WAIT", Inode: 8},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sockets, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("socket %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseProcNetTCP6MapsIPv4(t *testing.T) {
	// /proc/net/tcp6 shows IPv4 peers as ::ffff:a.b.c.d; the parser unmaps them so they
	// compare equal to the machines' IPv4 addresses.
	const in = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0000000000000000FFFF00000200007F:1F90 0000000000000000FFFF00000400007F:2000 01 00000000:00000000 00:00000000 00000000  1000        0 31 1 0 20 4 30 10 -1
`
	got, err := simnet.ParseProcNetTCP(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseProcNetTCP: %v", err)
	}
	want := simnet.Socket{Local: ap("127.0.0.2:8080"), Remote: ap("127.0.0.4:8192"), State: "ESTABLISHED", Inode: 31}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want [%+v]", got, want)
	}
}

func TestParseProcNetTCPRejectsBadLine(t *testing.T) {
	for _, line := range []string{
		"   1: ZZ00007F:1F90 0300007F:2000 01 00000000:00000000 00:00000000 00000000  1000        0 4242 1",
		"   1: 0200007F 0300007F:2000 01 00000000:00000000 00:00000000 00000000  1000        0 4242 1",
		"   1: 0200007F:1F90 0300007F:2000 01 00000000:00000000 00:00000000 00000000  1000        0 notanumber 1",
	} {
		if _, err := simnet.ParseProcNetTCP(strings.NewReader(line + "\n")); err == nil {
			t.Errorf("ParseProcNetTCP(%q) returned no error", line)
		}
	}
}

func TestParseLsof(t *testing.T) {
	// Output of: lsof -nP -a -p 101,202 -iTCP -FpTn
	const in = "p101\nf5\nTST=LISTEN\nn127.0.0.2:5000\nf7\nTST=ESTABLISHED\nTQR=0\nTQS=0\nn127.0.0.2:50001->127.0.0.4:6000\n" +
		"p202\nf3\nTST=SYN_SENT\nn127.0.0.3:50002->127.0.0.2:5000\nf4\nTST=CLOSE_WAIT\nn127.0.0.3:50003->127.0.0.1:7777\n"
	got, err := simnet.ParseLsof(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseLsof: %v", err)
	}
	want := []simnet.Socket{
		{Pid: 101, Local: ap("127.0.0.2:5000"), State: "LISTEN"},
		{Pid: 101, Local: ap("127.0.0.2:50001"), Remote: ap("127.0.0.4:6000"), State: "ESTABLISHED"},
		{Pid: 202, Local: ap("127.0.0.3:50002"), Remote: ap("127.0.0.2:5000"), State: "SYN_SENT"},
		{Pid: 202, Local: ap("127.0.0.3:50003"), Remote: ap("127.0.0.1:7777"), State: "CLOSE_WAIT"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sockets, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("socket %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseLsofRejectsBadInput(t *testing.T) {
	for _, in := range []string{
		"pabc\nn127.0.0.2:1->127.0.0.3:2\n",  // pid is not a number
		"p1\nn127.0.0.2:1->not-an-address\n", // remote is not ip:port
		"n127.0.0.2:1->127.0.0.3:2\n",        // socket before any process
	} {
		if _, err := simnet.ParseLsof(strings.NewReader(in)); err == nil {
			t.Errorf("ParseLsof(%q) returned no error", in)
		}
	}
}

func TestFindDirect(t *testing.T) {
	machines := []*simnet.Machine{
		{Name: "A", Pid: 10, Addr: "127.0.0.2"},
		{Name: "B", Pid: 11, Addr: "127.0.0.3"},
		{Name: "C", Pid: 12, Addr: "127.0.0.4"},
	}
	socks := []simnet.Socket{
		{Pid: 10, Local: ap("127.0.0.2:5000"), State: "LISTEN"},                                     // A's own listener
		{Pid: 10, Local: ap("127.0.0.2:40001"), Remote: ap("127.0.0.4:6000"), State: "ESTABLISHED"}, // A→C
		{Pid: 10, Local: ap("127.0.0.2:40002"), Remote: ap("127.0.0.2:5000"), State: "ESTABLISHED"}, // A to itself: allowed
		{Pid: 11, Local: ap("127.0.0.1:40003"), Remote: ap("127.0.0.2:5000"), State: "SYN_SENT"},    // B→A, default source address
		{Pid: 12, Local: ap("127.0.0.4:40004"), Remote: ap("127.0.0.1:7777"), State: "ESTABLISHED"}, // C→relay: allowed
		{Pid: 12, Local: ap("127.0.0.4:40005"), Remote: ap("127.0.0.3:5000"), State: "CLOSE_WAIT"},  // C→B
		{Pid: 99, Local: ap("127.0.0.1:40006"), Remote: ap("127.0.0.3:5000"), State: "ESTABLISHED"}, // not a machine (the test reading B)
		{Pid: 11, Local: ap("127.0.0.3:5000"), Remote: ap("127.0.0.1:40006"), State: "ESTABLISHED"}, // B accepting that read
	}
	got := simnet.FindDirect(machines, socks)
	want := []simnet.Direct{
		{From: "A", To: "C", Local: "127.0.0.2:40001", Remote: "127.0.0.4:6000"},
		{From: "B", To: "A", Local: "127.0.0.1:40003", Remote: "127.0.0.2:5000"},
		{From: "C", To: "B", Local: "127.0.0.4:40005", Remote: "127.0.0.3:5000"},
	}
	if len(got) != len(want) {
		t.Fatalf("FindDirect = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("direct %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFindDirectNoneWhenOnlyRelay(t *testing.T) {
	machines := []*simnet.Machine{{Name: "A", Pid: 10, Addr: "127.0.0.2"}, {Name: "B", Pid: 11, Addr: "127.0.0.3"}}
	socks := []simnet.Socket{
		{Pid: 10, Local: ap("127.0.0.2:40001"), Remote: ap("127.0.0.1:7777"), State: "ESTABLISHED"},
		{Pid: 11, Local: ap("127.0.0.3:40002"), Remote: ap("127.0.0.1:7777"), State: "ESTABLISHED"},
	}
	if got := simnet.FindDirect(machines, socks); len(got) != 0 {
		t.Fatalf("FindDirect = %+v, want none", got)
	}
}

func TestDirectErrorNamesEveryConnection(t *testing.T) {
	err := error(&simnet.DirectError{Direct: []simnet.Direct{
		{From: "A", To: "C", Local: "127.0.0.2:40001", Remote: "127.0.0.4:6000"},
		{From: "B", To: "A", Local: "127.0.0.3:40003", Remote: "127.0.0.2:5000"},
	}})
	msg := err.Error()
	for _, s := range []string{"A", "C", "127.0.0.2:40001", "127.0.0.4:6000", "B", "127.0.0.3:40003", "127.0.0.2:5000"} {
		if !strings.Contains(msg, s) {
			t.Errorf("error %q does not mention %s", msg, s)
		}
	}
	var de *simnet.DirectError
	if !errors.As(err, &de) || len(de.Direct) != 2 {
		t.Errorf("errors.As(*DirectError) failed for %v", err)
	}
}
