package show

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/docs/show"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags

	DocName string
}

func New(logger *slogutil.Logger, gFlags *flag.GlobalFlags) *cobra.Command {
	args := &Args{
		GlobalFlags: gFlags,
	}
	r := &runner{}
	return r.Command(logger, args)
}

type runner struct{}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [<doc>]",
		Short: "Output the content of a given document",
		Long: `Output document. This is useful for coding agent to read the document and solve problems.
This command needs a document name.
To see the name, list documents with "ghtkn docs list"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, positional []string) error {
			if len(positional) > 0 {
				args.DocName = positional[0]
			}
			return r.action(cmd.Context(), logger, args)
		},
	}
	return cmd
}

func (r *runner) action(_ context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	return show.New(os.Stdout).Show(args.DocName) //nolint:wrapcheck
}
