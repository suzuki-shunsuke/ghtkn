//go:build !windows

package proc

import (
	"os"
	"syscall"
	"testing"
)

func TestSignals(t *testing.T) {
	t.Parallel()
	sigs := signals()
	// An empty list would make signal.Notify deliver every signal, including the
	// SIGURG the Go runtime uses to preempt goroutines.
	if len(sigs) == 0 {
		t.Fatal("the list of forwarded signals must not be empty")
	}
	seen := map[os.Signal]bool{}
	for _, sig := range sigs {
		if seen[sig] {
			t.Errorf("%s is listed twice", sig)
		}
		seen[sig] = true
	}
	for _, sig := range []os.Signal{syscall.SIGTSTP, syscall.SIGCONT, syscall.SIGWINCH, syscall.SIGCHLD, syscall.SIGPIPE} {
		if seen[sig] {
			t.Errorf("%s must keep its default behavior", sig)
		}
	}
}

func TestEscalates(t *testing.T) {
	t.Parallel()
	// Interactive commands treat Ctrl-C as "cancel the current line", so a second
	// SIGINT must not kill them.
	if escalates(syscall.SIGINT) {
		t.Error("SIGINT must not escalate to a kill")
	}
	if !escalates(syscall.SIGTERM) {
		t.Error("SIGTERM must escalate to a kill")
	}
}

func TestSignalExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		sys    any
		want   int
		wantOK bool
	}{
		{
			name:   "killed by SIGINT",
			sys:    syscall.WaitStatus(int(syscall.SIGINT)),
			want:   130,
			wantOK: true,
		},
		{
			name:   "killed by SIGKILL",
			sys:    syscall.WaitStatus(int(syscall.SIGKILL)),
			want:   137,
			wantOK: true,
		},
		{
			name: "exited on its own",
			// The exit code is in the second byte of the wait status.
			sys: syscall.WaitStatus(3 << 8),
		},
		{
			name: "not a wait status",
			sys:  "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, ok := signalExitCode(tt.sys)
			if ok != tt.wantOK {
				t.Fatalf("killed by a signal = %v, want %v", ok, tt.wantOK)
			}
			if code != tt.want {
				t.Errorf("the exit code is %d, want %d", code, tt.want)
			}
		})
	}
}

func TestIsNotExecutable(t *testing.T) {
	t.Parallel()
	if !isNotExecutable(syscall.ENOEXEC) {
		t.Error("ENOEXEC means the file can't be executed")
	}
	if !isNotExecutable(syscall.EISDIR) {
		t.Error("EISDIR means the file can't be executed")
	}
	if isNotExecutable(syscall.ENOENT) {
		t.Error("ENOENT means the file doesn't exist")
	}
}
