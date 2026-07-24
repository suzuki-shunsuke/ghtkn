package reset

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/keyfile"
)

// resetEnv isolates the agent's key/token/socket paths under temp dirs and points
// the socket at an absent path so the stop step is a no-op (no agent running). It
// returns a getEnv stub to inject (instead of t.Setenv, which forbids t.Parallel)
// alongside the computed key file and token dir.
func resetEnv(t *testing.T) (getEnv func(string) string, keyFile, tokenDir string) {
	t.Helper()
	data := t.TempDir()
	cache := t.TempDir()
	socket := filepath.Join(t.TempDir(), "absent.sock")
	getEnv = func(k string) string {
		switch k {
		case "XDG_DATA_HOME":
			return data
		case "XDG_CACHE_HOME":
			return cache
		case "GHTKN_AGENT_SOCKET":
			return socket
		default:
			// XDG_RUNTIME_DIR, HOME, etc.: empty so path resolution stays within the
			// XDG_* dirs above and never falls back to the real environment.
			return ""
		}
	}
	return getEnv, filepath.Join(data, "ghtkn", "key"), filepath.Join(cache, "ghtkn", "agent")
}

// writeFile writes data to path, creating parent directories as needed.
func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReset_recreates(t *testing.T) {
	t.Parallel()
	getEnv, keyFile, tokenDir := resetEnv(t)

	// Seed an existing key file and a cached token.
	writeFile(t, keyFile, []byte("OLD-KEY-FILE"))
	writeFile(t, filepath.Join(tokenDir, "Iv1.x"), []byte("OLD-TOKEN"))

	c := New()
	c.getEnv = getEnv
	c.confirm = func(string) (bool, error) { return true, nil }
	c.readPassphrase = func(string) ([]byte, error) { return []byte("pw"), nil }

	if err := c.Run(t.Context(), slog.New(slog.DiscardHandler)); err != nil {
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
	// with the new passphrase. LoadOrCreateDataKey reads the existing file and
	// reports created=false when it unwraps successfully.
	blob, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) == "OLD-KEY-FILE" {
		t.Fatal("key file was not recreated")
	}
	if _, created, err := keyfile.LoadOrCreateDataKey(keyFile, []byte("pw")); err != nil || created {
		t.Fatalf("new key file must unwrap with the new passphrase (created=%v): %v", created, err)
	}
}

func TestReset_cancel(t *testing.T) {
	t.Parallel()
	getEnv, keyFile, _ := resetEnv(t)
	writeFile(t, keyFile, []byte("OLD-KEY-FILE"))

	c := New()
	c.getEnv = getEnv
	c.confirm = func(string) (bool, error) { return false, nil } // user answers no
	c.readPassphrase = func(string) ([]byte, error) {
		t.Fatal("passphrase must not be read when canceled")
		return nil, nil
	}

	if err := c.Run(t.Context(), slog.New(slog.DiscardHandler)); err != nil {
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
