//go:build linux

// Package harden hardens a process that holds ghtkn secrets in memory: the agent,
// which keeps the data key and decrypts tokens for as long as it runs, and the
// commands that read the agent passphrase from the terminal.
package harden

import (
	"log/slog"

	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"golang.org/x/sys/unix"
)

// Process marks the calling process non-dumpable (PR_SET_DUMPABLE=0). This makes
// /proc/<pid>/mem and /proc/<pid>/maps root-owned so a same-user, non-root process can no
// longer read its memory via ptrace or /proc, and it suppresses core dumps, so a crash
// cannot write the secrets it holds to disk. It does not stop root or a process with
// CAP_SYS_PTRACE, as noted in the agent's security caveats, nor does it prevent pages
// from reaching swap. It is best-effort: a failure is logged and the caller continues.
//
// Call it before the process holds anything worth protecting, since it only affects
// later reads. It costs a debugger too: delve cannot attach to a hardened process.
func Process(logger *slog.Logger) {
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		if logger != nil {
			slogerr.WithError(logger, err).Warn("mark the process non-dumpable")
		}
	}
}
