package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

func TestNewEnv(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name   string
		envMap map[string]string
		goos   string
		want   *config.Env
	}{
		{
			name: "Linux: all environment variables set",
			envMap: map[string]string{
				"XDG_CONFIG_HOME": "/home/user/.config",
				"GHTKN_APP":       "my-app",
				"HOME":            "/home/user",
			},
			goos: "linux",
			want: &config.Env{
				XDGConfigHome: "/home/user/.config",
				App:           "my-app",
				Home:          "/home/user",
				AppData:       "",
				UserProfile:   "",
				GOOS:          "linux",
			},
		},
		{
			name: "macOS: XDG_CONFIG_HOME and HOME",
			envMap: map[string]string{
				"XDG_CONFIG_HOME": "/custom/config",
				"HOME":            "/Users/testuser",
			},
			goos: "darwin",
			want: &config.Env{
				XDGConfigHome: "/custom/config",
				App:           "",
				Home:          "/Users/testuser",
				AppData:       "",
				UserProfile:   "",
				GOOS:          "darwin",
			},
		},
		{
			name: "Windows: with APPDATA and USERPROFILE",
			envMap: map[string]string{
				"APPDATA":     `C:\Users\testuser\AppData\Roaming`,
				"USERPROFILE": `C:\Users\testuser`,
				"GHTKN_APP":   "test-app",
			},
			goos: "windows",
			want: &config.Env{
				XDGConfigHome: "",
				App:           "test-app",
				Home:          "",
				AppData:       `C:\Users\testuser\AppData\Roaming`,
				UserProfile:   `C:\Users\testuser`,
				GOOS:          "windows",
			},
		},
		{
			name:   "no environment variables set",
			envMap: map[string]string{},
			goos:   "linux",
			want: &config.Env{
				XDGConfigHome: "",
				App:           "",
				Home:          "",
				AppData:       "",
				UserProfile:   "",
				GOOS:          "linux",
			},
		},
		{
			name: "mixed environment (Linux with Windows vars ignored)",
			envMap: map[string]string{
				"XDG_CONFIG_HOME": "/home/user/.config",
				"HOME":            "/home/user",
				"APPDATA":         `C:\Users\testuser\AppData\Roaming`,
				"USERPROFILE":     `C:\Users\testuser`,
				"GHTKN_APP":       "app1",
			},
			goos: "linux",
			want: &config.Env{
				XDGConfigHome: "/home/user/.config",
				App:           "app1",
				Home:          "/home/user",
				AppData:       `C:\Users\testuser\AppData\Roaming`,
				UserProfile:   `C:\Users\testuser`,
				GOOS:          "linux",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getEnv := func(key string) string {
				return tt.envMap[key]
			}
			got := config.NewEnv(getEnv, tt.goos)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("NewEnv() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
