package keystore

import (
	"path/filepath"
	"testing"
)

func TestTokenDir(t *testing.T) {
	t.Parallel()
	data := []struct {
		name string
		env  map[string]string
		goos string
		want string
	}{
		{
			name: "explicit token dir override",
			env:  map[string]string{"GHTKN_AGENT_TOKEN_DIR": "/custom/tokens", "XDG_CACHE_HOME": "/cache"},
			goos: "linux",
			want: "/custom/tokens",
		},
		{
			name: "xdg cache home",
			env:  map[string]string{"XDG_CACHE_HOME": "/cache"},
			goos: "linux",
			want: "/cache/ghtkn/agent",
		},
		{
			name: "home fallback",
			env:  map[string]string{"HOME": "/home/me"},
			goos: "linux",
			want: "/home/me/.cache/ghtkn/agent",
		},
		{
			name: "windows localappdata",
			env:  map[string]string{"LocalAppData": `C:\Users\me\AppData\Local`},
			goos: "windows",
			want: filepath.Join(`C:\Users\me\AppData\Local`, "cache", "ghtkn", "agent"),
		},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(k string) string { return d.env[k] }
			got, err := TokenDir(getEnv, d.goos)
			if err != nil {
				t.Fatal(err)
			}
			if got != filepath.FromSlash(d.want) {
				t.Fatalf("TokenDir = %q, want %q", got, d.want)
			}
		})
	}
}
