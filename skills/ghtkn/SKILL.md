---
name: ghtkn
description: |
  Read ghtkn's documentation with `ghtkn docs list` and `ghtkn docs show <name>` before answering.
  ghtkn is a CLI that creates short-lived (8h) GitHub user access tokens from GitHub Apps.
  Use for any question about ghtkn, `GHTKN_*` environment variables, `ghtkn get` / `exec` / `auth` / `agent` / `revoke`, the ghtkn git credential helper, or errors from ghtkn.
---

> [!WARNING]
> The token `ghtkn get` outputs is a secret. Never print, echo, log, or include it in your
> output or a commit, and don't run `ghtkn get` just to inspect it. Run tools with
> `ghtkn exec` (`ghtkn exec -e GH_TOKEN -- gh ...`), which never prints the token, or use
> `ghtkn git-credential` for git.

Don't answer from memory. Run `ghtkn docs list` to list the documentation, then
`ghtkn docs show <name>` to read the relevant topics before answering questions about
ghtkn or troubleshooting its errors. `ghtkn docs show` prints the whole topic; read it
through before concluding.

If `ghtkn docs` is rejected as an unknown command, the installed ghtkn is older than v0.3.4.
Tell the user to upgrade, and don't fall back to guessing.

If `ghtkn` is not installed at all, point them at the install guide:

https://github.com/suzuki-shunsuke/ghtkn/blob/main/docs/install.md
