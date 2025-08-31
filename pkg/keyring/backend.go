package keyring

import (
	"github.com/zalando/go-keyring"
)

type Backend struct{}

func NewAPI() *Backend {
	return &Backend{}
}

func (b *Backend) Get(service, user string) (string, error) {
	return keyring.Get(service, user) //nolint:wrapcheck
}

func (b *Backend) Set(service, user, password string) error {
	return keyring.Set(service, user, password) //nolint:wrapcheck
}

func (b *Backend) Delete(service, user string) error {
	return keyring.Delete(service, user) //nolint:wrapcheck
}
