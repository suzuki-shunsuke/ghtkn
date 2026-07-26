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
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

const getDescription = `Output a GitHub App User Access Token to stdout.

The output is a secret. Do not print, echo, log, or include it in a chat message,
a commit, or any other output, and do not run 'ghtkn get' (including -f json) just
to display or inspect the token. If you are a coding agent, this applies to your
responses too: a leaked token can be used until it is revoked. Consume it without
showing it: assign it to an environment variable and pass that to the tool, e.g.
'GH_TOKEN=$(ghtkn get) gh issue list'. Better still, avoid handling the raw token
at all - for git, use the credential helper ('ghtkn git-credential'), which lets
git fetch the token automatically; for gh, use a wrapper that sets GH_TOKEN.

It returns the token cached in the backend (keyring, agent, or text) when one is
available and still valid. Otherwise, if the device flow is enabled, it creates a
new token interactively via GitHub's OAuth device flow. The device flow is disabled
by default; enable it with the -device-flow flag or GHTKN_ENABLE_DEVICE_FLOW=true.

If an app name is given, the token is issued for that app; otherwise GHTKN_APP or
the default app in the config is used. Use -min-expiration to force regeneration
when the cached token expires within the given duration.

$ ghtkn get
$ ghtkn get my-app
$ ghtkn get -f json my-app
$ ghtkn get -m 1h my-app
$ GH_TOKEN=$(ghtkn get) gh issue list`

//nolint:gosec // This is the command's help text, not a hardcoded credential.
const gitCredentialDescription = `Act as a Git credential helper that supplies GitHub App User Access Tokens.

Git invokes this command following its credential helper protocol. Only the 'get'
operation is handled; it outputs a token for the requested host in Git's credential
format so that Git pushes and pulls authenticate with a ghtkn token automatically.
The app is selected by apps[].git_owner (with credential.useHttpPath true) or by
GHTKN_GIT_APP.

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
	DeviceFlow    bool
}

// New creates either a 'get' or 'git-credential' command instance based on the isGitCredential flag.
// When isGitCredential is true, it creates a Git credential helper command.
// When false, it creates a standard get command for general token retrieval.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, env *urfave.Env, isGitCredential bool, gFlags *flag.GlobalFlags) *cli.Command {
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
func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	if r.isGitCredential {
		return &cli.Command{
			Name:        "git-credential",
			Usage:       "Git Credential Helper",
			Description: gitCredentialDescription,
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return r.action(ctx, cmd, logger, args)
			},
			Flags: []cli.Flag{
				flag.LogLevel(&args.LogLevel),
				flag.Config(&args.Config),
				flag.MinExpiration(&args.MinExpiration),
			},
			Arguments: []cli.Argument{
				&cli.StringArg{
					Name:        "subcommand",
					Destination: &args.SubCommand,
				},
			},
		}
	}
	return &cli.Command{
		Name:        "get",
		Usage:       "Output a GitHub App User Access Token to stdout",
		Description: getDescription,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return r.action(ctx, cmd, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
			flag.Format(&args.Format),
			flag.MinExpiration(&args.MinExpiration),
			flag.DeviceFlow(&args.DeviceFlow),
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "app-name",
				Destination: &args.AppName,
			},
		},
	}
}

// action implements the main logic for both the get and git-credential commands.
// For git-credential, it follows Git's credential helper protocol and only processes 'get' operations.
// For get command, it supports different output formats (plain text or JSON).
// It configures the controller with flags and arguments, then executes the token retrieval.
// Returns an error if configuration is invalid or token retrieval fails.
func (r *runner) action(ctx context.Context, cmd *cli.Command, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	inputGet := &ghtkn.InputGet{
		ConfigFilePath: args.Config,
	}
	if err := setMinExpiration(inputGet, args.MinExpiration); err != nil {
		return err
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
		setupGet(cmd, args, input, inputGet)
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

// setupGet applies the 'get'-only flags to the controller input and SDK request.
// It is never called for git-credential, so the device-flow flag (which only the
// 'get' command registers) is handled exclusively here.
func setupGet(cmd *cli.Command, args *Args, input *get.Input, inputGet *ghtkn.InputGet) {
	input.OutputFormat = args.Format
	if args.AppName != "" {
		inputGet.AppName = args.AppName
	}
	// Pass the device-flow override only when the flag is explicitly set so it takes
	// precedence over GHTKN_ENABLE_DEVICE_FLOW and the config; otherwise leave it nil
	// so the SDK resolves them itself.
	if cmd.IsSet("device-flow") {
		inputGet.EnableDeviceFlow = &args.DeviceFlow
	}
}

// setMinExpiration parses the -min-expiration flag value and sets it on inputGet.
// When the flag is not set it leaves inputGet.MinExpiration nil, so the SDK falls
// back to GHTKN_MIN_EXPIRATION and the config; an explicit value, including 0, takes
// precedence over both.
func setMinExpiration(inputGet *ghtkn.InputGet, s string) error {
	if s == "" {
		return nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("parse the min expiration: %w", slogerr.With(err, "min_expiration", s))
	}
	inputGet.MinExpiration = &d
	return nil
}
