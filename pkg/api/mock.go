package api

import (
	"context"

	"github.com/suzuki-shunsuke/ghtkn/pkg/github"
)

func NewMockGitHub(user *github.User, err error) func(ctx context.Context, user string) GitHub {
	return func(ctx context.Context, user string) GitHub {
		return github.NewMock(&github.User{
			Login: user,
		}, nil)(ctx, user)
	}
}
