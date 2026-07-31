---
name: ghtkn-exec
description: Run a command with ghtkn access tokens in environment variables using ghtkn exec. Use to keep tokens out of your output instead of ghtkn get, to give one command tokens of several apps, or to understand its exit codes and signal handling.
---

> [!IMPORTANT]
> If you are a coding agent, prefer `ghtkn exec` over `ghtkn get`. Tokens leaked by agents printing what `ghtkn get` returned are the reason this command exists.

`ghtkn exec` runs a command with GitHub App User Access Tokens in its environment, so ghtkn never writes the token to stdout:

```sh
ghtkn exec -- gh pr view
ghtkn exec -e GH_TOKEN -- gh pr view
ghtkn exec -e PINACT_GITHUB_TOKEN:suzuki-shunsuke/read -e GH_TOKEN:suzuki-shunsuke/write -- bash foo.sh
```

- The `--` in front of the command is required; everything after it belongs to the command, so its own flags aren't parsed by ghtkn.
- The token goes to `GITHUB_TOKEN` by default. `-e` replaces that default entirely: `-e <env name>` uses the app ghtkn selects automatically, `-e <env name>:<app name>` uses that app, and `-e` can be repeated.
- The exit code is the command's own. 111 means no access token could be acquired, and the command was not run.
- The device flow is disabled by default, exactly as in `ghtkn get`. Run `ghtkn auth` first, or pass `-device-flow`.
- The token is an environment variable of the command, so a command that prints its environment still exposes it. `ghtkn exec` prevents accidents, it doesn't make the token unreadable.

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
