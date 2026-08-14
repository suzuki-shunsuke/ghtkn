package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/deviceflow"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/auth"
)

type mockClient struct {
	err error
	// input records what Run passed through, so the test can assert the controller
	// doesn't drop or rewrite the request.
	input *ghtkn.InputAuth
}

func (m *mockClient) Auth(_ context.Context, _ *slog.Logger, input *ghtkn.InputAuth) error {
	m.input = input
	return m.err
}

func (m *mockClient) SetCopyOnetimeCodeToClipboard(_ deviceflow.CopyTextToClipboard) {}

func TestController_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  *mockClient
		wantErr bool
	}{
		{
			name:   "successful authentication",
			client: &mockClient{},
		},
		{
			name:    "authentication error",
			client:  &mockClient{err: errors.New("device flow failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.DiscardHandler)
			input := &ghtkn.InputAuth{
				AppName:        "test-app",
				ConfigFilePath: "/path/to/config.yaml",
			}
			err := auth.New(&auth.Input{Client: tt.client}).Run(t.Context(), logger, input)
			if err != nil {
				if !tt.wantErr {
					t.Error(err)
				}
				return
			}
			if tt.wantErr {
				t.Error("expected error but got nil")
				return
			}
			if tt.client.input != input {
				t.Error("Run passed a different InputAuth to the client")
			}
		})
	}
}
