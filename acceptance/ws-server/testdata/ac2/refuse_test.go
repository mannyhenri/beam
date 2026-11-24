// Package ac2 pins AC-2 of ws-server: every request with a missing or wrong upgrade header
// is refused with the stated HTTP status, ws.Upgrade returns an error and no Conn, and no
// refused response looks like an upgrade. It lives under testdata so go vet ./... and
// go test ./... skip it; the AC-2 Node test runs it by path.
package ac2

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"beam/internal/ws"
)

type outcome struct {
	err    error
	gotNil bool
}

func TestAC2RefusesBadUpgrades(t *testing.T) {
	valid := map[string]string{
		"Upgrade":               "websocket",
		"Connection":            "Upgrade",
		"Sec-WebSocket-Version": "13",
		"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
	}
	cases := []struct {
		name    string
		method  string
		change  map[string]string // "" removes the header
		status  int
		extraOK func(h http.Header) string
	}{
		{"no Upgrade header", "GET", map[string]string{"Upgrade": ""}, 400, nil},
		{"Upgrade: h2c", "GET", map[string]string{"Upgrade": "h2c"}, 400, nil},
		{"no Connection: Upgrade", "GET", map[string]string{"Connection": "keep-alive"}, 400, nil},
		{"no Sec-WebSocket-Key", "GET", map[string]string{"Sec-WebSocket-Key": ""}, 400, nil},
		{"key decodes to 5 bytes", "GET", map[string]string{"Sec-WebSocket-Key": "c2hvcnQ="}, 400, nil},
		{"version 8", "GET", map[string]string{"Sec-WebSocket-Version": "8"}, 426, func(h http.Header) string {
			if v := h.Values("Sec-Websocket-Version"); len(v) != 1 || v[0] != "13" {
				return "426 must carry Sec-WebSocket-Version: 13, got " + strings.Join(v, ",")
			}
			return ""
		}},
		{"method POST", "POST", nil, 405, func(h http.Header) string {
			if !strings.Contains(h.Get("Allow"), "GET") {
				return "405 must carry Allow: GET, got " + h.Get("Allow")
			}
			return ""
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcomes := make(chan outcome, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, err := ws.Upgrade(w, r)
				if c != nil {
					c.Close()
				}
				outcomes <- outcome{err: err, gotNil: c == nil}
			}))
			defer srv.Close()

			headers := map[string]string{}
			for k, v := range valid {
				headers[k] = v
			}
			for k, v := range tc.change {
				if v == "" {
					delete(headers, k)
				} else {
					headers[k] = v
				}
			}
			var b strings.Builder
			b.WriteString(tc.method + " / HTTP/1.1\r\nHost: " + srv.Listener.Addr().String() + "\r\n")
			for k, v := range headers {
				b.WriteString(k + ": " + v + "\r\n")
			}
			if tc.method == "POST" {
				b.WriteString("Content-Length: 0\r\n")
			}
			b.WriteString("\r\n")

			conn, err := net.DialTimeout("tcp", srv.Listener.Addr().String(), 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(10 * time.Second))
			if _, err := io.WriteString(conn, b.String()); err != nil {
				t.Fatal(err)
			}
			resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
			if err != nil {
				t.Fatalf("no HTTP response: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != tc.status {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.status)
			}
			if resp.StatusCode == http.StatusSwitchingProtocols {
				t.Error("a refused request must never get 101")
			}
			if v := resp.Header.Get("Sec-Websocket-Accept"); v != "" {
				t.Errorf("refused response carries Sec-WebSocket-Accept: %q", v)
			}
			if tc.extraOK != nil {
				if msg := tc.extraOK(resp.Header); msg != "" {
					t.Error(msg)
				}
			}

			select {
			case o := <-outcomes:
				if o.err == nil {
					t.Error("ws.Upgrade returned no error for a refused request")
				}
				if !o.gotNil {
					t.Error("ws.Upgrade returned a Conn for a refused request")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("handler did not finish")
			}
		})
	}
}
