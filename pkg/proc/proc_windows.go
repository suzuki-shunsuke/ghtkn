//go:build windows

package proc

import (
	"os"
	"syscall"
)

// signals returns the signals ghtkn forwards to the command. Windows delivers only
// os.Interrupt, which the console sends on Ctrl-C and Ctrl-Break; SIGHUP, SIGQUIT and
// SIGUSR1 and SIGUSR2 don't exist there.
func signals() []os.Signal {
	return []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
	}
}

// killsCommand reports whether receiving sig should kill the command rather than
// forward it. Everything but an interrupt does, on the first occurrence.
//
// Windows reports CTRL_CLOSE_EVENT, CTRL_LOGOFF_EVENT and CTRL_SHUTDOWN_EVENT as
// SIGTERM, and a process cannot decline them: the system terminates ghtkn moments
// later. Waiting for a second occurrence to escalate on would therefore be waiting
// for something that never comes, and ghtkn would be killed with the command still
// running. Forwarding is no help either, because Process.Signal accepts nothing but a
// kill here.
//
// os.Interrupt, which is how Ctrl-C and Ctrl-Break arrive, is excluded: the console
// delivers those to the command itself, and killing an interactive command on the
// first Ctrl-C would destroy work the user didn't mean to lose.
func killsCommand(sig os.Signal, _ bool) bool {
	return sig != os.Interrupt
}

// forwardSignal does nothing on Windows: os.Process.Signal accepts only Kill there,
// and the console already delivers Ctrl-C and Ctrl-Break to every process attached to
// it, including the command. The escalation to Kill in forward still works.
func forwardSignal(_ signaler, _ os.Signal) error {
	return nil
}

// signalExitCode always reports false: Windows doesn't report termination by a
// signal, so ProcessState.ExitCode is the only source of the exit code.
func signalExitCode(_ any) (int, bool) {
	return 0, false
}

// isNotExecutable always reports false: Windows has no execute permission bit, so a
// file that can't be run is reported as some other failure.
func isNotExecutable(_ error) bool {
	return false
}
