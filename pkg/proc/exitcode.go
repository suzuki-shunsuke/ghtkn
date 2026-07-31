package proc

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
)

const (
	// exitCodeUnclassified means ghtkn couldn't map a failure to an exit code of its
	// own. The caller decides what to exit with.
	exitCodeUnclassified = 0
	// exitCodeUnknown is used when a command finished but reported neither an exit
	// code nor a signal, which shouldn't happen.
	exitCodeUnknown = 1
	// exitCodeNotExecutable follows the shell convention: the command was found but
	// couldn't be executed.
	exitCodeNotExecutable = 126
	// exitCodeNotFound follows the shell convention: the command wasn't found.
	exitCodeNotFound = 127
)

// startExitCode maps a failure to start the command to the exit code a shell uses for
// it, and returns exitCodeUnclassified for any other failure.
func startExitCode(err error) int {
	// Permission is checked first because a file found in PATH but not executable is
	// reported as fs.ErrPermission, not as "not found".
	if errors.Is(err, fs.ErrPermission) || isNotExecutable(err) {
		return exitCodeNotExecutable
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		return exitCodeNotFound
	}
	return exitCodeUnclassified
}

// waitExitCode returns the exit code of a command that has finished.
// The signal is checked first because ProcessState.ExitCode returns -1 for a command
// killed by a signal.
func waitExitCode(state *os.ProcessState) int {
	if code, ok := signalExitCode(state.Sys()); ok {
		return code
	}
	if code := state.ExitCode(); code >= 0 {
		return code
	}
	return exitCodeUnknown
}
