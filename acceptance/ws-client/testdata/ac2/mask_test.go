// Package ac2 pins AC-2 of ws-client: every frame a client writes has FIN=1, the given
// opcode, the MASK bit, the right length form and a fresh key, and unmasks to the
// payload. The relay side parses the raw bytes itself (no frame reader exists yet).
// It lives under testdata so go vet ./... and go test ./... skip it.
package ac2

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"beam/internal/ws"
)

type frame struct {
	fin, masked bool
	opcode      byte
	lenForm     byte // 7-bit value: the length itself, 126 or 127
	key         [4]byte
	wire        []byte // payload as sent
	payload     []byte // after unmasking
	err         error
}

func readFrame(r io.Reader) frame {
	var f frame
	hdr := make([]byte, 2)
	if _, f.err = io.ReadFull(r, hdr); f.err != nil {
		return f
	}
	f.fin = hdr[0]&0x80 != 0
	f.opcode = hdr[0] & 0x0f
	f.masked = hdr[1]&0x80 != 0
	f.lenForm = hdr[1] & 0x7f
	n := uint64(f.lenForm)
	switch f.lenForm {
	case 126:
		b := make([]byte, 2)
		if _, f.err = io.ReadFull(r, b); f.err != nil {
			return f
		}
		n = uint64(binary.BigEndian.Uint16(b))
	case 127:
		b := make([]byte, 8)
		if _, f.err = io.ReadFull(r, b); f.err != nil {
			return f
		}
		n = binary.BigEndian.Uint64(b)
	}
	if f.masked {
		if _, f.err = io.ReadFull(r, f.key[:]); f.err != nil {
			return f
		}
	}
	f.wire = make([]byte, n)
	if _, f.err = io.ReadFull(r, f.wire); f.err != nil {
		return f
	}
	f.payload = make([]byte, n)
	for i := range f.wire {
		f.payload[i] = f.wire[i] ^ f.key[i%4]
	}
	return f
}

func payloadOf(n int) []byte {
	p := make([]byte, n)
	for i := range p {
		p[i] = byte(i*7 + 1)
	}
	return p
}

func TestAC2ClientFramesAreMasked(t *testing.T) {
	sizes := []int{0, 1, 125, 126, 65535, 65536}
	const repeats = 20
	total := len(sizes) + repeats

	frames := make(chan frame, total)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := ws.Upgrade(w, r)
		if err != nil {
			frames <- frame{err: err}
			return
		}
		defer c.Close()
		c.NetConn().SetDeadline(time.Now().Add(20 * time.Second))
		for i := 0; i < total; i++ {
			f := readFrame(c)
			frames <- f
			if f.err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c, err := ws.Dial(ctx, "ws://"+srv.Listener.Addr().String()+"/", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	var sent [][]byte
	for _, n := range sizes {
		sent = append(sent, payloadOf(n))
	}
	for i := 0; i < repeats; i++ {
		sent = append(sent, bytes.Repeat([]byte{0xAB}, 16))
	}
	for i, p := range sent {
		before := append([]byte(nil), p...)
		if err := c.WriteFrame(ws.OpBinary, p); err != nil {
			t.Fatalf("WriteFrame #%d (%d bytes): %v", i, len(p), err)
		}
		if !bytes.Equal(p, before) {
			t.Fatalf("WriteFrame #%d modified the caller's slice", i)
		}
	}

	keys := map[[4]byte]bool{}
	for i, p := range sent {
		var f frame
		select {
		case f = <-frames:
		case <-time.After(20 * time.Second):
			t.Fatalf("relay did not receive frame #%d", i)
		}
		if f.err != nil {
			t.Fatalf("frame #%d: %v", i, f.err)
		}
		if !f.fin || f.opcode != ws.OpBinary || !f.masked {
			t.Errorf("frame #%d: fin=%v opcode=%d masked=%v, want true 2 true", i, f.fin, f.opcode, f.masked)
		}
		var wantForm byte
		switch n := len(p); {
		case n <= 125:
			wantForm = byte(n)
		case n <= 65535:
			wantForm = 126
		default:
			wantForm = 127
		}
		if f.lenForm != wantForm {
			t.Errorf("frame #%d (%d bytes): length form %d, want %d", i, len(p), f.lenForm, wantForm)
		}
		if !bytes.Equal(f.payload, p) {
			t.Errorf("frame #%d (%d bytes): unmasked payload differs from what was sent", i, len(p))
		}
		// With 4+ bytes the wire equals the payload only if the whole key is zero (1 in 2^32);
		// a 1-byte payload is skipped here, since a zero first key byte is a legal 1 in 256.
		if len(p) >= 4 && bytes.Equal(f.wire, p) {
			t.Errorf("frame #%d (%d bytes): payload went over the wire unmasked", i, len(p))
		}
		if i >= len(sizes) {
			keys[f.key] = true
		}
	}
	if len(keys) < 2 {
		t.Errorf("all %d masking keys were the same: %v; each frame needs a fresh key", repeats, keys)
	}
}
