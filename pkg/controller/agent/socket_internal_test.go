package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSocketPath(t *testing.T) {
	data := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "xdg runtime dir",
			env:  map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"},
			want: "/run/user/1000/ghtkn/socket",
		},
		{
			name: "xdg cache home fallback",
			env:  map[string]string{"XDG_CACHE_HOME": "/home/me/.cache"},
			want: "/home/me/.cache/ghtkn/agent.sock",
		},
		{
			name: "home fallback",
			env:  map[string]string{"HOME": "/home/me"},
			want: "/home/me/.cache/ghtkn/agent.sock",
		},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Setenv("XDG_RUNTIME_DIR", "")
			t.Setenv("XDG_CACHE_HOME", "")
			for k, v := range d.env {
				t.Setenv(k, v)
			}
			got, err := socketPath()
			if err != nil {
				t.Fatal(err)
			}
			if got != filepath.FromSlash(d.want) {
				t.Fatalf("socketPath() = %q, want %q", got, d.want)
			}
		})
	}
}

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
