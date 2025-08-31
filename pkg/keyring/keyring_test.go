package keyring_test

import (
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
	zkeyring "github.com/zalando/go-keyring"
)

// mockAPI is a mock implementation of the API interface for testing.
// It stores secrets in memory instead of using the system keyring.
type mockAPI struct {
	secrets map[string]string
}

// NewMockAPI creates a new mock API instance with the provided initial secrets.
// If secrets is nil, an empty map will be created when needed.
func newMockAPI(secrets map[string]string) keyring.API {
	return &mockAPI{
		secrets: secrets,
	}
}

// mockKey generates a unique key for storing secrets by combining service and user.
// The format is "service:user".
func mockKey(service, user string) string {
	return service + ":" + user
}

// Get retrieves a secret from the mock keyring.
// Returns keyring.ErrNotFound if the secret doesn't exist.
func (m *mockAPI) Get(service, user string) (string, error) {
	k := mockKey(service, user)
	s, ok := m.secrets[k]
	if !ok {
		return "", zkeyring.ErrNotFound
	}
	return s, nil
}

// Set stores a secret in the mock keyring.
// Creates the internal map if it doesn't exist.
func (m *mockAPI) Set(service, user, password string) error {
	if m.secrets == nil {
		m.secrets = make(map[string]string)
	}
	m.secrets[mockKey(service, user)] = password
	return nil
}

// Delete removes a secret from the mock keyring.
// No error is returned if the secret doesn't exist.
func (m *mockAPI) Delete(service, user string) error {
	k := mockKey(service, user)
	delete(m.secrets, k)
	return nil
}

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
