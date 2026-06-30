---
name: ghtkn-multiple-apps
description: Configure and switch between multiple GitHub Apps in ghtkn. Use when setting up per-organization apps, selecting an app by argument or GHTKN_APP, or switching apps per directory with direnv.
---

ghtkn supports multiple GitHub Apps in the `apps` section of the config. The first app is the default. Select an app by:

- command line argument: `ghtkn get suzuki-shunsuke/write`
- the `GHTKN_APP` environment variable (handy with [direnv](https://direnv.net/) to switch per directory via `.envrc`)

`GHTKN_APP` holds an app name, not a token, so it's safe to commit in `.envrc`. A default app with no permissions is still useful for reading public repos and avoiding rate limits.

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
