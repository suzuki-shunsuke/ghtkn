# Troubleshooting

## ghtkn doesn't work well

1. Check environment variables, ghtkn version, etc.

```sh
ghtkn info [<app name>]
```

If `ghtkn info` command isn't found or the version isn't latest, please upgrade ghtkn to the latest version.

2. Check the token and expiration date.

```sh
ghtkn get -f json [<app name>]
```

3. Check the access token is available.

```sh
env GH_TOKEN=$(ghtkn get) gh auth status
```

Please confirm the prefix of the token is `ghu_`.
If the prefix isn't `ghu_`, another type of token is used.

```
github.com
  o Logged in to github.com account suzuki-shunsuke (GH_TOKEN)
  - Active account: true
  - Git operations protocol: https
  - Token: ghu_************************************
```

4. Check the access token is valid using curl.

```sh
curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $(ghtkn get)" \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  https://api.github.com/user
```

5. Check the configuration file is correct.

## The wrapper of `gh` command doesn't work well

1. [Check if `ghtkn` works well](#ghtkn-doesnt-work-well)
1. Check if the wrapper is invoked correctly.

```sh
command -v gh
```

1. Check if another access token like personal access token is set
1. Add the debug log to the wrapper.

e.g.

```sh
if [ -z "${GH_TOKEN:-}" ] && [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "[WARN] skip ghtkn because GH_TOKEN or GITHUB_TOKEN is set" >&2 # Add the debug log
  GH_TOKEN="$(ghtkn get)" 
  export GH_TOKEN
fi
```

## ghtkn returns an expired token (401)

If `ghtkn get` returns an expired token, you can renew it by running `ghtkn auth`.

```sh
ghtkn auth
```

## The device flow asks the verification code, but the code isn't shown anywhere

When ghtkn is run in the background process, the verification code is not displayed in the terminal.
In that case, you need to:

1. Cancel the process `A`
1. Run `ghtkn auth [app for process A]` manually to renew the access token
1. Re-run the process `A`

As of ghtkn v0.3.0, the automatic device flow is disabled by default, so this kind of issue no longer happens. When the token expires, you need to run `ghtkn auth` to renew it.

## A browser opens when using tools like cmux and warp

When using [cmux](https://github.com/manaflow-ai/cmux) and [warp](https://github.com/warpdotdev/warp), ghtkn may open a browser on its own.
Worse, the one-time code isn't shown anywhere, so you can't complete the device flow and have to close the browser tab.

As of ghtkn v0.3.0, the automatic device flow is disabled by default, so this kind of issue no longer happens. When the token expires, you need to run `ghtkn auth` to renew it.

## Limitations

ghtkn obtains a user access token, but unfortunately it has some limitations so a personal access token is required for some operations.

1. Packages API requires a classic personal access token
1. It's difficult to write other user's repositories

### Packages API requires a classic personal access token

- https://docs.github.com/en/rest/packages/packages?apiVersion=2026-03-10
- > To use the REST API to manage GitHub Packages, you must authenticate using a personal access token (classic).

### It's difficult to write other user's repositories

To write other users' repositories, a GitHub App installed on the target repository and its client id is required.
It's hard to ask others to install a GitHub App on their repository and share the client id with you.

For instance, it's difficult to create pull requests to other users' repositories by `gh pr create` command.
In that case, the `--web` option of `gh pr create` is useful.
