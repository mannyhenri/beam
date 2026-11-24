// Package ac3 pins AC-3 of ws-client: ws.Dial refuses a wrong upgrade answer — 200 OK,
// a 101 whose Sec-WebSocket-Accept belongs to another key, and a 101 without
// Upgrade: websocket — with an error and no Conn. A raw TCP fake relay sends each answer.
// It lives under testdata so go vet ./... and go test ./... skip it.
package ac3

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"beam/internal/ws"
)

func accept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// fakeRelay answers one upgrade request with answer(clientKey).
func fakeRelay(t *testing.T, answer func(key string) string) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(10 * time.Second))
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			return
		}
		io.WriteString(conn, answer(req.Header.Get("Sec-WebSocket-Key")))
		io.Copy(io.Discard, conn) // hold the connection until the client gives up
	}()
	return ln.Addr().String()
}

func TestAC3RefusesWrongUpgradeAnswers(t *testing.T) {
	cases := []struct {
		name    string
		answer  func(key string) string
		errWant string // the error must mention this (case-insensitive); "" = any error
	}{
		{"200 OK", func(string) string {
			return "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"
		}, "200"},
		{"101 with accept for another key", func(string) string {
			return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + accept("dGhlIHNhbXBsZSBub25jZQ==") + "\r\n\r\n"
		}, "Sec-WebSocket-Accept"},
		{"101 without Upgrade: websocket", func(key string) string {
			return "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\n" +
				"Sec-WebSocket-Accept: " + accept(key) + "\r\n\r\n"
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr := fakeRelay(t, tc.answer)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			c, err := ws.Dial(ctx, "ws://"+addr+"/", nil)
			if c != nil {
				c.Close()
				t.Error("Dial returned a Conn for a wrong upgrade answer")
			}
			if err == nil {
				t.Fatal("Dial returned no error for a wrong upgrade answer")
			}
			if tc.errWant != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.errWant)) {
				t.Errorf("error %q does not mention %q", err, tc.errWant)
			}
		})
	}
}
