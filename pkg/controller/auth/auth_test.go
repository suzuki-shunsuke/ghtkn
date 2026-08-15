package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/auth"
)

// errDeviceFlow stands in for whatever the SDK's Auth failed with, so the test can
// check that Run wraps it rather than replacing it.
var errDeviceFlow = errors.New("device flow failed")

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
			client:  &mockClient{err: errDeviceFlow},
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
					return
				}
				// The cause must survive the wrapping: the SDK's error is what tells the
				// user why authenticating failed, and errors.Is is how a caller detects
				// a specific one.
				if !errors.Is(err, tt.client.err) {
					t.Errorf("Run() error = %v, want it to wrap %v", err, tt.client.err)
				}
				if !strings.Contains(err.Error(), "create an access token") {
					t.Errorf("Run() error = %v, want it to say what failed", err)
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
