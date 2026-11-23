// Package ac3 pins the exact values AC-3 expects from the Linux /proc/net/tcp parser.
// It lives under testdata so `go vet ./...` and `go test ./...` skip it; AC-3 runs it by path.
package ac3

import (
	"net/netip"
	"strings"
	"testing"

	"beam/internal/simnet"
)

const procNetTCP = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0400007F:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 99 1 0 100 0 0 10 0
   1: 0200007F:1F90 0300007F:2000 01 00000000:00000000 00:00000000 00000000  1000        0 4242 1 0 20 4 30 10 -1
`

func TestAC3ProcNetTCP(t *testing.T) {
	got, err := simnet.ParseProcNetTCP(strings.NewReader(procNetTCP))
	if err != nil {
		t.Fatalf("ParseProcNetTCP: %v", err)
	}
	want := []simnet.Socket{
		{Local: netip.MustParseAddrPort("127.0.0.4:80"), Remote: netip.MustParseAddrPort("0.0.0.0:0"), State: "LISTEN", Inode: 99},
		{Local: netip.MustParseAddrPort("127.0.0.2:8080"), Remote: netip.MustParseAddrPort("127.0.0.3:8192"), State: "ESTABLISHED", Inode: 4242},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sockets, want %d (the header line is not a socket): %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("socket %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
