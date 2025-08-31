package keyring

import (
	"github.com/zalando/go-keyring"
)

type mockAPI struct {
	secrets map[string]string
}

func NewMockAPI(secrets map[string]string) API {
	return &mockAPI{
		secrets: secrets,
	}
}

func mockKey(service, user string) string {
	return service + ":" + user
}

func (m *mockAPI) Get(service, user string) (string, error) {
	k := mockKey(service, user)
	s, ok := m.secrets[k]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return s, nil
}

func (m *mockAPI) Set(service, user, password string) error {
	if m.secrets == nil {
		m.secrets = make(map[string]string)
	}
	m.secrets[mockKey(service, user)] = password
	return nil
}

func (m *mockAPI) Delete(service, user string) error {
	k := mockKey(service, user)
	delete(m.secrets, k)
	return nil
}
