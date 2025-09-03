//nolint:forcetypeassert,funlen
package api_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

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
		clientID     string
	}{
		{
			name: "successful token creation without persistence",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "test-token-123",
						ExpirationDate: keyring.FormatDate(futureTime),
					},
				}
				input.Keyring = &mockKeyring{}
				return input
			},
			clientID:   "test-client-id",
			wantErr:    false,
			wantOutput: "test-token-123\n",
		},
		{
			name: "successful token retrieval from keyring",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: keyring.FormatDate(futureTime),
					},
				}
				input.Keyring = &mockKeyring{
					tokens: map[string]*keyring.AccessToken{
						"test-client-id": {
							App:            "test-app",
							AccessToken:    "cached-token",
							ExpirationDate: keyring.FormatDate(futureTime),
							Login:          "cached-user",
						},
					},
				}
				return input
			},
			clientID:   "test-client-id",
			wantErr:    false,
			wantOutput: "cached-token\n",
		},
		{
			name: "expired token in keyring triggers new token creation",
			setupInput: func() *api.Input {
				expiredTime := fixedTime.Add(30 * time.Minute)
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: keyring.FormatDate(futureTime),
					},
				}
				input.Keyring = &mockKeyring{
					tokens: map[string]*keyring.AccessToken{
						"test-client-id": {
							App:            "test-app",
							AccessToken:    "expired-token",
							ExpirationDate: keyring.FormatDate(expiredTime),
						},
					},
				}
				return input
			},
			clientID:     "test-client-id",
			wantErr:      false,
			wantOutput:   "new-token\n",
			checkKeyring: true,
		},
		{
			name: "config read error",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					err: errors.New("app token client error"),
				}
				input.Keyring = &mockKeyring{}
				return input
			},
			clientID: "test-client-id",
			wantErr:  true,
		},
		{
			name: "invalid config",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					err: errors.New("token creation failed"),
				}
				input.Keyring = &mockKeyring{}
				return input
			},
			clientID: "test-client-id",
			wantErr:  true,
		},
		{
			name: "token creation error",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					err: errors.New("token creation failed"),
				}
				input.Keyring = &mockKeyring{}
				return input
			},
			clientID: "test-client-id",
			wantErr:  true,
		},
		{
			name: "GitHub API GetUser error",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "test-token-123",
						ExpirationDate: keyring.FormatDate(futureTime),
					},
				}
				input.Keyring = &mockKeyring{}
				input.NewGitHub = api.NewMockGitHub(nil, errors.New("GitHub API error"))
				return input
			},
			clientID: "test-client-id",
			wantErr:  true,
		},
		{
			name: "cached token without login and GitHub API error",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "new-token",
						ExpirationDate: keyring.FormatDate(futureTime),
					},
				}
				input.Keyring = &mockKeyring{
					tokens: map[string]*keyring.AccessToken{
						"test-client-id": {
							App:            "test-app",
							AccessToken:    "cached-token",
							ExpirationDate: keyring.FormatDate(futureTime),
							// Login is empty, will trigger GetUser call
						},
					},
				}
				input.NewGitHub = api.NewMockGitHub(nil, errors.New("GitHub API rate limit exceeded"))
				return input
			},
			clientID: "test-client-id",
			wantErr:  true,
		},
		{
			name: "JSON output format",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					token: &apptoken.AccessToken{
						AccessToken:    "test-token-json",
						ExpirationDate: keyring.FormatDate(futureTime),
					},
				}
				input.Keyring = &mockKeyring{}
				return input
			},
			clientID: "test-client-id",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			controller := api.New(input)
			ctx := t.Context()
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			_, err := controller.Get(ctx, logger, tt.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
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
