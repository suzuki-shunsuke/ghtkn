---
name: ghtkn-sandbox
description: Configure a coding agent's sandbox (Claude Code, Codex) to run ghtkn. Use when ghtkn fails inside the sandbox, e.g. the agent socket is "operation not permitted", a token can't be stored, or TLS fails with OSStatus -26276.
---

ghtkn doesn't work inside a coding agent's OS-level sandbox with the default settings. What it needs depends on the backend:

- `agent` (`>= 0.3.4`): allow the agent socket, and nothing else. The agent owns the token lifecycle and runs outside the sandbox, so no allowed domains and no write access are needed. Before v0.3.4 the client minted tokens itself, so minting from the sandbox also needs network access.
- `keyring` / `text`: reading a cached token needs little or nothing. Storing one needs write access to the keychain or the token directory, but that only happens via the interactive device flow, so run `ghtkn auth` outside the sandbox instead.
- Reaching GitHub from the sandboxed client (minting on `keyring`/`text`, `ghtkn revoke`) needs `github.com` and `api.github.com` allowed. On Claude Code this additionally fails TLS on macOS with `OSStatus -26276`, like every Go CLI; Codex doesn't have that problem. The `agent` backend on v0.3.4 or later avoids both.

The exact settings differ completely between tools - Claude Code uses `sandbox.*` in `settings.json`, Codex uses permissions profiles in `config.toml` - so read the file for the tool you're configuring rather than translating between them.

Both tools read their settings at startup: restart after changing them.

## Reference

Read the following files in this skill directory for the details:

- [claude_code.md](claude_code.md): read it to configure [Claude Code's sandbox](https://code.claude.com/docs/en/sandboxing) (`sandbox.enabled` in `settings.json`) - per-backend settings, why the agent backend needs so little, and the macOS TLS issue.
- [codex.md](codex.md): read it to configure [Codex](https://developers.openai.com/codex/security) (permissions profiles in `config.toml`) - per-backend settings, the difference between enabling a profile's network policy and allowlisting with the network proxy, what gates a project-scoped `.codex/config.toml`, and why `codex doctor` can't validate any of it.
