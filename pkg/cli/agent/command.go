// Package agent implements the 'ghtkn agent' command and its subcommands.
// The agent is a long-running process that caches GitHub App access tokens and
// serves them to clients over a Unix domain socket. It is intended for environments
// where the OS keyring is unavailable (containers, VMs, minimal Linux, etc.).
//
// This package currently provides the 'start' subcommand, which launches the agent
// in the foreground. Token caching is kept in memory only; on-disk encryption is
// planned for a later change.
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
		Usage: "Start the ghtkn agent in the foreground",
		Description: `Start the ghtkn agent in the foreground.

The agent listens on a Unix domain socket and serves cached access tokens to clients.
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
