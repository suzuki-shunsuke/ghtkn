package crypt_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/crypt"
)

func TestAtomicWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	data := []byte("payload")

	if err := crypt.AtomicWrite(path, data); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("content = %q, want %q", got, data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// 0600 spelled out rather than read from the package: the file holds encrypted
	// tokens, so "current user only" is the contract, not whatever the constant says.
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want %o", perm, 0o600)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("dir has %d entries, want 1 (no leftover temp file)", len(entries))
	}
}
