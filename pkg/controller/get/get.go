package get

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

// Run executes the main logic for retrieving a GitHub App access token.
// It reads configuration, checks for cached tokens, creates new tokens if needed,
// retrieves the authenticated user's login for Git Credential Helper if necessary,
// and outputs the result in the requested format.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger, input *ghtkn.InputGet) error {
	token, app, err := c.input.Client.Get(ctx, logger, input)
	if err != nil {
		return fmt.Errorf("get or create access token: %w", err)
	}

	// Output access token
	if err := c.output(app.Name, token); err != nil {
		return err
	}

	return nil
}
