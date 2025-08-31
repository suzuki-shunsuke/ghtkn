package config_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
)

func TestConfig_Validate(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "config is required",
		},
		{
			name: "empty apps",
			cfg: &config.Config{
				Apps: []*config.App{},
			},
			wantErr: true,
			errMsg:  "apps is required",
		},
		{
			name: "valid config",
			cfg: &config.Config{
				Persist: true,
				Apps: []*config.App{
					{
						Name:     "test-app",
						ClientID: "client123",
						Default:  true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid app",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name: "test-app",
						// Missing ClientID
					},
				},
			},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name: "multiple apps with one invalid",
			cfg: &config.Config{
				Apps: []*config.App{
					{
						Name:     "app1",
						ClientID: "client1",
					},
					{
						Name: "app2",
						// Missing ClientID
					},
				},
			},
			wantErr: true,
			errMsg:  "client_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.cfg.Validate(); err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message does not contain expected string\ngot: %v\nwant substring: %v", err.Error(), tt.errMsg)
				}
				return
			}
			if tt.wantErr {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}

func TestApp_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		app     *config.App
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid app",
			app: &config.App{
				Name:     "test-app",
				ClientID: "client123",
				Default:  true,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			app: &config.App{
				ClientID: "client123",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing client_id",
			app: &config.App{
				Name: "test-app",
			},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name:    "empty app",
			app:     &config.App{},
			wantErr: true,
			errMsg:  "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.app.Validate(); err != nil {
				if tt.wantErr {
					if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("error message does not contain expected string\ngot: %v\nwant substring: %v", err.Error(), tt.errMsg)
					}
					return
				}
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}

func TestReader_Read(t *testing.T) { //nolint:funlen
	t.Parallel()
	tests := []struct {
		name           string
		configContent  string
		configFilePath string
		setupFS        func(fs afero.Fs, content string)
		wantErr        bool
		errMsg         string
		want           *config.Config
	}{
		{
			name:           "empty config path",
			configFilePath: "",
			wantErr:        false,
			want:           &config.Config{},
		},
		{
			name: "valid yaml config",
			configContent: `persist: true
apps:
  - name: test-app
    client_id: client123
    default: true
  - name: another-app
    client_id: client456
`,
			configFilePath: "/config/ghtkn.yaml",
			setupFS: func(fs afero.Fs, content string) {
				_ = fs.MkdirAll("/config", 0o755)
				_ = afero.WriteFile(fs, "/config/ghtkn.yaml", []byte(content), 0o644)
			},
			wantErr: false,
			want: &config.Config{
				Persist: true,
				Apps: []*config.App{
					{
						Name:     "test-app",
						ClientID: "client123",
						Default:  true,
					},
					{
						Name:     "another-app",
						ClientID: "client456",
						Default:  false,
					},
				},
			},
		},
		{
			name:           "file not found",
			configFilePath: "/config/nonexistent.yaml",
			setupFS:        func(_ afero.Fs, _ string) {},
			wantErr:        true,
			errMsg:         "open a configuration file",
		},
		{
			name:           "invalid yaml",
			configContent:  "invalid: yaml: content:",
			configFilePath: "/config/invalid.yaml",
			setupFS: func(fs afero.Fs, content string) {
				_ = fs.MkdirAll("/config", 0o755)
				_ = afero.WriteFile(fs, "/config/invalid.yaml", []byte(content), 0o644)
			},
			wantErr: true,
			errMsg:  "decode a configuration file as YAML",
		},
		{
			name: "minimal valid config",
			configContent: `apps:
  - name: minimal
    client_id: min123
`,
			configFilePath: "/config/minimal.yaml",
			setupFS: func(fs afero.Fs, content string) {
				_ = fs.MkdirAll("/config", 0o755)
				_ = afero.WriteFile(fs, "/config/minimal.yaml", []byte(content), 0o644)
			},
			wantErr: false,
			want: &config.Config{
				Persist: false,
				Apps: []*config.App{
					{
						Name:     "minimal",
						ClientID: "min123",
						Default:  false,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := afero.NewMemMapFs()
			if tt.setupFS != nil {
				tt.setupFS(fs, tt.configContent)
			}

			reader := config.NewReader(fs)
			cfg := &config.Config{}
			err := reader.Read(cfg, tt.configFilePath)

			if tt.wantErr { //nolint:nestif
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message does not contain expected string\ngot: %v\nwant substring: %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.want != nil {
					if diff := cmp.Diff(tt.want, cfg); diff != "" {
						t.Errorf("config mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestDefault(t *testing.T) {
	t.Parallel()
	// Test that Default constant contains valid YAML
	if !strings.Contains(config.Default, "persist: true") {
		t.Error("Default config should contain 'persist: true'")
	}
	if !strings.Contains(config.Default, "apps:") {
		t.Error("Default config should contain 'apps:'")
	}
	if !strings.Contains(config.Default, "client_id:") {
		t.Error("Default config should contain 'client_id:'")
	}
	if !strings.Contains(config.Default, "default: true") {
		t.Error("Default config should contain 'default: true'")
	}
}
