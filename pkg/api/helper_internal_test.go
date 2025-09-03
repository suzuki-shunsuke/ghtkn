//nolint:funlen,revive
package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

type testAppTokenClient struct {
	token *apptoken.AccessToken
	err   error
}

func (m *testAppTokenClient) Create(_ context.Context, logger *slog.Logger, clientID string) (*apptoken.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

type testKeyring struct {
	tokens map[string]*keyring.AccessToken
	getErr error
	setErr error
}

func (m *testKeyring) Get(key string) (*keyring.AccessToken, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.tokens[key], nil
}

func (m *testKeyring) Set(key string, token *keyring.AccessToken) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.tokens == nil {
		m.tokens = make(map[string]*keyring.AccessToken)
	}
	m.tokens[key] = token
	return nil
}

func TestController_checkExpired(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name          string
		exDate        string
		minExpiration time.Duration
		now           time.Time
		want          bool
		wantErr       bool
	}{
		{
			name:          "not expired - future date",
			exDate:        keyring.FormatDate(fixedTime.Add(2 * time.Hour)),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          false,
			wantErr:       false,
		},
		{
			name:          "expired - within min expiration",
			exDate:        keyring.FormatDate(fixedTime.Add(30 * time.Minute)),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          true,
			wantErr:       false,
		},
		{
			name:          "expired - past date",
			exDate:        keyring.FormatDate(fixedTime.Add(-time.Hour)),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          true,
			wantErr:       false,
		},
		{
			name:          "exactly at threshold",
			exDate:        keyring.FormatDate(fixedTime.Add(time.Hour)),
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          false,
			wantErr:       false,
		},
		{
			name:          "invalid date format",
			exDate:        "invalid-date",
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &Input{
				MinExpiration: tt.minExpiration,
				Now:           func() time.Time { return tt.now },
			}
			controller := &TokenManager{input: input}

			got, err := controller.checkExpired(tt.exDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkExpired() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_createToken(t *testing.T) {
	t.Parallel()

	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		clientID string
		client   AppTokenClient
		want     *keyring.AccessToken
		wantErr  bool
	}{
		{
			name:     "successful token creation",
			clientID: "test-client-id",
			client: &testAppTokenClient{
				token: &apptoken.AccessToken{
					AccessToken:    "new-token",
					ExpirationDate: keyring.FormatDate(futureTime),
				},
			},
			want: &keyring.AccessToken{
				AccessToken:    "new-token",
				ExpirationDate: keyring.FormatDate(futureTime),
			},
			wantErr: false,
		},
		{
			name:     "token creation error",
			clientID: "test-client-id",
			client: &testAppTokenClient{
				err: errors.New("creation failed"),
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &Input{
				AppTokenClient: tt.client,
			}
			controller := &TokenManager{input: input}

			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			got, err := controller.createToken(ctx, logger, tt.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("createToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.AccessToken != tt.want.AccessToken || got.ExpirationDate != tt.want.ExpirationDate {
					t.Errorf("createToken() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestController_getAccessTokenFromKeyring(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	futureTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	expiredTime := fixedTime.Add(30 * time.Minute)

	tests := []struct {
		name          string
		clientID      string
		keyring       Keyring
		minExpiration time.Duration
		now           time.Time
		want          *keyring.AccessToken
		wantErr       bool
	}{
		{
			name:     "valid token from keyring",
			clientID: "test-client-id",
			keyring: &testKeyring{
				tokens: map[string]*keyring.AccessToken{
					"test-client-id": {
						AccessToken:    "cached-token",
						ExpirationDate: keyring.FormatDate(futureTime),
					},
				},
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want: &keyring.AccessToken{
				AccessToken:    "cached-token",
				ExpirationDate: keyring.FormatDate(futureTime),
			},
			wantErr: false,
		},
		{
			name:     "expired token in keyring",
			clientID: "test-client-id",
			keyring: &testKeyring{
				tokens: map[string]*keyring.AccessToken{
					"test-client-id": {
						AccessToken:    "expired-token",
						ExpirationDate: keyring.FormatDate(expiredTime),
					},
				},
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          nil,
			wantErr:       false,
		},
		{
			name:     "token not found in keyring",
			clientID: "test-client-id",
			keyring: &testKeyring{
				tokens: map[string]*keyring.AccessToken{},
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          nil,
			wantErr:       false,
		},
		{
			name:     "keyring error",
			clientID: "test-client-id",
			keyring: &testKeyring{
				getErr: errors.New("keyring error"),
			},
			minExpiration: time.Hour,
			now:           fixedTime,
			want:          nil,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &Input{
				Keyring:       tt.keyring,
				MinExpiration: tt.minExpiration,
				Now:           func() time.Time { return tt.now },
				Logger:        NewLogger(),
			}
			controller := &TokenManager{input: input}

			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			got, err := controller.getAccessTokenFromKeyring(logger, tt.clientID)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAccessTokenFromKeyring() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != nil {
				if got.AccessToken != tt.want.AccessToken || got.ExpirationDate != tt.want.ExpirationDate {
					t.Errorf("getAccessTokenFromKeyring() = %v, want %v", got, tt.want)
				}
			}
			if !tt.wantErr && got == nil && tt.want != nil {
				t.Error("getAccessTokenFromKeyring() returned nil, expected token")
			}
		})
	}
}
