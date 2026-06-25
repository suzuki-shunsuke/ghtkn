package revoke_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/revoke"
)

type mockClient struct {
	gotInput *ghtkn.InputRevoke
	err      error
}

func (m *mockClient) Revoke(_ context.Context, _ *slog.Logger, input *ghtkn.InputRevoke) error {
	m.gotInput = input
	return m.err
}

func TestController_Run(t *testing.T) {
	t.Parallel()

	t.Run("passes the input to the client", func(t *testing.T) {
		t.Parallel()
		client := &mockClient{}
		c := revoke.New(&revoke.Input{Client: client})
		input := &ghtkn.InputRevoke{AppNames: []string{"test"}, Tokens: []string{"ghu_x"}}
		if err := c.Run(t.Context(), slog.New(slog.DiscardHandler), input); err != nil {
			t.Fatal(err)
		}
		if client.gotInput != input {
			t.Errorf("Run() passed %+v, want %+v", client.gotInput, input)
		}
	})

	t.Run("propagates client errors", func(t *testing.T) {
		t.Parallel()
		c := revoke.New(&revoke.Input{Client: &mockClient{err: errors.New("boom")}})
		if err := c.Run(t.Context(), slog.New(slog.DiscardHandler), &ghtkn.InputRevoke{}); err == nil {
			t.Error("Run() expected an error, got nil")
		}
	})
}
