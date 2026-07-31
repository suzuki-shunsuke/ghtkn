//go:build !windows

package proc

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// helperOS runs the helper modes that only exist on Unix.
func helperOS(mode string) int {
	if mode != "self-sigterm" {
		return 0
	}
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		return 1
	}
	// The signal is delivered asynchronously, so wait for it rather than exiting
	// normally and reporting an exit code of its own.
	time.Sleep(time.Minute)
	return 1
}

// TestRunner_Run_signaled checks the exit code of a command killed by a signal, which
// a process reports as -1 rather than as a code of its own.
func TestRunner_Run_signaled(t *testing.T) {
	t.Parallel()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, impl := range runnerImpls() {
		t.Run(impl.name, func(t *testing.T) {
			t.Parallel()
			got := runRunner(t, impl.child, self, commandModeEnv+"=self-sigterm")
			// The shell convention: 128 plus the number of the signal that killed it.
			// Under execve(2) the shell derives it from the command itself; the child
			// implementation has to compute it, which is what waitExitCode does.
			if want := signalExitCodeBase + int(syscall.SIGTERM); got.code != want {
				t.Errorf("the exit code is %d, want %d: %s", got.code, want, got.stderr)
			}
		})
	}
}
