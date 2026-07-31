package exec_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/exec"
	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
)

// mockClient returns the access token registered for the requested app name, an
// empty name meaning the app ghtkn selects automatically.
type mockClient struct {
	tokens map[string]string
	errs   map[string]error
	// nilApp returns no app configuration, which is what the SDK does when
	// GHTKN_GITHUB_TOKEN is set and its token is returned for every app.
	nilApp bool
	// gotAppNames records the requested app names in order, so that a test can tell
	// how many times an app was asked for.
	gotAppNames []string
}

func (m *mockClient) Get(_ context.Context, _ *slog.Logger, input *ghtkn.InputGet) (*ghtkn.AccessToken, *ghtkn.AppConfig, error) {
	m.gotAppNames = append(m.gotAppNames, input.AppName)
	if err, ok := m.errs[input.AppName]; ok {
		return nil, nil, err
	}
	token := &ghtkn.AccessToken{
		AccessToken: m.tokens[input.AppName],
	}
	if m.nilApp {
		return token, nil, nil
	}
	return token, &ghtkn.AppConfig{Name: input.AppName}, nil
}

type mockRunner struct {
	called  bool
	gotEnv  []string
	gotName string
	gotArgs []string
	code    int
	err     error
}

func (m *mockRunner) Run(_ *slog.Logger, env []string, name string, args ...string) (int, error) {
	m.called = true
	m.gotEnv = env
	m.gotName = name
	m.gotArgs = args
	return m.code, m.err
}

var errGetToken = errors.New("app is not found in the config")

type runTestCase struct {
	name            string
	envVars         []*exec.EnvVar
	command         []string
	continueOnError bool
	tokens          map[string]string
	errs            map[string]error
	environ         []string
	nilApp          bool
	runnerCode      int
	runnerErr       error
	wantAppNames    []string
	wantRunner      bool
	wantEnv         []string
	wantErr         bool
	wantExitCode    int
	wantSilent      bool
}

func runTestCases() []*runTestCase { //nolint:funlen // A table of test cases.
	return []*runTestCase{
		{
			name:         "the token of the app ghtkn selects goes to GITHUB_TOKEN",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			tokens:       map[string]string{"": "token-1"},
			environ:      []string{"PATH=/bin"},
			wantAppNames: []string{""},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "GITHUB_TOKEN=token-1"},
		},
		{
			name:         "two environment variables of one app need one token",
			envVars:      []*exec.EnvVar{{Name: "A"}, {Name: "B"}},
			tokens:       map[string]string{"": "token-1"},
			environ:      []string{"PATH=/bin"},
			wantAppNames: []string{""},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "A=token-1", "B=token-1"},
		},
		{
			name: "two apps are asked for in the order they were given",
			envVars: []*exec.EnvVar{
				{Name: "A", AppName: "app-a"},
				{Name: "B", AppName: "app-b"},
			},
			tokens:       map[string]string{"app-a": "token-a", "app-b": "token-b"},
			environ:      []string{"PATH=/bin"},
			wantAppNames: []string{"app-a", "app-b"},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "A=token-a", "B=token-b"},
		},
		{
			name:         "an inherited environment variable is replaced, not duplicated",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			tokens:       map[string]string{"": "token-1"},
			environ:      []string{"GITHUB_TOKEN=old", "PATH=/bin"},
			wantAppNames: []string{""},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "GITHUB_TOKEN=token-1"},
		},
		{
			// The SDK returns no app configuration then, which must not stop the
			// command from running.
			name:         "GHTKN_GITHUB_TOKEN overrides the requested app",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN", AppName: "app-a"}},
			tokens:       map[string]string{"app-a": "token-a"},
			nilApp:       true,
			environ:      []string{"PATH=/bin"},
			wantAppNames: []string{"app-a"},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "GITHUB_TOKEN=token-a"},
		},
		{
			name:         "the command isn't run when a token can't be acquired",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			errs:         map[string]error{"": errGetToken},
			environ:      []string{"PATH=/bin"},
			wantAppNames: []string{""},
			wantErr:      true,
			wantExitCode: 111,
		},
		{
			name: "-continue-on-error keeps the inherited value of the failed variable",
			envVars: []*exec.EnvVar{
				{Name: "A", AppName: "app-a"},
				{Name: "GITHUB_TOKEN", AppName: "app-b"},
			},
			continueOnError: true,
			tokens:          map[string]string{"app-b": "token-b"},
			errs:            map[string]error{"app-a": errGetToken},
			environ:         []string{"A=inherited", "GITHUB_TOKEN=old", "PATH=/bin"},
			wantAppNames:    []string{"app-a", "app-b"},
			wantRunner:      true,
			wantEnv:         []string{"A=inherited", "PATH=/bin", "GITHUB_TOKEN=token-b"},
		},
		{
			name: "-continue-on-error asks a failed app only once",
			envVars: []*exec.EnvVar{
				{Name: "A", AppName: "app-a"},
				{Name: "B", AppName: "app-a"},
			},
			continueOnError: true,
			errs:            map[string]error{"app-a": errGetToken},
			environ:         []string{"PATH=/bin"},
			wantAppNames:    []string{"app-a"},
			wantRunner:      true,
			wantEnv:         []string{"PATH=/bin"},
		},
		{
			name:            "an interruption stops even with -continue-on-error",
			envVars:         []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			continueOnError: true,
			errs:            map[string]error{"": context.Canceled},
			environ:         []string{"PATH=/bin"},
			wantAppNames:    []string{""},
			wantErr:         true,
			wantExitCode:    130,
			wantSilent:      true,
		},
		{
			name:         "the exit code of the command is propagated silently",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			tokens:       map[string]string{"": "token-1"},
			environ:      []string{"PATH=/bin"},
			runnerCode:   3,
			wantAppNames: []string{""},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "GITHUB_TOKEN=token-1"},
			wantErr:      true,
			wantExitCode: 3,
			wantSilent:   true,
		},
		{
			name:         "a command that can't be run is reported with its exit code",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			tokens:       map[string]string{"": "token-1"},
			environ:      []string{"PATH=/bin"},
			runnerCode:   127,
			runnerErr:    errors.New("executable file not found in $PATH"),
			wantAppNames: []string{""},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "GITHUB_TOKEN=token-1"},
			wantErr:      true,
			wantExitCode: 127,
		},
		{
			name:         "a failure without an exit code of its own exits 1",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			tokens:       map[string]string{"": "token-1"},
			environ:      []string{"PATH=/bin"},
			runnerErr:    errors.New("wait for the command"),
			wantAppNames: []string{""},
			wantRunner:   true,
			wantEnv:      []string{"PATH=/bin", "GITHUB_TOKEN=token-1"},
			wantErr:      true,
			wantExitCode: 1,
		},
		{
			name:         "no command to run",
			envVars:      []*exec.EnvVar{{Name: "GITHUB_TOKEN"}},
			command:      []string{},
			environ:      []string{"PATH=/bin"},
			wantAppNames: nil,
			wantErr:      true,
			wantExitCode: 1,
		},
	}
}

