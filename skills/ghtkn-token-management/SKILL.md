---
name: ghtkn-token-management
description: Manage the ghtkn access token lifecycle - regeneration, ghtkn auth, the automatic device flow, and clipboard. Use when tokens expire, configuring -min-expiration, authenticating, or copying the one-time code.
---

ghtkn caches access tokens in the backend and regenerates them via Device Flow on expiry (8-hour validity). Key points:

- `ghtkn get` returns a cached token; use `-min-expiration`/`-m` (or `GHTKN_MIN_EXPIRATION`, `min_expiration`) to force regeneration when too little validity remains.
- `ghtkn auth` always runs the device flow and caches the token without printing it.
- The automatic device flow is disabled by default (v0.3.0+); `ghtkn get` / `git-credential` fail fast instead of blocking. Run `ghtkn auth` interactively to authenticate.
- `ghtkn auth` can copy the one-time code to the clipboard (`-clipboard`/`-p`, `GHTKN_CLIPBOARD`, or `clipboard.enable`).

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
