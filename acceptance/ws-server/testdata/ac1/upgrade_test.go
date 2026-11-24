// Package ac1 pins AC-1 of ws-server: a valid upgrade gets 101 with the RFC 6455 accept
// key and a taken-over connection that carries raw bytes both ways, including bytes the
// client sent in the same write as the request. It lives under testdata so go vet ./...,
// go list ./... and go test ./... skip it; the AC-1 Node test runs it by path.
package ac1

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"beam/internal/ws"
)

type result struct {
	err     error
	got     string
	netConn bool
}

func TestAC1UpgradeAndRawBytes(t *testing.T) {
	cases := []struct{ name, upgrade, connection string }{
		{"canonical", "websocket", "Upgrade"},
		{"mixed-case upgrade token", "WebSocket", "Upgrade"},
		{"connection token list", "websocket", "keep-alive, Upgrade"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := make(chan result, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := ws.Upgrade(w, r)
				if err != nil {
					results <- result{err: err}
					return
				}
				defer c.Close()
				var rwc io.ReadWriteCloser = c
				buf := make([]byte, 4)
				if _, err := io.ReadFull(rwc, buf); err != nil {
					results <- result{err: err}
					return
				}
				if _, err := rwc.Write([]byte("pong")); err != nil {
					results <- result{err: err}
					return
				}
				results <- result{got: string(buf), netConn: c.NetConn() != nil}
			}))
			defer srv.Close()

			conn, err := net.DialTimeout("tcp", srv.Listener.Addr().String(), 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(10 * time.Second))

			req := "GET / HTTP/1.1\r\n" +
				"Host: " + srv.Listener.Addr().String() + "\r\n" +
				"Upgrade: " + tc.upgrade + "\r\n" +
				"Connection: " + tc.connection + "\r\n" +
				"Sec-WebSocket-Version: 13\r\n" +
				"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
				"\r\n" +
				"ping" // same write as the request: must not be lost
			if _, err := io.WriteString(conn, req); err != nil {
				t.Fatal(err)
			}

			br := bufio.NewReader(conn)
			status, err := br.ReadString('\n')
			if err != nil {
				t.Fatalf("reading status line: %v", err)
			}
			if status != "HTTP/1.1 101 Switching Protocols\r\n" {
				t.Fatalf("status line = %q, want %q", status, "HTTP/1.1 101 Switching Protocols\r\n")
			}
			hdr, err := textproto.NewReader(br).ReadMIMEHeader()
			if err != nil {
				t.Fatalf("reading headers: %v", err)
			}
			for k, want := range map[string]string{
				"Upgrade":              "websocket",
				"Connection":           "Upgrade",
				"Sec-Websocket-Accept": "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=",
			} {
				if got := hdr.Values(k); len(got) != 1 || got[0] != want {
					t.Errorf("header %s = %q, want exactly %q", k, got, want)
				}
			}
			for _, k := range []string{"Sec-Websocket-Protocol", "Sec-Websocket-Extensions"} {
				if v := hdr.Get(k); v != "" {
					t.Errorf("unexpected header %s: %q", k, v)
				}
			}

			pong := make([]byte, 4)
			if _, err := io.ReadFull(br, pong); err != nil {
				t.Fatalf("reading bytes from the handler: %v", err)
			}
			if string(pong) != "pong" {
				t.Errorf("client got %q, want %q", pong, "pong")
			}

			select {
			case r := <-results:
				if r.err != nil {
					t.Fatalf("handler: %v", r.err)
				}
				if r.got != "ping" {
					t.Errorf("handler got %q, want %q (bytes sent with the request were lost)", r.got, "ping")
				}
				if !r.netConn {
					t.Error("Conn.NetConn() returned nil")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("handler did not finish")
			}
		})
	}
}
