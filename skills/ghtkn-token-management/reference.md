# Token Management

## Access Token Regeneration

ghtkn stores generated access tokens and their expiration dates in the backend.
`ghtkn get` retrieves these, and if the expiration has passed, regenerates the access token through Device Flow.
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

By the way, if you set the value to 8 hours or more, you can replicate how ghtkn regenerates the access token.
This could be useful if you want to test how `ghtkn` behaves.

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
