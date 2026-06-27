package getcode

import (
	"context"
	"fmt"
	"io"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

type Args struct {
	*flag.GlobalFlags
}

func New(logger *slogutil.Logger, env *urfave.Env, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	r := &runner{
		stdin: env.Stdin,
	}
	return r.Command(logger, args)
}

type runner struct {
	isGitCredential bool
	stdin           io.Reader
}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	return &cli.Command{
		Name:  "get-code",
		Usage: "Output the oldest device code captured during device flow",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.action(ctx, logger, args)
		},
	}
}

func (r *runner) action(ctx context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	inputGet := &ghtkn.InputGet{}
	inputGet.ConfigFilePath = args.Config

	input := get.NewInput()
	if inputGet.ConfigFilePath == "" {
		p, err := ghtkn.GetConfigPath()
		if err != nil {
			return fmt.Errorf("get the config path: %w", err)
		}
		inputGet.ConfigFilePath = p
	}
	if err := input.Validate(); err != nil {
		return err //nolint:wrapcheck
	}
	return get.New(input).Run(ctx, logger.Logger, inputGet) //nolint:wrapcheck
}
