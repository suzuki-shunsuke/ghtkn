//nolint:revive
package api_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
	"github.com/suzuki-shunsuke/ghtkn/pkg/keyring"
)

type mockAppTokenClient struct {
	token *apptoken.AccessToken
	err   error
}

func (m *mockAppTokenClient) Create(_ context.Context, logger *slog.Logger, clientID string) (*apptoken.AccessToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

type mockKeyring struct {
	tokens map[string]*keyring.AccessToken
	getErr error
	setErr error
}

func (m *mockKeyring) Get(key string) (*keyring.AccessToken, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.tokens[key], nil
}

func (m *mockKeyring) Set(key string, token *keyring.AccessToken) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.tokens == nil {
		m.tokens = make(map[string]*keyring.AccessToken)
	}
	m.tokens[key] = token
	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	input := &api.Input{}
	controller := api.New(input)
	if controller == nil {
		t.Error("New() returned nil")
	}
}

func TestNewInput(t *testing.T) {
	t.Parallel()

	input := api.NewInput()
	if input == nil {
		t.Error("NewInput() returned nil")
		return
	}

	if input.AppTokenClient == nil {
		t.Error("NewInput().AppTokenClient is nil")
	}

	if input.Stdout == nil {
		t.Error("NewInput().Stdout is nil")
	}

	if input.Keyring == nil {
		t.Error("NewInput().Keyring is nil")
	}

	if input.Now == nil {
		t.Error("NewInput().Now is nil")
	}
}

func TestInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		outputFormat string
		wantErr      bool
	}{
		{
			name:         "valid json format",
			outputFormat: "json",
			wantErr:      false,
		},
		{
			name:         "valid empty format",
			outputFormat: "",
			wantErr:      false,
		},
		{
			name:         "invalid format",
			outputFormat: "yaml",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := &api.Input{}

			err := input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
