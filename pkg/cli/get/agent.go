package get

import (
	"context"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn"
	"github.com/suzuki-shunsuke/ghtkn/pkg/agent/server"
	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/status"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
)

// warnStaleAgent warns when the running ghtkn agent was built from a different ghtkn
// version than this one. The agent is a long-running process, so upgrading ghtkn leaves
// it serving tokens from the old binary until it is restarted, and nothing else in the
// 'get' path would ever say so: the token comes back fine, just from stale code.
//
// It is best-effort and never fails the command. A config that cannot be read, a backend
// that is not the agent, an agent that is not running, and a socket error all mean there
// is nothing to warn about, so they are logged at debug and ignored.
//
// This runs before the token is fetched, so the warning is visible even when the fetch
// then fails against the stale agent.
func (r *runner) warnStaleAgent(ctx context.Context, logger *slogutil.Logger, configFilePath string) {
	cfg, err := ghtkn.LoadConfig(&ghtkn.InputLoadConfig{ConfigFilePath: configFilePath})
	if err != nil {
		slogerr.WithError(logger.Logger, err).Debug("resolve the backend for the agent version check")
		return
	}
	if cfg.Backend == nil || cfg.Backend.Type != "agent" {
		return
	}
	resp, running, err := status.Query(ctx, r.getEnv)
	if err != nil {
		slogerr.WithError(logger.Logger, err).Debug("query the agent version")
		return
	}
	if !running || !staleAgent(resp.Version, r.version) {
		return
	}
	logger.Warn(`the running ghtkn agent was built from a different ghtkn version than this one, `+
		`so it serves tokens from that other binary. Restart it with 'ghtkn agent stop', then start `+
		`and unlock it again to run this version.`,
		"agent_version", resp.Version, "ghtkn_version", r.version)
}

// staleAgent reports whether the running agent was built from a different ghtkn version
// than the one talking to it. It is a plain inequality rather than an ordering: either
// direction means the agent is not running this ghtkn's code, and a restart is the fix
// either way. Ordering would need semantic version parsing to say which side is older,
// which buys nothing here.
//
// The check is disabled unless both versions are known. An empty agentVersion is an agent
// too old to report one, and server.UnknownVersion on either side is a binary built
// without version information (e.g. with `go install`): comparing those would warn on
// every 'ghtkn get' for a development build, where the two binaries are usually the same
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
