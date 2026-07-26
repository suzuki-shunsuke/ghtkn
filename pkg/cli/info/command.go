// Package info implements the 'ghtkn info' command.
// It prints environment information useful for troubleshooting, such as the OS,
// architecture, ghtkn version, relevant environment variables (with tokens
// redacted), the selected backend, the target app, and the resolved
// configuration file path.
package info

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	agentapi "github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/backend/agent"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/server"
	"github.com/suzuki-shunsuke/ghtkn/pkg/cli/flag"
	"github.com/suzuki-shunsuke/ghtkn/pkg/config"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/status"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/info"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

// Args holds the flag and argument values for the info command.
type Args struct {
	*flag.GlobalFlags

	AppName string // positional argument for 'info' command
	Version string
}

func New(logger *slogutil.Logger, env *urfave.Env, gFlags *flag.GlobalFlags) *cli.Command {
	args := &Args{
		GlobalFlags: gFlags,
		Version:     env.Version,
	}
	r := &runner{
		stdin: env.Stdin,
	}
	return r.Command(logger, args)
}

type runner struct {
	stdin io.Reader
}

func (r *runner) Command(logger *slogutil.Logger, args *Args) *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "Output information about the environment which is useful for troubleshooting",
		Description: `Output environment information useful for troubleshooting.

It prints, as JSON, the OS, architecture, ghtkn version, relevant GHTKN_* and
related environment variables (with token values redacted), the selected backend,
the target app, and the resolved configuration file path. It does not authenticate
or modify any state.

With the agent backend it also reports the running agent's state, including the
ghtkn version it was built from and the agent protocol versions it speaks. The
agent keeps running the binary it was started with, so an agent version older than
the ghtkn version above it means the agent must be restarted; that case is called
out with a warning printed on stderr after the JSON, which leaves the output on
stdout untouched.

$ ghtkn info
$ ghtkn info my-app
$ ghtkn info | jq .envs`,
		Action: func(ctx context.Context, _ *cli.Command) error {
			return r.action(ctx, logger, args)
		},
		Flags: []cli.Flag{
			flag.LogLevel(&args.LogLevel),
			flag.Config(&args.Config),
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "app-name",
				Destination: &args.AppName,
			},
		},
	}
}

// action implements the 'info' command. It sets the log level, resolves the
// configuration file path, and delegates to the info controller to print the
// environment information. Returns an error if the config path can't be
// resolved or the controller fails.
func (r *runner) action(ctx context.Context, logger *slogutil.Logger, args *Args) error {
	if err := logger.SetLevel(args.LogLevel); err != nil {
		return fmt.Errorf("set log level: %w", err)
	}
	// Resolve the config path here (honoring the -c flag, then the default)
	// so the controller stays free of environment lookups and is easy to test.
	configPath, err := config.ResolvePath(args.Config)
	if err != nil {
		return err //nolint:wrapcheck
	}
	// Resolve the effective config (config file plus environment overrides) so info can
	// report the settings that actually take effect. The path resolved above is passed
	// in, so the reported config_path is always the file the reported config came from.
	// It is best-effort: a config that can't be loaded must not fail the troubleshooting
	// command, so warn and omit it.
	cfg, err := ghtkn.LoadConfig(&ghtkn.InputLoadConfig{ConfigFilePath: configPath})
	if err != nil {
		logger.Warn("load the config for the info output; omitting the config section", "error", err)
		cfg = nil
	}
	// When the effective backend is the agent, include its state. This is best-effort:
	// info is a troubleshooting command and must never fail because the agent is down.
	agentBackend := cfg != nil && cfg.Backend != nil && cfg.Backend.Type == "agent"
	agent := buildAgentStatus(ctx, agentBackend, logger)
	err = info.New(os.Stdout, os.Getenv).Info(configPath, args.AppName, args.Version, cfg, agent)
	// After the output, not before: the JSON runs to dozens of lines, and a warning
	// scrolled off the top of it is a warning nobody reads. It is emitted even when the
	// output failed, since a stale agent is worth knowing about either way.
	warnStaleAgent(logger, agent, args.Version)
	return err //nolint:wrapcheck
}

