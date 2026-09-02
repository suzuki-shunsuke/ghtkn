// Package cli provides the command-line interface layer for ghtkn.
// This package serves as the main entry point for all CLI operations,
// handling command parsing, flag processing, and routing to appropriate subcommands.
// It orchestrates the overall CLI structure using cobra and delegates
// actual business logic to controller packages.
package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/cobra-util/cobrautil"
	"github.com/suzuki-shunsuke/cobra-util/jsonschema"
	schema "github.com/suzuki-shunsuke/ghtkn/json-schema"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/auth"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/docs"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/exec"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/info"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/initcmd"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/revoke"
	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// Run creates and executes the main ghtkn CLI application.
// It configures the command structure with global flags and subcommands,
// then runs the CLI with the provided arguments.
// Returns an error if command parsing or execution fails.
func Run(ctx context.Context, logger *slogutil.Logger, env *cobrautil.Env) error {
	gFlags := &flag.GlobalFlags{}
	// The context reaches the commands through ExecuteContext below, which is what
	// cobra hands to the action as cmd.Context(); building the tree needs none.
	cmd := newCommand(logger, env, gFlags) //nolint:contextcheck
	if err := cmd.ExecuteContext(ctx); err != nil {
		return withHelp(err)
	}
	return nil
}

// newCommand builds the command tree. It is separate from Run so that a test can run
// the real tree with its output captured, which Run can't offer: the commands write
// to os.Stdout.
func newCommand(logger *slogutil.Logger, env *cobrautil.Env, gFlags *flag.GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ghtkn",
		Short: "Create GitHub App User Access Tokens for secure local development. https://github.com/suzuki-shunsuke/ghtkn",
		Long: `Create GitHub App User Access Tokens for secure local development.

ghtkn issues short-lived GitHub App User Access Tokens via the OAuth device flow
and caches them in a backend (OS keyring, the ghtkn agent, or a plain text file).
Use it to authenticate the gh CLI, Git, and other tools without long-lived
Personal Access Tokens.

See each subcommand's help with 'ghtkn help <command>'.
See https://github.com/suzuki-shunsuke/ghtkn for details.

If you are a coding agent, run 'ghtkn docs list' to list the documentation and
'ghtkn docs show <doc>' to read it before answering questions about ghtkn or
troubleshooting its errors.`,
	}
	// --log-level and --config are persistent, so they can be given either before or
	// after the subcommand. Every subcommand reads them through the same GlobalFlags,
	// so nothing has to register them again.
	flag.LogLevel(cmd.PersistentFlags(), &gFlags.LogLevel)
	flag.Config(cmd.PersistentFlags(), &gFlags.Config)
	cmd.AddCommand(
		initcmd.New(logger, gFlags),
		get.New(logger, env, true, gFlags),
		get.New(logger, env, false, gFlags),
		exec.New(logger, gFlags),
		auth.New(logger, gFlags),
		agent.New(logger, env, gFlags),
		revoke.New(logger, gFlags),
		info.New(logger, env, gFlags),
		docs.New(logger, gFlags),
	)
	// json-schema is added on its own because it is the one command that takes
	// neither the logger nor the global flags: it only writes the embedded schema.
	// With names the program in the example in its help, taking the name from cmd.
	jsonschema.With(cmd, schema.Schema)
	return cobrautil.Command(env, cmd, &cobrautil.Options{
		AfterVersion: func() {
			hintDocsOnVersion(logger, gFlags)
		},
	})
}

// withHelp adds the hint about the documentation to an error while keeping any exit
// code it carries.
//
// The code has to be read before slogerr.With and attached again afterwards, because
// slogerr.With collapses the chain to the innermost error holding attributes and
// drops every wrapper around it, including the one holding the code. Exit codes come
// from 'ghtkn exec', which propagates the exit code of the command it ran.
func withHelp(err error) error {
	code := ecerror.GetExitCode(err)
	err = slogerr.With(err, "help",
		"Run `ghtkn docs list` to see documentation and `ghtkn docs show <name>` to read it; this may help resolve the error.")
	if code == 1 {
		// 1 is also what an error carrying no code exits with, so there is nothing to
		// preserve.
		return err //nolint:wrapcheck // The error is the command's own, only annotated.
	}
	return ecerror.Wrap(err, code)
}

// hintDocsOnVersion makes `ghtkn -v`, `ghtkn --version`, and `ghtkn version` log
// docs.Hint. Checking the version is often the only ghtkn command a coding agent
// runs before it starts answering, so without this it never learns that
// `ghtkn docs` exists and reads the source code or the website instead.
// ghtkn appends a similar hint to command errors, but an agent that only asks
// about ghtkn without reproducing a failure never sees that one.
//
// The hint goes to stderr as a log rather than to stdout so that it doesn't break
// scripts that parse the version, and it's logged at the info level so that
// `--log-level warn` silences it.
//
// GHTKN_LOG_LEVEL needs no special handling: cobrautil prints the version from the
// root command's action, which runs after the environment variables have been
// applied to the flags, so the log level is already resolved here.
func hintDocsOnVersion(logger *slogutil.Logger, gFlags *flag.GlobalFlags) {
	// An invalid log level is ignored here because the version output must
	// succeed regardless. Subcommands report it when they set the level.
	_ = logger.SetLevel(gFlags.LogLevel)
	logger.Info(docs.Hint)
}
