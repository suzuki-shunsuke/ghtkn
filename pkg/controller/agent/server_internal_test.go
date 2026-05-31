package agent

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type handleTestCase struct {
	name     string
	requests []string
	want     []*Response
}

var handleTestCases = []handleTestCase{ //nolint:gochecknoglobals // test fixture
	{
		name:     "status empty",
		requests: []string{`{"command":"STATUS"}`},
		want:     []*Response{{OK: true}},
	},
	{
		name: "set then get",
		requests: []string{
			`{"command":"SET","client_id":"X","token":{"access_token":"abc"}}`,
			`{"command":"GET","client_id":"X"}`,
		},
		want: []*Response{
			{OK: true},
			{OK: true, Token: []byte(`{"access_token":"abc"}`)},
		},
	},
	{
		name:     "get missing",
		requests: []string{`{"command":"GET","client_id":"missing"}`},
		want:     []*Response{{Error: errMsgNotFound}},
	},
	{
		name:     "unknown command",
		requests: []string{`{"command":"NOPE"}`},
		want:     []*Response{{Error: errMsgUnknownCommand}},
	},
	{
		name:     "invalid json",
		requests: []string{`{not json`},
		want:     []*Response{{Error: errMsgInvalidRequest}},
	},
	{
		name:     "empty request",
		requests: []string{""},
		want:     []*Response{{Error: errMsgEmptyRequest}},
	},
	{
		name: "set then status counts",
		requests: []string{
			`{"command":"SET","client_id":"X","token":{"access_token":"abc"}}`,
			`{"command":"STATUS"}`,
		},
		want: []*Response{
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
				got := c.handle(strings.NewReader(req + "\n"))
				if diff := cmp.Diff(d.want[i], got); diff != "" {
					t.Fatalf("request %d (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}
