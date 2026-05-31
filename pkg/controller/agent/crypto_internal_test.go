package agent

import (
	"bytes"
	"errors"
	"testing"
)

// testDataKey returns a deterministic 32-byte key for tests.
func testDataKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, dataKeyLen)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestSealOpen_roundTrip(t *testing.T) {
	t.Parallel()
	key := testDataKey(t)
	plaintext := []byte(`{"access_token":"secret"}`)
	blob, err := seal(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(blob, plaintext) {
		t.Fatal("ciphertext must not contain the plaintext")
	}
	got, err := open(key, blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("open = %q, want %q", got, plaintext)
	}
}

func TestOpen_wrongKey(t *testing.T) {
	t.Parallel()
	blob, err := seal(testDataKey(t), []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, dataKeyLen)
	if _, err := open(wrong, blob); !errors.Is(err, errDecrypt) {
		t.Fatalf("err = %v, want errDecrypt", err)
	}
}

func TestOpen_tooShort(t *testing.T) {
	t.Parallel()
	if _, err := open(testDataKey(t), []byte{1, 2, 3}); !errors.Is(err, errDecrypt) {
		t.Fatalf("err = %v, want errDecrypt", err)
	}
}

func TestSeal_uniqueNonce(t *testing.T) {
	t.Parallel()
	key := testDataKey(t)
	a, err := seal(key, []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := seal(key, []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("two seals of the same plaintext must differ (random nonce)")
	}
}

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
