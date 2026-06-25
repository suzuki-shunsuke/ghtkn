package keyfile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateDataKey(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "key")
	pass := []byte("correct horse")

	key, created, err := LoadOrCreateDataKey(path, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first call must report created=true")
	}
	if len(key) != dataKeyLen {
		t.Fatalf("data key len = %d, want %d", len(key), dataKeyLen)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != keyFilePerm {
		t.Fatalf("key file perm = %o, want %o", perm, keyFilePerm)
	}

	again, created, err := LoadOrCreateDataKey(path, pass)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("second call must report created=false")
	}
	if !bytes.Equal(key, again) {
		t.Fatal("reloaded data key must match the created one")
	}
}

func TestLoadOrCreateDataKey_wrongPassphrase(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "key")
	if _, _, err := LoadOrCreateDataKey(path, []byte("right")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadOrCreateDataKey(path, []byte("wrong")); !errors.Is(err, ErrIncorrectPassphrase) {
		t.Fatalf("err = %v, want ErrIncorrectPassphrase", err)
	}
}

func TestKeyPath(t *testing.T) {
	t.Parallel()
	data := []struct {
		name string
		env  map[string]string
		goos string
		want string
	}{
		{
			name: "explicit key override",
			env:  map[string]string{"GHTKN_AGENT_KEY": "/custom/key", "XDG_DATA_HOME": "/data"},
			goos: "linux",
			want: "/custom/key",
		},
		{
			name: "xdg data home",
			env:  map[string]string{"XDG_DATA_HOME": "/data"},
			goos: "linux",
			want: "/data/ghtkn/key",
		},
		{
			name: "home fallback",
			env:  map[string]string{"HOME": "/home/me"},
			goos: "linux",
			want: "/home/me/.local/share/ghtkn/key",
		},
		{
			name: "windows localappdata",
			env:  map[string]string{"LocalAppData": `C:\Users\me\AppData\Local`},
			goos: "windows",
			want: filepath.Join(`C:\Users\me\AppData\Local`, "ghtkn", "key"),
		},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(k string) string { return d.env[k] }
			got, err := KeyPath(getEnv, d.goos)
			if err != nil {
				t.Fatal(err)
			}
			if got != d.want {
				t.Fatalf("KeyPath = %q, want %q", got, d.want)
			}
		})
	}
}
