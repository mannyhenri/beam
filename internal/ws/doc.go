// Package ws implements the WebSocket framing Beam uses to talk to the relay.
//
// Server side: Upgrade turns a valid RFC 6455 upgrade request into a *Conn, a
// raw connection taken over from net/http with http.Hijacker. Invalid requests
// get an ordinary HTTP error (405, 400, 426 or 500) and are not taken over.
// Frames (opcodes, masking, ping/pong, close, the 1 MB cap) and the client side
// are separate features built on top of Conn.
package ws
