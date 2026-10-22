---
description: Why use ghtkn even though GitHub CLI now offers expiring, auto-refreshing OAuth tokens (gh auth login --expiring), and the risks that remain with them. Use when asked whether GitHub CLI's expiring tokens make ghtkn unnecessary, or when weighing the security of gh's tokens against ghtkn's.
---

# Advantages over GitHub CLI's expiring tokens

GitHub CLI can issue expiring, auto-refreshing OAuth access tokens, opted into with `gh auth login --expiring` or `gh config set expiring_credentials enabled`.
It was released as an opt-in public preview in gh v2.103.0 ([changelog](https://github.blog/changelog/2026-10-21-github-cli-opt-in-to-expiring-auto-refreshing-oauth-tokens/), [docs](https://docs.github.com/github-cli/github-cli/authenticating-with-github-cli#using-expiring-refreshable-tokens)), and the changelog says it will become the default for new sign-ins in a future release.

This is welcome: a leaked access token stops working within hours instead of never.
But it only shortens the access token's lifetime. Everything else about the token stays the same, and storing a long-lived refresh token on your machine adds a new risk.
ghtkn still has a clear advantage in security, and this document explains why.

The description of GitHub CLI is based on gh v2.103.0 (October 2026) and may become outdated in later versions; the research behind it is in [#651](https://github.com/suzuki-shunsuke/ghtkn/issues/651).

## Risks that remain with GitHub CLI's expiring tokens

### Your organization can't control or audit it

The tokens are still issued to the GitHub CLI OAuth app, one of GitHub's [privileged OAuth apps](https://docs.github.com/en/apps/oauth-apps/using-oauth-apps/privileged-oauth-apps).
According to that document, users can authorize and use a privileged app even when an organization has restricted OAuth app access to its data, so the organization can't turn GitHub CLI off.
Its use is recorded in the user's security log but not in the organization's audit log, so administrators can't see it being authorized or used.

### The token can reach everything you can, with broad permissions

The minimum scopes are still `repo`, `read:org`, and `gist`, and `--scopes` can add scopes but can't go below that minimum.
The token reaches every repository of every organization you belong to, so a leaked token exposes all of them at once.

### The refresh token can be stolen and used off your machine

gh stores the access token and the refresh token in the OS keyring.
If writing to the keyring fails, gh falls back to plain text in `hosts.yml`, prints a one-line warning, and the login succeeds anyway. On headless Linux or in containers without a keyring, this is what you are likely to get, and it is easy to miss.
On macOS, the keyring is accessed through `/usr/bin/security`, so any process running as your user can read the item without a confirmation dialog.

So if your user account is compromised, for example by a malicious package in a supply chain attack, the refresh token itself can be exfiltrated.
Unlike an access token, it keeps minting new access tokens off your machine until it is revoked or rejected, which makes the short access-token lifetime much less meaningful.

### A leak goes unnoticed

Refresh tokens rotate, so if an attacker uses yours first, your next refresh is rejected.
gh then deletes your credential and prints only `the token for <host> has expired; please run 'gh auth login' to re-authenticate`.
Nothing tells you the token may have leaked, and the deleted credential leaves nothing to investigate.

### In some cases the token doesn't expire

`--expiring` is only a request, and gh falls back to a non-expiring token without failing when it can't get an expiring one.
On github.com and ghe.com this rarely matters, but it happens in two cases:

- GitHub Enterprise Server older than 3.23, which doesn't support expiring credentials.
- Pasting a token into `gh auth login` instead of signing in through the browser. The pasted token is stored as is, even with `expiring_credentials` enabled.

Run `gh auth status` to check whether your token expires: it shows when an expiring access token expires.
Even when you do get an expiring token, the documentation says only that the access token expires "within hours", and gives no lifetime for the refresh token.

## Why use ghtkn

### Least privilege, under your organization's control

ghtkn issues user access tokens from a GitHub App that you or your organization create and manage.
A token can only reach the accounts and repositories the App is installed on, such as a single organization, and only with the permissions the App is granted.
You can keep several Apps with different permissions and pick one per task (see [Using Multiple Apps](multiple-apps.md)), and the organization decides where its Apps are installed.

### Tokens are never stored in plain text unless you choose to

ghtkn stores access tokens in plain text only when you explicitly select the text backend (see [Backend](backend.md)).
Refresh tokens are supported only on the agent backend, so they are never stored in plain text.
The agent encrypts tokens with AES-256-GCM using a data key that is itself wrapped by a key derived from your passphrase, and the passphrase is never saved to disk.
There is no API that returns a refresh token: a client can only obtain an access token.

### A compromise has a short, controllable window

What an attacker on your account can take from ghtkn is an access token valid for at most eight hours, never the refresh token, so they can't keep minting tokens off your machine.
ghtkn also narrows the window in which even that is possible (see [Refreshing tokens](refresh-token.md)):

- `ghtkn agent lock` makes tokens unobtainable altogether until you unlock the agent with your passphrase.
- Tokens that have gone unused for `--refresh-token-ttl` (seven days by default) are removed from the backend automatically.
- On Linux, the agent marks its process non-dumpable (`PR_SET_DUMPABLE=0`), so a same-user, non-root process can't read its memory.

### A suspected leak is reported

When a refresh token that is still within its expiration fails to refresh, the agent returns an incident warning that the client shows to you, instead of silently asking you to log in again.

### Your tools can share the token

Tools such as aqua, pinact, ghir, and ghaperf embed the ghtkn Go SDK and get tokens from ghtkn directly, so you don't need a PAT for them (see [Go SDK](go-sdk.md)).

## Comparison table

| Aspect | GitHub CLI (`--expiring`) | ghtkn |
| --- | --- | --- |
| Client | GitHub CLI's OAuth app (a GitHub-owned privileged app) | A GitHub App created and managed by the user or organization |
| Organization control | Exempt from OAuth app access restrictions, so it can't be disabled; not recorded in the audit log | The organization manages the App's installations |
| Access scope | Every repository the user can access | Can be limited to the repositories the App is installed on |
| Permissions | `repo`, `read:org`, `gist` at minimum | Can be narrowed to the minimum needed via the App's permissions |
| Access token lifetime | "Within hours" per the docs; non-expiring on GHES older than 3.23 or with a pasted token | 8 hours |
| Refresh token storage | OS keyring; plain text in `hosts.yml` on keyring failure or with `--insecure-storage` | Agent backend only, stored encrypted |
| Encryption at rest | None (relies on the keyring) | AES-256-GCM; the passphrase is never persisted |
| API to extract the refresh token | Readable directly from storage | None |
| Locking | None | `ghtkn agent lock` makes tokens unobtainable altogether |
| Removal of unused tokens | None | Removed automatically after `--refresh-token-ttl` (7 days by default) |
| Leak detection | Only deletes the credential and prompts re-login | Emits an incident warning when a refresh token still within its expiry is rejected |
| Memory protection | None | `PR_SET_DUMPABLE=0` on Linux |
| Supported OS | All | Refresh on macOS and Linux only (intentionally unsupported on Windows, where the defenses don't hold) |

## What ghtkn does not protect against

Don't overstate the advantage. While the ghtkn agent is unlocked, a process running as your user can still obtain access tokens (see [Refreshing tokens](refresh-token.md)), so it is not true that tokens can't leak when your user account is compromised.
The precise difference is:

- ghtkn: only access tokens valid for at most eight hours can be taken, never the refresh token. Locking the agent, the TTL-based removal, and incident warnings shorten the window an attacker has.
- GitHub CLI: the refresh token itself can be taken from a plain-text file or from a keyring readable without confirmation, so it can be used off the machine for a long time.
