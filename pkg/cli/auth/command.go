// Package auth implements the 'ghtkn auth' command.
// It authenticates to GitHub and caches a GitHub App User Access Token without
// printing it to stdout. When no valid token is cached, it runs the OAuth device
// flow to create one. Unlike 'ghtkn get', the device flow is always allowed,
// regardless of GHTKN_ENABLE_DEVICE_FLOW, because authentication is inherently
// interactive.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the auth command.
type Args struct {
	*flag.GlobalFlags

	MinExpiration string
	AppName       string // positional argument
}

// New creates a new auth command instance with the provided logger.
// It returns a CLI command that can be registered with the main CLI application.
func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	return &cli.Command{
		Name:  "auth",
		Usage: "Authenticate to GitHub and cache an access token without outputting it",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
			flag.MinExpiration(&args.MinExpiration),
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "app-name",
				Destination: &args.AppName,
			},
		},
	}
}

// action authenticates to GitHub and caches an access token without printing it.
// It reuses the get controller with Silent enabled, and always allows the device
// flow so authentication works even when GHTKN_ENABLE_DEVICE_FLOW is false.
func action(ctx context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	inputGet := &ghtkn.InputGet{}
	if args.MinExpiration != "" {
		d, err := time.ParseDuration(args.MinExpiration)
		if err != nil {
			return fmt.Errorf("parse the min expiration: %w", slogerr.With(err, "min_expiration", args.MinExpiration))
		}
		inputGet.MinExpiration = d
	}
	inputGet.ConfigFilePath = args.Config
	if args.AppName != "" {
		inputGet.AppName = args.AppName
	}
	// auth always allows the device flow, overriding GHTKN_ENABLE_DEVICE_FLOW,
	// because it is an explicit, interactive authentication command.
	enable := true
	inputGet.EnableDeviceFlow = &enable

	input, err := get.NewInput()
	if err != nil {
		return fmt.Errorf("create the controller input: %w", err)
	}
	if err := resolveConfigFilePath(inputGet); err != nil {
		return err
	}
	if err := input.Validate(); err != nil {
		return err //nolint:wrapcheck
	}
	return get.New(input).Run(ctx, logger.Logger, &get.InputRun{ //nolint:wrapcheck
		Silent:   true,
		InputGet: inputGet,
	})
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
