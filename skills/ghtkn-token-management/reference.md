# Token Management

## Do not leak the access token

The token `ghtkn get` writes to stdout is a live secret: anyone who sees it can act as you on GitHub until it expires (up to eight hours) or you revoke it. Coding agents in particular have leaked it by printing what `ghtkn get` returned.

Rules:

- Never print, echo, log, or include the token in your output, a response to the user, a chat message, a commit message, or an issue/PR. This applies to `ghtkn get` and `ghtkn get -f json` alike.
- Do not run `ghtkn get` just to display or inspect the token. There is no reason to look at its value; if you need to check things, use `ghtkn info` or `env GH_TOKEN=$(ghtkn get) gh auth status`, neither of which prints the token.
- Consume the token without showing it. Assign it to an environment variable and pass that to the tool that needs it:

  ```sh
  GH_TOKEN=$(ghtkn get) gh issue list
  ```

Better still, avoid handling the raw token at all:

- For `git`, configure the [git credential helper](../ghtkn-git-credential-helper/reference.md) (`ghtkn git-credential`). Git then fetches a token itself for each operation, so you never run `ghtkn get` or see the token.
- For `gh` and similar tools, use a small wrapper that sets `GH_TOKEN=$(ghtkn get)` before invoking the tool (see the ghtkn-troubleshooting skill), so the token stays in an environment variable and never reaches your output.

If a token is exposed, revoke it immediately with `ghtkn revoke` (see the ghtkn-revoke-tokens skill).

## Access Token Regeneration

ghtkn stores generated access tokens and their expiration dates in the backend.
`ghtkn get` retrieves these, and if the expiration has passed, regenerates the access token through Device Flow.
(With the agent backend and refresh enabled, an expired token is instead refreshed silently from the stored refresh token when a usable one exists, and Device Flow runs only otherwise. See the ghtkn-refresh-token skill.)
The access token validity period is 8 hours.

By default, if the access token hasn't expired, it returns it, but this may result in a short-lived access token being returned.
By specifying `-min-expiration (-m) <minimum required validity period. Not a datetime but remaining time>`, the access token will be regenerated if its validity period is shorter than the specified duration.

```sh
ghtkn get -m 1h
```

`2h`, `30m`, `30s` etc. are also valid. Units are required.

You can also set this using an environment variable.

```sh
export GHTKN_MIN_EXPIRATION=10m
```

Or in the configuration file.

```yaml
min_expiration: 1h
```

If you're only using the GitHub CLI to call an API, it usually finishes in an instant, so you probably won't need to set this.
However, if you're passing the access token to a script that takes, say, 30 minutes to run, setting it to something like `50m` will prevent the token from expiring in the middle of the script.

By the way, if you set the value to more than 8 hours (the maximum lifetime of a user access token), you can replicate how ghtkn regenerates the access token, because no real token can ever satisfy the requirement.
This could be useful if you want to test how `ghtkn` behaves.

## Non-expiring GitHub Apps (strongly discouraged)

A GitHub App can be configured with user access token expiration disabled, in which case it issues access tokens that never expire.
ghtkn works with such an App, but using one is strongly discouraged.
The whole value of ghtkn is to hand out short-lived access tokens, so that a leaked token is only usable for a short window; a non-expiring token throws that away, and a leak is then effectively permanent until you notice it and revoke the token manually.
Keep user access token expiration enabled on your GitHub App.

For completeness, ghtkn does handle a non-expiring token correctly.
It is stored with no expiration date and served from the cache, instead of running the device flow on every `ghtkn get` (a token with no expiration would otherwise read as already expired and be regenerated every time).
`ghtkn auth` still regenerates it, because it asks for more validity than any token can have (see above), so you can replace a revoked non-expiring token by running `ghtkn auth`.

## ghtkn auth

`ghtkn auth` command authenticates to GitHub and caches an access token without printing it to stdout.
It regenerates the token regardless of any cached token. Regeneration normally runs the device flow, but with the agent backend and refresh enabled it silently refreshes from the stored refresh token when one is available, running the device flow only when no usable refresh token exists.
Unlike `ghtkn get`, the device flow is always allowed even though it is disabled by default (and even when `GHTKN_ENABLE_DEVICE_FLOW=false`).
Also unlike `ghtkn get`, it does not accept `-min-expiration (-m)`, nor read `GHTKN_MIN_EXPIRATION` or `min_expiration` in the configuration file.

## Disabling Automatic Device Flow

> [!IMPORTANT]
> As of v0.3.0, the automatic device flow is disabled by default.
> We plan to remove the `GHTKN_ENABLE_DEVICE_FLOW=true` / `ghtkn get -d` opt-in entirely at v0.4.0.
> For details, see https://github.com/suzuki-shunsuke/ghtkn/issues/474

`ghtkn` obtains a GitHub App User access token via the OAuth Device Flow, which is interactive: it prints a one-time (user) code and waits for the user.
A coding agent (or any background / non-interactive process) cannot complete this, so it would block until a device code expires.
The automatic device flow is disabled by default, so `ghtkn get` and `git-credential` fail fast with an actionable error instead of blocking. Run `ghtkn auth` explicitly in your own interactive terminal to authenticate.

If you want `ghtkn get` to start the device flow automatically (the behavior before v0.3.0), enable it explicitly with `GHTKN_ENABLE_DEVICE_FLOW=true` or `ghtkn get -d`.

## :bulb: Copying a one-time code to clipboard automatically

[#446](https://github.com/suzuki-shunsuke/ghtkn/issues/446) [#309](https://github.com/suzuki-shunsuke/ghtkn/issues/309#issuecomment-4726483175)

> [!WARNING]
> Some applications, such as coding agents, cmux, and Warp, can start the device flow via ghtkn automatically. However, it is dangerous to use a one-time code when you didn't execute ghtkn explicitly, as this could be a phishing attack.
> An attacker could initiate the device flow, copy the one-time code to your clipboard, trick you into submitting it, and compromise your access token.
> As of v0.3.0 the automatic device flow is disabled by default, which mitigates this; if you use this tip, keep it disabled and always start the device flow explicitly with `ghtkn auth`.

`ghtkn auth` can copy the one-time code to the system clipboard for you.
This is only available on `ghtkn auth`, the explicit, interactive authentication command - not on `ghtkn get` or `ghtkn git-credential`, which must not start the device flow on your behalf.
It is disabled by default; enable it with the `-clipboard` (`-p`) flag, the `GHTKN_CLIPBOARD` environment variable, or the `clipboard.enable` config field.

```sh
ghtkn auth -p
```

```sh
export GHTKN_CLIPBOARD=true
```

```yaml
clipboard:
  enable: true
```

### Shell-based alternative

If you prefer not to enable the built-in feature, you can extract the code from stderr and
pipe it to the clipboard yourself. Feel free to change the function name and customize it.

For macOS:

```sh
ghauth() {
  ghtkn auth "$@" 2>&1 |
    tee >(grep -oE "[A-Z0-9]{4}-[A-Z0-9]{4}" --line-buffered | head -n1 | tr -d "\n" | pbcopy)
}
```

For WSL:

```sh
ghauth() {
  ghtkn auth "$@" 2>&1 |
    tee >(grep -oE "[A-Z0-9]{4}-[A-Z0-9]{4}" --line-buffered | head -n1 | tr -d "\n" | clip.exe)
}
```
