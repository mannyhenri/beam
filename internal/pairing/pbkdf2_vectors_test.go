package pairing

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// pbkdf2Vector is one PBKDF2-HMAC-SHA256 test vector from RFC 7914 section 11. dk is hex
// copied as the RFC prints it, space separated, one RFC line per string.
type pbkdf2Vector struct {
	password, salt string
	iterations     int
	dkLen          int
	dk             string
}

var pbkdf2Vectors = []pbkdf2Vector{
	// RFC 7914 section 11, first vector:
	// PBKDF2-HMAC-SHA-256 (P="passwd", S="salt", c=1, dkLen=64).
	{
		password:   "passwd",
		salt:       "salt",
		iterations: 1,
		dkLen:      64,
		dk: "55 ac 04 6e 56 e3 08 9f ec 16 91 c2 25 44 b6 05" +
			"f9 41 85 21 6d de 04 65 e6 8b 9d 57 c2 0d ac bc" +
			"49 ca 9c cc f1 79 b6 45 99 16 64 b3 9d 77 ef 31" +
			"7c 71 b8 45 b1 e3 0b d5 09 11 20 41 d3 a1 97 83",
	},
	// RFC 7914 section 11, second vector:
	// PBKDF2-HMAC-SHA-256 (P="Password", S="NaCl", c=80000, dkLen=64).
	{
		password:   "Password",
		salt:       "NaCl",
		iterations: 80000,
		dkLen:      64,
		dk: "4d dc d8 f6 0b 98 be 21 83 0c ee 5e f2 27 01 f9" +
			"64 1a 44 18 d0 4c 04 14 ae ff 08 87 6b 34 ab 56" +
			"a1 d4 25 a1 22 58 33 54 9a db 84 1b 51 c9 b3 17" +
			"6a 27 2b de bb a1 d0 78 47 8f 62 b3 97 f3 3c 8d",
	},
}

// TestPBKDF2SHA256Vectors checks crypto/pbkdf2 with SHA-256 against both PBKDF2-HMAC-SHA256
// vectors in RFC 7914 section 11, byte for byte.
func TestPBKDF2SHA256Vectors(t *testing.T) {
	for _, v := range pbkdf2Vectors {
		want, err := hex.DecodeString(strings.ReplaceAll(v.dk, " ", ""))
		if err != nil {
			t.Fatalf("P=%q c=%d: bad hex: %v", v.password, v.iterations, err)
		}
		if len(want) != v.dkLen {
			t.Fatalf("P=%q c=%d: DK is %d octets, dkLen = %d", v.password, v.iterations, len(want), v.dkLen)
		}
		dk, err := pbkdf2.Key(sha256.New, v.password, []byte(v.salt), v.iterations, v.dkLen)
		if err != nil {
			t.Fatalf("P=%q c=%d: pbkdf2.Key: %v", v.password, v.iterations, err)
		}
		if !bytes.Equal(dk, want) {
			t.Errorf("P=%q S=%q c=%d: DK = %x, want %x", v.password, v.salt, v.iterations, dk, want)
		}
	}
}
