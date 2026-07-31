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

// escalates reports whether receiving sig a second time should kill the command.
// As on Unix, os.Interrupt doesn't escalate so that a second Ctrl-C doesn't destroy
// the work of an interactive command.
func escalates(sig os.Signal) bool {
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
