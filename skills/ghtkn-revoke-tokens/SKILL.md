---
name: ghtkn-revoke-tokens
description: Revoke leaked or compromised ghtkn access tokens. Use during incident response, when a token leaks, or to revoke an app's stored token via ghtkn revoke or the GitHub REST API.
---

If an access token leaks, invalidate it immediately:

- `ghtkn revoke <app name>` revokes and deletes an app's stored token from the backend.
- `ghtkn revoke ghu_xxx` revokes a raw token directly (args with token prefixes like `ghu_`, `ghp_`, `gho_`, `ghr_`, `github_pat_` are treated as raw tokens; others as app names).
- `ghtkn revoke` with no args falls back to `GHTKN_APP` or the default app.
- `ghtkn revoke --all` revokes every app's stored token (incident response).
- Alternatively, call the GitHub REST API `POST /credentials/revoke` (no client secret required).

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
