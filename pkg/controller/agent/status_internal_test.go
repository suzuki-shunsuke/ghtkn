package agent

import (
	"log/slog"
	"path/filepath"
	"testing"
)

func TestQueryStatus_notRunning(t *testing.T) {
	t.Parallel()
	running, count, err := queryStatus(t.Context(), filepath.Join(t.TempDir(), "absent.sock"))
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("queryStatus must report not running when the socket is absent")
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestQueryStatus_running(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "agent.sock")
	c := New()
	listener, err := listen(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	logger := slog.New(slog.DiscardHandler)
	go c.serve(listener, logger) //nolint:errcheck // serve returns nil once the listener is closed

	running, count, err := queryStatus(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Fatal("queryStatus must report running")
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}

	if _, err := request(t.Context(), path, &Request{Command: CommandSet, ClientID: "X", Token: []byte(`{"access_token":"abc"}`)}); err != nil {
		t.Fatal(err)
	}

	running, count, err = queryStatus(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Fatal("queryStatus must report running after SET")
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
