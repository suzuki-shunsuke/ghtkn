//nolint:cyclop,funlen
package keyring_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
	zkeyring "github.com/zalando/go-keyring"
)

// TestParseDate tests the ParseDate function with various inputs.
func TestParseDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "valid RFC3339 date",
			input: "2024-01-15T10:30:00Z",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "valid RFC3339 date with timezone",
			input: "2024-06-20T15:45:30+09:00",
			want:  time.Date(2024, 6, 20, 15, 45, 30, 0, time.FixedZone("", 9*60*60)),
		},
		{
			name:  "valid RFC3339 date with negative timezone",
			input: "2024-12-31T23:59:59-05:00",
			want:  time.Date(2024, 12, 31, 23, 59, 59, 0, time.FixedZone("", -5*60*60)),
		},
		{
			name:  "valid RFC3339 date with nanoseconds",
			input: "2024-03-10T08:15:30.123456789Z",
			want:  time.Date(2024, 3, 10, 8, 15, 30, 123456789, time.UTC),
		},
		{
			name:    "invalid format - not RFC3339",
			input:   "2024-01-15 10:30:00",
			wantErr: true,
		},
		{
			name:    "invalid format - missing time",
			input:   "2024-01-15",
			wantErr: true,
		},
		{
			name:    "invalid format - missing timezone",
			input:   "2024-01-15T10:30:00",
			wantErr: true,
		},
		{
			name:    "invalid date string",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "2024-13-01T10:30:00Z",
			wantErr: true,
		},
		{
			name:    "invalid day",
			input:   "2024-02-30T10:30:00Z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := keyring.ParseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatDate tests the FormatDate function.
func TestFormatDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Time
		want  string
	}{
		{
			name:  "UTC time",
			input: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			want:  "2024-01-15T10:30:00Z",
		},
		{
			name:  "time with positive timezone",
			input: time.Date(2024, 6, 20, 15, 45, 30, 0, time.FixedZone("", 9*60*60)),
			want:  "2024-06-20T15:45:30+09:00",
		},
		{
			name:  "time with negative timezone",
			input: time.Date(2024, 12, 31, 23, 59, 59, 0, time.FixedZone("", -5*60*60)),
			want:  "2024-12-31T23:59:59-05:00",
		},
		{
			name:  "time with nanoseconds (truncated by RFC3339)",
			input: time.Date(2024, 3, 10, 8, 15, 30, 123456789, time.UTC),
			want:  "2024-03-10T08:15:30Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := keyring.FormatDate(tt.input)
			if got != tt.want {
				t.Errorf("FormatDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeyring_Get tests the Get method of Keyring.
func TestKeyring_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		secrets map[string]*keyring.AccessToken
		want    *keyring.AccessToken
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful get",
			key:  "test-key",
			secrets: map[string]*keyring.AccessToken{
				"github.com/suzuki-shunsuke/ghtkn:test-key": {
					App:            "test-app",
					AccessToken:    "token123",
					ExpirationDate: "2024-12-31T23:59:59Z",
				},
			},
			want: &keyring.AccessToken{
				App:            "test-app",
				AccessToken:    "token123",
				ExpirationDate: "2024-12-31T23:59:59Z",
			},
		},
		{
			name:    "key not found",
			key:     "non-existent",
			secrets: map[string]*keyring.AccessToken{},
			wantErr: true,
			errMsg:  "get a GitHub Access token in keyring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &keyring.Input{
				KeyService: "github.com/suzuki-shunsuke/ghtkn",
				API:        keyring.NewMockAPI(tt.secrets),
			}
			kr := keyring.New(input)

			got, err := kr.Get(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Get() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && !errors.Is(err, zkeyring.ErrNotFound) {
					// Check error message contains expected text
					if err.Error() == "" || err.Error() != "" && len(err.Error()) < len(tt.errMsg) {
						t.Errorf("Get() error message too short")
					}
				}
				return
			}
			if err != nil {
				t.Errorf("Get() unexpected error = %v", err)
				return
			}
			if got.App != tt.want.App || got.AccessToken != tt.want.AccessToken || got.ExpirationDate != tt.want.ExpirationDate {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeyring_Set tests the Set method of Keyring.
func TestKeyring_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		token   *keyring.AccessToken
		wantErr bool
	}{
		{
			name: "successful set",
			key:  "test-key",
			token: &keyring.AccessToken{
				App:            "test-app",
				AccessToken:    "token123",
				ExpirationDate: "2024-12-31T23:59:59Z",
			},
		},
		{
			name: "set with empty fields",
			key:  "empty-key",
			token: &keyring.AccessToken{
				App:            "",
				AccessToken:    "",
				ExpirationDate: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := keyring.NewMockAPI(nil)
			input := &keyring.Input{
				KeyService: "github.com/suzuki-shunsuke/ghtkn",
				API:        api,
			}
			kr := keyring.New(input)

			err := kr.Set(tt.key, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the token was stored correctly
				storedStr, err := api.Get("github.com/suzuki-shunsuke/ghtkn", tt.key)
				if err != nil {
					t.Errorf("Failed to verify stored token: %v", err)
					return
				}

				var storedToken keyring.AccessToken
				if err := json.Unmarshal([]byte(storedStr), &storedToken); err != nil {
					t.Errorf("Failed to unmarshal stored token: %v", err)
					return
				}

				if storedToken.App != tt.token.App || storedToken.AccessToken != tt.token.AccessToken || storedToken.ExpirationDate != tt.token.ExpirationDate {
					t.Errorf("Stored token = %v, want %v", storedToken, tt.token)
				}
			}
		})
	}
}

// TestNew tests the New function.
func TestNew(t *testing.T) {
	t.Parallel()

	input := &keyring.Input{
		KeyService: "test-service",
		API:        keyring.NewMockAPI(nil),
	}

	kr := keyring.New(input)
	if kr == nil {
		t.Error("New() returned nil")
	}
}

const serviceKey = "github.com/suzuki-shunsuke/ghtkn"

// TestNewInput tests the NewInput function.
func TestNewInput(t *testing.T) {
	t.Parallel()

	input := keyring.NewInput(serviceKey)
	if input == nil {
		t.Error("NewInput() returned nil")
		return
	}

	if input.KeyService != "github.com/suzuki-shunsuke/ghtkn" {
		t.Errorf("NewInput().KeyService = %v, want %v", input.KeyService, "github.com/suzuki-shunsuke/ghtkn")
	}

	if input.API == nil {
		t.Error("NewInput().API is nil")
	}
}
