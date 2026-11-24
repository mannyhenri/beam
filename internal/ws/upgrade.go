package ws

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// acceptGUID is the fixed value RFC 6455 section 4.2.2 appends to the client's key.
const acceptGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Conn is one end of a WebSocket connection: the server side from Upgrade or
// the client side from Dial. Reads go through the buffer net/http already
// filled, so bytes that arrived with the handshake are not lost. Read and Write
// carry raw bytes; WriteFrame writes one frame, masked on the client side.
type Conn struct {
	nc     net.Conn
	r      io.Reader
	w      io.Writer
	closer io.Closer
	client bool

	writeMu sync.Mutex
}

// Read reads raw bytes from the peer.
func (c *Conn) Read(p []byte) (int, error) { return c.r.Read(p) }

// Write writes raw bytes to the peer.
func (c *Conn) Write(p []byte) (int, error) { return c.w.Write(p) }

// Close closes the connection. The caller owns it: http.Server's shutdown does
// not reach hijacked connections.
func (c *Conn) Close() error { return c.closer.Close() }

// NetConn returns the underlying network connection, for deadlines and addresses.
func (c *Conn) NetConn() net.Conn { return c.nc }

// Upgrade checks that r is a WebSocket upgrade request, answers 101 Switching
// Protocols and takes over the connection. On any problem it writes an HTTP
// error through w, returns an error and leaves the connection to net/http:
// 405 (with Allow: GET) for a method other than GET, 400 without the token
// websocket in Upgrade or Upgrade in Connection, 426 (with
// Sec-WebSocket-Version: 13) for any version but 13, 400 for a
// Sec-WebSocket-Key that is not base64 of 16 bytes, and 500 if w cannot be
// hijacked. No origin, subprotocol or extension is checked or offered.
func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		return nil, refuse(w, http.StatusMethodNotAllowed, "websocket upgrade needs GET")
	}
	if !hasToken(r.Header, "Upgrade", "websocket") {
		return nil, refuse(w, http.StatusBadRequest, "missing Upgrade: websocket")
	}
	if !hasToken(r.Header, "Connection", "Upgrade") {
		return nil, refuse(w, http.StatusBadRequest, "missing Connection: Upgrade")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		return nil, refuse(w, http.StatusUpgradeRequired, "unsupported Sec-WebSocket-Version")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if raw, err := base64.StdEncoding.DecodeString(key); err != nil || len(raw) != 16 {
		return nil, refuse(w, http.StatusBadRequest, "bad Sec-WebSocket-Key")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, refuse(w, http.StatusInternalServerError, "connection cannot be taken over")
	}

	nc, rw, err := hj.Hijack()
	if err != nil {
		return nil, fmt.Errorf("ws: hijack: %w", err)
	}
	// From here on w is dead: the response goes straight onto the connection.
	rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey(key) + "\r\n\r\n")
	if err := rw.Flush(); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ws: writing 101: %w", err)
	}
	return &Conn{nc: nc, r: rw.Reader, w: nc, closer: nc}, nil
}

// refuse writes an HTTP error and returns it as a Go error.
func refuse(w http.ResponseWriter, status int, msg string) error {
	http.Error(w, msg, status)
	return errors.New("ws: " + msg)
}

// hasToken reports whether any value of header name contains token in its
// comma-separated list, ignoring case.
func hasToken(h http.Header, name, token string) bool {
	for _, v := range h.Values(name) {
		for _, t := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(t), token) {
				return true
			}
		}
	}
	return false
}

// acceptKey computes Sec-WebSocket-Accept for a client key (RFC 6455 4.2.2).
func acceptKey(key string) string {
	sum := sha1.Sum([]byte(key + acceptGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}