// warnStaleAgent warns when the running agent was built from a different ghtkn version
// than this one. The two versions are already in the JSON output, but info is the command
// people run when something is off, and a stale agent is the kind of thing that is easy to
// read past: the output looks healthy, the tokens come back fine, they just come from the
// binary the agent was started with.
//
// The warning goes to the logger, i.e. stderr, so it never mixes into the JSON on stdout,
// and it is emitted after the output so it is the last thing on screen rather than the
// first line scrolled away by the JSON.
func warnStaleAgent(logger *slogutil.Logger, agent *info.AgentStatus, version string) {
	if agent == nil || !agent.Running || !staleAgent(agent.Version, version) {
		return
	}
	logger.Warn(`the running ghtkn agent was built from a different ghtkn version than this one, `+
		`so it serves tokens from that other binary. Restart it with 'ghtkn agent stop', then start `+
		`and unlock it again to run this version.`,
		"agent_version", agent.Version, "ghtkn_version", version)
}

// staleAgent reports whether the running agent was built from a different ghtkn version
// than the one talking to it. It is a plain inequality rather than an ordering: either
// direction means the agent is not running this ghtkn's code, and a restart is the fix
// either way. Ordering would need semantic version parsing to say which side is older,
// which buys nothing here.
//
// The check is disabled unless both versions are known. An empty agentVersion is an agent
// too old to report one, and server.UnknownVersion on either side is a binary built
// without version information (e.g. with `go install`), where the two are usually the same
// working tree anyway.
func staleAgent(agentVersion, ghtknVersion string) bool {
	if agentVersion == "" || ghtknVersion == "" {
		return false
	}
	if agentVersion == server.UnknownVersion || ghtknVersion == server.UnknownVersion {
		return false
	}
	return agentVersion != ghtknVersion
}

// buildAgentStatus queries the running agent and builds the info output's agent section,
// or returns nil when the backend is not the agent. It is best-effort: a not-running or
// erroring agent yields {running:false} (with a warning) rather than failing info.
func buildAgentStatus(ctx context.Context, agentBackend bool, logger *slogutil.Logger) *info.AgentStatus {
	if !agentBackend {
		return nil
	}
	resp, running, err := status.Query(ctx, os.Getenv)
	if err != nil {
		slogerr.WithError(logger.Logger, err).Warn("query the agent status for the info output")
		return &info.AgentStatus{Running: false}
	}
	return agentStatusFromResponse(running, resp)
}

// agentStatusFromResponse maps an agent STATUS response to the info output's agent
// section. Locked and RefreshToken describe the unlocked agent, so they are set only when
// the agent is running (Locked) and unlocked (RefreshToken). It is a pure function so the
// shaping is testable without a socket.
//
// The version fields describe the agent process, so they are set whenever it is running,
// locked or not. An agent too old to report its version leaves Version empty, which drops
// it from the output; its protocol versions are still reported, because both fields
// decode to a value that is true of such an agent (0, i.e. pre-versioning, for the one it
// speaks and the minimum it accepts).
func agentStatusFromResponse(running bool, resp *agentapi.Response) *info.AgentStatus {
	if !running {
		return &info.AgentStatus{Running: false}
	}
	st := &info.AgentStatus{
		Running: true, Locked: new(resp.Locked), Version: resp.Version,
		ProtocolVersion: new(resp.ProtocolVersion), MinProtocolVersion: new(resp.MinProtocolVersion),
	}
	if !resp.Locked {
		// enabled/ttl describe the unlocked, refresh-enabled state; TTL is present only
		// when the agent reports one (an older agent omits it).
		rt := &info.AgentRefreshToken{Enabled: resp.RefreshTokenEnabled}
		if resp.RefreshTokenTTL > 0 {
			rt.TTL = formatTTLDays(resp.RefreshTokenTTL)
		}
		st.RefreshToken = rt
	}
	return st
}

// formatTTLDays renders a refresh-token TTL in days, e.g. "3d" or "10.5d".
func formatTTLDays(d time.Duration) string {
	return strconv.FormatFloat(d.Hours()/24, 'g', -1, 64) + "d"
}
