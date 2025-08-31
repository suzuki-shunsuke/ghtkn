package config_test

import (
	"path/filepath"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

func TestGetPath(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name    string
		env     *config.Env
		want    string
		wantErr bool
	}{
		// Linux/macOS tests
		{
			name: "Linux: standard XDG config path",
			env: &config.Env{
				XDGConfigHome: "/home/user/.config",
				GOOS:          "linux",
			},
			want:    filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "macOS: custom XDG config path",
			env: &config.Env{
				XDGConfigHome: "/custom/config/dir",
				GOOS:          "darwin",
			},
			want:    filepath.Join("/custom", "config", "dir", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: fallback to HOME when XDG not set",
			env: &config.Env{
				XDGConfigHome: "",
				Home:          "/home/user",
				GOOS:          "linux",
			},
			want:    filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: error when both XDG and HOME are empty",
			env: &config.Env{
				XDGConfigHome: "",
				Home:          "",
				GOOS:          "linux",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "macOS: XDG config with app field set",
			env: &config.Env{
				XDGConfigHome: "/home/user/.config",
				App:           "my-app",
				GOOS:          "darwin",
			},
			want:    filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: root config path",
			env: &config.Env{
				XDGConfigHome: "/",
				GOOS:          "linux",
			},
			want:    filepath.Join("/", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: relative path",
			env: &config.Env{
				XDGConfigHome: "relative/config",
				GOOS:          "linux",
			},
			want:    filepath.Join("relative", "config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "macOS: path with spaces",
			env: &config.Env{
				XDGConfigHome: "/path with spaces/config",
				GOOS:          "darwin",
			},
			want:    filepath.Join("/path with spaces", "config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: HOME fallback with spaces",
			env: &config.Env{
				XDGConfigHome: "",
				Home:          "/home with spaces/user",
				GOOS:          "linux",
			},
			want:    filepath.Join("/home with spaces", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Linux: AppData ignored on Linux",
			env: &config.Env{
				XDGConfigHome: "/home/user/.config",
				AppData:       `C:\Users\testuser\AppData\Roaming`,
				GOOS:          "linux",
			},
			want:    filepath.Join("/home", "user", ".config", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},

		// Windows tests
		{
			name: "Windows: with AppData",
			env: &config.Env{
				AppData: filepath.Join("C:", "Users", "testuser", "AppData", "Roaming"),
				GOOS:    "windows",
			},
			want:    filepath.Join("C:", "Users", "testuser", "AppData", "Roaming", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Windows: without AppData",
			env: &config.Env{
				AppData: "",
				GOOS:    "windows",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Windows: custom AppData path",
			env: &config.Env{
				AppData: filepath.Join("D:", "CustomAppData"),
				GOOS:    "windows",
			},
			want:    filepath.Join("D:", "CustomAppData", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Windows: ignores XDG_CONFIG_HOME",
			env: &config.Env{
				AppData:       filepath.Join("C:", "Users", "testuser", "AppData", "Roaming"),
				XDGConfigHome: "/home/user/.config",
				GOOS:          "windows",
			},
			want:    filepath.Join("C:", "Users", "testuser", "AppData", "Roaming", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
		{
			name: "Windows: ignores HOME",
			env: &config.Env{
				AppData: filepath.Join("C:", "Users", "testuser", "AppData", "Roaming"),
				Home:    "/home/user",
				GOOS:    "windows",
			},
			want:    filepath.Join("C:", "Users", "testuser", "AppData", "Roaming", "ghtkn", "ghtkn.yaml"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := config.GetPath(tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
