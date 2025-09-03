//nolint:cyclop,funlen
package keyring

import (
	zkeyring "github.com/zalando/go-keyring"
)

// Mock is a mock implementation of the API interface for testing.
// It stores secrets in memory instead of using the system keyring.
type Mock struct {
	secrets map[string]string
}

// NewMock creates a new mock API instance with the provided initial secrets.
// If secrets is nil, an empty map will be created when needed.
func NewMock(secrets map[string]string) API {
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
		return "", zkeyring.ErrNotFound
	}
	return s, nil
}

// Set stores a secret in the mock keyring.
// Creates the internal map if it doesn't exist.
func (m *Mock) Set(service, user, password string) error {
	if m.secrets == nil {
		m.secrets = make(map[string]string)
	}
	m.secrets[mockKey(service, user)] = password
	return nil
}

// Delete removes a secret from the mock keyring.
// No error is returned if the secret doesn't exist.
func (m *Mock) Delete(service, user string) error {
	k := mockKey(service, user)
	delete(m.secrets, k)
	return nil
}
