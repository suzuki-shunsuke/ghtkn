// Package cli provides the command-line interface layer for ghtkn.
// This package serves as the main entry point for all CLI operations,
// handling command parsing, flag processing, and routing to appropriate subcommands.
// It orchestrates the overall CLI structure using urfave/cli framework and delegates
// actual business logic to controller packages.
package cli

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/initcmd"
	"github.com/suzuki-shunsuke/go-stdutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

// Run creates and executes the main ghtkn CLI application.
// It configures the command structure with global flags and subcommands,
// then runs the CLI with the provided arguments.
//
// Parameters:
//   - ctx: context for cancellation and timeout control
//   - logE: logrus entry for structured logging
//   - ldFlags: linker flags containing build information
//   - args: command line arguments to parse and execute
//
// Returns an error if command parsing or execution fails.
func Run(ctx context.Context, logger *slog.Logger, ldFlags *stdutil.LDFlags, args ...string) error {
	return urfave.Command(ldFlags, &cli.Command{ //nolint:wrapcheck
		Name:  "ghtkn",
		Usage: "Create GitHub App User Access Tokens for secure local development. https://github.com/suzuki-shunsuke/ghtkn",
		Flags: []cli.Flag{
			flag.LogLevel(),
			flag.Config(),
		},
		Commands: []*cli.Command{
			initcmd.New(logger, ldFlags.Version),
			get.New(logger, ldFlags.Version, true),
			get.New(logger, ldFlags.Version, false),
		},
	}).Run(ctx, args)
}