func TestController_Run(t *testing.T) {
	t.Parallel()
	for _, tt := range runTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testControllerRun(t, tt)
		})
	}
}

func testControllerRun(t *testing.T, tt *runTestCase) {
	t.Helper()
	client := &mockClient{
		tokens: tt.tokens,
		errs:   tt.errs,
		nilApp: tt.nilApp,
	}
	runner := &mockRunner{
		code: tt.runnerCode,
		err:  tt.runnerErr,
	}
	ctrl := exec.New(&exec.Input{
		Client:  client,
		Runner:  runner,
		Environ: func() []string { return tt.environ },
	})
	command := tt.command
	if command == nil {
		command = []string{"gh", "pr", "view"}
	}

	err := ctrl.Run(t.Context(), slog.New(slog.DiscardHandler), &exec.InputRun{
		InputGet:        &ghtkn.InputGet{},
		EnvVars:         tt.envVars,
		Command:         command,
		ContinueOnError: tt.continueOnError,
	})

	checkError(t, tt, err)
	if diff := cmp.Diff(tt.wantAppNames, client.gotAppNames); diff != "" {
		t.Errorf("requested app names are unexpected (-want +got):\n%s", diff)
	}
	if runner.called != tt.wantRunner {
		t.Fatalf("the command is run = %v, want %v", runner.called, tt.wantRunner)
	}
	if !tt.wantRunner {
		return
	}
	if diff := cmp.Diff(tt.wantEnv, runner.gotEnv); diff != "" {
		t.Errorf("the command's environment is unexpected (-want +got):\n%s", diff)
	}
	if runner.gotName != "gh" {
		t.Errorf("the command is %q, want gh", runner.gotName)
	}
	if diff := cmp.Diff([]string{"pr", "view"}, runner.gotArgs); diff != "" {
		t.Errorf("the command's arguments are unexpected (-want +got):\n%s", diff)
	}
}

func checkError(t *testing.T, tt *runTestCase, err error) {
	t.Helper()
	if !tt.wantErr {
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	if err == nil {
		t.Fatal("an error must be returned")
	}
	if code := ecerror.GetExitCode(err); code != tt.wantExitCode {
		t.Errorf("the exit code is %d, want %d: %v", code, tt.wantExitCode, err)
	}
	// A silent error keeps ghtkn from logging a failure of its own, either because
	// the command already reported it or because the user interrupted ghtkn.
	if silent := err.Error() == ""; silent != tt.wantSilent {
		t.Errorf("the error is silent = %v, want %v: %v", silent, tt.wantSilent, err)
	}
}
