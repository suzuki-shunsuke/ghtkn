package revoke_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/revoke"
)

type mockClient struct {
	called   bool
	gotInput *ghtkn.InputRevoke
	err      error
}

func (m *mockClient) Revoke(_ context.Context, _ *slog.Logger, input *ghtkn.InputRevoke) error {
	m.called = true
	m.gotInput = input
	return m.err
}

type mockRevoker struct {
	called    bool
	gotTokens []string
	err       error
}

func (m *mockRevoker) Revoke(_ context.Context, tokens []string) error {
	m.called = true
	m.gotTokens = tokens
	return m.err
}

func checkRevoker(t *testing.T, revoker *mockRevoker, wantCalled bool, wantTokens []string) {
	t.Helper()
	if revoker.called != wantCalled {
		t.Errorf("revoker called = %v, want %v", revoker.called, wantCalled)
	}
	if wantCalled {
		if diff := cmp.Diff(wantTokens, revoker.gotTokens); diff != "" {
			t.Errorf("revoked tokens mismatch (-want +got):\n%s", diff)
		}
	}
}

func checkClient(t *testing.T, client *mockClient, wantCalled bool, wantAppNames []string, wantAll bool) {
	t.Helper()
	if client.called != wantCalled {
		t.Errorf("client called = %v, want %v", client.called, wantCalled)
	}
	if wantCalled {
		if diff := cmp.Diff(wantAppNames, client.gotInput.AppNames); diff != "" {
			t.Errorf("app names mismatch (-want +got):\n%s", diff)
		}
		if client.gotInput.All != wantAll {
			t.Errorf("All = %v, want %v", client.gotInput.All, wantAll)
		}
	}
}

type runTestCase struct {
	name         string
	input        *revoke.InputRevoke
	clientErr    error
	revokerErr   error
	wantClient   bool
	wantRevoker  bool
	wantTokens   []string
	wantAppNames []string
	wantAll      bool
	wantErr      bool
}

func runTestCases() []runTestCase {
	return []runTestCase{
		{
			name:        "only raw tokens: revoker is called, the SDK is not",
			input:       &revoke.InputRevoke{Tokens: []string{"ghu_x"}},
			wantRevoker: true,
			wantTokens:  []string{"ghu_x"},
			wantClient:  false,
		},
		{
			name:         "app names: the SDK is called, the revoker is not",
			input:        &revoke.InputRevoke{AppNames: []string{"test", "test2"}},
			wantClient:   true,
			wantAppNames: []string{"test", "test2"},
			wantRevoker:  false,
		},
		{
			name:        "neither: the SDK is called for the fallback app",
			input:       &revoke.InputRevoke{},
			wantClient:  true,
			wantRevoker: false,
		},
		{
			name:         "both: the revoker and the SDK are called",
			input:        &revoke.InputRevoke{Tokens: []string{"ghu_x"}, AppNames: []string{"test"}},
			wantRevoker:  true,
			wantTokens:   []string{"ghu_x"},
			wantClient:   true,
			wantAppNames: []string{"test"},
		},
		{
			name:        "--all: the SDK is called with All set, the revoker is not",
			input:       &revoke.InputRevoke{All: true},
			wantClient:  true,
			wantAll:     true,
			wantRevoker: false,
		},
		{
			name:        "--all with raw tokens: both are called, the SDK with All set",
			input:       &revoke.InputRevoke{All: true, Tokens: []string{"ghu_x"}},
			wantRevoker: true,
			wantTokens:  []string{"ghu_x"},
			wantClient:  true,
			wantAll:     true,
		},
		{
			name:       "revoker error is propagated",
			input:      &revoke.InputRevoke{Tokens: []string{"ghu_x"}},
			revokerErr: errors.New("boom"),
			wantErr:    true,
		},
		{
			name:      "client error is propagated",
			input:     &revoke.InputRevoke{AppNames: []string{"test"}},
			clientErr: errors.New("boom"),
			wantErr:   true,
		},
	}
}

func TestController_Run(t *testing.T) {
	t.Parallel()

	for _, tt := range runTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := &mockClient{err: tt.clientErr}
			revoker := &mockRevoker{err: tt.revokerErr}
			c := revoke.New(&revoke.Input{Client: client, Revoker: revoker})

			err := c.Run(t.Context(), slog.New(slog.DiscardHandler), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Run() expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			checkRevoker(t, revoker, tt.wantRevoker, tt.wantTokens)
			checkClient(t, client, tt.wantClient, tt.wantAppNames, tt.wantAll)
		})
	}
}
