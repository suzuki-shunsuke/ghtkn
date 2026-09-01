// Package agent implements the 'ghtkn agent' command and its subcommands.
// The agent is a long-running process that caches GitHub App access tokens and
// serves them to clients over a Unix domain socket. It is intended for environments
// where the OS keyring is unavailable (containers, VMs, minimal Linux, etc.).
//
// This package provides the 'start', 'stop', 'status', 'unlock', and 'reset'
// subcommands. The agent starts locked and is unlocked with a passphrase via
// 'unlock'; tokens are encrypted at rest. The agent server lives in
// pkg/agent/server.
package agent

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/cobra-util/cobrautil"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/server"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/lock"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/reset"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/status"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/stop"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/unlock"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// New creates the 'agent' parent command for the CLI application.
// The parent command groups the agent subcommands (currently only 'start').
// The returned command can be added to the CLI application's command list.
func New(logger *slogutil.Logger, env *cobrautil.Env, gFlags *flag.GlobalFlags) *cobra.Command {
	r := &runner{
		logger:  logger,
		flags:   gFlags,
		version: env.Version,
	}
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage the ghtkn agent that caches access tokens and serves them over a Unix socket",
		Long: `Manage the ghtkn agent.

The agent is a long-running process that caches GitHub App access tokens and serves
them to clients over a Unix domain socket. It is intended for environments where the
OS keyring is unavailable (containers, VMs, minimal Linux, etc.). Select it with
GHTKN_BACKEND=agent or backend.type: agent in the config.

The agent starts locked; unlock it with a passphrase to make cached tokens available.
Tokens are encrypted at rest with AES-256-GCM.`,
	}
	cmd.AddCommand(
		r.startCommand(),
		r.stopCommand(),
		r.statusCommand(),
		r.unlockCommand(),
		r.lockCommand(),
		r.resetCommand(),
	)
	return cmd
}

// runner holds the dependencies for the agent subcommands.
type runner struct {
	logger *slogutil.Logger
	flags  *flag.GlobalFlags
	// version is the ghtkn version, handed to the agent server so it can report which
	// binary the long-running process runs (see the STATUS response and 'ghtkn info').
	version string
}

// unlockArgs holds the flag values for the 'agent unlock' subcommand.
// The other subcommands have no flags of their own, so they read the global flags
// from the runner directly.
type unlockArgs struct {
	*flag.GlobalFlags

	EnableRefresh   bool
	RefreshTokenTTL string
}

// warnIfBackendNotAgent logs a warning when the resolved storage backend is not the
// agent. Running any 'ghtkn agent' subcommand implies the user means to use the agent
// backend; if it is not selected, 'ghtkn get'/'auth' silently use another backend (the
// OS keyring by default) and never touch this agent. It is best-effort: a resolution
// error is logged at debug and does not fail the command. It reflects this process's
// environment and config, which can differ from the environment a later 'ghtkn get'/
// 'auth' runs in.
func (r *runner) warnIfBackendNotAgent() {
	// The global --config flag selects the config file; when it is empty the SDK falls
	// back to GHTKN_CONFIG and then the default path.
	cfg, err := ghtkn.LoadConfig(&ghtkn.InputLoadConfig{ConfigFilePath: r.flags.Config})
	if err != nil {
		slogerr.WithError(r.logger.Logger, err).Debug("resolve the backend for the agent-usage check")
		return
	}
	backend := "keyring"
	if cfg.Backend != nil && cfg.Backend.Type != "" {
		backend = cfg.Backend.Type
	}
	if backend != "agent" {
		r.logger.Warn(`the configured backend is not "agent", so ghtkn get/auth will not use the ghtkn agent. `+
			`Set GHTKN_BACKEND=agent or backend.type: agent to use it.`,
			"backend", backend)
	}
}

// startCommand returns the CLI command definition for the 'agent start' subcommand.
// It configures the command name, usage description, and action handler.
func (r *runner) startCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the ghtkn agent in the foreground (locked)",
		Args:  cobra.NoArgs,
		Long: `Start the ghtkn agent in the foreground.

The agent starts locked and listens on a Unix domain socket without asking for a
passphrase, so it can run as a background service (e.g. systemd). Use
'ghtkn agent unlock' to enter the passphrase and make cached tokens available.
It keeps running until it receives SIGINT or SIGTERM, then removes the socket and exits.

$ ghtkn agent start`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return r.start(cmd.Context())
		},
	}
}

// start executes the 'agent start' command logic.
// It configures the log level and runs the agent controller until the process is signaled.
func (r *runner) start(ctx context.Context) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	r.warnIfBackendNotAgent()
	return server.New(r.version).Start(ctx, r.logger.Logger) //nolint:wrapcheck
}

// stopCommand returns the CLI command definition for the 'agent stop' subcommand.
func (r *runner) stopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running ghtkn agent",
		Args:  cobra.NoArgs,
		Long: `Stop the running ghtkn agent.

It connects to the agent's Unix domain socket and asks it to shut down.

$ ghtkn agent stop`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return r.stop(cmd.Context())
		},
	}
}

// stop executes the 'agent stop' command logic.
// It configures the log level and asks the running agent to shut down.
func (r *runner) stop(ctx context.Context) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	r.warnIfBackendNotAgent()
	return stop.New().Run(ctx, r.logger.Logger) //nolint:wrapcheck
}

