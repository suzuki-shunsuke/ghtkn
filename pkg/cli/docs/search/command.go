package search

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs/search"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags

	Query string
}

func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	r := &runner{}
	return r.Command(logger, args)
}

type runner struct {
	stdin io.Reader
}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "Search document",
		Description: `Search documents by query. This is useful for coding agent to search document and solve problems.
This command outputs a list of documents including the query.
The output only includes the document ID and description.
To read all contents, please run "ghtkn docs show <id>"
	`,
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "query",
				Destination: &args.Query,
			},
		},
	}
}

func (r *runner) action(_ context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return search.New(os.Stdout).Search(args.Query) //nolint:wrapcheck
}
