package tokenstore_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
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

	if err := tokenstore.New(key, dir).Set("Iv1.abc", token); err != nil {
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
	got, ok, err := tokenstore.New(key, dir).Get("Iv1.abc")
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
	got, ok, err := tokenstore.New(testDataKey(t), t.TempDir()).Get("Iv1.absent")
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
	if err := tokenstore.New(testDataKey(t), dir).Set("Iv1.abc", json.RawMessage(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, 32)
	if _, _, err := tokenstore.New(wrong, dir).Get("Iv1.abc"); err == nil {
		t.Fatal("decrypting with the wrong key must fail")
	}
}

func TestStore_invalidClientID(t *testing.T) {
	t.Parallel()
	s := tokenstore.New(testDataKey(t), t.TempDir())
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
	s := tokenstore.New(key, dir)

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
	if _, ok, err := tokenstore.New(key, dir).Get("Iv1.abc"); err != nil || ok {
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
	s := tokenstore.New(key, dir)
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

// TestStore_getReturnsCallerOwnedBuffer verifies that the token Get returns is a private
// copy the caller may scrub, so zeroing it (to shorten the plaintext's lifetime) does not
// corrupt the store's data for a later read.
func TestStore_getReturnsCallerOwnedBuffer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := tokenstore.New(testDataKey(t), dir)
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

// TestStore_deleteIf_miss verifies that DeleteIf is a no-op for a missing token and never
// calls the predicate.
func TestStore_deleteIf_miss(t *testing.T) {
	t.Parallel()
	s := tokenstore.New(testDataKey(t), t.TempDir())
	deleted, err := s.DeleteIf("Iv1.absent", func(json.RawMessage) bool {
		t.Error("predicate must not run for a missing token")
		return true
	})
	if err != nil || deleted {
		t.Fatalf("DeleteIf on miss = (%v, %v), want (false, nil)", deleted, err)
	}
}

// TestStore_deleteIf_keep verifies that a false predicate keeps the token and receives
// its decrypted bytes.
func TestStore_deleteIf_keep(t *testing.T) {
	t.Parallel()
	token := json.RawMessage(`{"access_token":"abc"}`)
	s := tokenstore.New(testDataKey(t), t.TempDir())
	if err := s.Set("Iv1.keep", token); err != nil {
		t.Fatal(err)
	}
	deleted, err := s.DeleteIf("Iv1.keep", func(raw json.RawMessage) bool {
		if string(raw) != string(token) {
			t.Errorf("predicate got %q, want %q", raw, token)
		}
		return false
	})
	if err != nil || deleted {
		t.Fatalf("DeleteIf with a false predicate = (%v, %v), want (false, nil)", deleted, err)
	}
	if _, ok, err := s.Get("Iv1.keep"); err != nil || !ok {
		t.Fatalf("a token kept by a false predicate must remain, got ok=%v err=%v", ok, err)
	}
}

// TestStore_deleteIf_drop verifies that a true predicate deletes the token from disk.
func TestStore_deleteIf_drop(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	key := testDataKey(t)
	s := tokenstore.New(key, dir)
	if err := s.Set("Iv1.drop", json.RawMessage(`{"access_token":"abc"}`)); err != nil {
		t.Fatal(err)
	}
	deleted, err := s.DeleteIf("Iv1.drop", func(json.RawMessage) bool { return true })
	if err != nil || !deleted {
		t.Fatalf("DeleteIf with a true predicate = (%v, %v), want (true, nil)", deleted, err)
	}
	if _, ok, err := tokenstore.New(key, dir).Get("Iv1.drop"); err != nil || ok {
		t.Fatalf("a token dropped by a true predicate must be gone, got ok=%v err=%v", ok, err)
	}
}

// TestStore_deleteIf_invalidClientID verifies that an invalid client ID is rejected
// without calling the predicate.
func TestStore_deleteIf_invalidClientID(t *testing.T) {
	t.Parallel()
	s := tokenstore.New(testDataKey(t), t.TempDir())
	if _, err := s.DeleteIf("../escape", func(json.RawMessage) bool {
		t.Error("predicate must not run for an invalid client id")
		return true
	}); err == nil {
		t.Fatal("DeleteIf must reject an invalid client id")
	}
}

// TestStore_zero verifies that Zero scrubs the store's data key so it can no longer
// decrypt, while leaving the encrypted token on disk intact for a fresh store.
func TestStore_zero(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	token := json.RawMessage(`{"access_token":"abc"}`)
	s := tokenstore.New(testDataKey(t), dir)
	if err := s.Set("Iv1.abc", token); err != nil {
		t.Fatal(err)
	}

	s.Zero()
	// The scrubbed store can no longer decrypt its own token.
	if _, _, err := s.Get("Iv1.abc"); err == nil {
		t.Fatal("Get after Zero must fail to decrypt")
	}

	// The on-disk token is untouched: a fresh store with the same key still reads it.
	got, ok, err := tokenstore.New(testDataKey(t), dir).Get("Iv1.abc")
	if err != nil || !ok {
		t.Fatalf("a fresh store must still read the token, got ok=%v err=%v", ok, err)
	}
	if string(got) != string(token) {
		t.Fatalf("token = %q, want %q", got, token)
	}
}

// TestStore_ClientIDs verifies that ClientIDs lists every stored token, ignoring
// temporary files and invalid names.
func TestStore_ClientIDs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := tokenstore.New(testDataKey(t), dir)
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
// (the cost of not caching the plaintext in memory).
func BenchmarkStore_Get(b *testing.B) {
	dir := b.TempDir()
	key := make([]byte, 32)
	s := tokenstore.New(key, dir)
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
