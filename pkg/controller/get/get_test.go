//nolint:forcetypeassert,funlen
package get_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
)

type mockClient struct {
	token *ghtkn.AccessToken
	app   *ghtkn.AppConfig
	err   error
}

func (m *mockClient) Get(_ context.Context, _ *slog.Logger, _ *ghtkn.InputGet) (*ghtkn.AccessToken, *ghtkn.AppConfig, error) {
	return m.token, m.app, m.err
}

func TestController_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupInput   func() *get.Input
		wantErr      bool
		wantOutput   string
		checkKeyring bool
	}{
		{
			name: "successful token creation",
			setupInput: func() *get.Input {
				return &get.Input{
					OutputFormat: "",
					Stdout:       &bytes.Buffer{},
					Client: &mockClient{
						token: &ghtkn.AccessToken{
							AccessToken:    "test-token-123",
							ExpirationDate: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
						},
						app: &ghtkn.AppConfig{
							Name: "test",
						},
					},
				}
			},
			wantErr:    false,
			wantOutput: "test-token-123\n",
		},
		{
			name: "token creation error",
			setupInput: func() *get.Input {
				return &get.Input{
					OutputFormat: "",
					Stdout:       &bytes.Buffer{},
					Client: &mockClient{
						err: errors.New("failed to create token"),
					},
				}
			},
			wantErr: true,
		},
		{
			name: "JSON output format",
			setupInput: func() *get.Input {
				return &get.Input{
					OutputFormat: "json",
					Stdout:       &bytes.Buffer{},
					Client: &mockClient{
						token: &ghtkn.AccessToken{
							AccessToken:    "test-token-123",
							ExpirationDate: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
						},
						app: &ghtkn.AppConfig{
							Name: "test",
						},
					},
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.setupInput()
			controller := get.New(input)
			ctx := t.Context()
			logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

			err := controller.Run(ctx, logger, &ghtkn.InputGet{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && input.OutputFormat != "json" {
				output := input.Stdout.(*bytes.Buffer).String()
				if output != tt.wantOutput {
					t.Errorf("Run() output = %v, want %v", output, tt.wantOutput)
				}
			}
		})
	}
}
