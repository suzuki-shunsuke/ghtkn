// Package agent provides the controller for the 'ghtkn agent' command.
// It implements the agent server: a long-running process that caches GitHub App
// access tokens and serves them to clients over a Unix domain socket.
//
// The agent starts locked: it listens without a passphrase so it can run as a
// background service (e.g. systemd) that needs no terminal. A separate
// 'ghtkn agent unlock' command prompts for the passphrase on a terminal and sends
// it over the socket; only then is the data key loaded and tokens become readable.
//
// Tokens are encrypted at rest with AES-256-GCM under a data key that is itself
// wrapped with a passphrase-derived (Argon2id) key-encryption key. The derived
// keys live only in memory. The socket protocol (in ghtkn-go-sdk/ghtkn/backend/agent)
// is the contract between the agent and its clients.
package agent

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent/tokenstore"
	"github.com/suzuki-shunsuke/go-github-device-flow/deviceflow"
	"github.com/suzuki-shunsuke/go-revoke-github-access-token/revoke"
)

// goosWindows is the runtime.GOOS value for Windows.
const goosWindows = "windows"

// RefreshTokenSupported reports whether the refresh-token feature may be enabled on
// goos. It is unsupported on Windows because the defenses that keep a stored refresh
// token from leaking are POSIX-specific: the 0600 permissions the agent sets on the
// key, the token files, and the socket are effectively a no-op there, and there is no
// equivalent of the PR_SET_DUMPABLE hardening that stops a same-user process from
// reading the agent's memory (see harden.Process). A refresh token outlives the 8-hour
// access token by months, so it is not worth storing without them.
//
// This is the single source of truth for the restriction: the CLI rejects
// --enable-refresh up front by calling this, and handleUnlock refuses an UNLOCK that
// asks for it, so no client can turn the feature on.
func RefreshTokenSupported(goos string) bool {
	return goos != goosWindows
}

// githubHTTPTimeout bounds a single HTTP request to GitHub (device-code request, token
// refresh, and credential revocation, plus each individual device-flow poll request). Go's
// default transport bounds only dial and TLS handshake, so a peer that accepts the
// connection but never sends response headers would otherwise block the handler goroutine
// forever. GitHub normally responds in well under a second, so this is a generous backstop.
// It bounds each request, not the whole device-flow poll loop, which legitimately runs for
// the device code's lifetime.
const githubHTTPTimeout = 30 * time.Second

// revoker revokes raw GitHub access tokens via GitHub's credential revocation API.
type revoker interface {
	Revoke(ctx context.Context, tokens []string) error
}

// Controller runs the ghtkn agent server.
type Controller struct {
	// mu guards store, which is swapped from nil (locked) to a disk store on unlock.
	mu    sync.RWMutex
	store *tokenstore.Store // nil while locked

	// shutdown cancels the serve loop. It is set while the server is running
	// (see Start) and invoked when a STOP command is received.
	shutdown context.CancelFunc
	// logger is the server logger, set in Start so socket handlers can log.
	logger *slog.Logger
	// keyFile and tokenDir are the server's on-disk locations, set in Start.
	keyFile  string
	tokenDir string

	// statusMu guards status.
	statusMu sync.Mutex
	// status tracks the device flows in progress, keyed by client ID. An entry holds
	// the one-time code info so any client polling GET can display it. An entry means
	// a flow is running; it is deleted when the flow finishes (success or failure).
	status map[string]*deviceFlowState
	// client runs the GitHub device flow (device-code request and access-token poll).
	client *deviceflow.Client
	// revoker revokes stored tokens via GitHub's credential revocation API.
	revoker revoker
	// enableRefreshToken lets an expiring access token be refreshed with a stored
	// refresh token instead of re-running the device flow. It is part of the unlocked
	// state (guarded by mu): set from the passphrase-authenticated UNLOCK and read per
	// GET, never from the environment.
	enableRefreshToken bool
	// refreshTokenTTL is how long a stored token may sit unused before the periodic
	// sweep discards it (see sweep.go). It is part of the unlocked state (guarded by mu),
	// set from UNLOCK, and only used when enableRefreshToken is set.
	refreshTokenTTL time.Duration
	// sweepCancel stops the refresh-token sweep started at unlock. It is part of the
	// unlocked state (guarded by mu): UNLOCK sets it to a cancel func derived from the
	// server context, and LOCK calls it so the sweep goroutine does not outlive the
	// unlocked state (which would leak a goroutine and run multiple sweeps across
	// lock/unlock cycles). It is nil when refresh is disabled (no sweep runs).
	sweepCancel context.CancelFunc
	// goos is the GOOS the agent runs on, set in New and overridable in tests. It gates
	// the refresh-token feature (see RefreshTokenSupported); it is read-only after New,
	// so it needs no lock.
	goos string
}

// refreshEnabled reports whether refreshing expiring access tokens with stored refresh
// tokens is enabled. It is set at unlock time.
func (c *Controller) refreshEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enableRefreshToken
}

// refreshState reports whether refresh is enabled and, if so, the sweep TTL, read
// together under one lock so STATUS reports a consistent pair. The TTL is meaningful
// only when refresh is enabled (i.e. the agent is unlocked with it on).
func (c *Controller) refreshState() (bool, time.Duration) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enableRefreshToken, c.refreshTokenTTL
}

// New creates a new agent Controller. The server starts locked (no token store);
// it is unlocked later via the UNLOCK command.
//
// The controller reads the clock with time.Now rather than an injectable hook: tests
// that need a controlled clock run inside a testing/synctest bubble, where the time
// package itself is fake.
func New() *Controller {
	// One HTTP client, shared by the device-flow and revoke clients, with a per-request
	// timeout so no GitHub call can block a handler goroutine indefinitely.
	httpClient := &http.Client{Timeout: githubHTTPTimeout}
	return &Controller{
		status:  map[string]*deviceFlowState{},
		client:  deviceflow.New(&deviceflow.Input{HTTPClient: httpClient}),
		revoker: revoke.New(httpClient),
		goos:    runtime.GOOS,
	}
}
