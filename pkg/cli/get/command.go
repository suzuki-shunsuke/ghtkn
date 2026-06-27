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
			Name:  "git-credential",
			Usage: "Git Credential Helper",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return r.action(ctx, cmd, logger, args)
			},
			Flags: []cli.Flag{
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
		Name:  "get",
		Usage: "Output a GitHub App User Access Token to stdout",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return r.action(ctx, cmd, logger, args)
		},
		Flags: []cli.Flag{
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
		if err := r.handleGitCredential(ctx, logger.Logger, args.SubCommand, input, inputGet); err != nil {
			return err
		}
	} else {
		input.OutputFormat = args.Format
		if args.AppName != "" {
			inputGet.AppName = args.AppName
		}
		// Only the 'get' command exposes --device-flow. Pass the override only when the
		// flag is explicitly set so it takes precedence over GHTKN_ENABLE_DEVICE_FLOW
		// and the config; otherwise leave it nil so the SDK resolves them itself.
		// git-credential never registers the flag, so IsSet is always false there.
		if cmd.IsSet("device-flow") {
			inputGet.EnableDeviceFlow = &args.DeviceFlow
		}
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
