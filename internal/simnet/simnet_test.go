//go:build !windows

package simnet_test

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"beam/internal/simnet"
)

// Helper modes for the test binary when the harness runs it as a machine.
const (
	envDump     = "SIMNET_TEST_ENVDUMP"
	envNoTerm   = "SIMNET_TEST_IGNORE_TERM"
	loopbackAll = "127.0.0.1"
)

func TestMain(m *testing.M) {
	switch {
	case os.Getenv(envDump) == "1":
		for _, k := range []string{"HOME", "XDG_CONFIG_HOME", "BEAM_SIMNET_NAME", "BEAM_SIMNET_ADDR"} {
			fmt.Printf("%s=%s\n", k, os.Getenv(k))
		}
		os.Exit(0)
	case os.Getenv(envNoTerm) == "1":
		signal.Ignore(syscall.SIGTERM, os.Interrupt)
		fmt.Println("ignoring SIGTERM")
		select {}
	}
	// The harness re-runs this binary as the stub machine; this returns when not in stub mode.
	simnet.MaybeRunStub()
	os.Exit(m.Run())
}

// newHarness builds a harness in a temp dir; every machine is stopped when the test ends.
func newHarness(t *testing.T, cfg simnet.Config) *simnet.Harness {
	t.Helper()
	if cfg.Dir == "" {
		cfg.Dir = t.TempDir()
	}
	h, err := simnet.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { h.StopAll() })
	return h
}

func threeLocal() []string { return []string{loopbackAll, loopbackAll, loopbackAll} }

func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

type answer struct {
	Name string `json:"name"`
	Pid  int    `json:"pid"`
	Home string `json:"home"`
}

func ask(t *testing.T, addr string) answer {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		t.Fatalf("read from %s: %v", addr, err)
	}
	var a answer
	if err := json.Unmarshal([]byte(line), &a); err != nil {
		t.Fatalf("answer from %s is not JSON: %q", addr, line)
	}
	return a
}

func aliasesMissing() string {
	for _, a := range simnet.DefaultAddrs {
		ln, err := net.Listen("tcp", a+":0")
		if err != nil {
			return a
		}
		ln.Close()
	}
	return ""
}

func TestDefaultAddrs(t *testing.T) {
	want := []string{"127.0.0.2", "127.0.0.3", "127.0.0.4"}
	if strings.Join(simnet.DefaultAddrs, ",") != strings.Join(want, ",") {
		t.Fatalf("DefaultAddrs = %v, want %v", simnet.DefaultAddrs, want)
	}
}

func TestStartAssignsAddressesInOrder(t *testing.T) {
	if a := aliasesMissing(); a != "" {
		t.Skipf("loopback alias %s missing; run: sudo ifconfig lo0 alias %s up", a, a)
	}
	h := newHarness(t, simnet.Config{})
	for i, name := range []string{"A", "B", "C"} {
		m, err := h.Start(name)
		if err != nil {
			t.Fatalf("Start(%s): %v", name, err)
		}
		if m.Addr != simnet.DefaultAddrs[i] {
			t.Errorf("%s Addr = %s, want %s", name, m.Addr, simnet.DefaultAddrs[i])
		}
		host, _, err := net.SplitHostPort(m.ListenAddr)
		if err != nil || host != simnet.DefaultAddrs[i] {
			t.Errorf("%s ListenAddr = %q, want host %s", name, m.ListenAddr, simnet.DefaultAddrs[i])
		}
	}
}

func TestStartGivesEachMachineOwnProcessHomeAndLog(t *testing.T) {
	dir := t.TempDir()
	h := newHarness(t, simnet.Config{Dir: dir, Addrs: threeLocal()})
	seen := map[int]bool{}
	for _, name := range []string{"A", "B", "C"} {
		m, err := h.Start(name)
		if err != nil {
			t.Fatalf("Start(%s): %v", name, err)
		}
		if m.Name != name {
			t.Errorf("Name = %q, want %q", m.Name, name)
		}
		if m.Pid <= 0 || m.Pid == os.Getpid() || seen[m.Pid] || !alive(m.Pid) {
			t.Errorf("%s pid %d is not a distinct live child", name, m.Pid)
		}
		seen[m.Pid] = true
		if want := filepath.Join(dir, name, "home"); m.Home != want {
			t.Errorf("%s Home = %s, want %s", name, m.Home, want)
		}
		if fi, err := os.Stat(m.Home); err != nil || !fi.IsDir() {
			t.Errorf("%s home %s is not a directory: %v", name, m.Home, err)
		}
		if want := filepath.Join(dir, name, "machine.log"); m.Log != want {
			t.Errorf("%s Log = %s, want %s", name, m.Log, want)
		}
		a := ask(t, m.ListenAddr)
		if a.Name != name || a.Pid != m.Pid || a.Home != m.Home {
			t.Errorf("%s answered %+v, want name %s pid %d home %s", name, a, name, m.Pid, m.Home)
		}
		log, err := os.ReadFile(m.Log)
		if err != nil || !strings.Contains(string(log), name) || !strings.Contains(string(log), m.ListenAddr) {
			t.Errorf("%s log %s should name the machine and its address; got %q (%v)", name, m.Log, log, err)
		}
	}
	if got := len(h.Machines()); got != 3 {
		t.Errorf("Machines() has %d entries, want 3", got)
	}
}

