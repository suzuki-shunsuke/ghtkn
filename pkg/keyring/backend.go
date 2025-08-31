package keyring

import (
	"github.com/zalando/go-keyring"
)

// Backend is the real implementation of the API interface.
// It wraps the zalando/go-keyring library to interact with the system keyring.
type Backend struct{}

// NewAPI creates a new Backend instance for accessing the system keyring.
// This is the production implementation that uses the actual OS keyring service.
func NewAPI() *Backend {
	return &Backend{}
}

// Get retrieves a password from the system keyring.
// It delegates to the zalando/go-keyring library's Get function.
func (b *Backend) Get(service, user string) (string, error) {
	return keyring.Get(service, user) //nolint:wrapcheck
}

// Set stores a password in the system keyring.
// It delegates to the zalando/go-keyring library's Set function.
func (b *Backend) Set(service, user, password string) error {
	return keyring.Set(service, user, password) //nolint:wrapcheck
}

// Delete removes a password from the system keyring.
// It delegates to the zalando/go-keyring library's Delete function.
func (b *Backend) Delete(service, user string) error {
	return keyring.Delete(service, user) //nolint:wrapcheck
}
