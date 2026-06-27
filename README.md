# ghtkn (GH-Token)

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/suzuki-shunsuke/ghtkn)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/main/LICENSE) | [Install](INSTALL.md) | [Usage](USAGE.md)

**Stop risking token leaks - Use secure, short-lived GitHub tokens for local development**

## :warning: The Security Problem

Are you still using Personal Access Tokens (PATs) or GitHub CLI OAuth tokens stored on your local machine? These long-lived tokens pose **significant security risks**:
- **Indefinite or months-long validity** - A leaked token remains dangerous for extended periods
- **Broad permissions** - Often configured with wide access for convenience
- **Difficult to rotate** - Manual management leads to tokens being used far longer than they should

## :white_check_mark: The ghtkn Solution

ghtkn generates **8-hour User Access Tokens** from GitHub Apps using Device Flow - a fundamentally more secure approach:
- **Short-lived tokens** - Only 8 hours validity minimizes damage from any potential leak
- **No secrets required** - Only needs a Client ID (which isn't secret), no Private Keys or Client Secrets
- **User-attributed actions** - Operations are performed as you, not as an app
- **Automatic token management** - Integrates with the backend (the default is OS keyring) for secure storage and reuse

ghtkn (pronounced `GH-Token`) allows you to manage multiple GitHub Apps through configuration files and securely store tokens using OS keyring (Windows Credential Manager, macOS Keychain, or GNOME Keyring) or another backend.

## :rocket: Getting Started

1. [Install ghtkn](INSTALL.md)
2. [Create a GitHub App](https://github.com/settings/apps/new?url=https://github.com/suzuki-shunsuke/ghtkn&device_flow_enabled=true&webhook_active=false&public=false)

- Enable Device Flow
- Disable Webhook
- Homepage URL: https://github.com/suzuki-shunsuke/ghtkn (You can change this freely. If you share the GitHub App in your development team, it's good to prepare the document and set it to Homepage URL)
- `Only on this account`
- Permissions: Nothing
- Repositories: Nothing

You don't need to create secrets such as Client Secrets and Private Keys.

3. Create a configuration file by `ghtkn init` and modify it

```sh
ghtkn init
```

- Windows: `%APPDATA%\ghtkn\ghtkn.yaml`
- macOS, Linux: `${XDG_CONFIG_HOME:-${HOME}/.config}/ghtkn/ghtkn.yaml`

```yaml:ghtkn.yaml
apps:
  - name: suzuki-shunsuke/none
    client_id: xxx # Mandatory. GitHub App Client ID
```

> [!NOTE]  
> The GitHub App Client ID is not a secret, so there's generally no problem writing it in plain text in local configuration files.

4. Run `ghtkn get` and create a user access token

```sh
ghtkn get
```

https://github.com/login/device will open in your browser, so enter the code displayed in the terminal and approve it.
Then a user access token starting with `ghu_` is outputted.
You can close the opened tab.

With Device Flow, access tokens cannot be generated in non-interactive environments like CI.
ghtkn is primarily intended for local development.

If you run the same command immediately, it will now run without the authorization flow because ghtkn stores access tokens into the backend and reuse them.

```sh
ghtkn get
```

5. Run `gh issue create` using the access token

```sh
REPO=suzuki-shunsuke/ghtkn # Please change this to your public repository
env GH_TOKEN=$(ghtkn get) gh issue create -R "$REPO" --title "Hello, ghtkn" --body "This is created by ghtkn"
```

Then it fails due to the permission error even if you have the permission.

```
GraphQL: Resource not accessible by integration (createIssue)
```

Please grant the permission `issues:write` to the GitHub App and run again, then it still fails.
Please install the app to the repository and run again, then it succeeds.
At this time, the issue creator will be you, not the App.

The permissions (Permissions and Repositories) of a user access token are held by both the authorized user (i.e. you) and the GitHub App.
Therefore, as shown above, the GitHub App cannot perform operations that it is not permitted to perform, and conversely, the user cannot perform operations that they are not authorized to perform.

## Wrapping commands

You can wrap commands using shell functions or scripts.

Shell functions:

```sh
gh() {
    env GH_TOKEN=$(ghtkn get) command gh "$@" # Be careful to use 'command' to avoid infinite loops
}
```

Shell scripts:

1. Put shell scripts in $PATH:

e.g. ~/.local/bin/gh:

```sh
#!/usr/bin/env bash

set -eu

# If GH_TOKEN or GITHUB_TOKEN is set, use it.
if [ -z "${GH_TOKEN:-}" ] && [ -z "${GITHUB_TOKEN:-}" ]; then
  # echo "[WARN] skip ghtkn because GH_TOKEN or GITHUB_TOKEN is set" >&2
  GH_TOKEN="$(ghtkn get)" 
  export GH_TOKEN
fi

exec /opt/homebrew/bin/gh "$@" # Specify the absolute path to avoid infinite loop
```

If the command is managed by [aqua](https://aquaproj.github.io/), `aqua exec` is useful:

```sh
exec aqua exec -- gh "$@"
```

2. Make scripts executable

```sh
chmod +x ~/.local/bin/gh
```

It's useful to wrap `gh` using shell script as gh always requires GitHub access tokens.

## Git Credential Helper

`version >= v0.1.2`

You can use ghtkn as a [Git Credential Helper](https://git-scm.com/book/en/v2/Git-Tools-Credential-Storage):

```sh
git config --global credential.helper '!ghtkn git-credential'
```

```ini
[credential]
	helper =
	helper = !ghtkn git-credential
```

> [!IMPORTANT]
> `helper =` is necessary to disable other helpers.
> https://git-scm.com/docs/gitcredentials#_configuration_options
> > If credential.helper is configured to the empty string, this resets the helper list to empty
> > (so you may override a helper set by a lower-priority config file by configuring the empty-string helper, followed by whatever set of helpers you would like).

### Switching GitHub Apps by repository owner

If you want to switch GitHub Apps by repository owner,

1. Set `.apps[].git_owner` in a configuration file
1. Configure Git `git config credential.useHttpPath true`

```sh
git config --global credential.useHttpPath true
```

```yaml
apps:
  - name: suzuki-shunsuke/write
    client_id: xxx
    git_owner: suzuki-shunsuke # Using this app if the repository owner is suzuki-shunsuke
```

> [!WARNING]
> `git_owner` must be unique.
> Please set `git_owner` to only one app per repository owner (organization and user).
> For instance, if you use a read-only app and a write app for a repository owner and you want to push commits, you should set `git_owner` to the write app.
>
> ```yaml
> apps:
>   - name: suzuki-shunsuke/write
>     client_id: xxx
>     git_owner: suzuki-shunsuke # Using this app if the repository owner is suzuki-shunsuke
>   - name: suzuki-shunsuke/read-only
>     client_id: xxx
>     # git_owner: suzuki-shunsuke # Don't set `git_owner` to read-only app to push commits
> ```

#### Switching GitHub Apps to access fork repositories

Unfortunately, `.apps[].git_owner` doesn't match when accessing fork repositories.
For instance, when you checkout a pull request from a fork repository by [gh pr checkout](https://cli.github.com/manual/gh_pr_checkout) command and push commits to the fork repository, `.apps[].git_owner` doesn't work unless you configure fork repositories in `ghtkn.yaml`.

As of ghtkn v0.2.6, the environment variable `GHTKN_GIT_APP` is useful.
`GHTKN_GIT_APP` is similar to `GHTKN_APP` but it's used for Git Credential Helper.

e.g.

```sh
export GHTKN_GIT_APP=suzuki-shunsuke/git
```

The priority of the app used for Git Credential Helper is as follows:

1. `.apps[].git_owner` if git credential helper's username matches
1. `GHTKN_GIT_APP`
1. `GHTKN_APP` if `GHTKN_GIT_APP` is not set
1. The default app

### :warning: Troubleshooting of Git Credential Helper on macOS

If Git Credential Helper doesn't work on macOS, please check if osxkeychain is used.

You can check the trace log of Git by `GIT_TRACE=1 GIT_CURL_VERBOSE=1`.

```sh
GIT_TRACE=1 GIT_CURL_VERBOSE=1 git push origin
```

If git outputs the following log, Git uses `git-credential-osxkeychain`, not ghtkn.

```
09:25:49.373133 git.c:750               trace: exec: git-credential-osxkeychain get
09:25:49.373152 run-command.c:655       trace: run_command: git-credential-osxkeychain get
```

Please check the git config.

```sh
git config --get-all --show-origin credential.helper
```

The following output shows osxkeychain is used by the system setting `/Library/Developer/CommandLineTools/usr/share/git-core/gitconfig`.

```
file:/Library/Developer/CommandLineTools/usr/share/git-core/gitconfig   osxkeychain
file:/Users/shunsukesuzuki/.gitconfig   !ghtkn git-credential
```

To solve the problem, please set credential.helper to the empty string.

```ini
[credential]
	helper =
	helper = !ghtkn git-credential
```

https://git-scm.com/docs/gitcredentials#_configuration_options

> If credential.helper is configured to the empty string, this resets the helper list to empty
> (so you may override a helper set by a lower-priority config file by configuring the empty-string helper, followed by whatever set of helpers you would like).

## Using Multiple Apps

You can configure multiple GitHub Apps in the `apps` section of the configuration file and create and use different Apps for each Organization or User.
By default, the first App in `apps` is used.

You can specify the App by command line argument:

```sh
ghtkn get suzuki-shunsuke/write
```

The value is the app name defined in the configuration file.
Alternatively, you can specify it with the environment variable `GHTKN_APP`.
For example, it might be convenient to switch `GHTKN_APP` for each directory using a tool like [direnv](https://direnv.net/).

I check out my repositories from [https://github.com/suzuki-shunsuke](https://github.com/suzuki-shunsuke) into the `~/repos/src/github.com/suzuki-shunsuke` directory.
I then place a `.envrc` file in that directory with the following content:

```sh
source_up

export GHTKN_APP=suzuki-shunsuke/write
```

Similarly, I place a `.envrc` file in `~/repos/src/github.com/aquaproj` as well:

```sh
source_up

export GHTKN_APP=aquaproj/write
```

I've also set up a default App that has no permissions.
While some might think an access token with no permissions is useless, it can still be used to read public repositories and helps you avoid hitting API rate limits compared to not using an access token at all.
So, it's quite useful.

```yaml
apps:
  - name: suzuki-shunsuke/none
    client_id: xxx
```

With this setup, the access token is transparently switched depending on the working directory. What's written in the `.envrc` is the `GHTKN_APP`, not the access token itself, which is safe because it's not a secret.

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
It always runs the device flow to regenerate the token, regardless of any cached token.
Unlike `ghtkn get`, the device flow is always allowed even when `GHTKN_ENABLE_DEVICE_FLOW=false` or `device_flow.enable: false` in the configuration file.
Also unlike `ghtkn get`, it does not accept `-min-expiration (-m)`, nor read `GHTKN_MIN_EXPIRATION` or `min_expiration` in the configuration file.

## Disabling Device Flow

`ghtkn` obtains a GitHub App User access token via the OAuth Device Flow, which is interactive: it prints a one-time (user) code and waits for the user.
A coding agent (or any background / non-interactive process) cannot complete this, so it would block until a device code expires.
The device flow is enabled by default.
By setting `GHTKN_ENABLE_DEVICE_FLOW` to `false`, `ghtkn` will fail fast with an actionable error instead of blocking.

You can also disable it in the configuration file.

```yaml
device_flow:
  enable: false
```

```sh
ghtkn get -d
```

## :bulb: Copying a one-time code to clipboard automatically

[#446](https://github.com/suzuki-shunsuke/ghtkn/issues/446) [#309](https://github.com/suzuki-shunsuke/ghtkn/issues/309#issuecomment-4726483175)

> [!WARNING]
> Some applications, such as coding agents, cmux, and Warp, can start the device flow via ghtkn automatically. However, it is dangerous to use a one-time code when you didn't execute ghtkn explicitly, as this could be a phishing attack.
> An attacker could initiate the device flow, copy the one-time code to your clipboard, trick you into submitting it, and compromise your access token.
> To prevent this, if you use this tip, we recommend setting `GHTKN_ENABLE_DEVICE_FLOW` to `false` and always starting the device flow explicitly.

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

## Backend

By default ghtkn stores access tokens in the OS keyring, but you can change this.
This is useful in environments where the OS keyring is hard to use, such as containers and microVMs.
The following backends are supported:

- `keyring`: OS keyring (default)
- `text`: Store tokens as plaintext files
- `agent`: Store tokens encrypted via the ghtkn agent

For more details, see the [backend documentation](docs/backend.md).

## Disabling Browser Open

`version > 0.2.7` [#453](https://github.com/suzuki-shunsuke/ghtkn/issues/453)

By default, ghtkn opens the browser automatically for the device flow if commands such as `xdg-open` exist on PATH.
You can disable this behavior by setting the `GHTKN_OPEN_BROWSER` environment variable or `.open_browser.enable` in a configuration file to `false`.

```sh
export GHTKN_OPEN_BROWSER=false
```

```yaml
open_browser:
  enable: false
```

This is useful in environments where those commands exist on PATH but don't work.
For example, in WSL `xdg-open` exists but doesn't work.
In that case, please open the browser yourself.

## Configuration

Some settings can be configured via multiple sources.
The priority order of configuration sources is as follows:

1. command line arguments
2. environment variables
3. configuration files

## Using ghtkn in Enterprise Organizations

When using ghtkn in a company's GitHub Organization, it may not be practical for each developer to create their own GitHub App in organizations with a certain scale. In such cases, you can create a shared GitHub App and share the Client ID within the company.

User Access Tokens cannot generate tokens with permissions beyond what the user has, and users cannot impersonate others. API rate limits are also per-user.

Therefore, the risk of sharing within a limited internal space is considered to be low.

From a company's perspective, this can prevent the leakage of developers' PATs or GitHub CLI OAuth App access tokens that have access to the company's Organization. Even if a Client ID is leaked outside the company, it doesn't provide direct access to the company's Organization, and even if an access token is leaked, the risk can be minimized due to its short validity period (8 hours).

## Using personal access token for one-off operations

If the `GHTKN_GITHUB_TOKEN` environment variable is set, `ghtkn` will use it as the GitHub token.
This is useful when a personal access token is required due to [the limitations of user access tokens](#limitations).

## Go SDK

You can enable your CLI application to create GitHub User Access Tokens using [ghtkn Go SDK](pkg.go.dev/github.com/suzuki-shunsuke/ghtkn-go-sdk).
ghtkn itself uses this.
If SDK doesn't work well, please check if the version is latest.

## How does ghtkn work?

ghtkn gets and outputs an access token in the following way:

1. Read command line options and environment variables
2. Read a configuration file. It has pairs of app name and client id
3. [Determine the GitHub App](#using-multiple-apps)
4. Get the client id from the configuration file
5. Get the access token by client id from the backend
6. If the access token isn't found in the backend or the access token expires, [creating a new access token through Device Flow. A user need to input the verification code and approve the request](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app#using-the-device-flow-to-generate-a-user-access-token)
7. Get the authenticated user login by GitHub API for Git Credential Helper
8. Store the access token, expiration date, and authenticated user login in the backend
9. Output the access token

## How To Revoke Access Tokens

If an access token is leaked, it must be immediately invalidated.
[You can confirm if the leaked access token expires or not by GitHub API.](https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28#get-the-authenticated-user)

### `ghtkn revoke`

`version >= v0.2.7`

The simplest way is the `ghtkn revoke` command:

```sh
ghtkn revoke <app name>        # revoke the token stored for an app and delete it from the backend
ghtkn revoke ghu_xxx           # revoke a raw token directly (e.g. a leaked one)
ghtkn revoke ghu_a ghu_b foo   # revoke multiple tokens and an app's stored token at once
ghtkn revoke                   # revoke the token stored for GHTKN_APP or the default app
ghtkn revoke --all             # revoke the stored tokens of every app in the config
```

Each argument is classified by its prefix: arguments starting with a GitHub token prefix (`ghp_`, `github_pat_`, `gho_`, `ghu_`, `ghr_`) are revoked directly as raw access tokens, and all other arguments are treated as app names whose stored tokens are revoked and removed from the [backend](#backend).
When no argument is given, it falls back to `GHTKN_APP` or the default app; when only raw tokens are given, the fallback is not used, so revoking a raw token never touches an unrelated app's stored token.

The `--all` flag revokes the stored tokens of every app in the config at once. This is meant for incident response: when the environment running ghtkn is compromised, you can revoke all stored tokens immediately. With `--all`, app name arguments are ignored, but raw access tokens are still revoked.

### GitHub REST API

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

## Enabling the GitHub Account Picker

`version >= v0.2.7`

ghtkn skips GitHub's account picker by opening the authorization URL with the `skip_account_picker=true` query parameter.

https://github.com/login/device?skip_account_picker=true

Note that this query parameter is undocumented and may not be supported in the future.

Most users don't need to choose a different GitHub account.
However, if you do want to choose another account, set skip_account_picker: false in the configuration file.

~/.config/ghtkn/ghtkn.yaml

```yaml
skip_account_picker: false
```

## Comparison between GitHub App User Access Token and other access tokens

### GitHub CLI OAuth App access token

https://cli.github.com/manual/gh_auth_token

This can be easily generated with `gh auth login`, `gh auth token` in GitHub CLI.
You don't need to generate Personal Access Tokens, and it's convenient.
Also, when scopes across Users or Organizations are needed, it's difficult with non-Public GitHub Apps, but installing GitHub CLI OAuth App across multiple Users or Organizations solves such problems.

However, this access token is not very good from a security perspective.
While you can restrict the scope (permission) and target Organizations, these tend to be quite broad for convenience.
Also, it's basically indefinite.
Therefore, the risk when this token is leaked is very high.

So, a more secure mechanism is needed.

### fine-grained Personal Access Token

https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens

We'll ignore Legacy PAT as it's almost the same as OAuth App tokens.

Fine-grained access tokens have the following disadvantages compared to User Access Tokens:

- Regular rotation is cumbersome
- Management is cumbersome
- High risk when leaked
  - While the validity period is not indefinite, it tends to be quite long
    - Since short periods make rotation cumbersome, it tends to be 1 year or 6 months
    - Not on the order of a few hours

### GitHub App installation access token

https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation

- Pros
  - Can change permissions, repositories, and validity period when generating tokens
- Cons
  - Cannot operate as a User
    - e.g., PR creator becomes the App
  - Private Key management is cumbersome
  - High risk when Private Key is leaked
  
## :warning: Troubleshooting

### ghtkn doesn't work well

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

### The wrapper of `gh` command doesn't work well

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

### ghtkn returns an expired token (401)

If `ghtkn get` returns an expired token, you can renew it by running `ghtkn auth`.

```sh
ghtkn auth
```

### The device flow asks the verification code, but the code isn't shown anywhere

When ghtkn is run in the background process, the verification code is not displayed in the terminal.
In that case, you need to:

1. Cancel the process `A`
1. Run `ghtkn auth [app for process A]` manually to renew the access token
1. Re-run the process `A`

As of ghtkn v0.2.5, you can prevent this kind of issue by setting `GHTKN_ENABLE_DEVICE_FLOW` to false.

```sh
export GHTKN_ENABLE_DEVICE_FLOW=false
```

When the token expires, you need to run `ghtkn auth` to renew it.

### A browser opens when using tools like cmux and warp

When using [cmux](https://github.com/manaflow-ai/cmux) and [warp](https://github.com/warpdotdev/warp), ghtkn may open a browser on its own.
Worse, the one-time code isn't shown anywhere, so you can't complete the device flow and have to close the browser tab.

As of ghtkn v0.2.5, you can prevent this kind of issue by setting `GHTKN_ENABLE_DEVICE_FLOW` to false.

```sh
export GHTKN_ENABLE_DEVICE_FLOW=false
```

When the token expires, you need to run `ghtkn auth` to renew it.

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

## :memo: Note

### API rate limit

https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#primary-rate-limit-for-github-app-installations

> Primary rate limits for GitHub App user access tokens (as opposed to installation access tokens) are dictated by the primary rate limits for the authenticated user.
> This rate limit is combined with any requests that another GitHub App or OAuth app makes on that user's behalf and any requests that the user makes with a personal access token.
> For more information, see Rate limits for the REST API.

The rate limit for authenticated users is 5,000 per hour, so it should be fine for normal use.

> All of these requests count towards your personal rate limit of 5,000 requests per hour.
