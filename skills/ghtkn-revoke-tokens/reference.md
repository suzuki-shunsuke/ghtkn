# How To Revoke Access Tokens

If an access token is leaked, it must be immediately invalidated.
[You can confirm if the leaked access token expires or not by GitHub API.](https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28#get-the-authenticated-user)

## `ghtkn revoke`

`version >= v0.2.7`

The simplest way is the `ghtkn revoke` command:

```sh
ghtkn revoke <app name>        # revoke the token stored for an app and delete it from the backend
ghtkn revoke ghu_xxx           # revoke a raw token directly (e.g. a leaked one)
ghtkn revoke ghu_a ghu_b foo   # revoke multiple tokens and an app's stored token at once
ghtkn revoke                   # revoke the token stored for GHTKN_APP or the default app
ghtkn revoke --all             # revoke the stored tokens of every app in the config
```

Each argument is classified by its prefix: arguments starting with a GitHub token prefix (`ghp_`, `github_pat_`, `gho_`, `ghu_`, `ghr_`) are revoked directly as raw access tokens, and all other arguments are treated as app names whose stored tokens are revoked and removed from the backend.
When no argument is given, it falls back to `GHTKN_APP` or the default app; when only raw tokens are given, the fallback is not used, so revoking a raw token never touches an unrelated app's stored token.

The `--all` flag revokes the stored tokens of every app in the config at once. This is meant for incident response: when the environment running ghtkn is compromised, you can revoke all stored tokens immediately. With `--all`, app name arguments are ignored, but raw access tokens are still revoked.

## GitHub REST API

You can also revoke access tokens directly via the GitHub REST API.

[You can revoke access tokens by GitHub REST API.](https://docs.github.com/en/rest/credentials/revoke?apiVersion=2022-11-28#revoke-a-list-of-credentials)

```sh
curl -L \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2026-03-10" \
  https://api.github.com/credentials/revoke \
  -d '{"credentials":["ghu_<REDACTED>"]}'
```

> [!NOTE]
> We Updated the guide at 2026-06-17. Previously, we misunderstood that the REST API doesn't support User Access Tokens and a client secret is required to revoke them.
> But actually, a client secret is unnecessary.
