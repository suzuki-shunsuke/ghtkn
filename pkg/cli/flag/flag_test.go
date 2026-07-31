package flag_test

import (
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
)

func TestSetMinExpiration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		value   string
		want    *time.Duration
		wantErr bool
	}{
		{
			// Leaving it nil is what makes the SDK fall back to
			// GHTKN_MIN_EXPIRATION and the config.
			name:  "an unset flag changes nothing",
			value: "",
			want:  nil,
		},
		{
			name:  "a duration",
			value: "1h",
			want:  new(time.Hour),
		},
		{
			// An explicit zero must take precedence over the environment variable and
			// the config, so it can't be treated as unset.
			name:  "zero",
			value: "0",
			want:  new(time.Duration(0)),
		},
		{
			name:    "an invalid duration",
			value:   "invalid",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inputGet := &ghtkn.InputGet{}
			err := flag.SetMinExpiration(inputGet, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("an error must be returned")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case tt.want == nil && inputGet.MinExpiration != nil:
				t.Fatalf("the min expiration is %v, want nil", *inputGet.MinExpiration)
			case tt.want == nil:
			case inputGet.MinExpiration == nil:
				t.Fatalf("the min expiration is nil, want %v", *tt.want)
			case *inputGet.MinExpiration != *tt.want:
				t.Errorf("the min expiration is %v, want %v", *inputGet.MinExpiration, *tt.want)
			}
		})
	}
}
