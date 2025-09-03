//nolint:funlen
package api_test

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

const serviceKey = "github.com/suzuki-shunsuke/ghtkn"

func TestTokenManager_Get(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setupInput func() *api.Input
		wantErr    bool
		wantToken  *keyring.AccessToken
		input      *api.InputGet
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
				input.Keyring = keyring.New(keyring.NewMockInput(serviceKey, nil))
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: false,
			},
			wantErr: false,
			wantToken: &keyring.AccessToken{
				AccessToken:    "test-token-123",
				ExpirationDate: keyring.FormatDate(futureTime),
				Login:          "test-user",
			},
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
				input.Keyring = keyring.New(keyring.NewMockInput(serviceKey, map[string]*keyring.AccessToken{
					"test-client-id": {
						App:            "test-app",
						AccessToken:    "cached-token",
						ExpirationDate: keyring.FormatDate(futureTime),
						Login:          "cached-user",
					},
				}))
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: false,
			wantToken: &keyring.AccessToken{
				AccessToken:    "new-token",
				ExpirationDate: keyring.FormatDate(futureTime),
				Login:          "test-user",
			},
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
				input.Keyring = keyring.New(keyring.NewMockInput(serviceKey, map[string]*keyring.AccessToken{
					"test-client-id": {
						App:            "test-app",
						AccessToken:    "expired-token",
						ExpirationDate: keyring.FormatDate(expiredTime),
					},
				}))
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: false,
			wantToken: &keyring.AccessToken{
				AccessToken:    "new-token",
				ExpirationDate: keyring.FormatDate(futureTime),
				Login:          "test-user",
			},
		},
		{
			name: "token creation error",
			setupInput: func() *api.Input {
				input := api.NewMockInput()
				input.MinExpiration = time.Hour
				input.AppTokenClient = &mockAppTokenClient{
					err: errors.New("token creation failed"),
				}
				input.Keyring = keyring.New(keyring.NewMockInput(serviceKey, nil))
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: true,
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
				input.Keyring = keyring.New(keyring.NewMockInput(serviceKey, nil))
				input.NewGitHub = api.NewMockGitHub(nil, errors.New("GitHub API error"))
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: true,
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
				input.Keyring = keyring.New(keyring.NewMockInput(serviceKey, map[string]*keyring.AccessToken{
					"test-client-id": {
						App:            "test-app",
						AccessToken:    "cached-token",
						ExpirationDate: keyring.FormatDate(futureTime),
						// Login is empty, will trigger GetUser call
					},
				}))
				input.NewGitHub = api.NewMockGitHub(nil, errors.New("GitHub API rate limit exceeded"))
				return input
			},
			input: &api.InputGet{
				ClientID:   "test-client-id",
				UseKeyring: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			tm := api.New(input)
			ctx := t.Context()
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			token, err := tm.Get(ctx, logger, tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Error(err)
				}
				return
			}
			if tt.wantErr {
				t.Error("expected error but got nil")
				return
			}
			if diff := cmp.Diff(tt.wantToken, token); diff != "" {
				t.Error(diff)
			}
		})
	}
}
