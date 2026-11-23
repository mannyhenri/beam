package simnet

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DefaultAddrs are the machines' loopback addresses, assigned in start order.
var DefaultAddrs = []string{"127.0.0.2", "127.0.0.3", "127.0.0.4"}

const (
	defaultStopTimeout = 5 * time.Second
	readyTimeout       = 10 * time.Second
	readyPrefix        = "simnet: ready "
)

var validName = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// Config describes a harness. Only Dir is required.
type Config struct {
	Dir         string        // parent of every machine's folder
	Addrs       []string      // addresses in start order; DefaultAddrs if empty
	Command     []string      // program and arguments; the stub machine from os.Executable if empty
	StopTimeout time.Duration // how long StopAll waits per machine; 5 s if zero
}

// Harness starts and stops fake machines.
type Harness struct {
	cfg      Config
	mu       sync.Mutex
	machines []*Machine
}

// Machine is one running fake machine.
type Machine struct {
	Name       string
	Pid        int
	Addr       string // assigned loopback address
	ListenAddr string // stub machine's address:port; empty for a custom Command
	Home       string
	Log        string

	cmd  *exec.Cmd
	done chan struct{}
	err  error
}

// StopResult reports how one machine stopped.
type StopResult struct {
	Name  string `json:"name"`
	Pid   int    `json:"pid"`
	Clean bool   `json:"clean"` // exited with code 0 after the stop signal, without a forced kill
}

// New returns a harness rooted at cfg.Dir.
func New(cfg Config) (*Harness, error) {
	if cfg.Dir == "" {
		return nil, errors.New("simnet: Config.Dir is required")
	}
	dir, err := filepath.Abs(cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("simnet: %w", err)
	}
	cfg.Dir = dir
	if len(cfg.Addrs) == 0 {
		cfg.Addrs = DefaultAddrs
	}
	cfg.Addrs = append([]string(nil), cfg.Addrs...)
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = defaultStopTimeout
	}
	return &Harness{cfg: cfg}, nil
}

// Preflight checks that the first n addresses can be bound, before any machine starts.
func (h *Harness) Preflight(n int) error {
	if n > len(h.cfg.Addrs) {
		return fmt.Errorf("simnet: %d machines requested, only %d addresses", n, len(h.cfg.Addrs))
	}
	for _, a := range h.cfg.Addrs[:n] {
		if err := checkAddr(a); err != nil {
			return err
		}
	}
	return nil
}

// checkAddr binds addr on a free port to prove the address exists on this host.
func checkAddr(addr string) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(addr, "0"))
	if err != nil {
		msg := fmt.Sprintf("simnet: cannot bind loopback address %s: %v", addr, err)
		if runtime.GOOS == "darwin" {
			msg += fmt.Sprintf("; add it once (and after every reboot) with: sudo ifconfig lo0 alias %s up", addr)
		}
		return errors.New(msg)
	}
	return ln.Close()
}

// Machines returns the machines started and not yet stopped, in start order.
func (h *Harness) Machines() []*Machine {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*Machine(nil), h.machines...)
}

// Start starts the named machine on the next free address.
func (h *Harness) Start(name string) (*Machine, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !validName.MatchString(name) {
		return nil, fmt.Errorf("simnet: machine name %q must match [A-Za-z0-9-]+", name)
	}
	for _, m := range h.machines {
		if m.Name == name {
			return nil, fmt.Errorf("simnet: machine %q already started", name)
		}
	}
	if len(h.machines) >= len(h.cfg.Addrs) {
		return nil, fmt.Errorf("simnet: at most %d machines", len(h.cfg.Addrs))
	}
	addr := h.cfg.Addrs[len(h.machines)]
	if err := checkAddr(addr); err != nil {
		return nil, err
	}

	m, err := h.start(name, addr)
	if err != nil {
		return nil, err
	}
	h.machines = append(h.machines, m)
	return m, nil
}

