---
name: ghtkn-configuration
description: Configure ghtkn via config file, environment variables, and CLI args. Use for configuration priority, disabling browser open, the GitHub account picker, enterprise GitHub App sharing, or GHTKN_GITHUB_TOKEN.
---

ghtkn reads settings from command line arguments, environment variables, and configuration files (in this priority order). Topics covered:

- Disabling automatic browser open (`GHTKN_OPEN_BROWSER=false` or `open_browser.enable: false`).
- The GitHub account picker (`skip_account_picker`).
- Sharing a single GitHub App across an enterprise organization (share the Client ID, which isn't a secret).
- Using a personal access token for one-off operations via `GHTKN_GITHUB_TOKEN`.

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
