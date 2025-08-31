//nolint:wrapcheck
package initcmd_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/initcmd"
)

func TestController_Init(t *testing.T) { //nolint:gocognit,cyclop,funlen
	t.Parallel()
	tests := []struct {
		name            string
		configFilePath  string
		setupFS         func(fs afero.Fs)
		wantErr         bool
		errContains     string
		checkFile       bool
		wantLogContains string
	}{
		{
			name:            "create new config file",
			configFilePath:  "/home/user/.config/ghtkn/ghtkn.yaml",
			setupFS:         func(_ afero.Fs) {},
			wantErr:         false,
			checkFile:       true,
			wantLogContains: "The configuration file has been created",
		},
		{
			name:           "config file already exists",
			configFilePath: "/home/user/.config/ghtkn/ghtkn.yaml",
			setupFS: func(fs afero.Fs) {
				_ = fs.MkdirAll("/home/user/.config/ghtkn", 0o755)
				_ = afero.WriteFile(fs, "/home/user/.config/ghtkn/ghtkn.yaml", []byte("existing content"), 0o644)
			},
			wantErr:         false,
			checkFile:       false,
			wantLogContains: "The configuration file already exists",
		},
		{
			name:            "create config in nested directory",
			configFilePath:  "/home/user/.config/ghtkn/subdir/ghtkn.yaml",
			setupFS:         func(_ afero.Fs) {},
			wantErr:         false,
			checkFile:       true,
			wantLogContains: "The configuration file has been created",
		},
		{
			name:            "create config in current directory",
			configFilePath:  "ghtkn.yaml",
			setupFS:         func(_ afero.Fs) {},
			wantErr:         false,
			checkFile:       true,
			wantLogContains: "The configuration file has been created",
		},
		{
			name:            "create config with absolute path",
			configFilePath:  "/etc/ghtkn/ghtkn.yaml",
			setupFS:         func(_ afero.Fs) {},
			wantErr:         false,
			checkFile:       true,
			wantLogContains: "The configuration file has been created",
		},
		{
			name:           "directory exists but file doesn't",
			configFilePath: "/home/user/.config/ghtkn/ghtkn.yaml",
			setupFS: func(fs afero.Fs) {
				_ = fs.MkdirAll("/home/user/.config/ghtkn", 0o755)
			},
			wantErr:         false,
			checkFile:       true,
			wantLogContains: "The configuration file has been created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup filesystem
			fs := afero.NewMemMapFs()
			if tt.setupFS != nil {
				tt.setupFS(fs)
			}

			// Setup logger with buffer to capture logs
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			// Create controller
			env := &config.Env{
				XDGConfigHome: "/home/user/.config",
			}
			ctrl := initcmd.New(fs, env)

			// Execute Init
			err := ctrl.Init(logger, tt.configFilePath)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check log output
			logOutput := buf.String()
			if tt.wantLogContains != "" && !strings.Contains(logOutput, tt.wantLogContains) {
				t.Errorf("log output does not contain expected string\ngot: %v\nwant substring: %v", logOutput, tt.wantLogContains)
			}

			// Check file creation
			if tt.checkFile { //nolint:nestif
				exists, err := afero.Exists(fs, tt.configFilePath)
				if err != nil {
					t.Fatalf("failed to check file existence: %v", err)
				}
				if !exists {
					t.Errorf("config file was not created at %s", tt.configFilePath)
				}

				// Check file content
				content, err := afero.ReadFile(fs, tt.configFilePath)
				if err != nil {
					t.Fatalf("failed to read created file: %v", err)
				}

				// Should contain the default template
				if !strings.Contains(string(content), "persist:") {
					t.Error("created file does not contain expected 'persist:' field")
				}
				if !strings.Contains(string(content), "apps:") {
					t.Error("created file does not contain expected 'apps:' field")
				}
				if !strings.Contains(string(content), "client_id:") {
					t.Error("created file does not contain expected 'client_id:' field")
				}

				// Check file permissions
				info, err := fs.Stat(tt.configFilePath)
				if err != nil {
					t.Fatalf("failed to stat created file: %v", err)
				}
				mode := info.Mode()
				expectedMode := os.FileMode(0o644)
				if mode != expectedMode {
					t.Errorf("file permissions = %v, want %v", mode, expectedMode)
				}

				// Check directory exists
				dirPath := filepath.Dir(tt.configFilePath)
				if dirPath != "." && dirPath != "/" {
					dirInfo, err := fs.Stat(dirPath)
					if err != nil {
						t.Fatalf("failed to stat directory: %v", err)
					}
					// Check if it's a directory
					if !dirInfo.IsDir() {
						t.Errorf("expected %s to be a directory", dirPath)
					}
				}
			}
		})
	}
}

