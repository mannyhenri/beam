// Command simnet starts named fake machines with internal/simnet, prints one
// JSON line per machine, waits until stdin closes or it is told to stop, then
// stops them and prints {"stopped":[...]}. It exits 0 only if every machine
// stopped cleanly. It is a test tool and not part of the release build.
//
//	simnet --dir <dir> --names A,B,C [--addrs 127.0.0.2,127.0.0.3,127.0.0.4]
package main

import (
	"encoding/json"
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
	flag.Parse()
	if *dir == "" || *names == "" {
		fmt.Fprintln(os.Stderr, "usage: simnet --dir <dir> --names A,B,C [--addrs a,b,c]")
		return 2
	}

	// Listen for a stop request before any machine exists, so none is missed.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)

	h, err := simnet.New(simnet.Config{Dir: *dir, Addrs: strings.Split(*addrs, ",")})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	list := strings.Split(*names, ",")
	if err := h.Preflight(len(list)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	out := json.NewEncoder(os.Stdout)
	for _, name := range list {
		m, err := h.Start(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			h.StopAll()
			return 1
		}
		out.Encode(machineLine{Name: m.Name, Pid: m.Pid, Addr: m.ListenAddr, Home: m.Home, Log: m.Log})
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
	for _, r := range stopped {
		if !r.Clean {
			return 1
		}
	}
	return 0
}
