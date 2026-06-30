---
name: ghtkn-troubleshooting
description: Diagnose ghtkn problems and known limitations. Use when ghtkn or the gh wrapper misbehaves, a token is expired (401), the device flow code isn't shown, or hitting Packages API / cross-user repo limits.
---

Common checks when ghtkn doesn't work:

- `ghtkn info [<app name>]` to check env vars and version; upgrade if not latest.
- `ghtkn get -f json [<app name>]` to inspect the token and expiration.
- `env GH_TOKEN=$(ghtkn get) gh auth status` and confirm the token prefix is `ghu_`.
- Expired token (401): renew with `ghtkn auth`.
- gh wrapper issues: verify `command -v gh`, that no other token is set, and add debug logging.

Known limitations: the Packages API requires a classic PAT, and writing other users' repositories needs a GitHub App installed there.

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
