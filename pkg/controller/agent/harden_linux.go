//go:build linux

package agent

import (
	"log/slog"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"golang.org/x/sys/unix"
)

// hardenProcess marks the agent process non-dumpable (PR_SET_DUMPABLE=0). This makes
// /proc/<pid>/mem and /proc/<pid>/maps root-owned so a same-user, non-root process can no
// longer read the agent's memory via ptrace or /proc, and it suppresses core dumps. That
// protects the in-memory data key and decrypted tokens from a same-user attacker. It does
// not stop root or a process with CAP_SYS_PTRACE, as noted in the agent's security caveats,
// nor does it prevent pages from reaching swap. It is best-effort: a failure is logged and
// the agent still starts.
func hardenProcess(logger *slog.Logger) {
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		if logger != nil {
			slogerr.WithError(logger, err).Warn("mark the agent process non-dumpable")
		}
	}
}
