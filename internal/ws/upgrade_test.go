package ws

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

const sampleKey = "dGhlIHNhbXBsZSBub25jZQ==" // RFC 6455 section 1.3

func TestAcceptKeyRFCSample(t *testing.T) {
	if got, want := acceptKey(sampleKey), "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="; got != want {
		t.Fatalf("acceptKey(%q) = %q, want %q", sampleKey, got, want)
	}
}

func validRequest() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Sec-WebSocket-Version", "13")
	r.Header.Set("Sec-WebSocket-Key", sampleKey)
	return r
}

// Refusals happen before any hijack, so a ResponseRecorder (which cannot hijack) is enough.
func TestUpgradeRefusals(t *testing.T) {
	cases := []struct {
		name   string
		edit   func(r *http.Request)
		status int
		header map[string]string // headers the refusal must carry
	}{
		{"no Upgrade header", func(r *http.Request) { r.Header.Del("Upgrade") }, 400, nil},
		{"Upgrade: h2c", func(r *http.Request) { r.Header.Set("Upgrade", "h2c") }, 400, nil},
		{"no Connection: Upgrade", func(r *http.Request) { r.Header.Set("Connection", "keep-alive") }, 400, nil},
		{"no Connection header", func(r *http.Request) { r.Header.Del("Connection") }, 400, nil},
		{"no Sec-WebSocket-Key", func(r *http.Request) { r.Header.Del("Sec-WebSocket-Key") }, 400, nil},
		{"key decodes to 5 bytes", func(r *http.Request) { r.Header.Set("Sec-WebSocket-Key", "c2hvcnQ=") }, 400, nil},
		{"key is not base64", func(r *http.Request) { r.Header.Set("Sec-WebSocket-Key", "not base64 at all!!!!!!!") }, 400, nil},
		{"version 8", func(r *http.Request) { r.Header.Set("Sec-WebSocket-Version", "8") }, 426, map[string]string{"Sec-WebSocket-Version": "13"}},
		{"no version", func(r *http.Request) { r.Header.Del("Sec-WebSocket-Version") }, 426, map[string]string{"Sec-WebSocket-Version": "13"}},
		{"method POST", func(r *http.Request) { r.Method = http.MethodPost }, 405, map[string]string{"Allow": "GET"}},
		// Order of checks: method first, then version before key.
		{"POST with bad headers is 405", func(r *http.Request) { r.Method = http.MethodPost; r.Header.Del("Upgrade") }, 405, map[string]string{"Allow": "GET"}},
		{"version 8 with bad key is 426", func(r *http.Request) {
			r.Header.Set("Sec-WebSocket-Version", "8")
			r.Header.Set("Sec-WebSocket-Key", "c2hvcnQ=")
		}, 426, map[string]string{"Sec-WebSocket-Version": "13"}},
		// A valid request through a writer that cannot hijack: 500, no panic.
		{"writer without Hijacker", func(r *http.Request) {}, 500, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validRequest()
			tc.edit(r)
			w := httptest.NewRecorder()
			c, err := Upgrade(w, r)
			if err == nil {
				t.Error("Upgrade returned no error")
			}
			if c != nil {
				t.Error("Upgrade returned a Conn")
			}
			if w.Code != tc.status {
				t.Errorf("status = %d, want %d", w.Code, tc.status)
			}
			for k, v := range tc.header {
				if got := w.Header().Get(k); got != v {
					t.Errorf("header %s = %q, want %q", k, got, v)
				}
			}
			if got := w.Header().Get("Sec-WebSocket-Accept"); got != "" {
				t.Errorf("refusal carries Sec-WebSocket-Accept %q", got)
			}
		})
	}
}

// dialUpgrade sends a raw upgrade request with extra bytes in the same write and returns
// the status line, the response headers and a reader positioned after them.
func dialUpgrade(t *testing.T, addr string, headerLines []string, extra string) (string, textproto.MIMEHeader, *bufio.Reader, net.Conn) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	req := "GET / HTTP/1.1\r\nHost: " + addr + "\r\n" + strings.Join(headerLines, "\r\n") + "\r\n\r\n" + extra
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	status, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("reading status line: %v", err)
	}
	hdr, err := textproto.NewReader(br).ReadMIMEHeader()
	if err != nil {
		t.Fatalf("reading headers: %v", err)
	}
	return status, hdr, br, conn
}

func TestUpgradeHandsOverConnection(t *testing.T) {
	variants := []struct {
		name  string
		lines []string
	}{
		{"canonical", []string{"Upgrade: websocket", "Connection: Upgrade"}},
		{"mixed case", []string{"Upgrade: WebSocket", "Connection: upgrade"}},
		{"token list", []string{"Upgrade: websocket", "Connection: keep-alive, Upgrade"}},
		{"repeated Connection header", []string{"Upgrade: websocket", "Connection: keep-alive", "Connection: Upgrade"}},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			got := make(chan string, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := Upgrade(w, r)
				if err != nil {
					got <- "upgrade error: " + err.Error()
					return
				}
				defer c.Close()
				if c.NetConn() == nil {
					got <- "NetConn() is nil"
					return
				}
				buf := make([]byte, 4)
				if _, err := io.ReadFull(c, buf); err != nil {
					got <- "read: " + err.Error()
					return
				}
				c.Write([]byte("pong"))
				got <- string(buf)
			}))
			defer srv.Close()

			lines := append(v.lines, "Sec-WebSocket-Version: 13", "Sec-WebSocket-Key: "+sampleKey)
			status, hdr, br, _ := dialUpgrade(t, srv.Listener.Addr().String(), lines, "ping")
			if status != "HTTP/1.1 101 Switching Protocols\r\n" {
				t.Fatalf("status line = %q", status)
			}
			for k, want := range map[string]string{"Upgrade": "websocket", "Connection": "Upgrade", "Sec-Websocket-Accept": "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="} {
				if vals := hdr.Values(k); len(vals) != 1 || vals[0] != want {
					t.Errorf("header %s = %q, want exactly %q", k, vals, want)
				}
			}
			pong := make([]byte, 4)
			if _, err := io.ReadFull(br, pong); err != nil || string(pong) != "pong" {
				t.Errorf("client read %q, %v; want pong", pong, err)
			}
			select {
			case s := <-got:
				if s != "ping" {
					t.Errorf("handler: %s (want it to read ping)", s)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("handler did not finish")
			}
		})
	}
}

func TestConnCloseEndsTheConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := Upgrade(w, r); err == nil {
			c.Close()
		}
	}))
	defer srv.Close()
	_, _, br, _ := dialUpgrade(t, srv.Listener.Addr().String(),
		[]string{"Upgrade: websocket", "Connection: Upgrade", "Sec-WebSocket-Version: 13", "Sec-WebSocket-Key: " + sampleKey}, "")
	if _, err := br.ReadByte(); err != io.EOF {
		t.Fatalf("after Close the client read err = %v, want EOF", err)
	}
}
