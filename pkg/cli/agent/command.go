// Package agent implements the 'ghtkn agent' command and its subcommands.
// The agent is a long-running process that caches GitHub App access tokens and
// serves them to clients over a Unix domain socket. It is intended for environments
// where the OS keyring is unavailable (containers, VMs, minimal Linux, etc.).
//
// This package provides the 'start', 'stop', 'status', 'unlock', and 'reset'
// subcommands. The agent starts locked and is unlocked with a passphrase via
// 'unlock'; tokens are encrypted at rest. The agent server lives in
// pkg/controller/agent.
package agent

import (
	"context"
	"fmt"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// New creates the 'agent' parent command for the CLI application.
// The parent command groups the agent subcommands (currently only 'start').
// The returned command can be added to the CLI application's command list.
func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cli.Command {
	r := &runner{
		logger: logger,
		flags:  gFlags,
	}
	return &cli.Command{
		Name:  "agent",
		Usage: "Manage the ghtkn agent that caches access tokens and serves them over a Unix socket",
		Commands: []*cli.Command{
			r.startCommand(),
			r.stopCommand(),
			r.statusCommand(),
			r.unlockCommand(),
			r.resetCommand(),
		},
	}
}

// runner holds the dependencies for the agent subcommands.
type runner struct {
	logger *slogutil.Logger
	flags  *flag.GlobalFlags
}

// startCommand returns the CLI command definition for the 'agent start' subcommand.
// It configures the command name, usage description, and action handler.
func (r *runner) startCommand() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "Start the ghtkn agent in the foreground (locked)",
		Description: `Start the ghtkn agent in the foreground.

The agent starts locked and listens on a Unix domain socket without asking for a
passphrase, so it can run as a background service (e.g. systemd). Use
'ghtkn agent unlock' to enter the passphrase and make cached tokens available.
It keeps running until it receives SIGINT or SIGTERM, then removes the socket and exits.

$ ghtkn agent start`,
		Action: r.start,
	}
}

// start executes the 'agent start' command logic.
// It configures the log level and runs the agent controller until the process is signaled.
func (r *runner) start(ctx context.Context, _ *cli.Command) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return agent.New().Start(ctx, r.logger.Logger) //nolint:wrapcheck
}

// stopCommand returns the CLI command definition for the 'agent stop' subcommand.
func (r *runner) stopCommand() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "Stop the running ghtkn agent",
		Description: `Stop the running ghtkn agent.

It connects to the agent's Unix domain socket and asks it to shut down.

$ ghtkn agent stop`,
		Action: r.stop,
	}
}

// stop executes the 'agent stop' command logic.
// It configures the log level and asks the running agent to shut down.
func (r *runner) stop(ctx context.Context, _ *cli.Command) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return agent.New().Stop(ctx, r.logger.Logger) //nolint:wrapcheck
}

// statusCommand returns the CLI command definition for the 'agent status' subcommand.
func (r *runner) statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show whether the ghtkn agent is running",
		Description: `Show whether the ghtkn agent is running.

It connects to the agent's Unix domain socket and reports the number of cached
access tokens. It exits 0 whether or not the agent is running.

$ ghtkn agent status`,
		Action: r.status,
	}
}

// status executes the 'agent status' command logic.
// It configures the log level and reports whether the agent is running.
func (r *runner) status(ctx context.Context, _ *cli.Command) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return agent.New().Status(ctx, r.logger.Logger) //nolint:wrapcheck
}

// unlockCommand returns the CLI command definition for the 'agent unlock' subcommand.
func (r *runner) unlockCommand() *cli.Command {
	return &cli.Command{
		Name:  "unlock",
		Usage: "Unlock the running ghtkn agent by entering the passphrase",
		Description: `Unlock the running ghtkn agent.

The agent starts locked. This command prompts for the passphrase on the terminal
and sends it to the agent over the socket so it can decrypt cached tokens. On first
use it asks for a new passphrase twice to confirm it.

$ ghtkn agent unlock`,
		Action: r.unlock,
	}
}

// unlock executes the 'agent unlock' command logic.
// It configures the log level and unlocks the running agent.
func (r *runner) unlock(ctx context.Context, _ *cli.Command) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return agent.New().Unlock(ctx, r.logger.Logger) //nolint:wrapcheck
}

// resetCommand returns the CLI command definition for the 'agent reset' subcommand.
func (r *runner) resetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Reset the agent after a forgotten passphrase (deletes the key and cached tokens)",
		Description: `Reset the ghtkn agent when you have forgotten the passphrase.

It stops the agent if it is running, deletes the key file and all encrypted access
token files, and creates a new key from a freshly entered passphrase. The old
passphrase is not needed and the cached tokens are discarded (they are reminted from
GitHub on the next 'ghtkn get'). It asks for confirmation first.

$ ghtkn agent reset`,
		Action: r.reset,
	}
}

// reset executes the 'agent reset' command logic.
// It configures the log level and reinitializes the agent's key.
func (r *runner) reset(ctx context.Context, _ *cli.Command) error {
	if err := r.logger.SetLevel(r.flags.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return agent.New().Reset(ctx, r.logger.Logger) //nolint:wrapcheck
}
