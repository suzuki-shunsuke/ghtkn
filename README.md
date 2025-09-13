# ghtkn (GH-Token)

[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/main/LICENSE) | [Install](INSTALL.md) | [Usage](USAGE.md)

**Stop risking token leaks - Use secure, short-lived GitHub tokens for local development**

## ⚠️ The Security Problem

Are you still using Personal Access Tokens (PATs) or GitHub CLI OAuth tokens stored on your local machine? These long-lived tokens pose **significant security risks**:
- **Indefinite or months-long validity** - A leaked token remains dangerous for extended periods
- **Broad permissions** - Often configured with wide access for convenience
- **Difficult to rotate** - Manual management leads to tokens being used far longer than they should

## ✅ The ghtkn Solution

ghtkn generates **8-hour User Access Tokens** from GitHub Apps using Device Flow - a fundamentally more secure approach:
- **Short-lived tokens** - Only 8 hours validity minimizes damage from any potential leak
- **No secrets required** - Only needs a Client ID (which isn't secret), no Private Keys or Client Secrets
- **User-attributed actions** - Operations are performed as you, not as an app
- **Automatic token management** - Integrates with OS keychains for secure storage and reuse

ghtkn (pronounced `GH-Token`) allows you to manage multiple GitHub Apps through configuration files and securely store tokens using Windows Credential Manager, macOS Keychain, or GNOME Keyring.

> [!NOTE]
> In this document, we call Windows Credential Manger, macOS KeyChain, and GNOME Keyring as secret manager.

## Requirements

A secret manager is required.

## :rocket: Getting Started

1. [Install ghtkn](INSTALL.md)
2. Create a GitHub App

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
persist: true
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

If you run the same command immediately, it will now run without the authorization flow because ghtkn stores access tokens into the secret manager and reuse them.

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

e.g. ~/bin/gh:

```sh
#!/usr/bin/env bash

set -eu

GH_TOKEN="$(ghtkn get)" 
export GH_TOKEN
exec /opt/homebrew/bin/gh "$@" # Specify the absolute path to avoid infinite loop
```

If the command is managed by [aqua](https://aquaproj.github.io/), `aqua exec` is useful:

```sh
exec aqua exec -- gh "$@"
```

2. Make scripts executable

```sh
chmod +x ~/bin/gh
```

It's useful to wrap `gh` using shell script as gh always requires GitHub access tokens.

## Git Credential Helper

ghtkn >= v0.1.2

You can use ghtkn as a [Git Credential Helper](https://git-scm.com/book/en/v2/Git-Tools-Credential-Storage):

```sh
git config --global credential.helper '!ghtkn git-credential'
```

```ini
[credential]
	helper = !ghtkn git-credential
```

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

To solve the problem, please comment out the system setting.

```sh
sudo vi /Library/Developer/CommandLineTools/usr/share/git-core/gitconfig
```

```ini
# [credential]
# 	helper = osxkeychain
```

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

ghtkn stores generated access tokens and their expiration dates in the secret manager.
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

If you're only using the GitHub CLI to call an API, it usually finishes in an instant, so you probably won't need to set this.
However, if you're passing the access token to a script that takes, say, 30 minutes to run, setting it to something like `50m` will prevent the token from expiring in the middle of the script.

By the way, if you set the value to 8 hours or more, you can replicate how ghtkn regenerates the access token.
This could be useful if you want to test how `ghtkn` behaves.

## Using ghtkn in Enterprise Organizations

When using ghtkn in a company's GitHub Organization, it may not be practical for each developer to create their own GitHub App in organizations with a certain scale. In such cases, you can create a shared GitHub App and share the Client ID within the company.

User Access Tokens cannot generate tokens with permissions beyond what the user has, and users cannot impersonate others. API rate limits are also per-user.

Therefore, the risk of sharing within a limited internal space is considered to be low.

From a company's perspective, this can prevent the leakage of developers' PATs or GitHub CLI OAuth App access tokens that have access to the company's Organization. Even if a Client ID is leaked outside the company, it doesn't provide direct access to the company's Organization, and even if an access token is leaked, the risk can be minimized due to its short validity period (8 hours).

## Environment Variables

All environment variables are optional.

- GHTKN_LOG_LEVEL: Log level. One of `debug`, `info` (default), `warn`, `error`.
- GHTKN_OUTPUT_FORMAT: The output format of `ghtkn get` command
  - `json`: JSON Format
- GHTKN_APP: The app identifier to get an access token
- GHTKN_MIN_EXPIRATION: The minimum expiration duration of access token. If `ghtkn get` gets the access token from keying but the expiration duration is shorter than the minimum expiratino duration, `ghtkn get` creates a new access token via Device Flow
- GHTKN_CONFIG: The configuration file path
- XDG_CONFIG_HOME

## Go SDK

You can enable your CLI application to create GitHub User Access Tokens using [ghtkn Go SDK](pkg.go.dev/github.com/suzuki-shunsuke/ghtkn-go-sdk).
ghtkn itself uses this.

## How does ghtkn work?

ghtkn gets and outputs an access token in the following way:

1. Read command line options and environment variables
2. Read a configuration file. It has pairs of app name and client id
3. [Determine the GitHub App](#using-multiple-apps)
4. Get the client id from the configuration file
5. Get the access token by client id from the keyring
6. If the access token isn't found in the keyring or the access token expires, [creating a new access token through Device Flow. A user need to input the device code and approve the request](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app#using-the-device-flow-to-generate-a-user-access-token)
7. Get the authenticated user login by GitHub API for Git Credential Helper
8. Store the access token, expiration date, and authenticated user login in the keyring
9. Output the access token

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

## :memo: Note

### API rate limit

https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#primary-rate-limit-for-github-app-installations

> Primary rate limits for GitHub App user access tokens (as opposed to installation access tokens) are dictated by the primary rate limits for the authenticated user.
> This rate limit is combined with any requests that another GitHub App or OAuth app makes on that user's behalf and any requests that the user makes with a personal access token.
> For more information, see Rate limits for the REST API.

The rate limit for authenticated users is 5,000 per hour, so it should be fine for normal use.

> All of these requests count towards your personal rate limit of 5,000 requests per hour.

### Limitation

ghtkn doesn't support some operations that require Client Secrets as the risk of Client Secret leakage is high:

- [Refresh Token via GitHub API](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/refreshing-user-access-tokens#refreshing-a-user-access-token-with-a-refresh-token)
- [Revoke Access Tokens via GitHub API](https://docs.github.com/en/rest/apps/oauth-applications?apiVersion=2022-11-28#delete-an-app-token)

Instead of refreshing a token, ghtkn regenerates the token through Device Flow.
While you can't revoke a token directly with `ghtkn`, if you absolutely need to, you can either go to the GitHub App settings page and select **"Revoke all user tokens"** or temporarily generate a client secret and [use the API to revoke the token](https://docs.github.com/en/rest/apps/oauth-applications?apiVersion=2022-11-28#delete-an-app-token).

## LICENSE

[MIT](LICENSE)
