# Codex Sandbox Configuration

This guide describes the minimum Codex configuration needed to use `ghtkn`
from Codex. The examples assume a project-local `.codex/config.toml`.

Codex configuration uses TOML and snake_case keys. It is not the same syntax as
Claude Code's `settings.json`.

## Baseline: allow the `ghtkn` command

Codex command approval is separate from filesystem and network sandboxing.
If you want Codex to run `ghtkn` without asking every time, allow the command
prefix:

```toml
[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

You can allow narrower prefixes such as `ghtkn docs`, but a single `ghtkn`
prefix is usually more practical. It covers `ghtkn get`, `ghtkn docs`,
`ghtkn info`, version commands, and troubleshooting commands.

The token printed by `ghtkn get` is a secret. Codex must not print, echo, log,
or include it in a response. Consume it immediately, for example by passing it
to another command as an environment variable:

```sh
GH_TOKEN=$(ghtkn get) gh issue list
```

## text backend

For normal use with the `text` backend, no extra Codex filesystem or network
permission is needed after the token is cached.

Recommended workflow:

1. Configure `ghtkn` to use the `text` backend.
2. Run `ghtkn auth` in your own interactive terminal, outside Codex.
3. Let Codex run `ghtkn get` to read the cached token.

Minimum project config:

```toml
[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

The text backend stores tokens in this order:

1. `$GHTKN_TEXT_BACKEND_DIR`
2. `$XDG_CACHE_HOME/ghtkn/tokens`
3. `$HOME/.cache/ghtkn/tokens`

If your Codex read policy denies the resolved token directory, re-allow that
directory with a custom permission profile:

```toml
default_permissions = "ghtkn_text"

[permissions.ghtkn_text]
extends = ":workspace"
description = "Workspace profile plus ghtkn text backend token cache reads."

[permissions.ghtkn_text.filesystem]
"/Users/alice/.cache/ghtkn/tokens" = "read"

[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

Use the actual resolved token directory, especially if `GHTKN_TEXT_BACKEND_DIR`
or `XDG_CACHE_HOME` is set.

### Writing text backend tokens

Writing is not needed for normal cached-token use. It is needed if a sandboxed
`ghtkn` command creates, refreshes, or deletes a text-backend token.

Allow the directory, not an individual token file. The text backend writes a
temporary file in the token directory and renames it into place.

```toml
default_permissions = "ghtkn_text"

[permissions.ghtkn_text]
extends = ":workspace"
description = "Workspace profile plus ghtkn text backend token cache writes."

[permissions.ghtkn_text.filesystem]
"/Users/alice/.cache/ghtkn/tokens" = "write"

[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

Creating a new access token from inside the sandbox also reaches GitHub. On
macOS, Go-based CLIs can hit platform TLS verification failures inside a
sandbox. Prefer running `ghtkn auth` in your own interactive terminal and keep
Codex on cached-token reads.

## agent backend

With the `agent` backend, the minimum configuration depends on where the agent
process runs.

If the `ghtkn` agent runs outside Codex, Codex only needs permission to connect
to the agent socket. The agent owns token storage, refresh, and GitHub network
access.

```toml
[network]
allow_unix_sockets = ["/Users/alice/.cache/ghtkn/agent.sock"]

[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

Allow the socket path that `ghtkn` actually resolves:

1. `$GHTKN_AGENT_SOCKET`
2. `$XDG_RUNTIME_DIR/ghtkn/agent.sock`
3. `$XDG_CACHE_HOME/ghtkn/agent.sock`
4. `$HOME/.cache/ghtkn/agent.sock`

Run and unlock the agent in your own terminal:

```sh
ghtkn agent start
ghtkn agent unlock
```

If the agent itself runs inside Codex's sandbox, token minting and refresh also
need GitHub network access from inside the sandbox. That is a broader setup than
the usual host-agent pattern.

## keyring backend

The keyring backend is platform-specific. The minimum permission depends on how
the OS keyring is exposed to sandboxed processes.

### macOS

On macOS, cached-token reads need access to the Keychain service. With Codex
0.145.0, enable the profile's network policy so its curated macOS policy allows
that service.

Enabling the network policy alone can also allow outbound traffic. Enable the
managed network proxy and allow only a reserved `.invalid` hostname to keep
real public destinations blocked.

Minimum project config:

```toml
default_permissions = "ghtkn_keyring_macos"
features.network_proxy = true

[permissions.ghtkn_keyring_macos]
extends = ":workspace"
description = "Workspace profile plus macOS Keychain service access for ghtkn."

[permissions.ghtkn_keyring_macos.network]
enabled = true

[permissions.ghtkn_keyring_macos.network.domains]
"ghtkn-keyring.invalid" = "allow"

[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

No explicit read permission for `~/Library/Keychains` is needed for cached-token
reads. See [SURVEY.md](SURVEY.md) for the investigation, implementation
references, and validation results.

Creating, refreshing, or deleting a token in the keychain needs write access to
the keychain directory. This is broader than `ghtkn`: every sandboxed command in
that permission profile can modify the login keychain.

```toml
default_permissions = "ghtkn_keyring_macos"
features.network_proxy = true

[permissions.ghtkn_keyring_macos]
extends = ":workspace"
description = "Workspace profile plus macOS Keychain writes for ghtkn keyring backend."

[permissions.ghtkn_keyring_macos.network]
enabled = true

[permissions.ghtkn_keyring_macos.network.domains]
"github.com" = "allow"

[permissions.ghtkn_keyring_macos.filesystem]
"/Users/alice/Library/Keychains" = "write"

[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

Prefer running `ghtkn auth` in your own terminal and letting Codex read cached
tokens. That keeps Keychain writes and the interactive device flow outside the
sandbox. If you intentionally run `ghtkn auth` inside Codex, also enable the
managed network proxy feature and allow `github.com` as shown above.

### Linux

On Linux, keyring implementations commonly use Secret Service over the D-Bus
session bus, which is a Unix socket. If Codex's Linux sandbox blocks Unix
sockets, even cached-token reads can fail.

Do not assume the macOS settings apply. Prefer the `agent` backend or the `text`
backend in Linux sandboxes, containers, and microVMs unless you have verified
the exact Unix-socket policy you want to allow.

### Windows

On Windows, the keyring backend uses the Windows credential store. Cached-token
reads require access to the credential APIs, and token creation or deletion
requires write access to that store.

This guide does not currently give an exact Windows Codex minimum because it
needs to be validated on Windows. Document Windows separately once the relevant
Codex sandbox controls are confirmed.

## GitHub network access

`ghtkn get` does not always need GitHub network access:

- `text` / `keyring`: reading a valid cached token is local.
- `agent`: if the agent runs outside Codex, the client talks only to the socket.

GitHub network access is needed when the sandboxed process itself reaches
GitHub, including:

- `ghtkn auth` or device-flow token creation with `text` / `keyring`
- `ghtkn revoke`
- an agent running inside Codex that refreshes access tokens
- `gh`, `git`, or other tools that use the token to call GitHub

Codex accepts domain allowlist configuration in the network section:

```toml
[network]
domains = [{ allow = "github.com" }, { allow = "api.github.com" }]
```

This only covers Codex's network policy. On macOS, Go-based CLIs can also need
access to the system trust service for TLS verification. If that becomes the
blocking issue, prefer running the interactive or incident-response command
outside Codex, or use an explicit one-off approval rather than adding broad
network weakening to the normal setup.

## Revoke and incident response

`ghtkn revoke` is not required for normal use. Document it separately from the
normal setup.

`ghtkn revoke` does two things:

1. Calls GitHub's credential revocation API.
2. Removes the stored token from the backend when revoking an app's cached token.

For the `text` backend, deleting the stored token requires write access to the
text token directory:

```toml
default_permissions = "ghtkn_text"

[permissions.ghtkn_text]
extends = ":workspace"
description = "Workspace profile plus ghtkn text backend token cache writes."

[permissions.ghtkn_text.filesystem]
"/Users/alice/.cache/ghtkn/tokens" = "write"

[network]
domains = [{ allow = "github.com" }, { allow = "api.github.com" }]

[[rules.prefix_rules]]
pattern = [{ token = "ghtkn" }]
decision = "allow"
justification = "Allow ghtkn commands; access tokens must not be printed or logged."
```

`ghtkn revoke --all` is for incident response and can invalidate every stored
token.

For the `keyring` backend, deleting the stored token requires write access to
the OS keyring. On macOS that means the `~/Library/Keychains` write permission
shown in the keyring section. It is reasonable to keep revoke workflows out of
the default Codex setup and run them deliberately when needed.

## Validation

Restart Codex after changing `.codex/config.toml`. Existing Codex sessions can
keep the old permission profile, so a `ghtkn get` failure in the same session
does not prove the new profile is wrong.

Check the effective Codex configuration:

```sh
codex doctor --summary --ascii
```

The `Configuration` section should report `config loaded`. Other doctor
failures, such as provider reachability or local state database issues, are
separate from whether this TOML parsed successfully.

Then verify `ghtkn` without printing the token:

```sh
ghtkn info
ghtkn get >/dev/null
```

If `ghtkn get >/dev/null` tries to start Device Flow and fails because Device
Flow is disabled, Codex did not find a valid cached token for the current
`ghtkn` app and backend. Run `ghtkn auth` in your own terminal with the same
configuration, especially the same `GHTKN_APP` or app argument, then retry the
non-printing check. If the same `ghtkn get` succeeds in a normal shell but
fails only in Codex, check that Codex was restarted after enabling the
profile's network policy.

On macOS keyring backend, `ghtkn` stores the token with service
`github.com/suzuki-shunsuke/ghtkn` and account set to the GitHub App Client ID,
not the app name. To test item access without printing the item or secret:

```sh
if security find-generic-password \
  -s github.com/suzuki-shunsuke/ghtkn \
  -a <client-id> >/dev/null 2>&1; then
  echo accessible
else
  echo unavailable
fi
```

Do not add `-w` or `-wa` when debugging in an agent session; those options
print the stored secret.
