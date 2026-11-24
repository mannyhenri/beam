//go:build !windows

package simnet_test

import (
	"errors"
	"net"
	"sort"
	"testing"
	"time"

	"beam/internal/simnet"
)

// startABC starts A, B and C on 127.0.0.2-.4, skipping if the aliases are missing.
func startABC(t *testing.T, cfg simnet.Config) (*simnet.Harness, map[string]*simnet.Machine) {
	t.Helper()
	if a := aliasesMissing(); a != "" {
		t.Skipf("loopback alias %s missing; run: sudo ifconfig lo0 alias %s up", a, a)
	}
	h := newHarness(t, cfg)
	ms := map[string]*simnet.Machine{}
	for _, name := range []string{"A", "B", "C"} {
		m, err := h.Start(name)
		if err != nil {
			t.Fatalf("Start(%s): %v", name, err)
		}
		ms[name] = m
	}
	return h, ms
}

func startRelay(t *testing.T) *simnet.Relay {
	t.Helper()
	r, err := simnet.StartRelay()
	if err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func dial(t *testing.T, m *simnet.Machine, addr string) {
	t.Helper()
	if err := m.Dial(addr); err != nil {
		t.Fatalf("%s.Dial(%s): %v", m.Name, addr, err)
	}
}

func TestRelayListensOnLoopbackOnly(t *testing.T) {
	r := startRelay(t)
	host, port, err := net.SplitHostPort(r.Addr)
	if err != nil || host != "127.0.0.1" || port == "0" {
		t.Fatalf("relay Addr = %q, want 127.0.0.1:<port>", r.Addr)
	}
	c, err := net.DialTimeout("tcp", r.Addr, 5*time.Second)
	if err != nil {
		t.Fatalf("relay does not accept: %v", err)
	}
	c.Close()
}

func TestCheckNoDirectFindsEveryDirectDial(t *testing.T) {
	h, ms := startABC(t, simnet.Config{})
	r := startRelay(t)
	dial(t, ms["A"], ms["C"].ListenAddr)
	dial(t, ms["B"], ms["A"].ListenAddr)
	dial(t, ms["C"], r.Addr)

	err := h.CheckNoDirect()
	var de *simnet.DirectError
	if !errors.As(err, &de) {
		t.Fatalf("CheckNoDirect = %v, want *DirectError", err)
	}
	got := map[string]simnet.Direct{}
	for _, d := range de.Direct {
		got[d.From+"→"+d.To] = d
	}
	if len(got) != 2 || len(de.Direct) != 2 {
		t.Fatalf("direct = %+v, want exactly A→C and B→A", de.Direct)
	}
	for _, p := range [][2]string{{"A", "C"}, {"B", "A"}} {
		d, ok := got[p[0]+"→"+p[1]]
		if !ok {
			t.Fatalf("%s→%s not reported: %+v", p[0], p[1], de.Direct)
		}
		if d.Remote != ms[p[1]].ListenAddr {
			t.Errorf("%s→%s remote = %s, want %s", p[0], p[1], d.Remote, ms[p[1]].ListenAddr)
		}
		if host, _, _ := net.SplitHostPort(d.Local); host != ms[p[0]].Addr {
			t.Errorf("%s→%s local = %s, want it on %s (the stub dials from its own address)", p[0], p[1], d.Local, ms[p[0]].Addr)
		}
	}
}

func TestCheckNoDirectPassesForRelayOnlyAndHarnessReads(t *testing.T) {
	h, ms := startABC(t, simnet.Config{})
	r := startRelay(t)
	for _, name := range []string{"A", "B", "C"} {
		dial(t, ms[name], r.Addr)
	}
	// The test process (not a machine) reading each stub is not a violation either.
	for _, name := range []string{"A", "B", "C"} {
		if a := ask(t, ms[name].ListenAddr); a.Name != name {
			t.Fatalf("stub %s answered as %q", name, a.Name)
		}
	}
	if err := h.CheckNoDirect(); err != nil {
		t.Fatalf("CheckNoDirect = %v, want nil", err)
	}
}

func TestDialFailsForUnreachableAddress(t *testing.T) {
	_, ms := startABC(t, simnet.Config{})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close() // nothing listens there now
	if err := ms["A"].Dial(addr); err == nil {
		t.Fatalf("A.Dial(%s) to a closed port returned no error", addr)
	}
}

func TestPollerRemembersDirectConnectionAfterStop(t *testing.T) {
	h, ms := startABC(t, simnet.Config{Poll: 50 * time.Millisecond})
	dial(t, ms["A"], ms["B"].ListenAddr)

	deadline := time.Now().Add(10 * time.Second)
	for len(h.Seen()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("poller did not record A→B within 10 s")
		}
		time.Sleep(20 * time.Millisecond)
	}
	h.StopAll()
	seen := h.Seen()
	sort.Slice(seen, func(i, j int) bool { return seen[i].From+seen[i].To < seen[j].From+seen[j].To })
	if len(seen) != 1 || seen[0].From != "A" || seen[0].To != "B" {
		t.Fatalf("Seen after StopAll = %+v, want only A→B", seen)
	}
}

func TestNoPollerFindingsWithoutDirectDials(t *testing.T) {
	h, ms := startABC(t, simnet.Config{Poll: 20 * time.Millisecond})
	r := startRelay(t)
	dial(t, ms["A"], r.Addr)
	// Let the poller run several times while A only talks to the relay.
	time.Sleep(200 * time.Millisecond)
	h.StopAll()
	if seen := h.Seen(); len(seen) != 0 {
		t.Fatalf("Seen = %+v, want none", seen)
	}
}
