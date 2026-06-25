package keyfile

import (
	"bytes"
	"testing"
)

func TestDeriveKEK(t *testing.T) {
	t.Parallel()
	salt := []byte("0123456789abcdef")
	k1 := deriveKEK([]byte("pass"), salt)
	k2 := deriveKEK([]byte("pass"), salt)
	if !bytes.Equal(k1, k2) {
		t.Fatal("deriveKEK must be deterministic for the same passphrase and salt")
	}
	if len(k1) != dataKeyLen {
		t.Fatalf("len = %d, want %d", len(k1), dataKeyLen)
	}
	if bytes.Equal(k1, deriveKEK([]byte("pass"), []byte("fedcba9876543210"))) {
		t.Fatal("different salt must yield a different key")
	}
	if bytes.Equal(k1, deriveKEK([]byte("other"), salt)) {
		t.Fatal("different passphrase must yield a different key")
	}
}
