---
description: Choose and configure where ghtkn stores access tokens (keyring, text, agent). Use when the OS keyring is unavailable (containers, microVMs), when setting up the ghtkn agent backend, or to look up where the socket, the tokens, and the encryption key are stored.
---

# Backend

By default ghtkn stores access tokens in the OS keyring.
You can change where they are stored with the `GHTKN_BACKEND` environment variable or the configuration file `.backend.type`.

```sh
export GHTKN_BACKEND=text
```

```yaml
backend:
  type: agent # text, keyring
```

This is useful in environments where the OS keyring is hard to use, such as containers and microVMs.
The following values are supported:

- `keyring`: OS keyring (default)
- `text`: Store tokens as plaintext files
- `agent`: Store tokens encrypted via the ghtkn agent

In a container the default backend typically fails like this, which is the usual reason to change it:

```console
$ ghtkn get
... ERR ghtkn failed error="get or create access token: get or create token: get a token from the backend: get a secret from the keyring: exec: \"dbus-launch\": executable file not found in $PATH"
```

## Which backend should you use?

The default is the OS keyring, and where it is available it stays a secure choice that needs no setup, so leaving the backend alone is perfectly reasonable.

Since v0.3.4, though, the `agent` backend is no longer just a workaround for environments without a keyring.
Only the agent backend supports [refresh tokens](refresh-token.md), which spare you the device flow every eight hours, and for security reasons there is no plan to support them on the other backends.
So the agent backend is worth considering on desktop environments too, and it may become the mainstream choice going forward.
It does take some effort in exchange: you need to keep the agent running and manage a passphrase.
Note also that refresh tokens are supported only on macOS and Linux, and shouldn't be enabled where malware can easily escalate to root; see [Refreshing tokens](refresh-token.md) for the caveats and the tradeoff involved.

In environments where the OS keyring is unavailable and you want to prioritize security, the `agent` backend, which encrypts access tokens with AES-256-GCM, is a good choice.
If you prefer simplicity over encryption at rest and don't need refresh tokens, the `text` backend, which needs neither an agent nor a passphrase, is a good choice.

## text Backend

```sh
export GHTKN_BACKEND=text
```

The text backend stores access tokens as plaintext files.
The files are created with permission `0600`, so other users can't read them, but to prevent leaks the following measures are recommended:

- Don't manage the storage directory with git
- Exclude it from cloud storage such as Dropbox
- Enable OS-level disk encryption

The access token storage location is resolved in the following order of precedence:

1. `$GHTKN_TEXT_BACKEND_DIR`
1. `$XDG_CACHE_HOME/ghtkn/tokens`
1. `$HOME/.cache/ghtkn/tokens`

On Windows:

1. `$GHTKN_TEXT_BACKEND_DIR`
1. `$LocalAppData\cache\ghtkn\tokens`

## agent Backend

The agent backend stores access tokens encrypted without depending on the OS keyring.

The encryption works as follows:

- Access tokens are encrypted with AES-256-GCM.
- The 32-byte data key used for encryption is generated randomly on the first `ghtkn agent unlock`, when you set the passphrase. `ghtkn agent start` doesn't create it, because deriving the key that wraps it needs the passphrase.
- The data key is encrypted (wrapped) with a key (KEK) derived from the passphrase via Argon2id and saved to a key file. Neither the passphrase nor the KEK is saved to disk, and both are wiped from memory as soon as the data key is unwrapped. Only the data key stays in the agent's memory while it is unlocked.

Start the agent with the `ghtkn agent start` command.

```sh
: Start the agent in the background
ghtkn agent start &
```

Even after the agent starts, you can't get access tokens until you enter the passphrase with the `ghtkn agent unlock` command.

```sh
: Enter the passphrase
ghtkn agent unlock
```

