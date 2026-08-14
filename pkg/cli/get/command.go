// Package get implements both the 'ghtkn get' command and 'ghtkn git-credential' command.
// These commands retrieve or create GitHub App User Access Tokens and output them to stdout.
// The 'get' command outputs tokens in plain text or JSON format for general use.
// The 'git-credential' command outputs tokens in Git's credential helper format for seamless Git authentication.
// Both commands handle token persistence, expiration checking, and automatic renewal when needed.
package get

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/completion"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cobrautil"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

const getDescription = `Output a GitHub App User Access Token to stdout.

Prefer 'ghtkn exec', which puts the token in the environment of a command instead of
printing it: 'ghtkn exec -e GH_TOKEN -- gh issue list'. ghtkn then writes the token
nowhere, though a command that prints its own environment still exposes it. For git,
prefer the credential helper ('ghtkn git-credential'): git runs it and reads the
credential itself, so the token never passes through your shell.

The output is a secret. Do not print, echo, log, or include it in a chat message,
a commit, or any other output, and do not run 'ghtkn get' (including -f json) just
to display or inspect the token. If you are a coding agent, this applies to your
responses too: a leaked token can be used until it is revoked.

It returns the token cached in the backend (keyring, agent, or text) when one is
available and still valid, and with the agent backend and refresh enabled it also
renews an expiring one silently from the stored refresh token. Otherwise it fails
immediately, because the only way left is GitHub's interactive OAuth device flow,
which only 'ghtkn auth' starts, so no command ever starts one on your behalf. Run
'ghtkn auth' in your own terminal to authenticate.

If an app name is given, the token is issued for that app; otherwise GHTKN_APP or
the default app in the config is used. Use --min-expiration to force regeneration
when the cached token expires within the given duration.

$ ghtkn get
$ ghtkn get my-app
$ ghtkn get -f json my-app
$ ghtkn get -m 1h my-app`

//nolint:gosec // This is the command's help text, not a hardcoded credential.
const gitCredentialDescription = `Act as a Git credential helper that supplies GitHub App User Access Tokens.

Git invokes this command following its credential helper protocol. Only the 'get'
operation is handled; it outputs a token for the requested host in Git's credential
format so that Git pushes and pulls authenticate with a ghtkn token automatically.
The app is selected by apps[].git_owner or apps[].git_owners (with
credential.useHttpPath true) or by GHTKN_GIT_APP.

Configure it in Git (an empty helper first disables other helpers):

$ git config --global credential.helper ''
$ git config --global --add credential.helper '!ghtkn git-credential'`

// Args holds the flag and argument values for the get and git-credential commands.
type Args struct {
	*flag.GlobalFlags

	Format        string
	MinExpiration string
	AppName       string // positional argument for 'get' command
	SubCommand    string // positional argument for 'git-credential' command (e.g., "get")
}

// New creates either a 'get' or 'git-credential' command instance based on the isGitCredential flag.
// When isGitCredential is true, it creates a Git credential helper command.
// When false, it creates a standard get command for general token retrieval.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, env *cobrautil.Env, isGitCredential bool, gFlags *flag.GlobalFlags) *cobra.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	r := &runner{
		isGitCredential: isGitCredential,
		stdin:           env.Stdin,
		getEnv:          env.Getenv,
	}
	return r.Command(logger, args)
}

// runner encapsulates the state and behavior for both the get and git-credential commands.
type runner struct {
	isGitCredential bool
	stdin           io.Reader
	getEnv          func(string) string
}

// Command returns the CLI command definition for either the get or git-credential subcommand.
// For git-credential, it creates a command compatible with Git's credential helper protocol.
// For get, it creates a standard command with output format options.
// It defines the command name, usage, action handler, and available flags.
func (r *runner) Command(logger *slogutil.Logger, args *Args) *cobra.Command {
	if r.isGitCredential {
		cmd := &cobra.Command{
			Use:   "git-credential [<subcommand>]",
			Short: "Git Credential Helper",
			Long:  gitCredentialDescription,
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, positional []string) error {
				if len(positional) > 0 {
					args.SubCommand = positional[0]
				}
				return r.action(cmd.Context(), logger, args)
			},
		}
		flag.MinExpiration(cmd.Flags(), &args.MinExpiration)
		return cmd
	}
	cmd := &cobra.Command{
		Use:   "get [<app-name>]",
		Short: "Output a GitHub App User Access Token to stdout",
		Long:  getDescription,
		Args:  cobra.MaximumNArgs(1),
		// git-credential takes no app name, so only 'get' completes one.
		ValidArgsFunction: completion.AppName(&args.Config),
		RunE: func(cmd *cobra.Command, positional []string) error {
			if len(positional) > 0 {
				args.AppName = positional[0]
			}
			return r.action(cmd.Context(), logger, args)
		},
	}
	flag.Format(cmd.Flags(), &args.Format)
	flag.MinExpiration(cmd.Flags(), &args.MinExpiration)
	return cmd
}

// action implements the main logic for both the get and git-credential commands.
// For git-credential, it follows Git's credential helper protocol and only processes 'get' operations.
// For get command, it supports different output formats (plain text or JSON).
// It configures the controller with flags and arguments, then executes the token retrieval.
// Returns an error if configuration is invalid or token retrieval fails.
func (r *runner) action(ctx context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	inputGet := &ghtkn.InputGet{
		ConfigFilePath: args.Config,
	}
	if err := flag.SetMinExpiration(inputGet, args.MinExpiration); err != nil {
		return err //nolint:wrapcheck
	}

	input, err := get.NewInput()
	if err != nil {
		return fmt.Errorf("create the controller input: %w", err)
	}
	if r.isGitCredential {
		skip, err := r.handleGitCredential(ctx, logger.Logger, args.SubCommand, input, inputGet)
		if err != nil {
			return err
		}
		if skip {
			return nil
		}
	} else {
		setupGet(args, input, inputGet)
	}
	p, err := config.ResolvePath(inputGet.ConfigFilePath)
	if err != nil {
		return err //nolint:wrapcheck
	}
	inputGet.ConfigFilePath = p
	if err := input.Validate(); err != nil {
		return err //nolint:wrapcheck
	}
	return get.New(input).Run(ctx, logger.Logger, &get.InputRun{ //nolint:wrapcheck
		InputGet: inputGet,
	})
}

// setupGet applies the 'get'-only flags and argument to the controller input and SDK
// request. git-credential registers no output format and takes no app name (it selects
// the app from the Git request), so it is never called for it.
func setupGet(args *Args, input *get.Input, inputGet *ghtkn.InputGet) {
	input.OutputFormat = args.Format
	if args.AppName != "" {
		inputGet.AppName = args.AppName
	}
}
