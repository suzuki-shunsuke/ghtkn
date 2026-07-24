// Package refreshtoken holds the policy constraints of the ghtkn agent's refresh-token
// feature that are shared between the agent server and the CLI's up-front validation.
// It is a leaf package (it depends only on the standard library), so a caller can check
// these constraints without pulling in the agent server.
package refreshtoken

import "time"

// goosWindows is the runtime.GOOS value for Windows.
const goosWindows = "windows"

// MaxTTL caps --refresh-token-ttl: a stored token is useless once its refresh token
// expires (GitHub issues refresh tokens that live about six months), so a larger TTL is
// clamped to this. A month is counted as 30 days. It is the single source of truth for
// the upper bound: the server clamps to it (see the sweep) and the CLI rejects larger
// values up front (see pkg/cli/agent).
const MaxTTL = 6 * 30 * 24 * time.Hour

// Supported reports whether the refresh-token feature may be enabled on goos. It is
// unsupported on Windows because the defenses that keep a stored refresh token from
// leaking are POSIX-specific: the 0600 permissions the agent sets on the key, the token
// files, and the socket are effectively a no-op there, and there is no equivalent of the
// PR_SET_DUMPABLE hardening that stops a same-user process from reading the agent's
// memory (see harden.Process). A refresh token outlives the 8-hour access token by
// months, so it is not worth storing without them.
//
// It is the single source of truth for the restriction: the CLI rejects --enable-refresh
// up front by calling this, and the agent refuses an UNLOCK that asks for it, so no
// client can turn the feature on.
func Supported(goos string) bool {
	return goos != goosWindows
}
