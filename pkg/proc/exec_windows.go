//go:build windows

package proc

import (
	"log/slog"
)

// run runs the command as a child process, because Windows has no execve(2) and so
// cannot hand ghtkn's process over to it. ghtkn stays alive for as long as the command
// does and reports the command's exit code as its own.
func (r *Runner) run(logger *slog.Logger, env []string, name string, args ...string) (int, error) {
	return r.runChild(logger, env, name, args...)
}
