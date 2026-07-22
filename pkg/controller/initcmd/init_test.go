package initcmd_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/initcmd"
)

// newLogger returns a logger writing into buf so tests can assert what the user is told.
func newLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// checkCreated asserts that path holds the default configuration and grants nothing
// beyond 0644.
func checkCreated(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the created file: %v", err)
	}
	for _, field := range []string{"apps:", "client_id:"} {
		if !strings.Contains(string(content), field) {
			t.Errorf("the created file does not contain %q:\n%s", field, content)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat the created file: %v", err)
	}
	// The file is created with 0644, but the process umask may clear bits from it, so
	// the exact mode is environment-dependent. What matters is that nothing beyond 0644
	// is granted: no write for group or other, and not executable.
	if perm := info.Mode().Perm(); perm|0o644 != 0o644 {
		t.Errorf("file permissions = %o, want no bits beyond 644", perm)
	}
}

// initCase is one Init scenario against a temporary directory.
type initCase struct {
	name string
	// path is joined onto the test's temporary directory.
	path string
	// setup prepares the temporary directory before Init runs.
	setup func(t *testing.T, dir string)
	// wantExisting is the content the file must keep, i.e. Init must not overwrite it.
	wantExisting    string
	wantLogContains string
}

// run executes the scenario in its own temporary directory.
func (tt *initCase) run(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if tt.setup != nil {
		tt.setup(t, dir)
	}
	path := filepath.Join(dir, tt.path)

	buf := &bytes.Buffer{}
	if err := initcmd.New().Init(newLogger(buf), path); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if !strings.Contains(buf.String(), tt.wantLogContains) {
		t.Errorf("log does not contain %q:\n%s", tt.wantLogContains, buf)
	}
	if tt.wantExisting != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read the existing file: %v", err)
		}
		if string(content) != tt.wantExisting {
			t.Errorf("the existing file was overwritten: %q", content)
		}
		return
	}
	checkCreated(t, path)
}

func TestController_Init(t *testing.T) {
	t.Parallel()
	tests := []initCase{
		{
			name:            "create the config file",
			path:            "ghtkn.yaml",
			wantLogContains: "The configuration file has been created",
		},
		{
			name:            "create the config file in a directory that does not exist yet",
			path:            filepath.Join("a", "b", "ghtkn.yaml"),
			wantLogContains: "The configuration file has been created",
		},
		{
			name: "the directory exists but the file does not",
			path: filepath.Join("cfg", "ghtkn.yaml"),
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, "cfg"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantLogContains: "The configuration file has been created",
		},
		{
			name: "an existing config file is kept",
			path: "ghtkn.yaml",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "ghtkn.yaml"), []byte("existing content"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantExisting:    "existing content",
			wantLogContains: "The configuration file already exists",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// TestController_Init_relativePath verifies that a config path with no directory part
// works: the directory to create is ".", which already exists.
//
//nolint:paralleltest // t.Chdir, which keeps the relative path inside a temporary directory, cannot be used in a parallel test.
func TestController_Init_relativePath(t *testing.T) {
	t.Chdir(t.TempDir())

	buf := &bytes.Buffer{}
	if err := initcmd.New().Init(newLogger(buf), "ghtkn.yaml"); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	checkCreated(t, "ghtkn.yaml")
}

// TestController_Init_statError verifies that a path whose existence cannot be
// determined is reported instead of being written over. A parent that is a regular file
// makes the check fail with ENOTDIR rather than "does not exist".
func TestController_Init_statError(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows reports a path under a regular file as not existing, which is a different branch")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := initcmd.New().Init(newLogger(&bytes.Buffer{}), filepath.Join(file, "ghtkn.yaml"))
	if err == nil {
		t.Fatal("Init() must fail when it cannot tell whether the config file exists")
	}
	if !strings.Contains(err.Error(), "check if a configuration file exists") {
		t.Errorf("Init() error = %v, want it to report the existence check", err)
	}
}

// TestController_Init_mkdirError verifies that a directory that cannot be created is
// reported as such.
func TestController_Init_mkdirError(t *testing.T) {
	t.Parallel()
	skipUnlessPermissionsDeny(t)
	dir := readOnlyDir(t)

	err := initcmd.New().Init(newLogger(&bytes.Buffer{}), filepath.Join(dir, "sub", "ghtkn.yaml"))
	if err == nil {
		t.Fatal("Init() must fail when the config directory cannot be created")
	}
	if !strings.Contains(err.Error(), "create config dir") {
		t.Errorf("Init() error = %v, want it to report the directory creation", err)
	}
}

// TestController_Init_writeError verifies that a file that cannot be written is reported
// as such. The directory already exists, so only the write fails.
func TestController_Init_writeError(t *testing.T) {
	t.Parallel()
	skipUnlessPermissionsDeny(t)
	dir := readOnlyDir(t)

	err := initcmd.New().Init(newLogger(&bytes.Buffer{}), filepath.Join(dir, "ghtkn.yaml"))
	if err == nil {
		t.Fatal("Init() must fail when the config file cannot be written")
	}
	if !strings.Contains(err.Error(), "create a configuration file") {
		t.Errorf("Init() error = %v, want it to report the file creation", err)
	}
}

// skipUnlessPermissionsDeny skips a test that makes a directory unwritable to provoke a
// failure, on the platforms where that does not deny the owner: root bypasses the
// permission bits, and Windows does not honor them this way.
func skipUnlessPermissionsDeny(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not deny writes through the permission bits set here")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses the permission bits set here")
	}
}

// readOnlyDir returns a directory the test process may read but not write.
func readOnlyDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "read-only")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	// TempDir's cleanup must be able to remove it again.
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Error(err)
		}
	})
	return dir
}
