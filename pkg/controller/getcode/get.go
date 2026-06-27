package getcode

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

func (c *Controller) Run(ctx context.Context, logger *slog.Logger, input *ghtkn.InputGet) error {
	// get dir path
	// get files
	// sort file names
	// remove expired files
	// read oldest file
	// remove oldest file
	// output code
	return nil
}
