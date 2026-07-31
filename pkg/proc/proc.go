// Package proc runs an external command on ghtkn's behalf and reports the exit code
// a shell would report for it.
//
// The command inherits ghtkn's standard input, output and error as they are, so it
// keeps ghtkn's terminal and interactive commands and pagers work. Instead of dying
// when ghtkn is signalled, ghtkn forwards the signal to the command and keeps
// waiting for it, so a wrapper process doesn't change how the command is terminated.
package proc

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
)

// signalBufferSize is the capacity of the channel receiving signals. Signals that
// arrive while the command starts are buffered until the forwarding goroutine runs,
// and the signal package drops them rather than blocking when the buffer is full.
const signalBufferSize = 16

// Runner runs external commands with ghtkn's standard streams.
type Runner struct {
	stdin  *os.File
	stdout *os.File
	stderr *os.File
}

// New creates a Runner running commands with the given standard streams.
// They are *os.File rather than io.Reader and io.Writer on purpose: os/exec passes
// the file descriptors to the command directly only for *os.File, so the command
// keeps the terminal. Pipes would break isatty checks in it.
func New(stdin, stdout, stderr *os.File) *Runner {
	return &Runner{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

// signaler is the subset of *os.Process the signal forwarding loop uses, so the loop
// can be tested without starting a process.
type signaler interface {
	Signal(sig os.Signal) error
	Kill() error
}

// Run runs name with args and env, and waits for it.
//
// A nil error means the command ran to completion; the returned code is then its exit
// code, which may still be non-zero. A non-nil error means ghtkn couldn't run the
// command, and the returned code is the exit code ghtkn should exit with: 127 when the
// command isn't found, 126 when it isn't executable, and exitCodeUnclassified when
// the failure is neither.
func (r *Runner) Run(logger *slog.Logger, env []string, name string, args ...string) (int, error) {
	// The context is deliberately not passed to the command: cancelling it would kill
	// the command, and the signals that cancel it are forwarded below instead, so that
	// the command is terminated the way it would be without ghtkn in front of it.
	//nolint:noctx // The command must not be tied to the context; see above.
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdin = r.stdin
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr

	// Register the handler before starting the command so that a signal arriving
	// while it starts neither kills ghtkn nor orphans the command.
	ch := make(chan os.Signal, signalBufferSize)
	signal.Notify(ch, signals()...)
	defer signal.Stop(ch)

	if err := cmd.Start(); err != nil {
		return startExitCode(err), fmt.Errorf("start the command: %w", err)
	}

	// done stops the forwarding goroutine once the command is gone. ch is never
	// closed: the signal package keeps a reference to it and sending on a closed
	// channel panics.
	done := make(chan struct{})
	defer close(done)
	// Start sets cmd.Process, so forwarding must not begin before it returns.
	go forward(logger, cmd.Process, ch, done)

	if err := cmd.Wait(); err != nil {
		exitErr := &exec.ExitError{}
		if !errors.As(err, &exitErr) {
			// The command ran but ghtkn failed to wait for it, so there is no exit
			// code to report.
			return exitCodeUnclassified, fmt.Errorf("wait for the command: %w", err)
		}
	}
	return waitExitCode(cmd.ProcessState), nil
}

// forward sends the signals ghtkn receives to the command until done is closed.
//
// Receiving the same signal a second time kills the command instead of forwarding it
// again, so a command ignoring the signal doesn't leave ghtkn waiting forever. SIGINT
// is excluded from that escalation; see escalates.
func forward(logger *slog.Logger, p signaler, ch <-chan os.Signal, done <-chan struct{}) {
	// sent is read and written only by this goroutine, so it needs no lock.
	sent := map[os.Signal]bool{}
	for {
		select {
		case <-done:
			return
		case sig := <-ch:
			if sent[sig] && escalates(sig) {
				logger.Warn("the command didn't exit, so kill it", "signal", sig.String())
				// SIGKILL can't be ignored, so Wait returns and the exit code becomes
				// 128 plus the signal number.
				_ = p.Kill()
				continue
			}
			sent[sig] = true
			// The command may have exited already, in which case this returns
			// os.ErrProcessDone. There is nothing to do about it.
			_ = forwardSignal(p, sig)
		}
	}
}
