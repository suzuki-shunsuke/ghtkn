# Unreleased Features

> [!WARNING]
> These features haven't been released yet.

## Get an access token without outputting it

Add `--silent (-s)` to the `ghtkn get` command to retrieve an access token (running the device flow if needed) without outputting it to stdout.

```sh
ghtkn get -s
```

This is similar to `ghtkn get >/dev/null`, but

- `-s` is shorter than `>/dev/null`
- `-s` works on Windows too

## Disable Device Flow

`ghtkn` obtains a GitHub App User access token via the OAuth Device Flow, which is interactive: it prints a one-time (user) code and waits for the user.
A coding agent (or any background / non-interactive process) cannot complete this, so it would block until a device code expires.
By setting `GHTKN_DISABLE_DEVICE_FLOW` to `true`, `ghtkn` will fail fast with an actionable error instead of blocking.

e.g.

~/.claude/settings.json:

```json
{
  "env": {
    "GHTKN_DISABLE_DEVICE_FLOW": "true"
  }
}
```

```console
❯ claude
▗ ▗   ▖ ▖  Claude Code v2.1.158
           Opus 4.8 (1M context) with high effort · Claude Max
  ▘▘ ▝▝    ~/repos/src/github.com/csm-actions/securefix-action

   Opus 4.8 is here! Now defaults to high effort · /effort xhigh for your hardest tasks

❯ gh repo view                                                                                                                                

⏺ Bash(gh repo view)
  ⎿  Error: Exit code 1
     May 31 14:27:42.175 ERR ghtkn failed program=ghtkn version=v3.0.0-local error="get or create access token: get or create token: create a
     GitHub App User Access Token: a GitHub App User access token can't be created via Device Flow because it's disabled by 
     GHTKN_DISABLE_DEVICE_FLOW. The Device Flow is interactive and can't be completed by a background or non-interactive process. If you are a
     coding agent, do NOT run `ghtkn get` yourself because it would fail the same way; instead, ask the user to run `ghtkn get` in their own 
     interactive terminal to authenticate"

⏺ gh repo view failed — it couldn't authenticate. The gh CLI here goes through ghtkn, which needs an interactive GitHub App user access token
  (Device Flow is disabled for non-interactive processes).

  Could you authenticate in your own terminal by typing:

  ! ghtkn get

  Once that succeeds, I'll re-run gh repo view.
```


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
