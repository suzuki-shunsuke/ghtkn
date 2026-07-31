//go:build !windows

package proc_test

import (
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/proc"
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
// ProcessState reports as -1 rather than as a code of its own.
func TestRunner_Run_signaled(t *testing.T) {
	t.Parallel()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	stdin, stdout, stderr := stdio(t)
	code, err := proc.New(stdin, stdout, stderr).Run(
		slog.New(slog.DiscardHandler), helperEnvs("self-sigterm"), self,
	)
	if err != nil {
		t.Fatal(err)
	}
	// The shell convention: 128 plus the number of the signal that killed it.
	if want := 128 + int(syscall.SIGTERM); code != want {
		t.Errorf("the exit code is %d, want %d", code, want)
	}
}
