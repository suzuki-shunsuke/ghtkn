---
name: ghtkn-sandbox
description: Configure Claude Code's sandbox to run ghtkn. Use when ghtkn fails inside the sandbox, e.g. the agent socket is "operation not permitted", a token can't be stored, or TLS fails with OSStatus -26276.
---

ghtkn doesn't work inside [Claude Code's sandbox](https://code.claude.com/docs/en/sandboxing) (`sandbox.enabled`) with the default settings. What it needs depends on the backend:

- `agent` (`>= 0.3.4`): allow the socket, and nothing else. macOS: `network.allowUnixSockets: ["~/.cache/ghtkn/agent.sock"]`. Linux: `network.allowAllUnixSockets: true` (seccomp can't filter by path). The agent owns the token lifecycle and runs outside the sandbox, so no allowed domains and no write access are needed. Before v0.3.4 the client minted tokens itself, so minting from the sandbox also needs the network settings below.
- `keyring` / `text`: reading a cached token works with no settings. Storing one needs `filesystem.allowWrite` (`~/Library/Keychains`, `~/.cache/ghtkn/tokens`), but that only happens via the interactive device flow, so run `ghtkn auth` outside the sandbox instead.
- Reaching GitHub from the sandboxed client (minting on `keyring`/`text`, `ghtkn revoke`) needs `network.allowedDomains: ["github.com", "api.github.com"]` to avoid the per-host prompt. On macOS it also fails TLS with `OSStatus -26276`, like every Go CLI: fix it with `excludedCommands: ["ghtkn *"]` or `enableWeakerNetworkIsolation`. The `agent` backend on v0.3.4 or later avoids both.

Sandbox settings are read at startup: restart Claude Code after changing them.

If this overview is enough, you don't need to read further.

## Reference

For details (per-backend settings, why the agent backend needs so little, and the macOS TLS issue), read [reference.md](reference.md) in this skill directory.
