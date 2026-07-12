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

// TestStore_diskModeNoPlaintextCache verifies that in disk mode the store never retains
// the decrypted token in memory: neither Set nor Get populates the in-memory map, so a
// memory dump cannot yield a plaintext access/refresh token from the cache.
func TestStore_diskModeNoPlaintextCache(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := New(testDataKey(t), dir)
	token := json.RawMessage(`{"access_token":"ghu_secret","expiration_date":"2999-01-01T00:00:00Z"}`)

	if err := s.Set("Iv1.abc", token); err != nil {
		t.Fatal(err)
	}
	if len(s.tokens) != 0 {
		t.Fatalf("Set must not cache the plaintext in disk mode; map has %d entries", len(s.tokens))
	}

	got, ok, err := s.Get("Iv1.abc")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if string(got) != string(token) {
		t.Fatalf("Get returned %q, want %q", got, token)
	}
	if len(s.tokens) != 0 {
		t.Fatalf("Get must not cache the plaintext in disk mode; map has %d entries", len(s.tokens))
	}
}

// TestStore_getReturnsCallerOwnedBuffer verifies that the token Get returns is a private
// copy the caller may scrub, so zeroing it (to shorten the plaintext's lifetime) does not
// corrupt the store's data for a later read.
func TestStore_getReturnsCallerOwnedBuffer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := New(testDataKey(t), dir)
	token := json.RawMessage(`{"access_token":"ghu_secret","expiration_date":"2999-01-01T00:00:00Z"}`)
	if err := s.Set("Iv1.abc", token); err != nil {
		t.Fatal(err)
	}

	got, ok, err := s.Get("Iv1.abc")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	for i := range got {
		got[i] = 0
	}

	again, ok, err := s.Get("Iv1.abc")
	if err != nil || !ok {
		t.Fatalf("second Get: ok=%v err=%v", ok, err)
	}
	if string(again) != string(token) {
		t.Fatalf("scrubbing a returned token corrupted the store: got %q", again)
	}
}

// TestStore_memoryModeOwnsBuffers verifies that in memory-only mode (New(nil, "")) the
// store neither aliases the caller's Set buffer nor hands out its own on Get: scrubbing
// the buffer passed to Set, and scrubbing a buffer returned by Get, both leave the stored
// token intact. Disk mode gets this for free (Set re-encrypts, Get re-decrypts), so this
// exercises the memory-mode copy branch specifically.
func TestStore_memoryModeOwnsBuffers(t *testing.T) {
	t.Parallel()
	s := New(nil, "")
	const want = `{"access_token":"ghu_secret","expiration_date":"2999-01-01T00:00:00Z"}`

	// Set must copy: scrubbing the caller's buffer must not corrupt the stored token.
	token := json.RawMessage(want)
	if err := s.Set("Iv1.abc", token); err != nil {
		t.Fatal(err)
	}
	for i := range token {
		token[i] = 0
	}

	// Get must copy: scrubbing a returned buffer must not corrupt the stored token.
	got, ok, err := s.Get("Iv1.abc")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if string(got) != want {
		t.Fatalf("scrubbing the Set buffer corrupted the store: got %q", got)
	}
	for i := range got {
		got[i] = 0
	}

	again, ok, err := s.Get("Iv1.abc")
	if err != nil || !ok {
		t.Fatalf("second Get: ok=%v err=%v", ok, err)
	}
	if string(again) != want {
		t.Fatalf("scrubbing a returned token corrupted the store: got %q", again)
	}
}

// TestStore_ClientIDs verifies that ClientIDs lists every stored token, ignoring
// temporary files and invalid names.
func TestStore_ClientIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := New(testDataKey(t), dir)
	for _, id := range []string{"Iv1.a", "Iv1.b"} {
		if err := s.Set(id, json.RawMessage(`{"access_token":"x"}`)); err != nil {
			t.Fatal(err)
		}
	}
	// A temp file and a dot file must be ignored.
	if err := os.WriteFile(filepath.Join(dir, ".ghtkn-tmp-123"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	ids, err := s.ClientIDs()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, id := range ids {
		got[id] = true
	}
	if len(got) != 2 || !got["Iv1.a"] || !got["Iv1.b"] {
		t.Fatalf("ClientIDs = %v, want exactly Iv1.a and Iv1.b", ids)
	}
}

// BenchmarkStore_Get measures the per-Get cost of reading and decrypting the token file
// in disk mode (the cost of not caching the plaintext in memory).
func BenchmarkStore_Get(b *testing.B) {
	dir := b.TempDir()
	key := make([]byte, 32)
	s := New(key, dir)
	if err := s.Set("Iv1.abc", json.RawMessage(`{"access_token":"ghu_secret","expiration_date":"2999-01-01T00:00:00Z","refresh_token":"ghr_secret","refresh_token_expiration_date":"2999-06-01T00:00:00Z"}`)); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, _, err := s.Get("Iv1.abc"); err != nil {
			b.Fatal(err)
		}
	}
}
