// Package ac1 pins AC-1 of ws-client: with HTTPS_PROXY set, ws.Dial reaches a wss://
// relay through the proxy (one CONNECT), and with NO_PROXY naming the relay it does not.
// The client runs in a fresh child process (this test binary re-executed) because
// http.ProxyFromEnvironment reads the environment once per process, and it dials
// relay.beam.invalid because loopback addresses are never proxied; only the test proxy
// can resolve that name. It lives under testdata so go vet ./... and go test ./... skip it.
package ac1

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"beam/internal/ws"
)

const relayName = "relay.beam.invalid"

// TestMain runs the client side when re-executed as a child.
func TestMain(m *testing.M) {
	if os.Getenv("AC1_CLIENT") == "1" {
		os.Exit(runClient())
	}
	os.Exit(m.Run())
}

// runClient dials AC1_URL trusting the PEM certificate in AC1_CERT, sends one binary
// frame "hello" and prints "OK" or "ERR <error>".
func runClient() int {
	pemBytes, err := os.ReadFile(os.Getenv("AC1_CERT"))
	if err != nil {
		fmt.Println("ERR reading cert:", err)
		return 0
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pemBytes)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c, err := ws.Dial(ctx, os.Getenv("AC1_URL"), &ws.DialConfig{TLS: &tls.Config{RootCAs: pool, ServerName: "example.com"}})
	if err != nil {
		fmt.Println("ERR", err)
		return 0
	}
	defer c.Close()
	if err := c.WriteFrame(ws.OpBinary, []byte("hello")); err != nil {
		fmt.Println("ERR write:", err)
		return 0
	}
	fmt.Println("OK")
	return 0
}

type relayResult struct {
	path    string
	payload string
	err     error
}

// startRelay starts a TLS relay whose handler upgrades and reads one masked frame.
func startRelay(t *testing.T) (*httptest.Server, chan relayResult) {
	results := make(chan relayResult, 4)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := ws.Upgrade(w, r)
		if err != nil {
			results <- relayResult{err: err}
			return
		}
		defer c.Close()
		c.NetConn().SetDeadline(time.Now().Add(10 * time.Second))
		hdr := make([]byte, 2)
		if _, err := io.ReadFull(c, hdr); err != nil {
			results <- relayResult{path: r.URL.Path, err: err}
			return
		}
		n := int(hdr[1] & 0x7f)
		key := make([]byte, 4)
		if hdr[1]&0x80 != 0 {
			io.ReadFull(c, key)
		}
		p := make([]byte, n)
		io.ReadFull(c, p)
		for i := range p {
			p[i] ^= key[i%4]
		}
		results <- relayResult{path: r.URL.Path, payload: string(p)}
	}))
	t.Cleanup(srv.Close)
	return srv, results
}

// startProxy starts a CONNECT proxy that sends relay.beam.invalid to the relay and
// records every CONNECT target.
func startProxy(t *testing.T, relayAddr string) (string, func() []string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var mu sync.Mutex
	var seen []string
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				br := bufio.NewReader(conn)
				req, err := http.ReadRequest(br)
				if err != nil {
					return
				}
				mu.Lock()
				seen = append(seen, req.Method+" "+req.Host)
				mu.Unlock()
				host, port, _ := net.SplitHostPort(req.Host)
				_, relayPort, _ := net.SplitHostPort(relayAddr)
				if req.Method != http.MethodConnect || host != relayName || port != relayPort {
					io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
					return
				}
				up, err := net.Dial("tcp", relayAddr)
				if err != nil {
					io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
					return
				}
				defer up.Close()
				io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
				go io.Copy(up, br)
				io.Copy(conn, up)
			}(conn)
		}
	}()
	return ln.Addr().String(), func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func writeCert(t *testing.T, srv *httptest.Server) string {
	path := filepath.Join(t.TempDir(), "relay.pem")
	block := &pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runChild(t *testing.T, env []string) string {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("client process failed: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func cleanEnv() []string {
	// The child must see only the proxy settings this test chooses.
	var drop []string
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "NO_PROXY", "no_proxy", "ALL_PROXY", "all_proxy"} {
		drop = append(drop, k+"=")
	}
	return drop
}

func TestAC1ProxyAndNoProxy(t *testing.T) {
	relay, results := startRelay(t)
	relayAddr := relay.Listener.Addr().String()
	_, relayPort, _ := net.SplitHostPort(relayAddr)
	proxyAddr, seen := startProxy(t, relayAddr)
	cert := writeCert(t, relay)
	url := "wss://" + relayName + ":" + relayPort + "/c/test"
	base := append(cleanEnv(), "AC1_CLIENT=1", "AC1_CERT="+cert, "AC1_URL="+url, "HTTPS_PROXY=http://"+proxyAddr)

	t.Run("through HTTPS_PROXY", func(t *testing.T) {
		out := runChild(t, base)
		if out != "OK" {
			t.Fatalf("client: %s", out)
		}
		want := []string{"CONNECT " + relayName + ":" + relayPort}
		if got := seen(); len(got) != 1 || got[0] != want[0] {
			t.Fatalf("proxy saw %q, want exactly %q", got, want)
		}
		select {
		case r := <-results:
			if r.err != nil {
				t.Fatalf("relay: %v", r.err)
			}
			if r.path != "/c/test" {
				t.Errorf("relay saw path %q, want /c/test", r.path)
			}
			if r.payload != "hello" {
				t.Errorf("relay got payload %q, want hello", r.payload)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("relay never received the upgrade")
		}
	})

	t.Run("NO_PROXY names the relay", func(t *testing.T) {
		before := len(seen())
		out := runChild(t, append(base, "NO_PROXY="+relayName))
		if !strings.HasPrefix(out, "ERR") {
			t.Fatalf("Dial succeeded without the proxy (%s); %s must not resolve directly", out, relayName)
		}
		if got := seen(); len(got) != before {
			t.Fatalf("proxy saw %q although NO_PROXY names the relay", got[before:])
		}
	})
}
