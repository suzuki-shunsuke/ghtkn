package info

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/info"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags

	AppName string // positional argument for 'info' command
	Version string
}

func New(logger *slogutil.Logger, env *urfave.Env, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
		Version:     env.Version,
	}
	r := &runner{
		stdin: env.Stdin,
	}
	return r.Command(logger, args)
}

type runner struct {
	stdin io.Reader
}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Output a information about the environment which is useful for troubleshooting",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
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
func (r *runner) action(_ context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	inputGet := &ghtkn.InputGet{}
	inputGet.ConfigFilePath = args.Config

	input, err := get.NewInput()
	if err != nil {
		return fmt.Errorf("create the controller input: %w", err)
	}
	if args.AppName != "" {
		inputGet.AppName = args.AppName
	}
	// Only the 'get' command exposes --device-flow; the override lets the flag
	// take precedence over GHTKN_ENABLE_DEVICE_FLOW. git-credential leaves this
	// nil so the SDK falls back to the environment variable.
	if err := resolveConfigFilePath(inputGet); err != nil {
		return err
	}
	if err := input.Validate(); err != nil {
		return err //nolint:wrapcheck
	}
	return info.New(os.Stdout, os.Getenv).Info(args.AppName, args.Version) //nolint:wrapcheck
}

// resolveConfigFilePath fills in inputGet.ConfigFilePath with the default
// configuration path when it has not been set by a flag.
func resolveConfigFilePath(inputGet *ghtkn.InputGet) error {
	if inputGet.ConfigFilePath != "" {
		return nil
	}
	p, err := ghtkn.GetConfigPath()
	if err != nil {
		return fmt.Errorf("get the config path: %w", err)
	}
	inputGet.ConfigFilePath = p
	return nil
}
