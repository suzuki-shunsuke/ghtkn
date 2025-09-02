//nolint:forcetypeassert,funlen,maintidx
package api_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/github"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

type mockGitHub struct {
	user *github.User
	err  error
}

func (m *mockGitHub) GetUser(_ context.Context) (*github.User, error) {
	return m.user, m.err
}

func mockNewGitHub(_ context.Context, _ string) api.GitHub {
	return &mockGitHub{
		user: &github.User{
			Login: "test-user",
		},
	}
}

func TestTokenManager_Get(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		setupInput   func() *api.Input
		wantErr      bool
		wantOutput   string
		checkKeyring bool
		app          *config.App
	}{
		{
			name: "successful token creation without persistence",
			setupInput: func() *api.Input {
				return &api.Input{
					OutputFormat:  "",
					MinExpiration: time.Hour,
					FS:            afero.NewMemMapFs(),
					Env:           &config.Env{App: "test-app"},
					AppTokenClient: &mockAppTokenClient{
						token: &apptoken.AccessToken{
							AccessToken:    "test-token-123",
							ExpirationDate: keyring.FormatDate(futureTime),
						},
					},
					Stdout:    &bytes.Buffer{},
					Keyring:   &mockKeyring{},
					Now:       func() time.Time { return fixedTime },
					NewGitHub: mockNewGitHub,
				}
			},
			wantErr:    false,
			wantOutput: "test-token-123\n",
		},
		{
			name: "successful token retrieval from keyring",
			setupInput: func() *api.Input {
				return &api.Input{
					OutputFormat:  "",
					MinExpiration: time.Hour,
					FS:            afero.NewMemMapFs(),
					Env:           &config.Env{App: "test-app"},
					AppTokenClient: &mockAppTokenClient{
						token: &apptoken.AccessToken{
							AccessToken:    "new-token",
							ExpirationDate: keyring.FormatDate(futureTime),
						},
					},
					Stdout: &bytes.Buffer{},
					Keyring: &mockKeyring{
						tokens: map[string]*keyring.AccessToken{
							"test-client-id": {
								App:            "test-app",
								AccessToken:    "cached-token",
								ExpirationDate: keyring.FormatDate(futureTime),
								Login:          "cached-user",
							},
						},
					},
					Now:       func() time.Time { return fixedTime },
					NewGitHub: mockNewGitHub,
				}
			},
			wantErr:    false,
			wantOutput: "cached-token\n",
		},
		{
			name: "expired token in keyring triggers new token creation",
			setupInput: func() *api.Input {
				expiredTime := fixedTime.Add(30 * time.Minute)
				return &api.Input{
					OutputFormat:  "",
					MinExpiration: time.Hour,
					FS:            afero.NewMemMapFs(),
					Env:           &config.Env{App: "test-app"},
					AppTokenClient: &mockAppTokenClient{
						token: &apptoken.AccessToken{
							AccessToken:    "new-token",
							ExpirationDate: keyring.FormatDate(futureTime),
						},
					},
					Stdout: &bytes.Buffer{},
					Keyring: &mockKeyring{
						tokens: map[string]*keyring.AccessToken{
							"test-client-id": {
								App:            "test-app",
								AccessToken:    "expired-token",
								ExpirationDate: keyring.FormatDate(expiredTime),
							},
						},
					},
					Now:       func() time.Time { return fixedTime },
					NewGitHub: mockNewGitHub,
				}
			},
			wantErr:      false,
			wantOutput:   "new-token\n",
			checkKeyring: true,
		},
		{
			name: "config read error",
			setupInput: func() *api.Input {
				return &api.Input{
					FS:     afero.NewMemMapFs(),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "invalid config",
			setupInput: func() *api.Input {
				return &api.Input{
					FS:     afero.NewMemMapFs(),
					Stdout: &bytes.Buffer{},
				}
			},
			wantErr: true,
		},
		{
			name: "token creation error",
			setupInput: func() *api.Input {
				return &api.Input{
					OutputFormat:  "",
					MinExpiration: time.Hour,
					FS:            afero.NewMemMapFs(),
					Env:           &config.Env{App: "test-app"},
					AppTokenClient: &mockAppTokenClient{
						err: errors.New("token creation failed"),
					},
					Stdout:    &bytes.Buffer{},
					Keyring:   &mockKeyring{},
					Now:       func() time.Time { return fixedTime },
					NewGitHub: mockNewGitHub,
				}
			},
			wantErr: true,
		},
		{
			name: "GitHub API GetUser error",
			setupInput: func() *api.Input {
				return &api.Input{
					OutputFormat:  "",
					MinExpiration: time.Hour,
					FS:            afero.NewMemMapFs(),
					Env:           &config.Env{App: "test-app"},
					AppTokenClient: &mockAppTokenClient{
						token: &apptoken.AccessToken{
							AccessToken:    "test-token-123",
							ExpirationDate: keyring.FormatDate(futureTime),
						},
					},
					Stdout:  &bytes.Buffer{},
					Keyring: &mockKeyring{},
					Now:     func() time.Time { return fixedTime },
					NewGitHub: func(_ context.Context, _ string) api.GitHub {
						return &mockGitHub{
							err: errors.New("GitHub API error"),
						}
					},
				}
			},
			wantErr: true,
		},
		{
			name: "cached token without login and GitHub API error",
			setupInput: func() *api.Input {
				return &api.Input{
					OutputFormat:  "",
					MinExpiration: time.Hour,
					FS:            afero.NewMemMapFs(),
					Env:           &config.Env{App: "test-app"},
					AppTokenClient: &mockAppTokenClient{
						token: &apptoken.AccessToken{
							AccessToken:    "new-token",
							ExpirationDate: keyring.FormatDate(futureTime),
						},
					},
					Stdout: &bytes.Buffer{},
					Keyring: &mockKeyring{
						tokens: map[string]*keyring.AccessToken{
							"test-client-id": {
								App:            "test-app",
								AccessToken:    "cached-token",
								ExpirationDate: keyring.FormatDate(futureTime),
								// Login is empty, will trigger GetUser call
							},
						},
					},
					Now: func() time.Time { return fixedTime },
					NewGitHub: func(_ context.Context, _ string) api.GitHub {
						return &mockGitHub{
							err: errors.New("GitHub API rate limit exceeded"),
						}
					},
				}
			},
			wantErr: true,
		},
		{
			name: "JSON output format",
			setupInput: func() *api.Input {
				return &api.Input{
					OutputFormat:  "json",
					MinExpiration: time.Hour,
					FS:            afero.NewMemMapFs(),
					Env:           &config.Env{App: "test-app"},
					AppTokenClient: &mockAppTokenClient{
						token: &apptoken.AccessToken{
							AccessToken:    "test-token-json",
							ExpirationDate: keyring.FormatDate(futureTime),
						},
					},
					Stdout:    &bytes.Buffer{},
					Keyring:   &mockKeyring{},
					Now:       func() time.Time { return fixedTime },
					NewGitHub: mockNewGitHub,
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			controller := api.New(input)
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			_, err := controller.Get(ctx, logger, tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && input.OutputFormat != "json" {
				output := input.Stdout.(*bytes.Buffer).String()
				if output != tt.wantOutput {
					t.Errorf("Run() output = %v, want %v", output, tt.wantOutput)
				}
			}

			if tt.checkKeyring {
				kr := input.Keyring.(*mockKeyring)
				if kr.tokens["test-client-id"] == nil {
					t.Error("Token was not stored in keyring")
				} else if kr.tokens["test-client-id"].AccessToken != "new-token" {
					t.Errorf("Stored token = %v, want new-token", kr.tokens["test-client-id"].AccessToken)
				}
			}
		})
	}
}
