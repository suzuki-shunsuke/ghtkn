package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

// Run authenticates to GitHub and stores an access token in the backend, running the
// device flow when it needs to. It always regenerates the token, so running it
// proactively refreshes the cached token before it expires.
//
// Nothing is written to stdout, and the SDK hands back no token to write: the token is a
// secret and the point of this command is to cache it, not to hand it out. Printing it
// is 'ghtkn get'.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger, input *ghtkn.InputAuth) error {
	if err := c.input.Client.Auth(ctx, logger, input); err != nil {
		return fmt.Errorf("create an access token: %w", err)
	}
	return nil
}
