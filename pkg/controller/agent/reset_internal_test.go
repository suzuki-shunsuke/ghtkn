package agent

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// resetEnv isolates the agent's key/token/socket paths under temp dirs and points
// the socket at an absent path so Stop is a no-op (no agent running).
func resetEnv(t *testing.T) (keyFile, tokenDir string) {
	t.Helper()
	cfg := t.TempDir()
	cache := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("GHTKN_AGENT_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))
	return filepath.Join(cfg, "ghtkn", "key"), filepath.Join(cache, "ghtkn", "agent")
}

func TestReset_recreates(t *testing.T) { //nolint:paralleltest // uses t.Setenv
	keyFile, tokenDir := resetEnv(t)

	// Seed an existing key file and a cached token.
	if err := atomicWrite(keyFile, []byte("OLD-KEY-FILE")); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(filepath.Join(tokenDir, "Iv1.x"), []byte("OLD-TOKEN")); err != nil {
		t.Fatal(err)
	}

	c := New()
	c.confirm = func(string) (bool, error) { return true, nil }
	c.readPassphrase = func(string) ([]byte, error) { return []byte("pw"), nil }

	if err := c.Reset(t.Context(), slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}

	// Token directory must be empty (token deleted).
	entries, err := os.ReadDir(tokenDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("token dir has %d entries, want 0", len(entries))
	}

	// Key file must exist and be newly created (not the old contents), unwrappable
	// with the new passphrase.
	blob, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) == "OLD-KEY-FILE" {
		t.Fatal("key file was not recreated")
	}
	if _, err := unwrapDataKey(blob, []byte("pw")); err != nil {
		t.Fatalf("new key file must unwrap with the new passphrase: %v", err)
	}
}

func TestReset_cancel(t *testing.T) { //nolint:paralleltest // uses t.Setenv
	keyFile, _ := resetEnv(t)
	if err := atomicWrite(keyFile, []byte("OLD-KEY-FILE")); err != nil {
		t.Fatal(err)
	}

	c := New()
	c.confirm = func(string) (bool, error) { return false, nil } // user answers no
	c.readPassphrase = func(string) ([]byte, error) {
		t.Fatal("passphrase must not be read when canceled")
		return nil, nil
	}

	if err := c.Reset(t.Context(), slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}

	// The key file must be untouched.
	blob, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) != "OLD-KEY-FILE" {
		t.Fatal("key file must be untouched when reset is canceled")
	}
}
