# Configuration

Some settings can be configured via multiple sources.
The priority order of configuration sources is as follows:

1. command line arguments
2. environment variables
3. configuration files

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

## Using ghtkn in Enterprise Organizations

When using ghtkn in a company's GitHub Organization, it may not be practical for each developer to create their own GitHub App in organizations with a certain scale. In such cases, you can create a shared GitHub App and share the Client ID within the company.

User Access Tokens cannot generate tokens with permissions beyond what the user has, and users cannot impersonate others. API rate limits are also per-user.

Therefore, the risk of sharing within a limited internal space is considered to be low.

From a company's perspective, this can prevent the leakage of developers' PATs or GitHub CLI OAuth App access tokens that have access to the company's Organization. Even if a Client ID is leaked outside the company, it doesn't provide direct access to the company's Organization, and even if an access token is leaked, the risk can be minimized due to its short validity period (8 hours).

## Using personal access token for one-off operations

If the `GHTKN_GITHUB_TOKEN` environment variable is set, `ghtkn` will use it as the GitHub token.
This is useful when a personal access token is required due to the limitations of user access tokens (see the troubleshooting reference).
