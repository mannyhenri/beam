package simnet

import (
	"net"
	"sync"
)

// Relay stands in for the Beam relay until it exists: a TCP listener on
// 127.0.0.1 that accepts connections and holds them open until Close.
type Relay struct {
	Addr string // 127.0.0.1:port

	ln    net.Listener
	mu    sync.Mutex
	conns []net.Conn
}

// StartRelay starts a relay stand-in on 127.0.0.1 with a free port.
func StartRelay() (*Relay, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	r := &Relay{Addr: ln.Addr().String(), ln: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			r.mu.Lock()
			r.conns = append(r.conns, c)
			r.mu.Unlock()
		}
	}()
	return r, nil
}

// Close stops listening and closes every held connection.
func (r *Relay) Close() error {
	err := r.ln.Close()
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.conns {
		c.Close()
	}
	r.conns = nil
	return err
}
