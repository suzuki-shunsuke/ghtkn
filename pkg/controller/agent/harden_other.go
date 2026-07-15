//go:build !linux

package agent

import "log/slog"

// hardenProcess is a no-op off Linux. On macOS the OS already blocks reading another
// process's memory (task_for_pid needs root plus a debugger entitlement and is gated by
// SIP) and disables core dumps by default, so there is no PR_SET_DUMPABLE counterpart to
// add here.
func hardenProcess(_ *slog.Logger) {}
