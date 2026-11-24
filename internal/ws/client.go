package ws

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
)

// DialConfig holds options for Dial. The zero value (or nil) is fine.
type DialConfig struct {
	// TLS is used for wss:// URLs; nil means the system roots and the URL's host name.
	TLS *tls.Config
}

// Dial opens a client WebSocket to a ws:// or wss:// URL. The upgrade is an
// ordinary net/http request, so the proxy named by HTTPS_PROXY (wss) or
// HTTP_PROXY (ws), and NO_PROXY, apply as for any web request; loopback
// addresses are never proxied (net/http's rule). The answer must be 101 with
// Upgrade: websocket, Connection: Upgrade and the Sec-WebSocket-Accept for the
// key sent; anything else closes the connection and returns an error. Use ctx
// for the handshake deadline.
func Dial(ctx context.Context, rawURL string, cfg *DialConfig) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("ws: %w", err)
	}
	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	default:
		return nil, fmt.Errorf("ws: URL %q must start with ws:// or wss://", rawURL)
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("ws: %w", err)
	}
	key := base64.StdEncoding.EncodeToString(raw)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("ws: %w", err)
	}
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", key)

	// Remember the TCP connection (to the relay, or to the proxy) for NetConn.
	var (
		mu  sync.Mutex
		tcp net.Conn
	)
	var d net.Dialer
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := d.DialContext(ctx, network, addr)
			if err == nil {
				mu.Lock()
				tcp = c
				mu.Unlock()
			}
			return c, err
		},
	}
	if cfg != nil && cfg.TLS != nil {
		tr.TLSClientConfig = cfg.TLS.Clone()
	}
	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("ws: %w", err)
	}
	fail := func(msg string) (*Conn, error) {
		resp.Body.Close()
		tr.CloseIdleConnections()
		return nil, errors.New("ws: " + msg)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fail(fmt.Sprintf("relay answered %d %s, want 101", resp.StatusCode, http.StatusText(resp.StatusCode)))
	}
	if !hasToken(resp.Header, "Upgrade", "websocket") {
		return fail("101 answer lacks Upgrade: websocket")
	}
	if !hasToken(resp.Header, "Connection", "Upgrade") {
		return fail("101 answer lacks Connection: Upgrade")
	}
	if resp.Header.Get("Sec-WebSocket-Accept") != acceptKey(key) {
		return fail("wrong Sec-WebSocket-Accept in the 101 answer")
	}
	rwc, ok := resp.Body.(io.ReadWriteCloser)
	if !ok {
		return fail("101 answer was not handed over as a connection")
	}
	mu.Lock()
	nc := tcp
	mu.Unlock()
	return &Conn{nc: nc, r: rwc, w: rwc, closer: rwc, client: true}, nil
}
