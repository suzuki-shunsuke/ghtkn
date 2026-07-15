---
name: ghtkn-refresh-token
description: Automatically refresh expiring GitHub access tokens with refresh tokens. Use when enabling refresh on the ghtkn agent, running `ghtkn agent unlock --enable-refresh`, setting `--refresh-token-ttl`, or reasoning about refresh-token removal and security.
---

ghtkn can refresh an expiring access token from a stored refresh token instead of running the device flow every 8 hours. Key points:

- Supported only on the `agent` backend on macOS and Linux (not Windows, keyring, or text) - intentionally, for security, and enforced: on Windows `--enable-refresh` is rejected and the agent refuses to enable it. Do not use it where malware can easily escalate to root (many dev containers/VMs).
- Enable it per unlock: `ghtkn agent unlock --enable-refresh`. Then `ghtkn get` / `ghtkn auth` silently refresh when a valid refresh token exists (refresh tokens are valid ~6 months), and fall back to the device flow otherwise.
- You must pass `--enable-refresh` on every `agent unlock`. If you omit it while valid refresh tokens are stored, unlock asks you to confirm dropping them (default No); answer No and rerun with `--enable-refresh` to keep them.
- The agent sweeps tokens unused past a TTL (default 1 week, `--refresh-token-ttl`, max ~6 months), deleting the whole encrypted file to avoid holding long-lived refresh tokens for rarely used apps.

If this overview is enough, you don't need to read further.

## Reference

For details, read [reference.md](reference.md) in this skill directory.