func (h *Harness) start(name, addr string) (*Machine, error) {
	base := filepath.Join(h.cfg.Dir, name)
	m := &Machine{
		Name: name,
		Addr: addr,
		Home: filepath.Join(base, "home"),
		Log:  filepath.Join(base, "machine.log"),
		done: make(chan struct{}),
	}
	if err := os.MkdirAll(m.Home, 0o700); err != nil {
		return nil, fmt.Errorf("simnet: %w", err)
	}
	logf, err := os.OpenFile(m.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("simnet: %w", err)
	}
	defer logf.Close() // the child and the copier hold their own handles

	stub := len(h.cfg.Command) == 0
	argv := h.cfg.Command
	if stub {
		self, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("simnet: %w", err)
		}
		argv = []string{self}
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = machineEnv(m, stub)
	cmd.Stderr = logf

	// The stub's stdout runs through a pipe so Start can wait for its ready line.
	var ready chan string
	var pr *os.File
	if stub {
		var pw *os.File
		if pr, pw, err = os.Pipe(); err != nil {
			return nil, fmt.Errorf("simnet: %w", err)
		}
		defer pw.Close()
		cmd.Stdout = pw
		ready = make(chan string, 1)
	} else {
		cmd.Stdout = logf
	}

	if err := cmd.Start(); err != nil {
		if pr != nil {
			pr.Close()
		}
		return nil, fmt.Errorf("simnet: start %s: %w", name, err)
	}
	m.cmd = cmd
	m.Pid = cmd.Process.Pid

	copied := make(chan struct{})
	if stub {
		logCopy, err := os.OpenFile(m.Log, os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			pr.Close()
			cmd.Process.Kill()
			cmd.Wait()
			return nil, fmt.Errorf("simnet: %w", err)
		}
		go copyStdout(pr, logCopy, ready, copied)
	} else {
		close(copied)
	}
	go func() {
		m.err = cmd.Wait()
		<-copied
		close(m.done)
	}()

	if !stub {
		return m, nil
	}
	select {
	case m.ListenAddr = <-ready:
		return m, nil
	case <-m.done:
		return nil, fmt.Errorf("simnet: machine %s exited before it was ready (%v); see %s", name, m.err, m.Log)
	case <-time.After(readyTimeout):
		cmd.Process.Kill()
		<-m.done
		return nil, fmt.Errorf("simnet: machine %s not ready after %s; see %s", name, readyTimeout, m.Log)
	}
}

// copyStdout appends the child's stdout to its log and reports the ready line's
// address once it has been written to the log.
func copyStdout(r io.ReadCloser, log io.WriteCloser, ready chan<- string, done chan<- struct{}) {
	defer close(done)
	defer log.Close()
	defer r.Close()
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		io.WriteString(log, line+"\n")
		if rest, ok := strings.CutPrefix(line, readyPrefix); ok && ready != nil {
			addr, _, _ := strings.Cut(rest, " ")
			ready <- addr
			ready = nil
		}
	}
}

// machineEnv is the parent's environment with the machine's own home and identity.
func machineEnv(m *Machine, stub bool) []string {
	set := map[string]string{
		"HOME":             m.Home,
		"XDG_CONFIG_HOME":  filepath.Join(m.Home, ".config"),
		"USERPROFILE":      m.Home,
		"APPDATA":          filepath.Join(m.Home, "AppData", "Roaming"),
		"LOCALAPPDATA":     filepath.Join(m.Home, "AppData", "Local"),
		"BEAM_SIMNET_NAME": m.Name,
		"BEAM_SIMNET_ADDR": m.Addr,
	}
	if stub {
		set[stubEnv] = "1"
	}
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if _, ours := set[k]; ours || k == stubEnv {
			continue
		}
		env = append(env, kv)
	}
	for k, v := range set {
		env = append(env, k+"="+v)
	}
	return env
}

// Wait blocks until the machine's process has exited and returns its exit error.
func (m *Machine) Wait() error {
	<-m.done
	return m.err
}

// StopAll stops every running machine and reports each in start order.
func (h *Harness) StopAll() []StopResult {
	h.mu.Lock()
	ms := h.machines
	h.machines = nil
	h.mu.Unlock()

	for _, m := range ms {
		terminate(m.cmd.Process)
	}
	res := make([]StopResult, 0, len(ms))
	for _, m := range ms {
		clean := true
		select {
		case <-m.done:
		case <-time.After(h.cfg.StopTimeout):
			clean = false
			m.cmd.Process.Kill()
			<-m.done
		}
		res = append(res, StopResult{Name: m.Name, Pid: m.Pid, Clean: clean && m.err == nil})
	}
	return res
}
