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

## Which backend should you use?

The default is the OS keyring, and where it is available it stays a secure choice that needs no setup, so leaving the backend alone is perfectly reasonable.

Since v0.3.4, though, the `agent` backend is no longer just a workaround for environments without a keyring.
Only the agent backend supports [refresh tokens](../ghtkn-refresh-token/reference.md), which spare you the device flow every eight hours, and for security reasons there is no plan to support them on the other backends.
So the agent backend is worth considering on desktop environments too, and it may become the mainstream choice going forward.
It does take some effort in exchange: you need to keep the agent running and manage a passphrase.
Note also that refresh tokens are supported only on macOS and Linux, and shouldn't be enabled where malware can easily escalate to root; see the refresh-token document for the caveats and the tradeoff involved.

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

### Using the host's agent from a container

Giving a container its own agent puts the key file and the encrypted tokens inside the container, which rules out [refresh tokens](../ghtkn-refresh-token/reference.md): they shouldn't be enabled where malware can escalate to root without a password, and development containers usually allow exactly that.

Running the agent on the host and letting the container talk to it as a client avoids this. Only access tokens cross the socket; the refresh token, the key file, and the passphrase stay on the host, so a container whose user can become root still can't reach them.

On a Linux host, bind mount the socket. It is `0600`, so the container has to run as the same uid as the host user:

```sh
docker run \
  --user "$(id -u)" \
  -v "$HOME/.cache/ghtkn/agent.sock:/tmp/agent.sock" \
  -e GHTKN_BACKEND=agent \
  -e GHTKN_AGENT_SOCKET=/tmp/agent.sock \
  ...
```

#### macOS hosts: mounting the socket doesn't work

On macOS the mount approach fails with every runtime (Docker Desktop, colima / Lima, Podman machine, OrbStack, Apple's `container`). Containers run in a Linux VM, and the host directory reaches it through a file-sharing layer (virtiofs, gRPC-FUSE, 9p) that passes the socket's inode through but not the endpoint behind it, which lives in the macOS kernel:

```console
$ ghtkn info
... error="query the agent status: connect to the ghtkn agent: dial unix /home/foo/.cache/ghtkn/agent.sock: connect: operation not supported"
```

Sharing the directory into the VM itself, for example with colima's `mounts:`, doesn't help. The same limit applies at the VM boundary, so even `stat` on the socket fails there:

```console
$ colima ssh -- ls -l ~/.cache/ghtkn/agent.sock
ls: cannot access '.../agent.sock': Operation not supported
```

What does work is forwarding the socket over SSH, the mechanism behind ssh-agent forwarding. The socket SSH creates in the VM is a real Linux socket, so a container running in that VM can bind mount it. With colima:

```sh
colima ssh-config > ~/.colima-ssh.config
VM_HOME=$(colima ssh -- printenv HOME)

: A socket file left behind by an earlier forward blocks a new one
colima ssh -- rm -f "$VM_HOME/ghtkn-agent.sock"

ssh -F ~/.colima-ssh.config -N -f \
  -o ControlMaster=no -o ControlPath=none \
  -R "$VM_HOME/ghtkn-agent.sock:$HOME/.cache/ghtkn/agent.sock" \
  colima

: The socket is created 0600 and owned by the VM user; relax it if the container runs as another uid
colima ssh -- chmod 666 "$VM_HOME/ghtkn-agent.sock"
```

Then mount it into the container and point ghtkn at it:

```sh
docker run \
  -v "$VM_HOME/ghtkn-agent.sock:/tmp/agent.sock" \
  -e GHTKN_BACKEND=agent \
  -e GHTKN_AGENT_SOCKET=/tmp/agent.sock \
  ...
```

`$VM_HOME` is a path inside the VM and doesn't exist on macOS, which is correct here: the source of `-v` is resolved by the docker daemon, and with colima the daemon runs in the VM. Run the command from macOS as usual. If you get the path wrong, `-v` silently creates a directory there instead of failing, and the container ends up with an empty directory where the socket should be, so `--mount type=bind,src=...,dst=/tmp/agent.sock` is worth using to make a typo an error.

`ghtkn info` in the container then reports `.agent.running` as `true`, and `ghtkn get` works as long as the agent is unlocked on the host.

Notes on the forward:

- Put the socket on a path that belongs to the VM, and get it with `colima ssh -- printenv HOME`. Two traps here. Lima names the guest home differently from the host user name, so `/home/$USER` doesn't exist and `-R` fails on it. And `colima ssh -- pwd` is not a way to find the home: colima enters the VM in the current directory, so it prints the shared host path (`/Users/...`), which is a virtiofs mount where the socket doesn't work and `chmod` fails with `Invalid argument`.
- Regenerate `colima ssh-config` after restarting colima. The port changes on restart, and the stale config fails with `connect to host 127.0.0.1 port <port>: Connection refused`.
- colima's ssh config enables `ControlMaster`, and the multiplexed connection rejects the forwarding request, so pass `-o ControlMaster=no -o ControlPath=none`.
- `ssh -f` exits `0` even when the remote bind fails. The stale socket file then answers with `connection refused`, so remove it beforehand and verify the connection rather than trusting the exit status.
- `StreamLocalBindUnlink` and `StreamLocalBindMask` have no effect on `-R` when set on the client; they are sshd options. That's why the `rm -f` and the `chmod` are explicit steps.
- Relaying the socket over TCP with `socat` works too, but then every container and every process in the VM can reach the agent over the network. The SSH forward keeps it a unix socket, which a container sees only if you mount it.

While the agent is unlocked and the forward is up, anything in the container can ask for access tokens repeatedly, including forcing a renewal, and a `0666` socket is reachable by any process in the VM as well. What it can't obtain is the refresh token, the data key, or the passphrase. Drop the forward when you don't need it, or run `ghtkn agent lock` on the host to close the window immediately.

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
