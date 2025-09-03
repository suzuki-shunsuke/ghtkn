//nolint:revive
package api_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn/pkg/api"
	"github.com/suzuki-shunsuke/ghtkn/pkg/apptoken"
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

	if input.Keyring == nil {
		t.Error("NewInput().Keyring is nil")
	}

	if input.Now == nil {
		t.Error("NewInput().Now is nil")
	}
}

func TestInput_Validate(t *testing.T) {
	t.Parallel()

	// Currently, Input.Validate() always returns nil
	// since there are no validation rules for the Input struct
	input := &api.Input{}

	err := input.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}
