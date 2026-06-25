// Package revoke provides functionality to revoke GitHub App User Access Tokens.
// It revokes tokens stored in the backend for the given apps and any raw tokens
// passed directly, and removes the revoked stored tokens from the backend.
// All the orchestration lives in the ghtkn SDK; this package only wires the SDK
// client into the CLI.
package revoke

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
)

// Client is the subset of the ghtkn SDK client used to revoke tokens.
type Client interface {
	Revoke(ctx context.Context, logger *slog.Logger, input *ghtkn.InputRevoke) error
}

// Input contains the dependencies needed by the Controller.
type Input struct {
	Client Client
}

// NewInput creates a new Input with default production dependencies.
func NewInput() (*Input, error) {
	client, err := ghtkn.New()
	if err != nil {
		return nil, fmt.Errorf("create a ghtkn client: %w", err)
	}
	return &Input{
		Client: client,
	}, nil
}

// Controller revokes GitHub App User Access Tokens.
type Controller struct {
	input *Input
}

// New creates a new Controller with the provided input.
func New(input *Input) *Controller {
	return &Controller{
		input: input,
	}
}

// Run revokes the requested tokens via the ghtkn SDK.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger, input *ghtkn.InputRevoke) error {
	if err := c.input.Client.Revoke(ctx, logger, input); err != nil {
		return fmt.Errorf("revoke access tokens: %w", err)
	}
	return nil
}
