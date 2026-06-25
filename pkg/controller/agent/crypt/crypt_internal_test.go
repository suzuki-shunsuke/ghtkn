package crypt

import (
	"bytes"
	"errors"
	"testing"
)

// testKey returns a deterministic 32-byte key for tests.
func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestSealOpen_roundTrip(t *testing.T) {
	t.Parallel()
	key := testKey(t)
	plaintext := []byte(`{"access_token":"secret"}`)
	blob, err := Seal(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(blob, plaintext) {
		t.Fatal("ciphertext must not contain the plaintext")
	}
	got, err := Open(key, blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("Open = %q, want %q", got, plaintext)
	}
}

func TestOpen_wrongKey(t *testing.T) {
	t.Parallel()
	blob, err := Seal(testKey(t), []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, 32)
	if _, err := Open(wrong, blob); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("err = %v, want ErrDecrypt", err)
	}
}

func TestOpen_tooShort(t *testing.T) {
	t.Parallel()
	if _, err := Open(testKey(t), []byte{1, 2, 3}); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("err = %v, want ErrDecrypt", err)
	}
}

func TestSeal_uniqueNonce(t *testing.T) {
	t.Parallel()
	key := testKey(t)
	a, err := Seal(key, []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Seal(key, []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("two seals of the same plaintext must differ (random nonce)")
	}
}
