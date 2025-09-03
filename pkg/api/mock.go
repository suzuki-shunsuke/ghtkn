package api

import (
	"context"
	"log/slog"

	"github.com/suzuki-shunsuke/ghtkn/pkg/github"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

func NewMockGitHub(user *github.User, err error) func(ctx context.Context, token string) GitHub {
	return func(ctx context.Context, token string) GitHub {
		return github.NewMock(user, err)(ctx, token)
	}
}

type MockTokenManager struct {
	token *keyring.AccessToken
	err   error
}

func NewMockTokenManager(token *keyring.AccessToken, err error) *MockTokenManager {
	return &MockTokenManager{
		token: token,
		err:   err,
	}
}

func (m *MockTokenManager) Get(_ context.Context, _ *slog.Logger, _ *InputGet) (*keyring.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}
