package ws

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func dialCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// rawRelay answers each upgrade request with answer(request) and reports the request.
func rawRelay(t *testing.T, answer func(r *http.Request) string) (string, <-chan *http.Request) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	reqs := make(chan *http.Request, 8)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(10 * time.Second))
				r, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					return
				}
				reqs <- r
				io.WriteString(conn, answer(r))
				io.Copy(io.Discard, conn)
			}(conn)
		}
	}()
	return ln.Addr().String(), reqs
}

func goodAnswer(r *http.Request) string {
	return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n\r\n"
}

func TestDialSendsUpgradeRequestWithFreshKey(t *testing.T) {
	addr, reqs := rawRelay(t, goodAnswer)
	var keys []string
	for i := 0; i < 2; i++ {
		c, err := Dial(dialCtx(t), "ws://"+addr+"/c/abc?x=1", nil)
		if err != nil {
			t.Fatalf("Dial #%d: %v", i, err)
		}
		c.Close()
		r := <-reqs
		if r.Method != http.MethodGet || r.URL.Path != "/c/abc" || r.URL.RawQuery != "x=1" {
			t.Errorf("request = %s %s?%s, want GET /c/abc?x=1", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		if !hasToken(r.Header, "Upgrade", "websocket") || !hasToken(r.Header, "Connection", "Upgrade") {
			t.Errorf("request headers Upgrade=%q Connection=%q", r.Header.Get("Upgrade"), r.Header.Get("Connection"))
		}
		if v := r.Header.Get("Sec-WebSocket-Version"); v != "13" {
			t.Errorf("Sec-WebSocket-Version = %q, want 13", v)
		}
		key := r.Header.Get("Sec-WebSocket-Key")
		if raw, err := base64.StdEncoding.DecodeString(key); err != nil || len(raw) != 16 {
			t.Errorf("Sec-WebSocket-Key %q is not base64 of 16 bytes", key)
		}
		keys = append(keys, key)
	}
	if keys[0] == keys[1] {
		t.Errorf("two Dials used the same key %q", keys[0])
	}
}

func TestDialRefusesWrongAnswers(t *testing.T) {
	cases := []struct {
		name, want string
		answer     func(r *http.Request) string
	}{
		{"200 OK", "200", func(*http.Request) string { return "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n" }},
		{"403", "403", func(*http.Request) string { return "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n" }},
		{"wrong accept", "Sec-WebSocket-Accept", func(*http.Request) string {
			return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + acceptKey(sampleKey) + "\r\n\r\n"
		}},
		{"missing accept", "Sec-WebSocket-Accept", func(*http.Request) string {
			return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"
		}},
		{"no Upgrade header", "", func(r *http.Request) string {
			return "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + acceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n\r\n"
		}},
		{"Upgrade: h2c", "", func(r *http.Request) string {
			return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: h2c\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + acceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n\r\n"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr, _ := rawRelay(t, tc.answer)
			c, err := Dial(dialCtx(t), "ws://"+addr+"/", nil)
			if c != nil {
				c.Close()
				t.Error("Dial returned a Conn")
			}
			if err == nil {
				t.Fatal("Dial returned no error")
			}
			if tc.want != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestDialRejectsOtherSchemes(t *testing.T) {
	for _, u := range []string{"http://127.0.0.1:1/", "ftp://127.0.0.1/", "127.0.0.1:1"} {
		if c, err := Dial(dialCtx(t), u, nil); err == nil || c != nil {
			t.Errorf("Dial(%q) = %v, %v; want an error and no Conn", u, c, err)
		}
	}
}

// parsed is one frame as read off the wire.
type parsed struct {
	b0, b1  byte
	length  uint64
	key     []byte
	wire    []byte
	payload []byte
}

func readWireFrame(t *testing.T, r io.Reader) parsed {
	t.Helper()
	var f parsed
	h := make([]byte, 2)
	if _, err := io.ReadFull(r, h); err != nil {
		t.Fatalf("reading frame header: %v", err)
	}
	f.b0, f.b1 = h[0], h[1]
	f.length = uint64(h[1] & 0x7f)
	switch f.length {
	case 126:
		b := make([]byte, 2)
		io.ReadFull(r, b)
		f.length = uint64(binary.BigEndian.Uint16(b))
	case 127:
		b := make([]byte, 8)
		io.ReadFull(r, b)
		f.length = binary.BigEndian.Uint64(b)
	}
	if h[1]&0x80 != 0 {
		f.key = make([]byte, 4)
		io.ReadFull(r, f.key)
	}
	f.wire = make([]byte, f.length)
	if _, err := io.ReadFull(r, f.wire); err != nil {
		t.Fatalf("reading payload: %v", err)
	}
	f.payload = append([]byte(nil), f.wire...)
	for i := range f.payload {
		if f.key != nil {
			f.payload[i] ^= f.key[i%4]
		}
	}
	return f
}

// pair dials an Upgrade server and returns the client Conn and the server Conn.
func pair(t *testing.T) (client, server *Conn) {
	t.Helper()
	srvConn := make(chan *Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Upgrade(w, r)
		if err != nil {
			srvConn <- nil
			return
		}
		srvConn <- c
	}))
	t.Cleanup(srv.Close)
	c, err := Dial(dialCtx(t), "ws://"+srv.Listener.Addr().String()+"/", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	s := <-srvConn
	if s == nil {
		t.Fatal("server Upgrade failed")
	}
	t.Cleanup(func() { s.Close() })
	return c, s
}

func TestClientFramesLengthFormsAndMasking(t *testing.T) {
	client, server := pair(t)
	cases := []struct {
		n    int
		form byte
	}{{0, 0}, {1, 1}, {125, 125}, {126, 126}, {65535, 126}, {65536, 127}}
	for _, tc := range cases {
		p := make([]byte, tc.n)
		for i := range p {
			p[i] = byte(i*13 + 7)
		}
		before := append([]byte(nil), p...)
		errc := make(chan error, 1)
		go func() { errc <- client.WriteFrame(OpText, p) }()
		f := readWireFrame(t, server)
		if err := <-errc; err != nil {
			t.Fatalf("WriteFrame(%d bytes): %v", tc.n, err)
		}
		if f.b0 != 0x80|OpText {
			t.Errorf("%d bytes: first byte %#x, want FIN|OpText %#x", tc.n, f.b0, 0x80|OpText)
		}
		if f.b1&0x80 == 0 {
			t.Errorf("%d bytes: MASK bit not set", tc.n)
		}
		if f.b1&0x7f != tc.form {
			t.Errorf("%d bytes: length form %d, want %d", tc.n, f.b1&0x7f, tc.form)
		}
		if f.length != uint64(tc.n) || !bytes.Equal(f.payload, before) {
			t.Errorf("%d bytes: payload does not round-trip (length %d)", tc.n, f.length)
		}
		if tc.n >= 4 && bytes.Equal(f.wire, before) {
			t.Errorf("%d bytes: payload sent unmasked", tc.n)
		}
		if !bytes.Equal(p, before) {
			t.Errorf("%d bytes: WriteFrame modified the caller's slice", tc.n)
		}
	}
}

func TestClientUsesFreshKeyPerFrame(t *testing.T) {
	client, server := pair(t)
	keys := map[string]bool{}
	p := bytes.Repeat([]byte{0x42}, 8)
	for i := 0; i < 20; i++ {
		go client.WriteFrame(OpBinary, p)
		f := readWireFrame(t, server)
		if f.key == nil {
			t.Fatalf("frame %d: no masking key", i)
		}
		keys[string(f.key)] = true
	}
	if len(keys) < 2 {
		t.Errorf("20 frames used %d distinct key(s); each frame needs a fresh key", len(keys))
	}
}

func TestServerFramesAreNotMasked(t *testing.T) {
	client, server := pair(t)
	p := []byte("from the relay")
	errc := make(chan error, 1)
	go func() { errc <- server.WriteFrame(OpBinary, p) }()
	f := readWireFrame(t, client)
	if err := <-errc; err != nil {
		t.Fatalf("server WriteFrame: %v", err)
	}
	if f.b0 != 0x80|OpBinary {
		t.Errorf("first byte %#x, want %#x", f.b0, 0x80|OpBinary)
	}
	if f.b1&0x80 != 0 || f.key != nil {
		t.Error("server frame has the MASK bit set (RFC 6455 forbids server masking)")
	}
	if !bytes.Equal(f.wire, p) {
		t.Errorf("server payload %q, want %q", f.wire, p)
	}
}
