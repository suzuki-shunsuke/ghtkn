---
description: Compare ghtkn with GitHub CLI's expiring, auto-refreshing OAuth tokens (gh auth login --expiring). Use when asked whether GitHub CLI's expiring tokens make ghtkn unnecessary, or to weigh gh's token storage, scopes, and organization control against ghtkn's.
---

# Advantages over GitHub CLI's expiring tokens

GitHub CLI can issue expiring, auto-refreshing OAuth access tokens, opted into with `gh auth login --expiring` or `gh config set expiring_credentials enabled`.
It was released as an opt-in public preview in gh v2.103.0 ([changelog](https://github.blog/changelog/2026-10-21-github-cli-opt-in-to-expiring-auto-refreshing-oauth-tokens/), [docs](https://docs.github.com/github-cli/github-cli/authenticating-with-github-cli#using-expiring-refreshable-tokens)), and the changelog says it will become the default for new sign-ins in a future release.

Short-lived tokens are one of the problems ghtkn solves, and official support is welcome: it reduces the impact of a leaked non-expiring token.
It does not make ghtkn unnecessary, though, because it only changes the token's lifetime.
This document compares the two. The description of GitHub CLI is based on gh v2.103.0 (October 2026) and may become outdated in later versions; the research behind it is in [#651](https://github.com/suzuki-shunsuke/ghtkn/issues/651).

## How GitHub CLI's expiring tokens work

`--expiring` adds the `offline_access` scope to the OAuth request, and the server may then issue an expiring access token together with a rotating, single-use refresh token.
gh refreshes the access token transparently wherever it is consumed: API calls, `gh auth token`, `gh auth status`, and the git credential helper.
To keep concurrent gh processes from spending the same refresh token twice, refreshes are serialized with an in-process mutex and a best-effort file lock.

## Differences

### The client is a privileged OAuth app

The tokens are still issued to the GitHub CLI OAuth app, which is one of GitHub's [privileged OAuth apps](https://docs.github.com/en/apps/oauth-apps/using-oauth-apps/privileged-oauth-apps).
According to that document, users can authorize and use a privileged app even when an organization has restricted OAuth app access to its data, so an organization cannot disable GitHub CLI that way.
Its use is recorded in the user's security log but not in the organization's audit log.

ghtkn uses a GitHub App that you or your organization create and manage, and the organization controls where it is installed.

### Scopes and permissions are unchanged

The minimum scopes are still `repo`, `read:org`, and `gist`; `--scopes` can add scopes but cannot go below that minimum.
The token can reach every repository the user can access, with the same broad permissions as before.

With ghtkn, the token is limited to the repositories the GitHub App is installed on and to the permissions the App is granted.

### How the refresh token is stored

gh stores the access token and the refresh token in the OS keyring.
If writing to the keyring fails, it falls back to plain text in `hosts.yml` and only prints a warning; `--insecure-storage` writes plain text on purpose.
On headless Linux or in containers without a keyring, plain text is what you are likely to get.
There is no encryption beyond the keyring, no passphrase, and no way to lock the credential.
On macOS the keyring is accessed through `/usr/bin/security`, so any process running as the same user can read the item without a confirmation dialog; this is also why ghtkn's keyring backend does not support refresh tokens.

So if the user account is compromised, the refresh token itself can be exfiltrated and used off the machine to keep minting access tokens.

ghtkn stores refresh tokens only in the agent backend, encrypted with AES-256-GCM using a data key that is itself wrapped by a key derived from your passphrase (the passphrase is never saved to disk), and offers no API that returns the refresh token.
See [Refreshing tokens](refresh-token.md) and [Backend](backend.md).

### Leak detection

Because refresh tokens rotate, if an attacker uses one first, the user's next refresh is rejected.
gh then deletes the credential and prints only `the token for <host> has expired; please run 'gh auth login' to re-authenticate`, with no hint that the token may have leaked, and the deleted credential leaves nothing to investigate.

ghtkn's agent emits an incident warning when a refresh token that is still within its expiration fails to refresh.

### No guarantee that an expiring token is issued

`--expiring` is only a request. If the server does not support expiring credentials, gh falls back to a non-expiring token.
The documentation says only that the access token expires "within hours", and gives no lifetime for the refresh token.

A ghtkn access token expires after eight hours.

## Comparison table

| Aspect | GitHub CLI (`--expiring`) | ghtkn |
| --- | --- | --- |
| Client | GitHub CLI's OAuth app (a GitHub-owned privileged app) | A GitHub App created and managed by the user or organization |
| Organization control | Exempt from OAuth app access restrictions, so it can't be disabled; not recorded in the audit log | The organization manages the App's installations |
| Access scope | Every repository the user can access | Can be limited to the repositories the App is installed on |
| Permissions | `repo`, `read:org`, `gist` at minimum | Can be narrowed to the minimum needed via the App's permissions |
| Access token lifetime | "Within hours" per the docs; up to the server (no guarantee it expires) | 8 hours |
| Refresh token storage | OS keyring; plain text in `hosts.yml` on keyring failure or with `--insecure-storage` | Agent backend only, stored encrypted |
| Encryption at rest | None (relies on the keyring) | AES-256-GCM; the passphrase is never persisted |
| API to extract the refresh token | Readable directly from storage | None |
| Locking | None | `ghtkn agent lock` makes tokens unobtainable altogether |
| Removal of unused tokens | None | Removed automatically after `--refresh-token-ttl` (7 days by default) |
| Leak detection | Only deletes the credential and prompts re-login | Emits an incident warning when a refresh token still within its expiry is rejected |
| Memory protection | None | `PR_SET_DUMPABLE=0` on Linux |
| Supported OS | All | Refresh on macOS and Linux only (intentionally unsupported on Windows, where the defenses don't hold) |

## What ghtkn does not protect against

Don't overstate the difference. While the ghtkn agent is unlocked, a process running as the same user can still obtain access tokens (see [Refreshing tokens](refresh-token.md)), so it is not true that tokens can't leak when the user account is compromised.
The precise difference is:

- ghtkn: only access tokens valid for at most eight hours can be taken, never the refresh token. Locking the agent, the TTL-based removal, and incident warnings shorten the window an attacker has.
- GitHub CLI: the refresh token itself can be taken from a plain-text file or from a keyring readable without confirmation, so it can be used off the machine for a long time.
