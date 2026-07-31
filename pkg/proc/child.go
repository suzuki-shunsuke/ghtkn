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

// signaler is the subset of *os.Process the signal forwarding loop uses, so the loop
// can be tested without starting a process.
type signaler interface {
	Signal(sig os.Signal) error
	Kill() error
}

// runChild runs the command as a child process and waits for it. It is what Windows
// uses, since it has no execve(2), and what the tests use to cover that path
// elsewhere.
//
// Everything execve(2) gets for free has to be arranged here: the signals ghtkn
// receives are forwarded to the command instead of killing ghtkn and orphaning it, and
// the command's exit code becomes ghtkn's.
func (r *Runner) runChild(logger *slog.Logger, env []string, name string, args ...string) (int, error) {
	// The context is deliberately not passed to the command: cancelling it would kill
	// the command, and the signals that cancel it are forwarded below instead, so that
	// the command is terminated the way it would be without ghtkn in front of it.
	//nolint:noctx // The command must not be tied to the context; see above.
	cmd := exec.Command(name, args...) //nolint:gosec // Running the command the user gave is what this package is for.
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

// forward sends the signals ghtkn receives to the command until done is closed, and
// kills the command instead when forwarding cannot end it; see killsCommand, which
// decides that per platform.
func forward(logger *slog.Logger, p signaler, ch <-chan os.Signal, done <-chan struct{}) {
	// sent is read and written only by this goroutine, so it needs no lock.
	sent := map[os.Signal]bool{}
	for {
		select {
		case <-done:
			return
		case sig := <-ch:
			if killsCommand(sig, sent[sig]) {
				logger.Warn("the command didn't exit, so kill it", "signal", sig.String())
				// A kill can't be declined, so Wait returns and the exit code becomes
				// 128 plus the signal number where signals are reported.
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
