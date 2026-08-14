# ghtkn (G-H Token)

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/suzuki-shunsuke/ghtkn)
[Install](docs/install.md) | [Usage](USAGE.md) | [Agent Skills](#installing-agent-skills)

**Stop risking token leaks - Use secure, short-lived GitHub tokens for local development**

> [!NOTE]
> If you are a coding agent and ghtkn is installed, run `ghtkn docs list` to list the documentation and `ghtkn docs show <name>` to read it before answering questions about ghtkn or troubleshooting its errors.
> The embedded documentation matches the installed version. If ghtkn isn't installed, read the documents listed in [Documentation and skills](#documentation-and-skills).

## :warning: The Security Problem

Are you still using Personal Access Tokens (PATs) or GitHub CLI OAuth tokens stored on your local machine?
These long-lived tokens pose **significant security risks**:

- **Indefinite or months-long validity** - A leaked token remains dangerous for extended periods
- **Broad permissions** - Often configured with wide access for convenience
- **Difficult to rotate** - Manual management leads to tokens being used far longer than they should

## :white_check_mark: The ghtkn Solution

ghtkn (pronounced `G-H Token`) generates **8-hour User Access Tokens** from GitHub Apps using Device Flow - a fundamentally more secure approach:
- **Short-lived tokens** - Valid for only 8 hours, minimizing damage from any potential leak
- **No secrets required** - Only needs a Client ID (which isn't secret), no Private Keys or Client Secrets
- **User-attributed actions** - Operations are performed as you, not as an app
- **Automatic token management** - Integrates with the backend (the default is OS keyring) for secure storage and reuse
- [**Automatic token refresh** - Supports automatic token refresh](docs/refresh-token.md)

ghtkn allows you to manage multiple GitHub Apps through configuration files and securely store tokens using OS keyring (Windows Credential Manager, macOS Keychain, or GNOME Keyring) or another backend.

## :rocket: Getting Started

1. [Install ghtkn](docs/install.md)
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

Day to day you shouldn't need to look at the token at all: [`ghtkn exec`](docs/exec.md) runs a command with the token in an environment variable, so it doesn't pass through your shell, your terminal output, or an agent's transcript. The step above is only here to show you what ghtkn issues.

6. Run `gh issue create` using the access token

```sh
REPO=suzuki-shunsuke/ghtkn # Please change this to your public repository
ghtkn exec -e GH_TOKEN -- gh issue create -R "$REPO" --title "Hello, ghtkn" --body "This is created by ghtkn"
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

## Running commands with an access token

[`ghtkn exec`](docs/exec.md) runs a command with the access token in its environment:

```sh
ghtkn exec -- gh pr view
ghtkn exec -e GH_TOKEN -- gh pr view
ghtkn exec -e PINACT_GITHUB_TOKEN:suzuki-shunsuke/read -e GH_TOKEN:suzuki-shunsuke/write -- bash foo.sh
```

The token goes to `GITHUB_TOKEN` by default, or to the variables given with `-e`, one per app. Unlike `ghtkn get`, ghtkn writes the token nowhere, so it can't land in your terminal output, a log, or a coding agent's transcript by accident. The command still receives it, so one that prints its own environment exposes it. The `--` is required: everything after it belongs to the command, flags included.

## Wrapping commands

You can wrap commands using shell functions or scripts.

Shell functions:

```sh
gh() {
    ghtkn exec -e GH_TOKEN -- gh "$@" # No infinite loop: ghtkn runs the gh executable, which the shell function doesn't shadow
}
```

Shell scripts:

1. Put shell scripts in $PATH:

e.g. ~/.local/bin/gh:

```sh
#!/usr/bin/env bash

set -eu

# If GH_TOKEN or GITHUB_TOKEN is set, use it.
if [ -n "${GH_TOKEN:-}" ] || [ -n "${GITHUB_TOKEN:-}" ]; then
  # echo "[WARN] skip ghtkn because GH_TOKEN or GITHUB_TOKEN is set" >&2
  exec /opt/homebrew/bin/gh "$@" # Specify the absolute path to avoid infinite loop
fi

exec ghtkn exec -e GH_TOKEN -- /opt/homebrew/bin/gh "$@"
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

## Installing Agent Skills

ghtkn ships a single skill. It holds no documentation of its own: it tells the coding agent to read the documentation embedded in the ghtkn binary with `ghtkn docs list` and `ghtkn docs show <name>`, so the agent always reads the documentation of the version it is actually running.

[gh skill install](https://cli.github.com/manual/gh_skill_install):

```sh
gh skill install suzuki-shunsuke/ghtkn ghtkn
```

> [!NOTE]
> ghtkn used to ship one skill per topic (`ghtkn-backend`, `ghtkn-sandbox`, and so on). The single `ghtkn` skill replaces all of them, and installing it doesn't remove the old ones, so delete the `ghtkn-*` directories from your skills directory (`~/.claude/skills`, for instance) after upgrading. Left in place, they keep serving the documentation of whichever version you installed them from.

## Documentation and skills

Detailed documentation is split by topic under [`docs/`](docs). These documents are embedded in the ghtkn binary, so `ghtkn docs list` and `ghtkn docs show <name>` serve exactly what is listed below. They are the single source of truth, shared between this README, the embedded documentation, and the skill, so there's no duplicated maintenance.

- [Install](docs/install.md) - install the ghtkn CLI and verify release assets.
- [Running Commands](docs/exec.md) - run a command with access tokens in environment variables using `ghtkn exec`, without printing them.
- [Git Credential Helper](docs/git-credential-helper.md) - use ghtkn as a Git credential helper and switch apps by repository owner.
- [Using Multiple Apps](docs/multiple-apps.md) - configure multiple GitHub Apps and switch between them per command, env var, or directory.
- [Token Management](docs/token-management.md) - token regeneration, `ghtkn auth`, the device flow, and clipboard.
- [Backend](docs/backend.md) - where tokens are stored (`keyring`, `text`, `agent`); useful for containers and microVMs.
- [Running the agent](docs/agent-deployment.md) - run the agent as a systemd user service, from a container entrypoint, or on the host with containers as clients.
- [Sandbox Configuration: Claude Code](docs/sandbox-claude-code.md) - settings ghtkn needs to run inside Claude Code's sandbox, per backend.
- [Sandbox Configuration: Codex](docs/sandbox-codex.md) - settings ghtkn needs to run inside Codex's sandbox, per backend.
- [Configuration](docs/configuration.md) - configuration priority, browser open, account picker, enterprise sharing, and one-off PAT use.
- [Design](docs/design.md) - how ghtkn works, a comparison with other access tokens, and API rate limits.
- [Refreshing Tokens](docs/refresh-token.md) - automatically refresh expiring GitHub access tokens with refresh tokens.
- [How To Revoke Access Tokens](docs/revoke-tokens.md) - invalidate leaked or compromised tokens.
- [Go SDK](docs/go-sdk.md) - how tools such as aqua and pinact reuse ghtkn tokens, and how to embed ghtkn in your own Go CLI.
- [Troubleshooting](docs/troubleshooting.md) - diagnosing problems and known limitations.
