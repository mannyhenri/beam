// Package ws implements the WebSocket transport Beam uses to talk to the relay.
//
// Server side: Upgrade turns a valid RFC 6455 upgrade request into a *Conn, a
// raw connection taken over from net/http with http.Hijacker. Invalid requests
// get an ordinary HTTP error (405, 400, 426 or 500) and are not taken over.
//
// Client side: Dial sends the upgrade as an ordinary net/http request, so the
// proxy in HTTPS_PROXY (wss://) or HTTP_PROXY (ws://), and NO_PROXY, apply as
// for any web request; only environment variables are read. The 101 answer is
// checked (Upgrade, Connection, Sec-WebSocket-Accept) before the *Conn is
// returned.
//
// WriteFrame writes one final frame. Client frames are masked with a fresh
// random key per frame; server frames are not. Reading frames, ping/pong/close
// and the 1 MB cap belong to the framing feature.
package ws
