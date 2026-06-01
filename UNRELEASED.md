# Unreleased Features

> [!WARNING]
> These features haven't been released yet.

## Backend

By default ghtkn stores access tokens in the OS keyring.
You can change where they are stored with the `GHTKN_BACKEND` environment variable.
This is useful in environments where the OS keyring is hard to use, such as containers and microVMs.
`GHTKN_BACKEND` supports the following values:

- `keyring`: OS keyring (default)
- `text`: Store tokens as plaintext files
- `agent`: Store tokens encrypted via the ghtkn agent

### Which backend should you use?

On desktop environments where the OS keyring is available, using the default OS keyring is the most secure and recommended option, so you usually don't need to worry about backends.
In environments where the OS keyring is unavailable and you want to prioritize security, the `agent` backend, which encrypts access tokens with AES-256-GCM, is a good choice.
However, the `agent` backend takes some effort: you need to start the agent and manage a passphrase.
If you prefer simplicity, the `text` backend, which avoids that effort, is a good choice.

### text Backend

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

### agent Backend

The agent backend stores access tokens encrypted without depending on the OS keyring.

The encryption works as follows:

- Access tokens are encrypted with AES-256-GCM.
- The 32-byte data key used for encryption is generated randomly when the agent first starts.
- The data key is encrypted (wrapped) with a key (KEK) derived from the passphrase via Argon2id and saved to a key file. The passphrase itself and the KEK are not saved to disk; they are kept only in the agent's memory after unlocking.

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

There are also `status` and `stop` commands.

```sh
: Check the agent status
ghtkn agent status
: Stop the agent
ghtkn agent stop
```

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

The socket, the encryption key, and the encrypted access tokens are created with permission `0600`, so other users can't read them or connect to the socket.

#### Socket path

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

#### Access token storage location

The access token storage location is resolved in the following order of precedence:

1. `$GHTKN_AGENT_TOKEN_DIR/<client-id>`
1. `$XDG_CACHE_HOME/ghtkn/agent/<client-id>`
1. `$HOME/.cache/ghtkn/agent/<client-id>`

On Windows:

1. `$GHTKN_AGENT_TOKEN_DIR\<client-id>`
1. `$LocalAppData\cache\ghtkn\agent\<client-id>`

#### Encryption key storage location

The encryption key storage location is resolved in the following order of precedence:

1. `$GHTKN_AGENT_KEY`
1. `$XDG_DATA_HOME/ghtkn/key`
1. `$HOME/.local/share/ghtkn/key`

On Windows:

1. `$GHTKN_AGENT_KEY`
1. `$LocalAppData\ghtkn\key`
