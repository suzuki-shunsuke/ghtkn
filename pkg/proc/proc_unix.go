//go:build !windows

package proc

import (
	"errors"
	"os"
	"syscall"
)

// signalExitCodeBase is the shell convention for a command killed by a signal:
// 128 plus the signal number.
const signalExitCodeBase = 128

// signals returns the signals ghtkn forwards to the command.
//
// SIGTSTP, SIGCONT, SIGWINCH, SIGCHLD and SIGPIPE are deliberately absent. The
// command runs in ghtkn's process group, so the terminal delivers its signals to the
// command directly, and catching SIGTSTP without implementing stop and continue
// ourselves would break Ctrl-Z.
//
// The list must never be empty either: signal.Notify with no signal at all delivers
// every signal, including the SIGURG the Go runtime uses to preempt goroutines.
func signals() []os.Signal {
	return []os.Signal{
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	}
}

// killsCommand reports whether receiving sig should kill the command rather than
// forward it. Only the second occurrence does, so that a command ignoring a signal
// doesn't leave ghtkn waiting for it forever.
//
// SIGINT never kills it. Interactive commands such as python, node and psql treat
// Ctrl-C as "cancel the current line" and keep running, so killing them on the second
// Ctrl-C would destroy work the user didn't mean to lose.
//
// SIGINT is still forwarded, because ghtkn is not always signalled by a terminal:
// 'kill -INT' aimed at ghtkn alone reaches the command only through the forwarding. A
// terminal's Ctrl-C does reach the command on its own, since it shares ghtkn's process
// group, so it arrives twice there. Delivering an interrupt twice is what a command
// handling Ctrl-C already copes with, which is the cheaper of the two problems.
func killsCommand(sig os.Signal, alreadySent bool) bool {
	return alreadySent && sig != syscall.SIGINT
}

// forwardSignal sends sig to the command.
func forwardSignal(p signaler, sig os.Signal) error {
	return p.Signal(sig) //nolint:wrapcheck // The caller only checks whether the command is gone, so the error is passed through.
}

// signalExitCode reports 128 plus the signal number when sys says the command was
// killed by a signal.
//
// It takes the value of ProcessState.Sys() rather than the state itself because a
// syscall.WaitStatus can be built in a test while an os.ProcessState can't.
func signalExitCode(sys any) (int, bool) {
	ws, ok := sys.(syscall.WaitStatus)
	if !ok || !ws.Signaled() {
		return 0, false
	}
	return signalExitCodeBase + int(ws.Signal()), true
}

// isNotExecutable reports whether err means the file exists but can't be executed as
// a program.
func isNotExecutable(err error) bool {
	return errors.Is(err, syscall.ENOEXEC) || errors.Is(err, syscall.EISDIR)
}
