---
name: ghtkn-backend
description: Choose and configure where ghtkn stores access tokens (keyring, text, agent). Use when the OS keyring is unavailable (containers, microVMs), or when setting up the ghtkn agent backend.
---

ghtkn stores access tokens in a backend, selected with `GHTKN_BACKEND` or `backend.type`. Supported backends:

- `keyring`: OS keyring (default).
- `text`: plaintext files (`0600`) - useful where the keyring is hard to use.
- `agent`: tokens encrypted (AES-256-GCM) via the ghtkn agent; after `ghtkn agent unlock` the agent holds the decryption key in memory only, never the passphrase. Intended for local use, not CI.

Pick `text` or `agent` for containers and microVMs where the OS keyring is unavailable.

If this overview is enough, you don't need to read further.

## Reference

For details (storage locations, running the agent as a systemd service or in a container, socket paths, and a full Docker example), read [reference.md](reference.md) in this skill directory.
