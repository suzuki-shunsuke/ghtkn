# Codex Sandbox Configuration

[Codex](https://developers.openai.com/codex/security) runs commands inside an OS-level sandbox that restricts filesystem and network access.
ghtkn needs extra settings inside it with every backend, except for reading a cached token with the `text` backend.
This page describes the minimum settings ghtkn needs, per backend.

> [!NOTE]
> Verified against Codex 0.145.0 on macOS by running `codex sandbox ghtkn agent status` under each configuration.
> Named permission profiles are still labelled beta in Codex's documentation, so check the settings here against your own version.
> Codex reads the configuration at startup, so restart it after editing.

The permissions ghtkn needs depend on the backend.

Whatever consumes the access token ghtkn returns, such as `gh`, `git`, or your own script, calls GitHub on its own, so it needs `github.com` and `api.github.com` allowed as well, whichever backend ghtkn uses.
Making ghtkn work inside the sandbox doesn't make those commands work.

Write settings like the following in `~/.codex/config.toml`, or in your project's `.codex/config.toml`.
A project's `.codex/config.toml` is ignored unless the project is trusted.

## agent backend

Run the agent itself outside the sandbox.
Storing and deleting access tokens, and the API calls for the device flow, refresh, and revoke, all happen in the agent, so the sandbox doesn't need to allow any of them.
In the sandbox, allow access to the socket.

```toml
default_permissions = "ghtkn"
features.network_proxy = true

[permissions.ghtkn]
extends = ":workspace"
description = "Workspace profile plus the ghtkn agent socket."

[permissions.ghtkn.network]
enabled = true

[permissions.ghtkn.network.unix_sockets]
"/Users/alice/.cache/ghtkn/agent.sock" = "allow" # Change this to your own path

# ghtkn itself needs no domains with this backend, but whatever consumes the token does
# [permissions.ghtkn.network.domains]
# "github.com" = "allow" # For git over HTTPS
# "api.github.com" = "allow" # For the GitHub API
```

Allow the path the agent actually uses (see [Socket path](../ghtkn-backend/reference.md#socket-path)).

Use an absolute path.
Unlike `filesystem` entries, `~` is not expanded here, and a `~` or relative entry fails at startup rather than being ignored:

```console
Error: failed to start managed network proxy: failed to build network proxy state: invalid network.allow_unix_sockets[0]
```

This assumes the agent is already running and unlocked outside the sandbox.
The agent starts locked, so start and unlock it before running Codex:

```sh
ghtkn agent start
ghtkn agent unlock
```

`ghtkn agent unlock` prompts for the passphrase, so run it in your own shell rather than through a coding agent.

## keyring backend

The settings differ per OS.

### macOS

Enable the profile's network policy.

```toml
default_permissions = "ghtkn"
features.network_proxy = true

[permissions.ghtkn]
extends = ":workspace"
description = "Workspace profile plus macOS Keychain access for ghtkn."

[permissions.ghtkn.network]
enabled = true

# To also allow storing and deleting access tokens, write access to the keychain directory is needed.
# [permissions.ghtkn.filesystem]
# "~/Library/Keychains" = "write"

# [permissions.ghtkn.network.domains]
# "github.com" = "allow" # To allow the device flow, and for git over HTTPS
# "api.github.com" = "allow" # To allow revoke, and for whatever consumes the token
```

### Linux

Not verified, as there is no environment to test it in.
It talks to the Secret Service over the D-Bus session bus, which is a Unix socket.

### Windows

Not verified, as there is no environment to test it in.
Access to the Windows credential store needs to be allowed.

## text backend

No settings are needed if you only read access tokens.
The sandbox doesn't restrict reads, so the token is readable whatever path it is stored at.
Settings are needed to allow storing an access token, revoke, or the device flow.

```toml
default_permissions = "ghtkn"

# To allow revoke or the device flow
# features.network_proxy = true
#
# [permissions.ghtkn.network]
# enabled = true

[permissions.ghtkn]
extends = ":workspace"
description = "Workspace profile plus ghtkn text backend access."

# To allow storing an access token or revoke
# Needed even for the default path, as long as it is outside the workspace
# [permissions.ghtkn.filesystem]
# "~/.cache/ghtkn/tokens" = "write"

# [permissions.ghtkn.network.domains]
# "github.com" = "allow" # To allow the device flow, and for git over HTTPS
# "api.github.com" = "allow" # To allow revoke, and for whatever consumes the token
```

Allow the path the backend actually resolves (see [text Backend](../ghtkn-backend/reference.md#text-backend)).
Allow the directory, not the file: the text backend writes a token by creating a temporary file next to it and renaming it into place.
