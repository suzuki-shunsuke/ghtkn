// Package cli provides the command-line interface layer for ghtkn.
// This package serves as the main entry point for all CLI operations,
// handling command parsing, flag processing, and routing to appropriate subcommands.
// It orchestrates the overall CLI structure using urfave/cli framework and delegates
// actual business logic to controller packages.
package cli

import (
	"context"

	sdkenv "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/env"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/auth"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/info"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/initcmd"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/revoke"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
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
	gFlags := &flag.GlobalFlags{}
	cmd := urfave.Command(env, &cli.Command{
		Name:  "ghtkn",
		Usage: "Create GitHub App User Access Tokens for secure local development. https://github.com/suzuki-shunsuke/ghtkn",
		Description: `Create GitHub App User Access Tokens for secure local development.

ghtkn issues short-lived GitHub App User Access Tokens via the OAuth device flow
and caches them in a backend (OS keyring, the ghtkn agent, or a plain text file).
Use it to authenticate the gh CLI, Git, and other tools without long-lived
Personal Access Tokens.

See each subcommand's help with 'ghtkn help <command>'.
See https://github.com/suzuki-shunsuke/ghtkn for details.`,
		Flags: []cli.Flag{
			flag.LogLevel(&gFlags.LogLevel),
			flag.Config(&gFlags.Config),
		},
		Commands: []*cli.Command{
			initcmd.New(logger, gFlags),
			get.New(logger, env, true, gFlags),
			get.New(logger, env, false, gFlags),
			auth.New(logger, gFlags),
			agent.New(logger, env, gFlags),
			revoke.New(logger, gFlags),
			info.New(logger, env, gFlags),
			docs.New(logger, gFlags),
		},
	})
	hintDocsOnVersion(cmd, logger, env, gFlags)
	if err := cmd.Run(ctx, env.Args); err != nil {
		return slogerr.With(err, "help", //nolint:wrapcheck
			"Run `ghtkn docs list` to see documentation and `ghtkn docs show <name>` to read it; this may help resolve the error.")
	}
	return nil
}

// docsHint points coding agents at `ghtkn docs`.
// ghtkn appends a similar hint to command errors, but an agent that only asks
// about ghtkn without reproducing a failure never sees that one.
const docsHint = "If you are a coding agent, run `ghtkn docs list` to list the documentation and `ghtkn docs show <name>` to read it before answering questions about ghtkn or troubleshooting its errors."

// hintDocsOnVersion makes `ghtkn -v`, `ghtkn --version`, and `ghtkn version` log
// docsHint. Checking the version is often the only ghtkn command a coding agent
// runs before it starts answering, so without this it never learns that
// `ghtkn docs` exists and reads the source code or the website instead.
//
// The hint goes to stderr as a log rather than to stdout so that it doesn't break
// scripts that parse the version, and it's logged at the info level so that
// `--log-level warn` silences it.
func hintDocsOnVersion(cmd *cli.Command, logger *slogutil.Logger, env *urfave.Env, gFlags *flag.GlobalFlags) {
	hint := func() {
		level := gFlags.LogLevel
		if level == "" {
			// cli handles -v before it applies the flags' environment variable
			// sources, so GHTKN_LOG_LEVEL has to be read directly here.
			level = env.Getenv(sdkenv.LogLevel)
		}
		// An invalid log level is ignored here because the version output must
		// succeed regardless. Subcommands report it when they set the level.
		_ = logger.SetLevel(level)
		logger.Info(docsHint)
	}
	// -v and --version run no action, so they need cli's printer hook.
	cli.VersionPrinter = func(c *cli.Command) {
		cli.DefaultPrintVersion(c)
		hint()
	}
	// The `version` subcommand comes from urfave-cli-v3-util, so wrap its action.
	for _, sub := range cmd.Commands {
		if sub.Name != "version" {
			continue
		}
		action := sub.Action
		sub.Action = func(ctx context.Context, c *cli.Command) error {
			if err := action(ctx, c); err != nil {
				return err
			}
			hint()
			return nil
		}
	}
}
