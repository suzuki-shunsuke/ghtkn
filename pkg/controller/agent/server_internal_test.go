package agent

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
)

type handleTestCase struct {
	name     string
	requests []string
	want     []*agentapi.Response
}

var handleTestCases = []handleTestCase{ //nolint:gochecknoglobals // test fixture
	{
		name:     "status empty",
		requests: []string{`{"command":"STATUS"}`},
		want:     []*agentapi.Response{{OK: true}},
	},
	{
		name: "set then get",
		requests: []string{
			`{"command":"SET","client_id":"X","token":{"access_token":"abc"}}`,
			`{"command":"GET","client_id":"X"}`,
		},
		want: []*agentapi.Response{
			{OK: true},
			{OK: true, Token: []byte(`{"access_token":"abc"}`)},
		},
	},
	{
		name:     "get missing",
		requests: []string{`{"command":"GET","client_id":"missing"}`},
		want:     []*agentapi.Response{{Error: agentapi.RespNotFound}},
	},
	{
		name:     "get invalid client id",
		requests: []string{`{"command":"GET","client_id":"../escape"}`},
		want:     []*agentapi.Response{{Error: errMsgInvalidClientID}},
	},
	{
		name:     "set invalid client id",
		requests: []string{`{"command":"SET","client_id":"a/b","token":{}}`},
		want:     []*agentapi.Response{{Error: errMsgInvalidClientID}},
	},
	{
		name:     "unknown command",
		requests: []string{`{"command":"NOPE"}`},
		want:     []*agentapi.Response{{Error: errMsgUnknownCommand}},
	},
	{
		name:     "invalid json",
		requests: []string{`{not json`},
		want:     []*agentapi.Response{{Error: errMsgInvalidRequest}},
	},
	{
		name:     "empty request",
		requests: []string{""},
		want:     []*agentapi.Response{{Error: errMsgEmptyRequest}},
	},
	{
		name: "set then status counts",
		requests: []string{
			`{"command":"SET","client_id":"X","token":{"access_token":"abc"}}`,
			`{"command":"STATUS"}`,
		},
		want: []*agentapi.Response{
			{OK: true},
			{OK: true, Count: 1},
		},
	},
}

func TestController_handle(t *testing.T) {
	t.Parallel()
	for _, d := range handleTestCases {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			c := New()
			for i, req := range d.requests {
				got, _ := c.handle(strings.NewReader(req + "\n"))
				if diff := cmp.Diff(d.want[i], got); diff != "" {
					t.Fatalf("request %d (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}

// TestController_handle_disk drives GET/SET against a disk-backed encrypted store
// and verifies the token round-trips through encryption and disk persistence.
func TestController_handle_disk(t *testing.T) {
	t.Parallel()
	c := New()
	c.store = newDiskStore(testDataKey(t), t.TempDir())

	set, _ := c.handle(strings.NewReader(`{"command":"SET","client_id":"Iv1.abc","token":{"access_token":"abc"}}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, set); diff != "" {
		t.Fatalf("SET (-want +got):\n%s", diff)
	}
	get, _ := c.handle(strings.NewReader(`{"command":"GET","client_id":"Iv1.abc"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true, Token: []byte(`{"access_token":"abc"}`)}, get); diff != "" {
		t.Fatalf("GET (-want +got):\n%s", diff)
	}
}

func TestController_handle_stop(t *testing.T) {
	t.Parallel()
	c := New()
	got, shutdown := c.handle(strings.NewReader(`{"command":"STOP"}` + "\n"))
	if diff := cmp.Diff(&agentapi.Response{OK: true}, got); diff != "" {
		t.Fatalf("response (-want +got):\n%s", diff)
	}
	if !shutdown {
		t.Fatal("STOP must request shutdown")
	}

	// Non-stop commands must not request shutdown.
	if _, shutdown := c.handle(strings.NewReader(`{"command":"STATUS"}` + "\n")); shutdown {
		t.Fatal("STATUS must not request shutdown")
	}
}
