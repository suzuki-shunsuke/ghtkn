---
name: ghtkn-backend
description: Choose and configure where ghtkn stores access tokens (keyring, text, agent). Use when the OS keyring is unavailable (containers, microVMs), or when setting up the ghtkn agent backend.
---

ghtkn stores access tokens in a backend, selected with `GHTKN_BACKEND` or `backend.type`. Supported backends:

- `keyring`: OS keyring (default).
- `text`: plaintext files (`0600`) - useful where the keyring is hard to use.
- `agent`: tokens encrypted (AES-256-GCM) via the ghtkn agent; after `ghtkn agent unlock` the agent holds the decryption key in memory only, never the passphrase. Intended for local use, not CI.

Pick `text` or `agent` for containers and microVMs where the OS keyring is unavailable.
On desktop environments the default keyring still works fine, but since v0.3.4 the `agent` backend is worth considering there too, because it's the only backend that supports refresh tokens (see the ghtkn-refresh-token skill).

If this overview is enough, you don't need to read further.

## Reference

Read the following files in this skill directory for the details:

- [reference.md](reference.md): read it to choose a backend, to drive the agent (`start`, `unlock`, `lock`, `stop`, `reset`), or to look up where the socket, the tokens, and the encryption key are stored.
- [agent_deployment.md](agent_deployment.md): read it to run the agent as a long-lived process - a systemd user service, a container entrypoint, or an agent on the host that containers use as a client (needed for refresh tokens, which shouldn't be enabled inside a container).
