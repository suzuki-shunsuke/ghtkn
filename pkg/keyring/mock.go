package keyring

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

// Mock is a mock implementation of the API interface for testing.
// It stores secrets in memory instead of using the system keyring.
type Mock struct {
	secrets map[string]*AccessToken
}

// NewMockAPI creates a new mock API instance with the provided initial secrets.
// If secrets is nil, an empty map will be created when needed.
func NewMockAPI(secrets map[string]*AccessToken) API {
	return &Mock{
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
func (m *Mock) Get(service, user string) (string, error) {
	k := mockKey(service, user)
	s, ok := m.secrets[k]
	if !ok {
		return "", keyring.ErrNotFound
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("marshal secret as JSON: %w", err)
	}
	return string(b), nil
}

// Set stores a secret in the mock keyring.
// Creates the internal map if it doesn't exist.
func (m *Mock) Set(service, user, password string) error {
	if m.secrets == nil {
		m.secrets = map[string]*AccessToken{}
	}
	token := &AccessToken{}
	if err := json.Unmarshal([]byte(password), token); err != nil {
		return fmt.Errorf("unmarshal secret as JSON: %w", err)
	}
	m.secrets[mockKey(service, user)] = token
	return nil
}
