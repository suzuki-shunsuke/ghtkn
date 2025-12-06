// Package cli provides the command-line interface layer for ghtkn.
// This package serves as the main entry point for all CLI operations,
// handling command parsing, flag processing, and routing to appropriate subcommands.
// It orchestrates the overall CLI structure using urfave/cli framework and delegates
// actual business logic to controller packages.
package cli

import (
	"context"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/initcmd"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

// Run creates and executes the main ghtkn CLI application.
// It configures the command structure with global flags and subcommands,
// then runs the CLI with the provided arguments.
// args are command line arguments to parse and execute
// Returns an error if command parsing or execution fails.
func Run(ctx context.Context, logger *slogutil.Logger, env *urfave.Env) error {
	return urfave.Command(env, &cli.Command{ //nolint:wrapcheck
		Name:  "ghtkn",
		Usage: "Create GitHub App User Access Tokens for secure local development. https://github.com/suzuki-shunsuke/ghtkn",
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
		},
		Commands: []*cli.Command{
			initcmd.New(logger),
			get.New(logger, env, true),
			get.New(logger, env, false),
		},
	}).Run(ctx, env.Args)
}