func TestController_Init_ErrorCases(t *testing.T) { //nolint:funlen
	t.Parallel()

	t.Run("filesystem error on exists check", func(t *testing.T) {
		t.Parallel()

		// Create a mock filesystem that returns an error
		fs := &errorFS{
			existsErr: true,
		}

		env := &config.Env{}
		ctrl := initcmd.New(fs, env)
		logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

		if err := ctrl.Init(logger, "/test/config.yaml"); err != nil {
			if !strings.Contains(err.Error(), "check if a configuration file exists") {
				t.Errorf("unexpected error message: %v", err)
			}
			return
		}
		t.Fatal("expected error but got nil")
	})

	t.Run("filesystem error on mkdir", func(t *testing.T) {
		t.Parallel()

		fs := &errorFS{
			mkdirErr: true,
		}

		env := &config.Env{}
		ctrl := initcmd.New(fs, env)
		logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

		if err := ctrl.Init(logger, "/test/config.yaml"); err != nil {
			if !strings.Contains(err.Error(), "create config dir") {
				t.Errorf("unexpected error message: %v", err)
			}
			return
		}
		t.Fatal("expected error but got nil")
	})

	t.Run("filesystem error on write file", func(t *testing.T) {
		t.Parallel()

		fs := &errorFS{
			writeErr: true,
		}

		env := &config.Env{}
		ctrl := initcmd.New(fs, env)
		logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

		if err := ctrl.Init(logger, "/test/config.yaml"); err != nil {
			if !strings.Contains(err.Error(), "create a configuration file") {
				t.Errorf("unexpected error message: %v", err)
			}
			return
		}
		t.Fatal("expected error but got nil")
	})
}

// errorFS is a mock filesystem that returns errors for testing
type errorFS struct {
	afero.Fs
	existsErr bool
	mkdirErr  bool
	writeErr  bool
}

func (fs *errorFS) Name() string {
	return "errorFS"
}

func (fs *errorFS) Create(name string) (afero.File, error) {
	if fs.writeErr {
		return nil, os.ErrPermission
	}
	return afero.NewMemMapFs().Create(name)
}

func (fs *errorFS) MkdirAll(_ string, _ os.FileMode) error {
	if fs.mkdirErr {
		return os.ErrPermission
	}
	return nil
}

func (fs *errorFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if fs.writeErr {
		return nil, os.ErrPermission
	}
	return afero.NewMemMapFs().OpenFile(name, flag, perm)
}

func (fs *errorFS) Stat(_ string) (os.FileInfo, error) {
	if fs.existsErr {
		return nil, os.ErrPermission
	}
	// Return not found for new files
	return nil, os.ErrNotExist
}

func (fs *errorFS) Remove(_ string) error {
	return nil
}

func (fs *errorFS) RemoveAll(_ string) error {
	return nil
}

func (fs *errorFS) Rename(_, _ string) error {
	return nil
}

func (fs *errorFS) Chmod(_ string, _ os.FileMode) error {
	return nil
}

func (fs *errorFS) Chown(_ string, _, _ int) error {
	return nil
}

func (fs *errorFS) Chtimes(_ string, _, _ time.Time) error {
	return nil
}

func (fs *errorFS) Open(name string) (afero.File, error) {
	return afero.NewMemMapFs().Open(name)
}

func (fs *errorFS) Mkdir(_ string, _ os.FileMode) error {
	return nil
}
