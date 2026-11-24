package ws

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// Opcodes for WriteFrame (RFC 6455 section 5.2).
const (
	OpText   = 1
	OpBinary = 2
)

// WriteFrame writes payload as one final frame with the given opcode. A client
// Conn masks every frame with a fresh random key; a server Conn never masks
// (RFC 6455 section 5.1). The caller's payload is not modified. Reading frames,
// control frames and the 1 MB cap belong to the framing feature.
func (c *Conn) WriteFrame(opcode byte, payload []byte) error {
	n := len(payload)
	frame := make([]byte, 0, 14+n)
	frame = append(frame, 0x80|opcode&0x0f)
	var maskBit byte
	if c.client {
		maskBit = 0x80
	}
	switch {
	case n <= 125:
		frame = append(frame, maskBit|byte(n))
	case n <= 0xffff:
		frame = append(frame, maskBit|126)
		frame = binary.BigEndian.AppendUint16(frame, uint16(n))
	default:
		frame = append(frame, maskBit|127)
		frame = binary.BigEndian.AppendUint64(frame, uint64(n))
	}
	if c.client {
		var key [4]byte
		if _, err := rand.Read(key[:]); err != nil {
			return fmt.Errorf("ws: masking key: %w", err)
		}
		frame = append(frame, key[:]...)
		for i, b := range payload {
			frame = append(frame, b^key[i%4])
		}
	} else {
		frame = append(frame, payload...)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err := c.w.Write(frame)
	return err
}
