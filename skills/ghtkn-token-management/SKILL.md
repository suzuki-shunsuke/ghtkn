---
name: ghtkn-token-management
description: Manage the ghtkn access token lifecycle - regeneration, ghtkn auth, the automatic device flow, and clipboard. Use when tokens expire, configuring -min-expiration, authenticating, or copying the one-time code. The token ghtkn get outputs is a secret; never print or log it.
---

> [!WARNING]
> The token `ghtkn get` outputs is a secret. Do not print, echo, log, or include it in your output, a chat message, or a commit, and do not run `ghtkn get` (including `-f json`) just to display or inspect it - a leaked token is usable until it is revoked. Consume it without showing it: assign it to an environment variable (`GH_TOKEN=$(ghtkn get) gh ...`), or avoid the raw token entirely with the git credential helper (`ghtkn git-credential`) for git and a `GH_TOKEN` wrapper for gh.

ghtkn caches access tokens in the backend and regenerates them via Device Flow on expiry (8-hour validity). Key points:

- `ghtkn get` returns a cached token; use `-min-expiration`/`-m` (or `GHTKN_MIN_EXPIRATION`, `min_expiration`) to force regeneration when too little validity remains.
- `ghtkn auth` regenerates and caches the token without printing it. It normally runs the device flow, but with the agent backend and refresh enabled it silently refreshes from the stored refresh token when one is available, and runs the device flow only otherwise.
- The automatic device flow is disabled by default (v0.3.0+); `ghtkn get` / `git-credential` fail fast instead of blocking. Run `ghtkn auth` interactively to authenticate.
- `ghtkn auth` can copy the one-time code to the clipboard (`-clipboard`/`-p`, `GHTKN_CLIPBOARD`, or `clipboard.enable`).

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
