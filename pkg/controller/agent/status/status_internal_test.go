package status

import (
	"path/filepath"
	"testing"
)

func TestQueryStatus_notRunning(t *testing.T) {
	t.Parallel()
	resp, running, err := queryStatus(t.Context(), filepath.Join(t.TempDir(), "absent.sock"))
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("queryStatus must report not running when the socket is absent")
	}
	if resp != nil {
		t.Fatalf("resp = %+v, want nil", resp)
	}
}
