package server

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupStaleSocket(t *testing.T) {
	t.Parallel()

	t.Run("no file", func(t *testing.T) {
		t.Parallel()
		if err := cleanupStaleSocket(t.Context(), filepath.Join(t.TempDir(), "absent.sock")); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("stale file removed", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "stale.sock")
		if err := os.WriteFile(path, nil, socketFilePerm); err != nil {
			t.Fatal(err)
		}
		if err := cleanupStaleSocket(t.Context(), path); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale socket file was not removed: err=%v", err)
		}
	})
}
