//go:build !windows

package proc

import (
	"fmt"
	"log/slog"
	"os/exec"

	"golang.org/x/sys/unix"
)

// run replaces ghtkn's process with the command through execve(2), so it does not
// return unless the command could not be run at all.
//
// ghtkn has nothing left to do at this point: the access tokens are already in env,
// and nothing it holds outlives the call. Handing the process over rather than
// wrapping the command means the command keeps ghtkn's pid, process group, session and
// terminal, so its exit code and the signals it receives are its own. There is no
// wrapper to forward signals to it, to translate its exit status, or to get stuck
// waiting for it.
//
// Note that this discards everything the caller deferred, ghtkn's own signal handlers
// and any buffered output it holds. Standard output and error are unbuffered and the
// standard streams are inherited across the call, so nothing ghtkn has written is
// lost.
func (r *Runner) run(logger *slog.Logger, env []string, name string, args ...string) (int, error) {
	// execve(2) takes a path, so the command has to be resolved here rather than by
	// os/exec, and a command that isn't found or isn't executable is reported the way
	// a shell reports it.
	path, err := exec.LookPath(name)
	if err != nil {
		return startExitCode(err), fmt.Errorf("look up the command: %w", err)
	}
	// argv[0] is the name as it was given, which is what the command sees in its own
	// usage messages, rather than the resolved path.
	argv := append([]string{name}, args...)
	// The standard streams are inherited as they are: execve(2) keeps every file
	// descriptor that isn't close-on-exec, and 0, 1 and 2 never are. A Runner built
	// with anything other than ghtkn's own streams would therefore be ignored here,
	// which is why New documents them as ghtkn's.
	logger.Debug("replacing ghtkn with the command", "command", path)
	if err := unix.Exec(path, argv, env); err != nil {
		// Reaching this means the process was not replaced, so ghtkn is still running
		// and reports the failure itself.
		return startExitCode(err), fmt.Errorf("execute the command: %w", err)
	}
	// Unreachable: unix.Exec only returns on failure.
	return exitCodeUnclassified, nil
}
