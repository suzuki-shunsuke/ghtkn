package tokenstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// testDataKey returns a deterministic 32-byte key for tests.
func testDataKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestStore_diskPersistence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	key := testDataKey(t)
	token := json.RawMessage(`{"access_token":"abc"}`)

	if err := New(key, dir).Set("Iv1.abc", token); err != nil {
		t.Fatal(err)
	}

	// The on-disk file must not contain the plaintext token.
	blob, err := os.ReadFile(filepath.Join(dir, "Iv1.abc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) == string(token) {
		t.Fatal("token file must be encrypted")
	}

	// A fresh store with the same key must decrypt the token from disk.
	got, ok, err := New(key, dir).Get("Iv1.abc")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("token must be found on disk")
	}
	if string(got) != string(token) {
		t.Fatalf("token = %q, want %q", got, token)
	}
}

func TestStore_getMissing(t *testing.T) {
	t.Parallel()
	got, ok, err := New(testDataKey(t), t.TempDir()).Get("Iv1.absent")
	if err != nil {
		t.Fatal(err)
	}
	if ok || got != nil {
		t.Fatalf("missing token must be (nil,false), got (%q,%v)", got, ok)
	}
}

func TestStore_wrongKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := New(testDataKey(t), dir).Set("Iv1.abc", json.RawMessage(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, 32)
	if _, _, err := New(wrong, dir).Get("Iv1.abc"); err == nil {
		t.Fatal("decrypting with the wrong key must fail")
	}
}

func TestStore_invalidClientID(t *testing.T) {
	t.Parallel()
	s := New(testDataKey(t), t.TempDir())
	if _, _, err := s.Get("../escape"); err == nil {
		t.Fatal("Get must reject an invalid client id")
	}
	if err := s.Set("a/b", json.RawMessage(`{}`)); err == nil {
		t.Fatal("Set must reject an invalid client id")
	}
}

func TestStore_delete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	key := testDataKey(t)
	s := New(key, dir)

	// Deleting a client ID with no cached token is a no-op.
	if err := s.Delete("Iv1.absent"); err != nil {
		t.Fatalf("Delete on miss must succeed, got %v", err)
	}

	if err := s.Set("Iv1.abc", json.RawMessage(`{"access_token":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("Iv1.abc"); err != nil {
		t.Fatal(err)
	}
	// The token file must be gone and a fresh store must not find it.
	if _, err := os.Stat(filepath.Join(dir, "Iv1.abc")); !os.IsNotExist(err) {
		t.Fatalf("token file must be removed, stat err = %v", err)
	}
	if _, ok, err := New(key, dir).Get("Iv1.abc"); err != nil || ok {
		t.Fatalf("deleted token must be gone, got ok=%v err=%v", ok, err)
	}

	// Delete rejects an invalid client ID.
	if err := s.Delete("../escape"); err == nil {
		t.Fatal("Delete must reject an invalid client id")
	}
}

func TestStore_lenCountsDiskFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	key := testDataKey(t)
	s := New(key, dir)
	if err := s.Set("Iv1.a", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("Iv1.b", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	// A leftover temp file and an invalid name must be ignored.
	if err := os.WriteFile(filepath.Join(dir, ".ghtkn-tmp-xyz"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := s.Len(); got != 2 {
		t.Fatalf("Len = %d, want 2", got)
	}
}

func TestValidClientID(t *testing.T) {
	t.Parallel()
	data := map[string]bool{
		"Iv1.abc": true,
		"Iv23xyz": true,
		"a_b-c.d": true,
		"":        false,
		".":       false,
		"..":      false,
		"a/b":     false,
		"a\x00b":  false,
		"a b":     false,
	}
	for id, want := range data {
		if got := validClientID(id); got != want {
			t.Errorf("validClientID(%q) = %v, want %v", id, got, want)
		}
	}
}
