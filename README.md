# ghtkn (GH-Token)

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/suzuki-shunsuke/ghtkn)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/main/LICENSE) | [Install](skills/ghtkn-install/reference.md) | [Usage](USAGE.md)

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

ghtkn (pronounced `G-H Token`) allows you to manage multiple GitHub Apps through configuration files and securely store tokens using OS keyring (Windows Credential Manager, macOS Keychain, or GNOME Keyring) or another backend.

## :rocket: Getting Started

1. [Install ghtkn](skills/ghtkn-install/reference.md)
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

4. Run `ghtkn auth` for authentication

```sh
ghtkn auth
```

https://github.com/login/device will open in your browser, so enter the code displayed in the terminal and approve it.

With Device Flow, access tokens cannot be generated in non-interactive environments like CI.
ghtkn is primarily intended for local development.

You can close the opened tab.

5. Run `ghtkn get` to get a user access token

```sh
ghtkn get
```

A user access token starting with `ghu_` is outputted.

6. Run `gh issue create` using the access token

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

## Documentation and skills

Detailed documentation is split by topic. Each topic lives in a skill directory under [`skills/`](skills) and contains an agent-facing `SKILL.md` and a shared reference document (`reference.md`). The reference documents below are the single source of truth, shared between this README and the skills, so there's no duplicated maintenance.

The skills can be installed with skill installers such as [`gh skill install`](https://cli.github.com/manual/gh_skill_install) or [`npx skills`](https://github.com/vercel-labs/skills), e.g. `gh skill install suzuki-shunsuke/ghtkn ghtkn-backend`.

- [Install](skills/ghtkn-install/reference.md) - install the ghtkn CLI and verify release assets.
- [Git Credential Helper](skills/ghtkn-git-credential-helper/reference.md) - use ghtkn as a Git credential helper and switch apps by repository owner.
- [Using Multiple Apps](skills/ghtkn-multiple-apps/reference.md) - configure multiple GitHub Apps and switch between them per command, env var, or directory.
- [Token Management](skills/ghtkn-token-management/reference.md) - token regeneration, `ghtkn auth`, the automatic device flow, and clipboard.
- [Backend](skills/ghtkn-backend/reference.md) - where tokens are stored (`keyring`, `text`, `agent`); useful for containers and microVMs.
- [Configuration](skills/ghtkn-configuration/reference.md) - configuration priority, browser open, account picker, enterprise sharing, and one-off PAT use.
- [Design](skills/ghtkn-design/reference.md) - how ghtkn works, a comparison with other access tokens, and API rate limits.
- [How To Revoke Access Tokens](skills/ghtkn-revoke-tokens/reference.md) - invalidate leaked or compromised tokens.
- [Troubleshooting](skills/ghtkn-troubleshooting/reference.md) - diagnosing problems and known limitations.

## Go SDK

You can enable your CLI application to create GitHub User Access Tokens using [ghtkn Go SDK](pkg.go.dev/github.com/suzuki-shunsuke/ghtkn-go-sdk).
ghtkn itself uses this.
If SDK doesn't work well, please check if the version is latest.