> [!NOTE]
> [There is a third-party tool `yokonao/ghtkn-touchid`, which unlocks a local ghtkn agent with a passphrase protected by Touch ID in macOS Keychain.](https://github.com/yokonao/ghtkn-touchid)
> This is a third-party tool, so we don't guarantee anything about this tool, but if you're interested in, please check it out.

There are also `status`, `stop`, and `lock` commands.

```sh
: Check the agent status
ghtkn agent status
: Stop the agent
ghtkn agent stop
: Discard the in-memory data key without stopping the agent
ghtkn agent lock
```

### Lock the agent to shrink the exposure window

`ghtkn agent lock` discards the data key the agent holds in memory and returns it to the locked state, without stopping the process, closing the socket, or deleting the key file.
Cached tokens become unreadable until you run `ghtkn agent unlock` again with the same passphrase, which re-derives the same data key from the key file, so tokens stored before the lock are readable again.
Unlike `ghtkn agent stop`, the process and socket keep running, and unlike `unlock`, locking needs no passphrase.

This lets you shrink the window in which the agent holds decrypted tokens: while the agent is unlocked, a process running as your user can ask it for access tokens, so locking it when you step away (or when you switch to activities more likely to run untrusted code) reduces exposure. Because it needs no passphrase, `ghtkn agent lock` can be wired to a screen-lock or logout hook to do this automatically.

To get access tokens, set `GHTKN_BACKEND` to `agent` and run `ghtkn get` or the ghtkn Go SDK.

```sh
export GHTKN_BACKEND=agent
ghtkn get
```

`ghtkn get` and the ghtkn Go SDK communicate with the agent over a socket to get access tokens.

If you forget the passphrase, the only option is to reset it with `ghtkn agent reset`.
Note that resetting deletes the existing key and access tokens.

```sh
: Stop the agent, delete the saved access tokens and key, and create a new key
ghtkn agent reset
```

Resetting leaves the agent stopped, so start it again and unlock it with the new passphrase.

```sh
ghtkn agent start
ghtkn agent unlock
```

The socket, the encryption key, and the encrypted access tokens are created with permission `0600`, so other users can't read them or connect to the socket.

### Restart the agent after upgrading ghtkn

Upgrading the `ghtkn` binary does not update an agent that is already running: the running process keeps executing the old binary, so a bug fix or a new feature (for example refresh-token support) does not take effect until the agent restarts.
A client that needs behavior the running agent is too old for even refuses to talk to it and asks you to restart it.

So after you upgrade ghtkn, restart the agent and unlock it again.
`ghtkn agent start` refuses to start while another agent is still running, so stop the old one first:

```sh
ghtkn agent stop
ghtkn agent start &
ghtkn agent unlock
```

`ghtkn agent status` and `ghtkn info` report the version the running agent was built from, so you can tell whether it still needs a restart:

```sh
ghtkn info | jq '{ghtkn: .version, agent: .agent.version}'
```

```json
{
  "ghtkn": "v0.3.4",
  "agent": "v0.3.1"
}
```

`ghtkn info` also warns on stderr when the two differ, so you don't have to compare them yourself.
It stays quiet when either side reports no version.

An agent version older than the ghtkn version means the agent is still running the old binary.
`unknown` means the agent binary carries no version information (for example one built with `go install`), and a missing `agent.version` means the agent predates this report and is certainly out of date.

### Where to run the agent

`ghtkn agent start &` runs the agent for the current shell session, which is enough while you are trying it out.
For a long-lived agent, read [agent-deployment.md](agent-deployment.md): it covers running the agent as a systemd user service, starting it from a container's entrypoint, and running it on the host so that containers use it as a client instead of holding their own key and tokens.

### Socket path

The socket path is resolved in the following order of precedence:

1. `$GHTKN_AGENT_SOCKET`
1. `$XDG_RUNTIME_DIR/ghtkn/agent.sock`
1. `$XDG_CACHE_HOME/ghtkn/agent.sock`
1. `$HOME/.cache/ghtkn/agent.sock`

On Windows:

1. `$GHTKN_AGENT_SOCKET`
1. `$XDG_RUNTIME_DIR\ghtkn\agent.sock`
1. `$XDG_CACHE_HOME\ghtkn\agent.sock`
1. `$LocalAppData\cache\ghtkn\agent.sock`

### Access token storage location

The access token storage location is resolved in the following order of precedence:

1. `$GHTKN_AGENT_TOKEN_DIR/<client-id>`
1. `$XDG_CACHE_HOME/ghtkn/agent/<client-id>`
1. `$HOME/.cache/ghtkn/agent/<client-id>`

On Windows:

1. `$GHTKN_AGENT_TOKEN_DIR\<client-id>`
1. `$LocalAppData\cache\ghtkn\agent\<client-id>`

### Encryption key storage location

The encryption key storage location is resolved in the following order of precedence:

1. `$GHTKN_AGENT_KEY`
1. `$XDG_DATA_HOME/ghtkn/key`
1. `$HOME/.local/share/ghtkn/key`

On Windows:

1. `$GHTKN_AGENT_KEY`
1. `$LocalAppData\ghtkn\key`
