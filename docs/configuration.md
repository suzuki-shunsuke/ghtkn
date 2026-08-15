---
description: Configure ghtkn via config file, environment variables, and CLI args. Use for configuration priority, disabling browser open, the GitHub account picker, enterprise GitHub App sharing, GHTKN_GITHUB_TOKEN, or shell completion.
---

# Configuration

Some settings can be configured via multiple sources.
The priority order of configuration sources is as follows:

1. command line arguments
2. environment variables
3. configuration files

## Shell Completion

ghtkn can output a shell completion script for bash, zsh, fish, and PowerShell.
The setup differs per shell, so run the following command and follow the instructions it prints for your shell:

```sh
ghtkn help completion
```

Once it is set up, `ghtkn get`, `ghtkn auth`, and `ghtkn revoke` complete their app name argument with the apps in your configuration file, so you don't have to remember the names:

```console
$ ghtkn get <TAB>
my-app  work-app
```

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

## Using ghtkn in Enterprise Organizations

When using ghtkn in a company's GitHub Organization, it may not be practical for each developer to create their own GitHub App in organizations with a certain scale. In such cases, you can create a shared GitHub App and share the Client ID within the company.

User Access Tokens cannot generate tokens with permissions beyond what the user has, and users cannot impersonate others. API rate limits are also per-user.

Therefore, the risk of sharing within a limited internal space is considered to be low.

From a company's perspective, this can prevent the leakage of developers' PATs or GitHub CLI OAuth App access tokens that have access to the company's Organization. Even if a Client ID is leaked outside the company, it doesn't provide direct access to the company's Organization, and even if an access token is leaked, the risk can be minimized due to its short validity period (8 hours).

A shared App owned by an enterprise is often installed on several Organizations. One app entry covers all of them: list the Organizations in `.apps[].git_owners` so the [Git Credential Helper](git-credential-helper.md) picks that app for each of them.

```yaml
apps:
  - name: my-company
    client_id: xxx
    git_owners:
      - my-company
      - my-company-sandbox
```

## Using personal access token for one-off operations

If the `GHTKN_GITHUB_TOKEN` environment variable is set, `ghtkn` will use it as the GitHub token.
This is useful when a personal access token is required due to the limitations of user access tokens (see [Troubleshooting](troubleshooting.md)).

`ghtkn auth` ignores `GHTKN_GITHUB_TOKEN` and authenticates as usual, since caching a GitHub App user access token is its whole job.

## JSON Schema

The configuration file has a JSON Schema, so editors such as VSCode can complete the settings and warn about invalid ones.

- [ghtkn.json](../json-schema/ghtkn.json)
- https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/refs/heads/main/json-schema/ghtkn.json

Add a `yaml-language-server` comment to the top of the configuration file. `ghtkn init` writes this comment for you.

Version: `main`

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/main/json-schema/ghtkn.json
```

Or pinning version:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/suzuki-shunsuke/ghtkn/v0.3.6/json-schema/ghtkn.json
```

As of v0.4.0, ghtkn also has a `json-schema` command that outputs the schema embedded in the binary, which is the schema of the configuration that version accepts:

```sh
ghtkn json-schema
```

## Disabling the GitHub Account Picker

`version >= v0.2.7`

ghtkn skips GitHub's account picker by opening the authorization URL with the `skip_account_picker=true` query parameter.

https://github.com/login/device?skip_account_picker=true

Note that this query parameter is undocumented and may not be supported in the future.

Most users don't need to choose a different GitHub account.
However, if you do want to choose another account, set `skip_account_picker: false` in the configuration file.

~/.config/ghtkn/ghtkn.yaml

```yaml
skip_account_picker: false
```
