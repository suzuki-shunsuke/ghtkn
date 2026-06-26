# Backend

By default ghtkn stores access tokens in the OS keyring.
You can change where they are stored with the `GHTKN_BACKEND` environment variable.
This is useful in environments where the OS keyring is hard to use, such as containers and microVMs.
`GHTKN_BACKEND` supports the following values:

- `keyring`: OS keyring (default)
- `text`: Store tokens as plaintext files
- `agent`: Store tokens encrypted via the ghtkn agent

## Which backend should you use?

On desktop environments where the OS keyring is available, using the default OS keyring is the most secure and recommended option, so you usually don't need to worry about backends.
In environments where the OS keyring is unavailable and you want to prioritize security, the `agent` backend, which encrypts access tokens with AES-256-GCM, is a good choice.
However, the `agent` backend takes some effort: you need to start the agent and manage a passphrase.
If you prefer simplicity, the `text` backend, which avoids that effort, is a good choice.

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

### Running the agent as a service

`ghtkn agent start &` runs the agent for the current shell session.
Whether it keeps running after you close the terminal depends on your shell, so for a long-lived agent run it under a service manager (or detach it explicitly with `nohup` / `disown`).
In every case the agent starts locked, so after it (re)starts you need to run `ghtkn agent unlock` once.

#### Linux (systemd user service)

On a VM, a microVM, or a minimal Linux box without a keyring, run the agent as a systemd user service.
Create `~/.config/systemd/user/ghtkn-agent.service`:

```ini
[Unit]
Description=ghtkn agent

[Service]
ExecStart=/path/to/ghtkn agent start
Restart=on-failure

[Install]
WantedBy=default.target
```

```sh
: Enable and start the service
systemctl --user enable --now ghtkn-agent
: Unlock it once after it starts
ghtkn agent unlock
```

Notes:

- Use the absolute path to `ghtkn` in `ExecStart`; the systemd user environment has a minimal `PATH`.
- Use `Restart=on-failure`, not `Restart=always`. `ghtkn agent stop` exits successfully, so `Restart=always` would immediately start the agent again.
- To keep the agent running even when you are not logged in, enable lingering with `loginctl enable-linger "$USER"`.

#### Containers (Docker / devcontainer)

Containers usually have no init system, so start the agent from the container's entrypoint.
Use a wrapper that starts the agent in the background and then runs the container's main process:

```sh
#!/usr/bin/env bash
# entrypoint.sh
set -eu
ghtkn agent start &
exec "$@"
```

```dockerfile
ENV GHTKN_BACKEND=agent
ENTRYPOINT ["entrypoint.sh"]
```

Setting `GHTKN_BACKEND=agent` with `ENV` in the Dockerfile selects the backend for every process in the container, so you don't have to `export` it in each shell.

After attaching to the container (for example `docker exec -it <container> bash`), unlock the agent once:

```sh
ghtkn agent unlock
ghtkn get
```

The encryption key and the encrypted access tokens live on the container's filesystem, so they are lost when the container is removed (the tokens are reminted on the next `ghtkn get`).
To persist them, mount a volume for `$XDG_DATA_HOME` (the key) and `$XDG_CACHE_HOME` (the tokens).

A microVM (Firecracker, Cloud Hypervisor, Kata Containers, Lima, etc.) fits one of the two patterns above: use the systemd service if it boots a minimal Linux with systemd, or the entrypoint approach if it runs a single application like a container.

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

## Example

In this example, we use ghtkn in a Docker container.

First, build a Docker image and run it.

Dockerfile:

```dockerfile
FROM mirror.gcr.io/ubuntu:24.04@sha256:1e622c5f073b4f6bfad6632f2616c7f59ef256e96fe78bf6a595d1dc4376ac02
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update
RUN apt-get install -y sudo ca-certificates curl
RUN echo 'foo ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers
RUN useradd -u 900 -m -r foo
USER foo
ENV PATH=/home/foo/.local/share/aquaproj-aqua/bin:$PATH
RUN mkdir /home/foo/workspace
WORKDIR /home/foo/workspace
RUN curl -sSfL -O https://raw.githubusercontent.com/aquaproj/aqua-installer/v4.0.5/aqua-installer
RUN echo "451028d56959cc738564885b1dbebc2691ea038ffde04e2472e4d486a3591146  aqua-installer" | sha256sum -c -
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
Jun  2 00:02:36.945 ERR ghtkn failed program=ghtkn version=0.2.6 error="get or create access token: get or create token: get a token from the backend: get a secret from the keyring: exec: \"dbus-launch\": executable file not found in $PATH"
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
