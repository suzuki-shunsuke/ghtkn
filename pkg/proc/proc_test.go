package proc_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/proc"
)

const (
	// helperEnv makes this test binary run as the command under test instead of
	// running the tests, so that the tests don't depend on a shell or on any command
	// being installed.
	helperEnv = "GHTKN_TEST_HELPER_PROCESS"
	// helperModeEnv selects what the helper process does.
	helperModeEnv = "GHTKN_TEST_HELPER_MODE"
	// tokenEnv is the variable the helper prints in the echo-env mode, standing in
	// for an access token.
	tokenEnv = "GHTKN_TEST_TOKEN" //nolint:gosec // This is the name of an environment variable, not a credential.
)

func TestMain(m *testing.M) {
	if os.Getenv(helperEnv) != "1" {
		os.Exit(m.Run())
	}
	os.Exit(helperMain(os.Getenv(helperModeEnv)))
}

// helperMain runs the helper process. Modes that only exist on some platforms are
// handled by helperOS.
func helperMain(mode string) int {
	switch mode {
	case "exit3":
		return 3
	case "echo-env":
		fmt.Fprintln(os.Stdout, os.Getenv(tokenEnv))
		return 0
	}
	return helperOS(mode)
}

// helperEnvs returns the environment running this test binary as the helper process
// in the given mode.
//
// It builds on the test process's own environment rather than replacing it, both
// because that is how ghtkn runs a command and because a Go program started with an
// environment of a few variables is fragile, on Windows in particular. The variables
// set here come last, so they win over any inherited value of the same name.
func helperEnvs(mode string, envs ...string) []string {
	return append(append(
		os.Environ(),
		helperEnv+"=1",
		helperModeEnv+"="+mode,
	), envs...)
}

// stdio returns three files standing in for ghtkn's standard streams. Run takes
// *os.File so that the command inherits the terminal, so the output has to be read
// back from a file.
func stdio(t *testing.T) (*os.File, *os.File, *os.File) {
	t.Helper()
	dir := t.TempDir()
	files := make([]*os.File, 0, 3)
	for _, name := range []string{"stdin", "stdout", "stderr"} {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			f.Close()
		})
		files = append(files, f)
	}
	return files[0], files[1], files[2]
}

func TestRunner_Run(t *testing.T) { //nolint:funlen // A table of test cases.
	t.Parallel()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		mode     string
		command  string
		envs     []string
		wantCode int
		wantErr  bool
		wantOut  string
	}{
		{
			name:     "a command that succeeds",
			mode:     "exit0",
			wantCode: 0,
		},
		{
			name:     "the exit code of the command is returned",
			mode:     "exit3",
			wantCode: 3,
		},
		{
			name:     "the command runs with the given environment",
			mode:     "echo-env",
			envs:     []string{tokenEnv + "=token-1"},
			wantCode: 0,
			wantOut:  "token-1\n",
		},
		{
			name:     "a command that isn't found",
			command:  "ghtkn-no-such-command-6b3f1a",
			wantCode: 127,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stdin, stdout, stderr := stdio(t)
			command := tt.command
			if command == "" {
				command = self
			}
			code, err := proc.New(stdin, stdout, stderr).Run(
				slog.New(slog.DiscardHandler), helperEnvs(tt.mode, tt.envs...), command,
			)
			checkRun(t, err, tt.wantErr, code, tt.wantCode)
			if tt.wantOut == "" {
				return
			}
			out, err := os.ReadFile(stdout.Name())
			if err != nil {
				t.Fatal(err)
			}
			if got := string(out); got != tt.wantOut {
				t.Errorf("the command wrote %q, want %q", got, tt.wantOut)
			}
		})
	}
}

func TestRunner_Run_notExecutable(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no execute permission bit")
	}
	path := filepath.Join(t.TempDir(), "command")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdin, stdout, stderr := stdio(t)
	code, err := proc.New(stdin, stdout, stderr).Run(slog.New(slog.DiscardHandler), nil, path)
	checkRun(t, err, true, code, 126)
}

func checkRun(t *testing.T, err error, wantErr bool, code, wantCode int) {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Error("an error must be returned")
		}
	} else if err != nil {
		t.Error(err)
	}
	if code != wantCode {
		t.Errorf("the exit code is %d, want %d", code, wantCode)
	}
}
