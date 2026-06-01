package agent

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

	key, created, err := loadOrCreateDataKey(path, pass)
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

	again, created, err := loadOrCreateDataKey(path, pass)
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
	if _, _, err := loadOrCreateDataKey(path, []byte("right")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadOrCreateDataKey(path, []byte("wrong")); !errors.Is(err, errIncorrectPassphrase) {
		t.Fatalf("err = %v, want errIncorrectPassphrase", err)
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
			got, err := keyPath(getEnv, d.goos)
			if err != nil {
				t.Fatal(err)
			}
			if got != d.want {
				t.Fatalf("keyPath = %q, want %q", got, d.want)
			}
		})
	}
}
