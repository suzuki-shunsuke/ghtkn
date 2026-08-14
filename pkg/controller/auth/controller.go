// Package auth provides functionality to authenticate to GitHub and cache a GitHub App
// User Access Token. It serves the 'auth' command, the only command that runs the OAuth
// device flow: 'get', 'exec', and 'git-credential' read the cached token and fail when
// there is none, so a device flow is never started on the user's behalf.
package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/clipboard"
)

// Controller manages the process of authenticating to GitHub and caching the token.
type Controller struct {
	input *Input
}

// New creates a new Controller instance with the provided input configuration.
func New(input *Input) *Controller {
	return &Controller{
		input: input,
	}
}

type Client interface {
	Auth(ctx context.Context, logger *slog.Logger, input *ghtkn.InputAuth) error
}

// Input contains all the dependencies and configuration needed by the Controller.
type Input struct {
	Client Client
}

// NewInput creates a new Input instance with default production values.
func NewInput() (*Input, error) {
	client, err := ghtkn.New()
	if err != nil {
		return nil, fmt.Errorf("create a ghtkn client: %w", err)
	}
	// The clipboard implementation is always injected; whether the one-time code is
	// actually copied is resolved by the SDK from InputAuth.Clipboard, GHTKN_CLIPBOARD,
	// and the config's clipboard.enable. It is wired here rather than in the CLI so that
	// the Client interface stays the one method the controller calls.
	client.SetCopyOnetimeCodeToClipboard(clipboard.New())
	return &Input{
		Client: client,
	}, nil
}
