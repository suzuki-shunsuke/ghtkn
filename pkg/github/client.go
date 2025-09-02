package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"
)

type User struct {
	Login string `json:"login"`
}

type Client struct {
	client *github.UsersService
}

func New(ctx context.Context, token string) *Client {
	return &Client{
		client: github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		))).Users,
	}
}

func (c *Client) GetUser(ctx context.Context) (*User, error) {
	user, _, err := c.client.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("get authenticated user: %w", err)
	}
	return &User{
		Login: user.GetLogin(),
	}, nil
}
