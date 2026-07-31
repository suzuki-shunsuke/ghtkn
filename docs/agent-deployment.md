---
description: Run the ghtkn agent as a long-lived process. Use to set it up as a systemd user service, to start it from a container entrypoint, or to run it on the host so that containers use it as a client, which is what refresh tokens need.
---

# Running the agent

Where to run the ghtkn agent process: under a service manager, inside a container, or on the host with containers talking to it as clients.
See [backend.md](backend.md) for what the agent backend is, how it encrypts tokens, and how to drive it with `ghtkn agent`.

## Running the agent as a service

`ghtkn agent start &` runs the agent for the current shell session.
Whether it keeps running after you close the terminal depends on your shell, so for a long-lived agent run it under a service manager (or detach it explicitly with `nohup` / `disown`).
In every case the agent starts locked, so after it (re)starts you need to run `ghtkn agent unlock` once.

### Linux (systemd user service)

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

### Containers (Docker / devcontainer)

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

## Using the host's agent from a container

Giving a container its own agent puts the key file and the encrypted tokens inside the container, which rules out [refresh tokens](refresh-token.md): they shouldn't be enabled where malware can escalate to root without a password, and development containers usually allow exactly that.

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

### macOS hosts: mounting the socket doesn't work

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
