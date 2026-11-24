// Command simnet starts named fake machines with internal/simnet, prints one
// JSON line per machine, waits until stdin closes or it is told to stop, then
// stops them and prints {"stopped":[...]}. It exits 0 only if every machine
// stopped cleanly. It is a test tool and not part of the release build.
//
//	simnet --dir <dir> --names A,B,C [--addrs 127.0.0.2,127.0.0.3,127.0.0.4]
//	       [--relay] [--dial A=C,B=relay]
//
// --relay starts a relay stand-in on 127.0.0.1 and prints {"relay":"127.0.0.1:P"}
// before the machine lines. --dial makes each FROM machine connect to the TO
// machine's address (or the relay) and hold it. With either flag the driver then
// prints {"check":{"ok":…,"direct":[{"from","to","local","remote"}]}} and exits 3
// if any direct machine-to-machine connection was seen while the machines ran.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"beam/internal/simnet"
)

type machineLine struct {
	Name string `json:"name"`
	Pid  int    `json:"pid"`
	Addr string `json:"addr"`
	Home string `json:"home"`
	Log  string `json:"log"`
}

func main() {
	simnet.MaybeRunStub()
	os.Exit(run())
}

func run() int {
	dir := flag.String("dir", "", "parent directory for the machines' folders (required)")
	names := flag.String("names", "", "comma-separated machine names, e.g. A,B,C (required)")
	addrs := flag.String("addrs", strings.Join(simnet.DefaultAddrs, ","), "comma-separated loopback addresses, in start order")
	relayFlag := flag.Bool("relay", false, "start a relay stand-in on 127.0.0.1")
	dialFlag := flag.String("dial", "", "comma-separated FROM=TO pairs; TO is a machine name or 'relay'")
	flag.Parse()
	if *dir == "" || *names == "" {
		fmt.Fprintln(os.Stderr, "usage: simnet --dir <dir> --names A,B,C [--addrs a,b,c] [--relay] [--dial A=C,B=relay]")
		return 2
	}
	list := strings.Split(*names, ",")
	dials, err := parseDials(*dialFlag, list, *relayFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	checking := *relayFlag || len(dials) > 0

	// Listen for a stop request before any machine exists, so none is missed.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)

	h, err := simnet.New(simnet.Config{Dir: *dir, Addrs: strings.Split(*addrs, ",")})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := h.Preflight(len(list)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	out := json.NewEncoder(os.Stdout)
	var relay *simnet.Relay
	if *relayFlag {
		if relay, err = simnet.StartRelay(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer relay.Close()
		out.Encode(map[string]string{"relay": relay.Addr})
	}
	machines := map[string]*simnet.Machine{}
	for _, name := range list {
		m, err := h.Start(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			h.StopAll()
			return 1
		}
		machines[name] = m
		out.Encode(machineLine{Name: m.Name, Pid: m.Pid, Addr: m.ListenAddr, Home: m.Home, Log: m.Log})
	}

	if checking {
		for _, d := range dials {
			target := relay.Addr
			if d.to != "relay" {
				target = machines[d.to].ListenAddr
			}
			if err := machines[d.from].Dial(target); err != nil {
				fmt.Fprintln(os.Stderr, err)
				h.StopAll()
				return 1
			}
		}
		direct := []simnet.Direct{}
		if err := h.CheckNoDirect(); err != nil {
			var de *simnet.DirectError
			if !errors.As(err, &de) {
				fmt.Fprintln(os.Stderr, err)
				h.StopAll()
				return 1
			}
			direct = de.Direct
		}
		out.Encode(map[string]any{"check": map[string]any{"ok": len(direct) == 0, "direct": direct}})
	}

	stdinClosed := make(chan struct{})
	go func() {
		io.Copy(io.Discard, os.Stdin)
		close(stdinClosed)
	}()
	select {
	case <-stdinClosed:
	case <-stop:
	}

	stopped := h.StopAll()
	out.Encode(map[string][]simnet.StopResult{"stopped": stopped})
	if checking && len(h.Seen()) > 0 {
		return 3
	}
	for _, r := range stopped {
		if !r.Clean {
			return 1
		}
	}
	return 0
}

type dial struct{ from, to string }

// parseDials checks "A=C,B=relay" against the machine names before anything starts.
func parseDials(spec string, names []string, relay bool) ([]dial, error) {
	if spec == "" {
		return nil, nil
	}
	known := map[string]bool{}
	for _, n := range names {
		known[n] = true
	}
	var out []dial
	for _, pair := range strings.Split(spec, ",") {
		from, to, ok := strings.Cut(pair, "=")
		switch {
		case !ok || !known[from]:
			return nil, fmt.Errorf("simnet: --dial %q: FROM must be one of %s", pair, strings.Join(names, ","))
		case to == "relay" && !relay:
			return nil, fmt.Errorf("simnet: --dial %q needs --relay", pair)
		case to != "relay" && !known[to]:
			return nil, fmt.Errorf("simnet: --dial %q: TO must be a machine name or relay", pair)
		}
		out = append(out, dial{from, to})
	}
	return out, nil
}