// statusCommand returns the CLI command definition for the 'agent status' subcommand.
func (r *runner) statusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether the ghtkn agent is running",
		Args:  cobra.NoArgs,
		Long: `Show whether the ghtkn agent is running.

It connects to the agent's Unix domain socket and reports the number of cached
access tokens, along with the ghtkn version the running agent was built from and
the agent protocol version it speaks. The agent keeps running the binary it was
started with, so an agent version older than 'ghtkn --version' means the agent
must be restarted. It exits 0 whether or not the agent is running.

$ ghtkn agent status`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return r.status(cmd.Context())
		},
	}
}

// status executes the 'agent status' command logic.
// It configures the log level and reports whether the agent is running.
func (r *runner) status(ctx context.Context) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	r.warnIfBackendNotAgent()
	return status.New().Run(ctx, r.logger.Logger) //nolint:wrapcheck
}

// unlockCommand returns the CLI command definition for the 'agent unlock' subcommand.
func (r *runner) unlockCommand() *cobra.Command {
	args := &unlockArgs{
		GlobalFlags: r.flags,
	}
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock the running ghtkn agent by entering the passphrase",
		Args:  cobra.NoArgs,
		Long: `Unlock the running ghtkn agent.

The agent starts locked. This command prompts for the passphrase on the terminal
and sends it to the agent over the socket so it can decrypt cached tokens. On first
use it asks for a new passphrase twice to confirm it.

Pass --enable-refresh to let the agent refresh an expiring access token with a
stored refresh token instead of re-running the device flow. This is bound to the
passphrase on purpose: it cannot be enabled without unlocking the agent. It is
unsupported on Windows, where --enable-refresh is rejected, because the file
permissions and process hardening that protect a stored refresh token are
POSIX-specific.

With refresh enabled, the agent periodically discards tokens left unused for longer
than --refresh-token-ttl (default 7 days) so an unused refresh token does not linger.
The TTL takes a number with a d (day), w (week), or m (30-day month) suffix, e.g.
14d, 4w, 2m, and must be less than 6 months.

$ ghtkn agent unlock`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return r.unlock(cmd.Context(), args)
		},
	}
	cmd.Flags().BoolVar(&args.EnableRefresh, "enable-refresh",
		false, "Enable refreshing expiring access tokens with stored refresh tokens")
	// No default value: an empty value means the flag was not given, which lets the
	// agent apply its own default (seven days) instead of duplicating it here, and lets
	// this command tell "not given" from an explicit value it must reject without
	// --enable-refresh.
	cmd.Flags().StringVar(&args.RefreshTokenTTL, "refresh-token-ttl",
		"", "How long a stored token may sit unused before the agent discards it, e.g. 14d/4w/2m (default 7d; only with --enable-refresh)")
	return cmd
}

// unlock executes the 'agent unlock' command logic.
// It configures the log level and unlocks the running agent.
func (r *runner) unlock(ctx context.Context, args *unlockArgs) error {
	if err := r.logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	r.warnIfBackendNotAgent()
	if err := checkRefreshTokenSupported(args.EnableRefresh, runtime.GOOS); err != nil {
		return err
	}
	ttl, err := refreshTokenTTL(args.EnableRefresh, args.RefreshTokenTTL)
	if err != nil {
		return err
	}
	return unlock.New().Run(ctx, r.logger.Logger, args.EnableRefresh, ttl) //nolint:wrapcheck
}

// lockCommand returns the CLI command definition for the 'agent lock' subcommand.
func (r *runner) lockCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "lock",
		Short: "Lock the running ghtkn agent by discarding its in-memory data key",
		Args:  cobra.NoArgs,
		Long: `Lock the running ghtkn agent.

It asks the agent to discard the data key it holds in memory and return to the
locked state, without stopping the process or deleting the key file. Cached tokens
become unreadable until you run 'ghtkn agent unlock' again with the same passphrase.
Unlike 'ghtkn agent stop', the process and socket keep running, and unlike 'unlock',
locking needs no passphrase, so it can be wired to a screen-lock or logout hook to
shrink the window in which the agent holds decrypted tokens.

$ ghtkn agent lock`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return r.lock(cmd.Context())
		},
	}
}

// lock executes the 'agent lock' command logic.
// It configures the log level and asks the running agent to discard its data key.
func (r *runner) lock(ctx context.Context) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	r.warnIfBackendNotAgent()
	return lock.New().Run(ctx, r.logger.Logger) //nolint:wrapcheck
}

// resetCommand returns the CLI command definition for the 'agent reset' subcommand.
func (r *runner) resetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset the agent after a forgotten passphrase (deletes the key and cached tokens)",
		Args:  cobra.NoArgs,
		Long: `Reset the ghtkn agent when you have forgotten the passphrase.

It stops the agent if it is running, deletes the key file and all encrypted access
token files, and creates a new key from a freshly entered passphrase. The old
passphrase is not needed and the cached tokens are discarded (they are reminted from
GitHub on the next 'ghtkn get'). It asks for confirmation first.

It leaves the agent stopped, so start it again and unlock it with the new passphrase
afterwards; until then every 'ghtkn get' fails.

$ ghtkn agent reset
$ ghtkn agent start
$ ghtkn agent unlock`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return r.reset(cmd.Context())
		},
	}
}

// reset executes the 'agent reset' command logic.
// It configures the log level and reinitializes the agent's key.
func (r *runner) reset(ctx context.Context) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	r.warnIfBackendNotAgent()
	return reset.New().Run(ctx, r.logger.Logger) //nolint:wrapcheck
}
