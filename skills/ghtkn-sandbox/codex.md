# Codex Sandbox Configuration

[Codex](https://developers.openai.com/codex/security) runs commands inside an OS-level sandbox that restricts filesystem and network access.
ghtkn doesn't work inside it with the default settings.
This page describes the minimum settings ghtkn needs, per backend.

> [!NOTE]
> Verified against Codex 0.145.0 on macOS by running `codex sandbox ghtkn agent status` under each configuration.
> Named permission profiles are still labelled beta in Codex's documentation, so check the settings here against your own version.
> Codex reads the configuration at startup, so restart it after editing.

## Where the settings go

User-level settings go in `$CODEX_HOME/config.toml`, which is `~/.codex/config.toml` by default.
Project-scoped overrides go in `.codex/config.toml`, and everything on this page works there too.
Codex walks from the project root down to your working directory and loads every `.codex/config.toml` on the way, with the closest one winning.

Two conditions gate the project-scoped file, and it is silently ignored when either fails:

- Codex has to find the project root, which it does with `project_root_markers` - `.git` by default. A directory that isn't a repository has no project root, so its `.codex/config.toml` is never read.
- The project has to be trusted. Codex asks the first time you run it in a directory and records the answer as `[projects."<absolute path>"] trust_level = "trusted"` in your user-level config. Project-local config, hooks, and exec policies are all gated on this.

Neither failure prints anything under `codex sandbox`, so if a project-scoped file seems to have no effect, check trust first.

A handful of keys are stripped from project-scoped config even when trusted, with a startup warning: `openai_base_url`, `chatgpt_base_url`, `apps_mcp_product_sku`, `model_provider`, `model_providers`, `notify`, `profile`, `profiles`, `experimental_realtime_ws_base_url`, `otel`, and `features.respect_system_proxy`. None of the settings on this page are among them.

Sandbox permissions are configured with named permissions profiles, not with individual toggles:

```toml
default_permissions = "ghtkn"

[permissions.ghtkn]
extends = ":workspace"
description = "Workspace profile plus what ghtkn needs."
```

A profile takes `description`, `extends`, `workspace_roots`, `filesystem`, and `network`.

`extends = ":workspace"` inherits one of the three built-in profiles - `:read-only`, `:workspace`, and `:danger-full-access` - so the profile only adds to the usual behaviour rather than replacing it.
It is a choice, not the default: when `default_permissions` is unset, Codex picks `:workspace` for a project it already knows the trust setting of and `:read-only` otherwise.
Neither built-in grants anything ghtkn needs, so the examples below start from `:workspace` and add the one setting that matters.

If you want to be stricter than the default rather than match it, extend `:read-only` instead. Every example on this page works unchanged that way: with `:read-only`, `ghtkn agent status` still reaches the socket, while writes to the working directory are refused. Nothing here relies on workspace-write.

Don't leave `extends` out to build a profile from scratch. A profile without it makes Codex abort with exit code 134 and no message at all, not report a configuration error.

Two things that look like configuration but aren't:

- There is no top-level `[network]` table. Network settings live inside a profile, as `[permissions.<name>.network]`.
- There is no `[[rules.prefix_rules]]` in `config.toml`. Command approval is a separate mechanism, and its rules are `.rules` files that Codex reads from the `rules/` directory of each config layer - `$CODEX_HOME/rules/*.rules`, and `.codex/rules/*.rules` in a trusted project. An administrator's `requirements.toml` does take `rules.prefix_rules`, but rejects `decision = "allow"` there by design.

Nothing on this page affects command approval, and nothing about approval affects the sandbox.
The sandbox never blocks *running* `ghtkn`: Seatbelt's `process-exec` is unrestricted, so no permission entry is needed to execute it, and none of the settings below grant one.
They control what `ghtkn` may reach once it is running - the agent socket, the keychain, the token directory, GitHub.
In the normal case it doesn't need configuring either: under the default `approval_policy = "on-request"`, approval defers to a restricted sandbox and allows a command that is neither flagged dangerous nor asking to escape the sandbox, without prompting. `ghtkn` is neither. Set `approval_policy = "untrusted"` and every unmatched command is prompted instead, but that is a decision about approval, not something the settings below can change.

Codex ignores unknown keys in `config.toml` without complaining, so either of those silently does nothing.
That makes a wrong setting hard to notice: see [Validating](#validating).

## What the default sandbox blocks

On macOS with no settings. This is the same under `:read-only` and `:workspace`, so it doesn't depend on which one your project resolves to:

| | Default sandbox |
| --- | --- |
| ghtkn agent socket | blocked |
| OS keychain | blocked |
| text backend token directory, reading | allowed |
| text backend token directory, writing | blocked |
| Outbound network | blocked |

## agent Backend

Allow the agent socket:

```toml
default_permissions = "ghtkn"
features.network_proxy = true

[permissions.ghtkn]
extends = ":workspace"
description = "Workspace profile plus the ghtkn agent socket."

[permissions.ghtkn.network]
enabled = true

[permissions.ghtkn.network.unix_sockets]
"/Users/alice/.cache/ghtkn/agent.sock" = "allow"
```

Without it, every command that talks to the agent fails:

```console
$ ghtkn agent status
ERR ghtkn failed error="query the agent status: connect to the ghtkn agent: dial unix /Users/alice/.cache/ghtkn/agent.sock: connect: operation not permitted"
```

Both `enabled = true` and `features.network_proxy = true` matter, and they do different jobs.
`enabled = true` turns on the profile's network policy, which is what lets a sandboxed command open a Unix socket at all.
On its own, though, it allows *every* Unix socket and *every* outbound host: it's the switch, not the allowlist.
`features.network_proxy = true` is what makes `unix_sockets` and `domains` behave as allowlists, so the socket above is the only one allowed and, since no domains are listed, no outbound host is reachable.

Use an absolute path.
`~` is not expanded here, and with the proxy enabled a relative entry is a startup error rather than a silent miss:

```console
Error: failed to start managed network proxy: failed to build network proxy state: invalid network.allow_unix_sockets[0]
```

Allow the path the agent actually uses (see [Socket path](../ghtkn-backend/reference.md#socket-path)).
If you set `GHTKN_AGENT_SOCKET` or `XDG_RUNTIME_DIR`, allow that path instead.

The agent runs outside the sandbox and owns the token lifecycle, so nothing else is needed: no allowed domains, no write access.
This assumes the agent is already running and unlocked.
`ghtkn agent unlock` needs a terminal, so run it in your own shell rather than through a coding agent:

```sh
ghtkn agent start
ghtkn agent unlock
```

## keyring Backend

### macOS

Reading a cached token needs the profile's network policy enabled, and nothing else:

```toml
default_permissions = "ghtkn"
features.network_proxy = true

[permissions.ghtkn]
extends = ":workspace"
description = "Workspace profile plus macOS Keychain access for ghtkn."

[permissions.ghtkn.network]
enabled = true
```

The keychain isn't a network resource, so this looks stranger than it is.
macOS enforces the sandbox with Seatbelt, and the keyring library shells out to the `security` command, which reaches the keychain over Mach IPC.
The Mach service it needs, `com.apple.SecurityServer`, is granted by Codex's network policy - the policy adds it so that TLS certificate verification works, and keychain access comes along with it. Enabling the profile's network policy is therefore what makes a cached token readable.

Running `security` itself needs no permission. Codex's Seatbelt profile is closed by default but allows `process-exec` unconditionally, so it restricts what a program may touch, not which program runs. (Whether Codex asks you before running a command is the separate approval layer described above, and `codex sandbox` bypasses it.)

No read permission for `~/Library/Keychains` is needed either: the keychain file is readable under the default read policy, and it is the Mach lookup, not the file, that the sandbox withholds.

`features.network_proxy = true` is not needed for the keychain itself, but keep it: without it, enabling the network policy also opens every outbound host.
With it and no `domains` entries, outbound stays blocked.

Storing a token additionally needs write access to the keychain directory:

```toml
[permissions.ghtkn.filesystem]
"~/Library/Keychains" = "write"
```

This lets every command in that profile modify your login keychain, not just ghtkn.
Storing a token means minting one, which means the interactive device flow, so prefer running `ghtkn auth` in your own terminal and letting Codex read what it cached.

### Linux

Not verified. The keyring backend talks to the Secret Service over the D-Bus session bus, which is a Unix socket, so the same `network.enabled` and `unix_sockets` settings are the ones to look at, but the D-Bus socket path differs per session.

In the environments where the Linux keyring is a problem to begin with (containers, microVMs), use the `agent` or `text` backend instead.

### Windows

Not verified. The keyring backend uses the Windows credential store, and the relevant Codex sandbox controls need to be confirmed on Windows before this page recommends anything.

## text Backend

Reading a cached token works with no settings: the token directory is readable under the default policy.

Writing needs the token directory:

```toml
default_permissions = "ghtkn"

[permissions.ghtkn]
extends = ":workspace"
description = "Workspace profile plus ghtkn text backend token writes."

[permissions.ghtkn.filesystem]
"~/.cache/ghtkn/tokens" = "write"
```

Unlike `unix_sockets`, `filesystem` paths do expand `~`.

Allow the directory, not the file.
The text backend writes a token by creating a temporary file next to it and renaming it into place, so allowing only `<dir>/<client-id>` isn't enough.

Allow the path the backend actually resolves (see [text Backend](../ghtkn-backend/reference.md#text-backend)).
If you set `GHTKN_TEXT_BACKEND_DIR` or `XDG_CACHE_HOME`, allow that path instead.

Writes only happen when a token is minted, so read-only use needs no settings at all.

## Network

The `agent` backend needs no allowed domains: the client talks only to the socket.

Domains are needed when the sandboxed process itself reaches GitHub - minting a token with the device flow on `keyring` or `text`, `ghtkn revoke`, and any `gh` or `git` command that uses the token:

| Host | Purpose |
| --- | --- |
| `github.com` | Device flow and token refresh |
| `api.github.com` | `ghtkn revoke` |

```toml
[permissions.ghtkn.network.domains]
"github.com" = "allow"
"api.github.com" = "allow"
```

Entries are a table of `"host" = "allow"` or `"deny"`, not an array.

Unlike Claude Code, Codex doesn't prompt for a host you haven't listed: with `features.network_proxy = true` a non-allowed host is refused by the proxy, and a Go program sees it as `Forbidden`.

Note that ghtkn working in the sandbox doesn't make `git push` or `gh` work: they reach GitHub themselves and need these hosts regardless of the backend.

There is no macOS TLS problem to work around here.
`OSStatus -26276`, which [Claude Code's sandbox](claude_code.md#tls-verification-fails-on-macos) hits with every Go CLI, does not occur under Codex: a Go program reaching an allowed host verifies certificates and succeeds.
The reason is the same policy that grants the keychain: Codex's network policy allows `com.apple.trustd.agent`, the trust service Go's platform verifier calls, whereas Claude Code withholds it until you ask.

## Revoke and incident response

`ghtkn revoke` is not needed for normal use, and it's reasonable to keep it out of the default setup and run it deliberately.

It calls GitHub's revocation API and removes the stored token, so it needs the domains above plus write access to the backend - the token directory for `text`, the keychain directory for `keyring`. `ghtkn revoke --all` is for incident response and can invalidate every stored token.

## Validating

`codex doctor --summary --ascii` reports `config loaded` even for a file made entirely of keys Codex doesn't recognise, so it can't tell you whether these settings took effect. Use it only to confirm the TOML parsed.

Check the sandbox itself instead, by running the command under it:

```sh
codex sandbox ghtkn agent status
codex sandbox ghtkn get >/dev/null
```

`codex sandbox` runs the command under the same sandbox a session would use, without calling the model, so it needs no Codex login or subscription. It works even when `codex doctor` reports `no Codex credentials were found`.

`ghtkn agent status` reports whether the socket is reachable without touching a token. Redirecting `ghtkn get` keeps the access token off the terminal - it is a secret, and a coding agent must not print, echo, or log it. Consume it in place instead:

```sh
GH_TOKEN=$(ghtkn get) gh issue list
```

If `ghtkn get` tries to start the device flow and fails because the device flow is disabled, Codex found no valid cached token for the current app and backend. Run `ghtkn auth` in your own terminal with the same configuration, especially the same `GHTKN_APP`, then retry.

If the same command succeeds in a normal shell but fails under `codex sandbox`, the profile isn't applying. Check that `default_permissions` names the profile you defined, and, if the settings are in a project-scoped `.codex/config.toml`, that the project is a repository and is trusted - see [Where the settings go](#where-the-settings-go). Moving the same settings to `$CODEX_HOME/config.toml` tells the two apart quickly.

On the macOS keyring backend, ghtkn stores the token with service `github.com/suzuki-shunsuke/ghtkn` and account set to the GitHub App Client ID, not the app name. To test item access without printing the secret:

```sh
if security find-generic-password \
  -s github.com/suzuki-shunsuke/ghtkn \
  -a <client-id> >/dev/null 2>&1; then
  echo accessible
else
  echo unavailable
fi
```

Do not add `-w` or `-wa` when debugging in an agent session; those options print the stored secret.
