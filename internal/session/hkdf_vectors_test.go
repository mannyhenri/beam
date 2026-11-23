package session

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// hkdfVector is one HKDF-SHA256 test case from RFC 5869 Appendix A. Byte strings are hex,
// split where the RFC breaks its lines.
type hkdfVector struct {
	name            string
	ikm, salt, info string
	l               int
	prk, okm        string
}

var hkdfVectors = []hkdfVector{
	{
		name: "A.1",
		// RFC 5869 Appendix A, A.1 (Test Case 1): IKM, 22 octets.
		ikm: "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
		// RFC 5869 Appendix A, A.1 (Test Case 1): salt, 13 octets.
		salt: "000102030405060708090a0b0c",
		// RFC 5869 Appendix A, A.1 (Test Case 1): info, 10 octets.
		info: "f0f1f2f3f4f5f6f7f8f9",
		// RFC 5869 Appendix A, A.1 (Test Case 1): L.
		l: 42,
		// RFC 5869 Appendix A, A.1 (Test Case 1): PRK, 32 octets.
		prk: "077709362c2e32df0ddc3f0dc47bba63" +
			"90b6c73bb50f9c3122ec844ad7c2b3e5",
		// RFC 5869 Appendix A, A.1 (Test Case 1): OKM, 42 octets.
		okm: "3cb25f25faacd57a90434f64d0362f2a" +
			"2d2d0a90cf1a5a4c5db02d56ecc4c5bf" +
			"34007208d5b887185865",
	},
	{
		name: "A.2",
		// RFC 5869 Appendix A, A.2 (Test Case 2): IKM, 80 octets.
		ikm: "000102030405060708090a0b0c0d0e0f" +
			"101112131415161718191a1b1c1d1e1f" +
			"202122232425262728292a2b2c2d2e2f" +
			"303132333435363738393a3b3c3d3e3f" +
			"404142434445464748494a4b4c4d4e4f",
		// RFC 5869 Appendix A, A.2 (Test Case 2): salt, 80 octets.
		salt: "606162636465666768696a6b6c6d6e6f" +
			"707172737475767778797a7b7c7d7e7f" +
			"808182838485868788898a8b8c8d8e8f" +
			"909192939495969798999a9b9c9d9e9f" +
			"a0a1a2a3a4a5a6a7a8a9aaabacadaeaf",
		// RFC 5869 Appendix A, A.2 (Test Case 2): info, 80 octets.
		info: "b0b1b2b3b4b5b6b7b8b9babbbcbdbebf" +
			"c0c1c2c3c4c5c6c7c8c9cacbcccdcecf" +
			"d0d1d2d3d4d5d6d7d8d9dadbdcdddedf" +
			"e0e1e2e3e4e5e6e7e8e9eaebecedeeef" +
			"f0f1f2f3f4f5f6f7f8f9fafbfcfdfeff",
		// RFC 5869 Appendix A, A.2 (Test Case 2): L.
		l: 82,
		// RFC 5869 Appendix A, A.2 (Test Case 2): PRK, 32 octets.
		prk: "06a6b88c5853361a06104c9ceb35b45c" +
			"ef760014904671014a193f40c15fc244",
		// RFC 5869 Appendix A, A.2 (Test Case 2): OKM, 82 octets.
		okm: "b11e398dc80327a1c8e7f78c596a4934" +
			"4f012eda2d4efad8a050cc4c19afa97c" +
			"59045a99cac7827271cb41c65e590e09" +
			"da3275600c2f09b8367793a9aca3db71" +
			"cc30c58179ec3e87c14c01d5c1f3434f" +
			"1d87",
	},
	{
		name: "A.3",
		// RFC 5869 Appendix A, A.3 (Test Case 3): IKM, 22 octets.
		ikm: "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
		// RFC 5869 Appendix A, A.3 (Test Case 3): salt and info, 0 octets each.
		salt: "",
		info: "",
		// RFC 5869 Appendix A, A.3 (Test Case 3): L.
		l: 42,
		// RFC 5869 Appendix A, A.3 (Test Case 3): PRK, 32 octets.
		prk: "19ef24a32c717b167f33a91d6f648bdf" +
			"96596776afdb6377ac434c1c293ccb04",
		// RFC 5869 Appendix A, A.3 (Test Case 3): OKM, 42 octets.
		okm: "8da4e775a563c18f715f802a063c5a31" +
			"b8a11f5c5ee1879ec3454e5f3c738d2d" +
			"9d201395faa4b61a96c8",
	},
}

func decodeHex(t *testing.T, field, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("%s: bad hex: %v", field, err)
	}
	return b
}

// TestHKDFSHA256Vectors checks crypto/hkdf with SHA-256 against RFC 5869 Appendix A,
// test cases 1-3: Extract gives the PRK, Key gives the OKM, and Expand(PRK) gives the OKM.
func TestHKDFSHA256Vectors(t *testing.T) {
	for _, v := range hkdfVectors {
		t.Run(v.name, func(t *testing.T) {
			ikm := decodeHex(t, "IKM", v.ikm)
			salt := decodeHex(t, "salt", v.salt)
			info := string(decodeHex(t, "info", v.info))
			wantPRK := decodeHex(t, "PRK", v.prk)
			wantOKM := decodeHex(t, "OKM", v.okm)
			if len(wantOKM) != v.l {
				t.Fatalf("OKM is %d octets, L = %d", len(wantOKM), v.l)
			}

			prk, err := hkdf.Extract(sha256.New, ikm, salt)
			if err != nil {
				t.Fatalf("hkdf.Extract: %v", err)
			}
			if !bytes.Equal(prk, wantPRK) {
				t.Errorf("hkdf.Extract PRK = %x, want %x", prk, wantPRK)
			}

			okm, err := hkdf.Key(sha256.New, ikm, salt, info, v.l)
			if err != nil {
				t.Fatalf("hkdf.Key: %v", err)
			}
			if !bytes.Equal(okm, wantOKM) {
				t.Errorf("hkdf.Key OKM = %x, want %x", okm, wantOKM)
			}

			expanded, err := hkdf.Expand(sha256.New, prk, info, v.l)
			if err != nil {
				t.Fatalf("hkdf.Expand: %v", err)
			}
			if !bytes.Equal(expanded, wantOKM) {
				t.Errorf("hkdf.Expand(PRK) = %x, want OKM %x", expanded, wantOKM)
			}
		})
	}
}
