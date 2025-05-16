package hash

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestSedutilHashCompatibility(t *testing.T) {
	got := HashSedutilDTA("dummy", "S2RBNB0HA12200B")
	want := []byte{
		0x4f, 0x2a, 0xcc, 0xfd, 0x1a, 0x17, 0x64, 0xdc, 0x5b, 0x5b, 0xb3, 0x8f, 0x40, 0xf9, 0x06, 0x8d,
		0x2d, 0x1a, 0x1f, 0x6d, 0xd5, 0x39, 0x27, 0x07, 0xde, 0xa1, 0x4c, 0x3b, 0xb7, 0xde, 0xea, 0xcc,
	}
	if !bytes.Equal(want, got) {
		t.Errorf("Unexpected PBKDF2 hash, got %s want %s", hex.EncodeToString(got), hex.EncodeToString(want))
	}
}

func TestSedutilSha512(t *testing.T) {
	got := HashSedutil512("dummy", "S2RBNB0HA12200B")
	want := []byte{
		85, 196, 70, 116, 162, 150, 160, 93, 174, 31, 202, 3, 60, 245, 89, 141, 90, 6,
		213, 174, 233, 186, 186, 106, 59, 233, 12, 222, 253, 226, 174, 42,
	}
	if !bytes.Equal(want, got) {
		t.Errorf("Unexpected PBKDF2 hash, got %s want %s", hex.EncodeToString(got), hex.EncodeToString(want))
	}
}
