package config_test

import (
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

func TestGetPath(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name string
		env  *config.Env
		want string
	}{
		{
			name: "standard XDG config path",
			env: &config.Env{
				XDGConfigHome: "/home/user/.config",
			},
			want: filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
		},
		{
			name: "custom XDG config path",
			env: &config.Env{
				XDGConfigHome: "/custom/config/dir",
			},
			want: filepath.Join("/custom", "config", "dir", "ghtkn", "ghtkn.yaml"),
		},
		{
			name: "empty XDG config home",
			env: &config.Env{
				XDGConfigHome: "",
			},
			want: filepath.Join("ghtkn", "ghtkn.yaml"),
		},
		{
			name: "XDG config with app field set",
			env: &config.Env{
				XDGConfigHome: "/home/user/.config",
				App:           "my-app",
			},
			want: filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
		},
		{
			name: "root config path",
			env: &config.Env{
				XDGConfigHome: "/",
			},
			want: filepath.Join("/", "ghtkn", "ghtkn.yaml"),
		},
		{
			name: "relative path",
			env: &config.Env{
				XDGConfigHome: "relative/config",
			},
			want: filepath.Join("relative", "config", "ghtkn", "ghtkn.yaml"),
		},
		{
			name: "path with spaces",
			env: &config.Env{
				XDGConfigHome: "/path with spaces/config",
			},
			want: filepath.Join("/path with spaces", "config", "ghtkn", "ghtkn.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := config.GetPath(tt.env); got != tt.want {
				t.Errorf("GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