func TestChildEnvironment(t *testing.T) {
	t.Setenv(envDump, "1")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	h := newHarness(t, simnet.Config{Dir: dir, Addrs: threeLocal(), Command: []string{self}})
	m, err := h.Start("env-check")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	m.Wait()
	log, err := os.ReadFile(m.Log)
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(dir, "env-check", "home")
	for _, want := range []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"BEAM_SIMNET_NAME=env-check",
		"BEAM_SIMNET_ADDR=" + loopbackAll,
	} {
		if !strings.Contains(string(log), want+"\n") {
			t.Errorf("child environment lacks %q; log:\n%s", want, log)
		}
	}
}

func TestFourthStartFails(t *testing.T) {
	h := newHarness(t, simnet.Config{Addrs: threeLocal()})
	for _, n := range []string{"A", "B", "C"} {
		if _, err := h.Start(n); err != nil {
			t.Fatalf("Start(%s): %v", n, err)
		}
	}
	if m, err := h.Start("D"); err == nil {
		t.Fatalf("4th Start succeeded (pid %d); want an error", m.Pid)
	}
	if got := len(h.Machines()); got != 3 {
		t.Errorf("Machines() has %d entries after failed 4th Start, want 3", got)
	}
}

func TestStartRejectsBadAndDuplicateNames(t *testing.T) {
	h := newHarness(t, simnet.Config{Addrs: threeLocal()})
	for _, bad := range []string{"", "a b", "../x", "A/B", "é", "A.B"} {
		if _, err := h.Start(bad); err == nil {
			t.Errorf("Start(%q) succeeded; want an error", bad)
		}
	}
	if _, err := h.Start("A-1"); err != nil {
		t.Fatalf("Start(A-1): %v", err)
	}
	if _, err := h.Start("A-1"); err == nil {
		t.Errorf("duplicate Start(A-1) succeeded; want an error")
	}
	if got := len(h.Machines()); got != 1 {
		t.Errorf("Machines() has %d entries, want 1", got)
	}
}

func TestStopAllStopsCleanly(t *testing.T) {
	h := newHarness(t, simnet.Config{Addrs: threeLocal()})
	var ms []*simnet.Machine
	for _, n := range []string{"A", "B", "C"} {
		m, err := h.Start(n)
		if err != nil {
			t.Fatalf("Start(%s): %v", n, err)
		}
		ms = append(ms, m)
	}
	res := h.StopAll()
	if len(res) != 3 {
		t.Fatalf("StopAll returned %d results, want 3", len(res))
	}
	for i, r := range res {
		if r.Name != ms[i].Name || r.Pid != ms[i].Pid || !r.Clean {
			t.Errorf("result %d = %+v, want name %s pid %d clean", i, r, ms[i].Name, ms[i].Pid)
		}
	}
	for _, m := range ms {
		if alive(m.Pid) {
			t.Errorf("%s pid %d still alive after StopAll", m.Name, m.Pid)
		}
		if c, err := net.DialTimeout("tcp", m.ListenAddr, time.Second); err == nil {
			c.Close()
			t.Errorf("%s still accepts on %s after StopAll", m.Name, m.ListenAddr)
		}
	}
	if again := h.StopAll(); len(again) != 0 {
		t.Errorf("second StopAll returned %v, want nothing", again)
	}
}

func TestStopForceKillsStubbornMachine(t *testing.T) {
	t.Setenv(envNoTerm, "1")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, simnet.Config{Addrs: threeLocal(), Command: []string{self}, StopTimeout: 300 * time.Millisecond})
	m, err := h.Start("stubborn")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Wait until the child has installed its SIGTERM handler.
	deadline := time.Now().Add(10 * time.Second)
	for {
		b, _ := os.ReadFile(m.Log)
		if strings.Contains(string(b), "ignoring SIGTERM") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("stubborn child never became ready; log %q", b)
		}
		time.Sleep(20 * time.Millisecond)
	}
	res := h.StopAll()
	if len(res) != 1 || res[0].Clean {
		t.Fatalf("StopAll = %+v, want one result with Clean=false", res)
	}
	if alive(m.Pid) {
		t.Errorf("stubborn pid %d still alive after forced stop", m.Pid)
	}
}

func TestMissingAliasStartsNothing(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("only macOS lacks 127.0.0.0/8 aliases by default")
	}
	h := newHarness(t, simnet.Config{Addrs: []string{loopbackAll, "127.0.0.250"}})

	err := h.Preflight(2)
	if err == nil {
		t.Fatal("Preflight(2) succeeded with 127.0.0.250 unbindable; want an error")
	}
	for _, want := range []string{"127.0.0.250", "sudo ifconfig lo0 alias 127.0.0.250 up"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Preflight error %q lacks %q", err, want)
		}
	}
	if got := len(h.Machines()); got != 0 {
		t.Errorf("Preflight started %d machines, want 0", got)
	}

	if _, err := h.Start("A"); err != nil {
		t.Fatalf("Start(A) on 127.0.0.1: %v", err)
	}
	_, err = h.Start("B")
	if err == nil || !strings.Contains(err.Error(), "sudo ifconfig lo0 alias 127.0.0.250 up") {
		t.Fatalf("Start(B) on 127.0.0.250 = %v, want the sudo fix", err)
	}
	if got := len(h.Machines()); got != 1 {
		t.Errorf("Machines() has %d entries after failed Start, want 1", got)
	}
}
