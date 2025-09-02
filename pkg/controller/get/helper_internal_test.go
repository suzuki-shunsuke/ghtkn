//nolint:cyclop,funlen,revive
package get

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

type testConfigReader struct {
	cfg *config.Config
	err error
}

func (m *testConfigReader) Read(cfg *config.Config, _ string) error {
	if m.err != nil {
		return m.err
	}
	if m.cfg != nil {
		*cfg = *m.cfg
	}
	return nil
}

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

func TestController_readConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		configReader   ConfigReader
		configFilePath string
		wantErr        bool
	}{
		{
			name: "successful config read",
			configReader: &testConfigReader{
				cfg: &config.Config{
					Apps: []*config.App{
						{
							Name:     "app1",
							ClientID: "client1",
						},
					},
				},
			},
			configFilePath: "test.yaml",
			wantErr:        false,
		},
		{
			name: "config read error",
			configReader: &testConfigReader{
				err: errors.New("read error"),
			},
			configFilePath: "test.yaml",
			wantErr:        true,
		},
		{
			name: "invalid config - no apps",
			configReader: &testConfigReader{
				cfg: &config.Config{
					Apps: []*config.App{},
				},
			},
			configFilePath: "test.yaml",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &Input{
				ConfigReader:   tt.configReader,
				ConfigFilePath: tt.configFilePath,
			}
			controller := &Controller{input: input}

			cfg := &config.Config{}
			err := controller.readConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("readConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
