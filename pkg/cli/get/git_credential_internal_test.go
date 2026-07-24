package get

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/get"
)

const (
	gcStdinWithPath    = "protocol=https\nhost=github.com\npath=suzuki-shunsuke/ghtkn\n\n"
	gcStdinWithoutPath = "protocol=https\nhost=github.com\n\n"
	gcSubcommandGet    = "get"
)

type gitCredentialTestCase struct {
	name         string
	arg          string
	stdin        string
	gitApp       string // value of GHTKN_GIT_APP
	wantSkip     bool
	wantAppName  string
	wantAppOwner string
}

func (d gitCredentialTestCase) run(t *testing.T) {
	t.Helper()
	r := &runner{
		isGitCredential: true,
		stdin:           strings.NewReader(d.stdin),
		getEnv: func(key string) string {
			if key == "GHTKN_GIT_APP" {
				return d.gitApp
			}
			return ""
		},
	}
	input := &get.Input{}
	inputGet := &ghtkn.InputGet{}
	skip, err := r.handleGitCredential(t.Context(), slog.New(slog.DiscardHandler), d.arg, input, inputGet)
	if err != nil {
		t.Fatalf("handleGitCredential() error: %v", err)
	}
	if skip != d.wantSkip {
		t.Errorf("skip = %v, want %v", skip, d.wantSkip)
	}
	if !input.IsGitCredential {
		t.Error("input.IsGitCredential = false, want true")
	}
	if inputGet.AppName != d.wantAppName {
		t.Errorf("inputGet.AppName = %q, want %q", inputGet.AppName, d.wantAppName)
	}
	if inputGet.AppOwner != d.wantAppOwner {
		t.Errorf("inputGet.AppOwner = %q, want %q", inputGet.AppOwner, d.wantAppOwner)
	}
}

func TestRunner_handleGitCredential(t *testing.T) {
	t.Parallel()
	tests := []gitCredentialTestCase{
		{
			name:         "set AppName from GHTKN_GIT_APP",
			arg:          gcSubcommandGet,
			stdin:        gcStdinWithPath,
			gitApp:       "suzuki-shunsuke/git",
			wantAppName:  "suzuki-shunsuke/git",
			wantAppOwner: "suzuki-shunsuke",
		},
		{
			name:         "GHTKN_GIT_APP is not set",
			arg:          gcSubcommandGet,
			stdin:        gcStdinWithPath,
			gitApp:       "",
			wantAppName:  "",
			wantAppOwner: "suzuki-shunsuke",
		},
		{
			name:         "non-get subcommand returns early",
			arg:          "store",
			stdin:        gcStdinWithPath,
			gitApp:       "suzuki-shunsuke/git",
			wantSkip:     true,
			wantAppName:  "",
			wantAppOwner: "",
		},
		{
			name:         "GHTKN_GIT_APP is applied even without path",
			arg:          gcSubcommandGet,
			stdin:        gcStdinWithoutPath,
			gitApp:       "suzuki-shunsuke/git",
			wantAppName:  "suzuki-shunsuke/git",
			wantAppOwner: "",
		},
	}
	for _, d := range tests {
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			d.run(t)
		})
	}
}
