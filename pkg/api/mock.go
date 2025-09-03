package api

import (
	"context"

	"github.com/suzuki-shunsuke/ghtkn/pkg/github"
)

func NewMockGitHub(user *github.User, err error) func(ctx context.Context, token string) GitHub {
	return func(ctx context.Context, token string) GitHub {
		return github.NewMock(user, err)(ctx, token)
	}
}
