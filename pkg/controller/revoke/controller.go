// Package revoke provides functionality to revoke GitHub App User Access Tokens.
// Raw tokens passed with --token are revoked directly via GitHub's credential
// revocation API. The token stored in the backend for a given app is revoked
// through the ghtkn SDK, which also removes it from the backend. Both paths use
// the same revocation library, keeping the behavior consistent.
package revoke

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/go-revoke-github-access-token/revoke"
)

// Client is the subset of the ghtkn SDK client used to revoke stored app tokens.
type Client interface {
	Revoke(ctx context.Context, logger *slog.Logger, input *ghtkn.InputRevoke) error
}

// Revoker revokes raw credentials via GitHub's credential revocation API.
type Revoker interface {
	Revoke(ctx context.Context, tokens []string) error
}

// Input contains the dependencies needed by the Controller.
type Input struct {
	Client  Client
	Revoker Revoker
}

// NewInput creates a new Input with default production dependencies.
func NewInput() (*Input, error) {
	client, err := ghtkn.New()
	if err != nil {
		return nil, fmt.Errorf("create a ghtkn client: %w", err)
	}
	return &Input{
		Client:  client,
		Revoker: revoke.New(nil),
	}, nil
}

// InputRevoke holds the values needed to revoke tokens.
type InputRevoke struct {
	// Tokens are raw access tokens given as positional arguments. They are revoked
	// directly and are not looked up in or deleted from any backend.
	Tokens []string
	// AppNames are the apps whose stored tokens should be revoked (empty if none given).
	AppNames []string
	// ConfigFilePath is the resolved configuration file path.
	ConfigFilePath string
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

// Run revokes the requested tokens.
//
// Raw access tokens are revoked directly via the credential revocation API. The
// SDK is invoked to revoke apps' stored tokens only when app names are given, or
// when nothing at all is given (the fallback to GHTKN_APP / the default app). When
// only raw tokens are given, the SDK is not called, so a raw token is never
// expanded into an unrelated app's stored token.
func (c *Controller) Run(ctx context.Context, logger *slog.Logger, input *InputRevoke) error {
	if len(input.Tokens) > 0 {
		if err := c.input.Revoker.Revoke(ctx, input.Tokens); err != nil {
			return fmt.Errorf("revoke access tokens: %w", err)
		}
	}

	if len(input.AppNames) > 0 || len(input.Tokens) == 0 {
		if err := c.input.Client.Revoke(ctx, logger, &ghtkn.InputRevoke{
			AppNames:       input.AppNames,
			ConfigFilePath: input.ConfigFilePath,
		}); err != nil {
			return fmt.Errorf("revoke access tokens stored in the backend: %w", err)
		}
	}
	return nil
}
