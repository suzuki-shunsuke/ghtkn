package proc

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const (
	// helperEnv makes this test binary run as a helper instead of running the tests,
	// so that the tests depend on no shell and on no command being installed.
	helperEnv = "GHTKN_TEST_HELPER_PROCESS"
	// helperModeEnv selects what the helper process does.
	helperModeEnv = "GHTKN_TEST_HELPER_MODE"
	// commandModeEnv holds the mode the runner helper gives to the command it runs.
	// It is a separate variable because the command is this test binary too: passing
	// the mode in helperModeEnv would make the command run as a runner again.
	commandModeEnv = "GHTKN_TEST_COMMAND_MODE"
	// tokenEnv is the variable the helper prints in the echo-env mode, standing in
	// for an access token.
	tokenEnv = "GHTKN_TEST_TOKEN" //nolint:gosec // This is the name of an environment variable, not a credential.
	// runnerCommandEnv holds the command the runner helper passes to Run.
	runnerCommandEnv = "GHTKN_TEST_RUNNER_COMMAND"
	// runnerChildEnv makes the runner helper run the command as a child process
	// rather than replacing itself with it.
	runnerChildEnv = "GHTKN_TEST_RUNNER_CHILD"
	// runnerPIDEnv holds the pid of the process that called Run, so that the command
	// can report whether it is running as that same process.
	runnerPIDEnv = "GHTKN_TEST_RUNNER_PID"
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
	case "exit0":
		return 0
	case "exit3":
		return 3
	case "echo-env":
		fmt.Fprintln(os.Stdout, os.Getenv(tokenEnv))
		return 0
	case "echo-pid":
		// The pid this command runs as, and the pid of the process that called Run.
		// They are the same process exactly when Run replaced it with this command.
		fmt.Fprintln(os.Stdout, os.Getpid(), os.Getenv(runnerPIDEnv))
		return 0
	case "runner":
		return helperRunner()
	}
	return helperOS(mode)
}

// helperRunner is the helper that calls Run.
//
// Every test of Run goes through a helper process, because where execve(2) is
// available Run replaces its process with the command and never returns: calling it
// from a test would replace the test binary itself. This helper's exit status is
// therefore the command's, which is what the tests assert on.
func helperRunner() int {
	runner := New(os.Stdin, os.Stdout, os.Stderr)
	runner.fallback = os.Getenv(runnerChildEnv) == "1"
	// The command is this test binary as well, so it is told what to do through
	// helperModeEnv. The existing entry has to be dropped rather than shadowed by a
	// second one: execve(2) passes the environment through as it is, and a reader
	// takes the first entry of a name, so appending would leave the command running
	// as a runner again. This is what buildEnv in the exec controller does for real.
	env := []string{}
	for _, e := range os.Environ() {
		if name, _, _ := strings.Cut(e, "="); name == helperModeEnv {
			continue
		}
		env = append(env, e)
	}
	env = append(env, helperModeEnv+"="+os.Getenv(commandModeEnv), runnerPIDEnv+"="+strconv.Itoa(os.Getpid()))
	code, err := runner.Run(slog.New(slog.DiscardHandler), env, os.Getenv(runnerCommandEnv))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return code
}

// runnerResult is what calling Run inside a helper process produced.
type runnerResult struct {
	code   int
	stdout string
	stderr string
}

// runRunner calls Run in a helper process, which runs command, and reports what came
// out. child selects the implementation Windows always uses, so that it stays covered
// where execve(2) is used instead.
func runRunner(t *testing.T, child bool, command string, envs ...string) *runnerResult {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("find the test binary: %v", err)
	}
	envs = append(envs, runnerCommandEnv+"="+command)
	if child {
		envs = append(envs, runnerChildEnv+"=1")
	}
	cmd := exec.Command(self) //nolint:noctx // Cancelling the helper would hide what the command did.
	cmd.Env = helperEnvs("runner", envs...)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	exitErr := &exec.ExitError{}
	if err := cmd.Run(); err != nil && !errors.As(err, &exitErr) {
		t.Fatalf("run the helper: %v", err)
	}
	return &runnerResult{
		// What a shell would report for the helper, so that both implementations are
		// compared on the same terms. Under execve(2) the helper is the command, so a
		// command killed by a signal kills the helper and ProcessState reports the
		// signal rather than a code; the child implementation computes the same number
		// itself, and the helper then exits with it normally. The arithmetic behind
		// this is covered on its own by TestSignalExitCode.
		code:   waitExitCode(cmd.ProcessState),
		stdout: stdout.String(),
		stderr: stderr.String(),
	}
}

// helperEnvs returns the environment running this test binary as a helper process in
// the given mode.
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

// runnerImpl is one of the two ways Run runs a command.
type runnerImpl struct {
	name string
	// child runs the command as a child process rather than replacing ghtkn with it.
	child bool
}

// runnerImpls returns both implementations. Every test of Run goes through all of
// them, so that the child one, which is the only one Windows has, stays covered on the
// platforms that use execve(2) instead.
func runnerImpls() []*runnerImpl {
	return []*runnerImpl{
		{name: "exec"},
		{name: "child", child: true},
	}
}

func TestRunner_Run(t *testing.T) {
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
			wantCode: exitCodeNotFound,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		for _, impl := range runnerImpls() {
			t.Run(tt.name+"/"+impl.name, func(t *testing.T) {
				t.Parallel()
				command := tt.command
				if command == "" {
					command = self
				}
				got := runRunner(t, impl.child, command, append([]string{commandModeEnv + "=" + tt.mode}, tt.envs...)...)
				if got.code != tt.wantCode {
					t.Errorf("the exit code is %d, want %d: %s", got.code, tt.wantCode, got.stderr)
				}
				if tt.wantErr && got.stderr == "" {
					t.Error("the failure must be reported")
				}
				if tt.wantOut != "" && got.stdout != tt.wantOut {
					t.Errorf("the command wrote %q, want %q", got.stdout, tt.wantOut)
				}
			})
		}
	}
}

// TestRunner_Run_replacesTheProcess checks that ghtkn hands its process over to the
// command rather than wrapping it, which is what makes the command's exit code, the
// signals it receives and its terminal its own.
//
// The command reports the pid it runs as together with the pid of the process that
// called Run. Comparing the two within one run is what makes this able to fail:
// comparing pids across two runs could not, because separate processes never share a
// pid whichever implementation ran them.
func TestRunner_Run_replacesTheProcess(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no execve(2), so the command always runs as a child")
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	for _, impl := range runnerImpls() {
		t.Run(impl.name, func(t *testing.T) {
			t.Parallel()
			got := runRunner(t, impl.child, self, commandModeEnv+"=echo-pid")
			commandPID, runnerPID, ok := strings.Cut(strings.TrimSpace(got.stdout), " ")
			if !ok {
				t.Fatalf("the command reported no pids: %q %q", got.stdout, got.stderr)
			}
			// The command is the process that called Run exactly when that process was
			// replaced with it.
			if replaced := commandPID == runnerPID; replaced == impl.child {
				t.Errorf("the command runs as the process that called Run = %v, want %v (command %s, caller %s)",
					replaced, !impl.child, commandPID, runnerPID)
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
	for _, impl := range runnerImpls() {
		t.Run(impl.name, func(t *testing.T) {
			t.Parallel()
			got := runRunner(t, impl.child, path)
			if got.code != exitCodeNotExecutable {
				t.Errorf("the exit code is %d, want %d: %s", got.code, exitCodeNotExecutable, got.stderr)
			}
		})
	}
}
