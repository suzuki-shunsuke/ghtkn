// Package proc runs an external command on ghtkn's behalf.
//
// Where the platform allows it, ghtkn replaces itself with the command through
// execve(2) rather than running it as a child. ghtkn has nothing left to do once the
// access tokens are in the command's environment, so there is no reason to keep a
// process around: the command then owns ghtkn's process, and its exit code, the
// signals it receives, its process group and its terminal are the ones it would have
// without ghtkn in front of it. Nothing has to be forwarded or translated, and no
// wrapper can get stuck holding a command that ignores a signal.
//
// Windows has no execve(2), so there the command runs as a child and ghtkn waits for
// it, reporting its exit code as its own.
package proc

import (
	"log/slog"
	"os"
)

// Runner runs external commands with ghtkn's standard streams.
type Runner struct {
	stdin  *os.File
	stdout *os.File
	stderr *os.File
	// fallback runs the command as a child even where execve(2) is available. It
	// exists so that the tests can exercise the child implementation, which is the
	// only one Windows uses, on every platform. Nothing outside the tests sets it.
	fallback bool
}

// New creates a Runner running commands with the given standard streams.
// They are *os.File rather than io.Reader and io.Writer on purpose: the command has to
// inherit the file descriptors themselves to keep ghtkn's terminal, and os/exec passes
// them through only for *os.File. Pipes would break isatty checks in the command.
func New(stdin, stdout, stderr *os.File) *Runner {
	return &Runner{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

// Run runs name with args and env.
//
// Where execve(2) is available it does not return on success: ghtkn's process becomes
// the command, so nothing deferred by the caller runs and the command reports its own
// exit code to the shell. Everything ghtkn has to do must therefore be done before
// calling Run.
//
// It returns when the command could not be run at all, and on Windows also when the
// command has finished. A nil error means the command ran to completion; the returned
// code is then its exit code, which may still be non-zero. A non-nil error means ghtkn
// could not run the command, and the returned code is the exit code ghtkn should exit
// with: 127 when the command is not found, 126 when it cannot be executed, and
// exitCodeUnclassified when the failure is neither.
func (r *Runner) Run(logger *slog.Logger, env []string, name string, args ...string) (int, error) {
	if r.fallback {
		return r.runChild(logger, env, name, args...)
	}
	return r.run(logger, env, name, args...)
}
