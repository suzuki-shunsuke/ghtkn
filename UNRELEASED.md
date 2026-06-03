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
The device flow is enabled by default. By setting `GHTKN_ENABLE_DEVICE_FLOW` to `false`, `ghtkn` will fail fast with an actionable error instead of blocking.

e.g.

~/.claude/settings.json:

```json
{
  "env": {
    "GHTKN_ENABLE_DEVICE_FLOW": "false"
  }
}
```

When you do need to authenticate in your own interactive terminal, you can re-enable the device flow for a single command with the `--device-flow` (`-d`) flag, which overrides the environment variable:

```sh
ghtkn get -d
```

```console
$ claude
  Claude Code v2.1.158
  Opus 4.8 (1M context) with high effort - Claude Max
  ~/repos/src/github.com/csm-actions/securefix-action

$ gh repo view

> Bash(gh repo view)
  | Error: Exit code 1
    May 31 14:27:42.175 ERR ghtkn failed program=ghtkn version=v3.0.0-local error="get or create access token: get or create token: create a
    GitHub App User Access Token: a GitHub App User access token can't be created via Device Flow because it's disabled by
    GHTKN_ENABLE_DEVICE_FLOW=false. The Device Flow is interactive and can't be completed by a background or non-interactive process. If you are a
    coding agent, do NOT run `ghtkn get` yourself because it would fail the same way; instead, ask the user to run `ghtkn get -s -d` in their own
    interactive terminal to authenticate"

> gh repo view failed - it couldn't authenticate. The gh CLI here goes through ghtkn, which needs an interactive GitHub App user access token
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

### Example

In this example, we use ghtkn in a Docker container.

First, build a Docker image and run it.

Dockerfile:

```dockerfile
FROM mirror.gcr.io/ubuntu:24.04@sha256:1e622c5f073b4f6bfad6632f2616c7f59ef256e96fe78bf6a595d1dc4376ac02
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update
RUN apt-get install -y sudo vim ca-certificates curl
RUN echo 'foo ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers
RUN useradd -u 900 -m -r foo
USER foo
ENV PATH=/home/foo/.local/share/aquaproj-aqua/bin:$PATH
RUN mkdir /home/foo/workspace
WORKDIR /home/foo/workspace
RUN curl -sSfL -O https://raw.githubusercontent.com/aquaproj/aqua-installer/v4.0.4/aqua-installer
RUN echo "acd21cbb06609dd9a701b0032ba4c21fa37b0e3b5cc4c9d721cc02f25ea33a28  aqua-installer" | sha256sum -c -
RUN chmod +x aqua-installer
RUN ./aqua-installer
```

```sh
docker build -t ghtkn .
docker run --name ghtkn --rm -ti ghtkn bash
```

In the container, install ghtkn using [aqua](https://aquaproj.github.io/).

```sh
aqua init
aqua g -i suzuki-shunsuke/ghtkn
vim aqua.yaml
```

aqua.yaml:

```yaml
- name: suzuki-shunsuke/ghtkn@v0.2.5-0
```

```sh
aqua i
ghtkn init
```

Copy ghtkn.yaml from the host to the container.

```sh
docker cp ~/.config/ghtkn/ghtkn.yaml ghtkn:/home/foo/.config/ghtkn/ghtkn.yaml
```

Before using `text` and `agent` backends, let's confirm that ghtkn doesn't work by default, because the OS keyring isn't available.

```console
foo@6b90309bf6a4:~/workspace$ ghtkn get
Jun  2 00:02:36.945 ERR ghtkn failed program=ghtkn version=0.2.5-0 error="get or create access token: get or create token: get a token from the backend: get a secret from the keyring: exec: \"dbus-launch\": executable file not found in $PATH"
```

Let's set `GHTKN_BACKEND` to `text` and try again.
You need to open the browser manually because ghtkn running in a container can't open a browser on the host.

```sh
export GHTKN_BACKEND=text
ghtkn get
```

Awesome! ghtkn is now working with the text backend.

Next, let's use the `agent` backend.

```sh
ghtkn agent start &
ghtkn agent status
ghtkn agent unlock
```

```sh
export GHTKN_BACKEND=agent
ghtkn get
```

Great! ghtkn is now working with the agent backend too.

## A browser opens on its own when using cmux

### Symptom

When using [cmux](https://github.com/manaflow-ai/cmux), ghtkn may open a browser on its own.
Worse, the one-time code isn't shown anywhere, so you can't complete the device flow and have to close the browser tab.

### Cause

When cmux's `Show Pull Requests in Sidebar` is enabled (it is enabled by default), cmux periodically calls the GitHub API per pane to poll the state of pull requests.
To get an access token for this, it runs `gh auth token`.
If you wrap the `gh` command so that `ghtkn get` is called, the device flow runs and a browser opens when the access token has expired or is not found.
Because cmux runs these commands in the background, you can't see the one-time code.

### Solutions

There are several options.

1. Disable `Show Pull Requests in Sidebar` - this is a cmux setting, so the details are omitted.
1. Set `GHTKN_ENABLE_DEVICE_FLOW=false` in your shell configuration (`.bashrc`, `.zshrc`, etc.) to disable the device flow by default. When the token expires, explicitly run `ghtkn get -s -d` to renew the access token (requires ghtkn v0.2.5 or later).
1. Modify your `gh` wrapper script so that it sets `GHTKN_ENABLE_DEVICE_FLOW=false` only for the commands cmux runs in the background.

#### Set `GHTKN_ENABLE_DEVICE_FLOW=false` in your shell configuration

This approach disables the device flow by default, so it affects more than just cmux.
Note that you have to run `ghtkn get -s -d` explicitly, which is extra work, and that a script will fail partway through if it runs ghtkn.
On the other hand, always running the device flow explicitly makes you less likely to fall for phishing.

```sh
export GHTKN_ENABLE_DEVICE_FLOW=false
```

```sh
ghtkn get -s -d
```

#### Modify your `gh` wrapper script

This approach limits the impact of the change more than option 2.
However, it depends on cmux's implementation, so it may stop working in the future.
Also, the device flow is disabled when a human runs the same command on cmux too.
This probably rarely happens, but it may be confusing when it does.

Make your wrapper like the following.
The important part is the `if` block that checks `CMUX_PANEL_ID`.
Adjust the script to fit your own environment.

```sh
#!/usr/bin/env bash

set -eu

# Prevent cmux's PR-badge polling (gh pr view ... --json number,state,url) from
# opening a browser on its own via ghtkn's device flow when the token has expired.
# Disable the device flow only inside a cmux pane (CMUX_PANEL_ID) and only for the
# probe-like arguments. Manual gh/git pass through (device flow stays enabled).
if [ -n "${CMUX_PANEL_ID:-}" ]; then
  case " $* " in
    *" pr view "*) case " $* " in
        *"--json number,state,url"*) export GHTKN_ENABLE_DEVICE_FLOW=false ;;
      esac ;;
  esac
fi

# If GH_TOKEN or GITHUB_TOKEN is set, use it.
if [ -z "${GH_TOKEN:-}" ] && [ -z "${GITHUB_TOKEN:-}" ]; then
  GH_TOKEN="$(ghtkn get)"
  export GH_TOKEN
fi

exec aqua exec -- gh "$@"
```
