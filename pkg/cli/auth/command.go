// Package auth implements the 'ghtkn auth' command.
// It authenticates to GitHub and caches a GitHub App User Access Token without
// printing it to stdout. It always runs the OAuth device flow to regenerate the
// token, regardless of any cached token, so that running it proactively refreshes
// the cached token before it expires. Unlike 'ghtkn get', the device flow is
// always allowed, regardless of GHTKN_ENABLE_DEVICE_FLOW, because authentication
// is inherently interactive. Unlike 'ghtkn get', it does not accept the
// -min-expiration flag nor read GHTKN_MIN_EXPIRATION; that knob is reserved for
// 'ghtkn get'.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// alwaysRenewMinExpiration forces 'ghtkn auth' to always regenerate the token via
// the device flow. It must exceed GitHub's 8h User Access Token TTL so the cached
// token is always treated as expired (see checkExpired in the SDK).
const alwaysRenewMinExpiration = 9 * time.Hour

// Args holds the flag and argument values for the auth command.
type Args struct {
	*flag.GlobalFlags

	AppName string // positional argument
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
	// auth always regenerates the token, so it ignores -min-expiration,
	// GHTKN_MIN_EXPIRATION and the config's min_expiration, and forces a min expiration
	// larger than the token TTL. Passing it as an explicit override (non-nil pointer)
	// makes it take precedence over the environment variable and config.
	inputGet.MinExpiration = new(alwaysRenewMinExpiration)
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
	p, err := config.ResolvePath(inputGet.ConfigFilePath)
	if err != nil {
		return err //nolint:wrapcheck
	}
	inputGet.ConfigFilePath = p
	if err := input.Validate(); err != nil {
		return err //nolint:wrapcheck
	}
	return get.New(input).Run(ctx, logger.Logger, &get.InputRun{ //nolint:wrapcheck
		Silent:   true,
		InputGet: inputGet,
	})
}
