// Package github provides a client for interacting with the GitHub API.
// It is used to retrieve authenticated user information for Git Credential Helper support.
package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"
)

// User represents a GitHub user with the minimal information needed for authentication.
// The Login field contains the GitHub username required for Git Credential Helper.
type User struct {
	Login string `json:"login"`
}

// Client wraps the GitHub API client to provide simplified access to user information.
type Client struct {
	client *github.UsersService
}

// New creates a new GitHub API client authenticated with the provided access token.
// The client is configured to use OAuth2 authentication for API requests.
func New(ctx context.Context, token string) *Client {
	return &Client{
		client: github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		))).Users,
	}
}

// Get retrieves the authenticated user's information from GitHub.
// It returns a User struct containing the login name, which is required for
// Git Credential Helper to properly authenticate with GitHub repositories.
func (c *Client) Get(ctx context.Context) (*User, error) {
	user, _, err := c.client.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	return &User{
		Login: user.GetLogin(),
	}, nil
}
