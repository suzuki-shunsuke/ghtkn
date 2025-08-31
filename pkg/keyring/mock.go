package keyring

import (
	"github.com/zalando/go-keyring"
)

// mockAPI is a mock implementation of the API interface for testing.
// It stores secrets in memory instead of using the system keyring.
type mockAPI struct {
	secrets map[string]string
}

// NewMockAPI creates a new mock API instance with the provided initial secrets.
// If secrets is nil, an empty map will be created when needed.
func NewMockAPI(secrets map[string]string) API {
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
		return "", keyring.ErrNotFound
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
