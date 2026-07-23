# Refreshing tokens

`>= 0.3.4`

ghtkn supports refreshing access tokens automatically with refresh tokens.
Device flow authentication is no longer needed every 8 hours, which greatly improves the UX.

This feature is supported only on the agent backend on macOS and Linux.
It is not supported on Windows, nor on the default keyring backend or the text backend.
This restriction is intentional, to keep the feature secure, and it is enforced rather than merely documented: on Windows `ghtkn agent unlock --enable-refresh` fails, and the agent refuses an unlock that asks to enable refresh tokens, so no client can turn the feature on there.
The reason is that the defenses described below are POSIX-specific: the `0600` permissions on the key and token files are effectively a no-op on Windows, and it has no equivalent of the `PR_SET_DUMPABLE` hardening.

We treat security as the top priority and take a range of measures to minimize the risk of a refresh token leaking.
Refresh tokens are stored encrypted by the agent backend, and the passphrase is never persisted.
A refresh token is held only inside ghtkn; there is no API that exposes it externally.
On Linux the agent also marks its process non-dumpable (`PR_SET_DUMPABLE=0`) at startup, so a same-user, non-root process cannot read its memory via `/proc/<pid>/mem` or ptrace and no core dump is written; macOS restricts this by default. This does not stop root, which is why the container/VM caveat below still applies.

The agent backend was originally developed to run ghtkn in environments where the OS keyring is
unavailable, and a normal desktop environment with a usable OS keyring was expected to use the
keyring backend. However, as noted above, only the agent backend supports refresh tokens, and for
security reasons there is no plan to support refresh tokens on the other backends, so the agent
backend may become the mainstream choice going forward, including on desktop environments.

## :warning: Caveats when running in a Linux container or VM

Do not use this feature in an environment where intruding malware can easily escalate to root.
A normal desktop environment usually requires a password, but development containers and VMs often run as root in the first place, or allow escalation to root via passwordless sudo.
Do not use it in such environments.

## The window enabling refresh widens, and how to limit it

Enabling refresh is a tradeoff worth understanding before you turn it on.

Everything here applies only while the agent is unlocked. While it is unlocked, a process running as your user (including malware that has landed in your session) can ask the agent for access tokens over the socket, whether or not refresh is enabled. If the agent is locked, nothing can be obtained. Enabling refresh widens that window in two ways.

For an app you are actively using, it does not change much. `ghtkn get` returns the cached access token when one is still valid (on average four hours of its eight-hour life remain), so an attacker that simply reads what `ghtkn get` returns gets the same short-lived token whether or not refresh is enabled. The token's life is only extended if the attacker deliberately forces a renewal (for example by running `ghtkn auth`), which is a targeted attack against ghtkn specifically; then it gains on average four and at most eight more hours.

For an app you have not used within the last eight hours, it changes more. Without refresh, that app's cached access token has expired, and renewing it needs the interactive device flow, so an attacker cannot obtain one. With refresh, the attacker can obtain a fresh access token as long as a valid refresh token is still stored, which is up to `--refresh-token-ttl` (three days by default, longer if you raise it). That TTL is therefore the window in which an idle app's token is reachable.

The default three-day TTL already keeps this window short, so for most people it needs no further action. If you want to reduce it further, you have a few options:

- Shorten `--refresh-token-ttl` below the three-day default. It leaves apps you use every few days untouched, because a token refreshed recently is not swept (see [refresh-token-ttl](#refresh-token-ttl-automatically-remove-unused-refresh-tokens-from-the-backend) below).
- Lock the agent when you will not need a token for a while, with `ghtkn agent lock`, which closes the window entirely until you unlock again. Because it needs no passphrase, it can be wired to a screen-lock or logout hook; re-enabling refresh on the next unlock needs `ghtkn agent unlock --enable-refresh` and the passphrase (see [Lock the agent to shrink the exposure window](../ghtkn-backend/reference.md#lock-the-agent-to-shrink-the-exposure-window)).
- If you want to close a particular app's window immediately instead of waiting for the TTL, you can revoke its token with `ghtkn revoke <app name>`, which revokes and deletes it so no refresh token is left to mint from (see [ghtkn revoke](../ghtkn-revoke-tokens/reference.md)). With the short default TTL this is rarely worth the trouble, and it sends you a notification email from GitHub, so treat it as an option for the extra-cautious rather than a routine step.

## Usage

1. Update the ghtkn CLI to v0.3.4 or later
1. Change the backend to the agent

```sh
export GHTKN_BACKEND=agent
```

Or

```yaml
# ~/.config/ghtkn/ghtkn.yaml
backend:
  type: agent
```

> [!WARNING]
> Update tools that depend on ghtkn-go-sdk, such as aqua and pinact, to their latest versions.
> Older SDKs either do not support the agent backend (supported since v0.3.0), or ignore the backend specified in the config file (respected since v0.4.0).
> If the SDK does not work correctly, check its version.

3. Run the ghtkn agent

```sh
ghtkn agent start
```

If the agent is already running, you need to stop and start it again to reflect the update of ghtkn and enable refresh tokens.

4. Pass the option `--enable-refresh` when you unlock the agent

```sh
ghtkn agent unlock --enable-refresh
```

You must pass `--enable-refresh` every time you run `agent unlock`.
If you omit it while the backend still holds a valid refresh token, `agent unlock` asks you to confirm before removing it:

```console
$ ghtkn agent unlock
Enter the agent passphrase:
Stored refresh tokens will be dropped (access tokens are kept; affected apps re-authenticate on next expiry). Rerun with --enable-refresh to keep them. Continue? (y/N):
```

Answer `y` to drop the refresh tokens and unlock with refresh disabled, or `N` (the default) to abort so you can rerun with `--enable-refresh` and keep them.
Only the refresh tokens are dropped; the access tokens stay, so the affected apps simply re-authenticate with the device flow when they next expire.
When no valid refresh token is stored, there is nothing to remove and no prompt appears.

Everything else works as before.

When refresh-token support is enabled and the access token has expired but a valid refresh token exists, `ghtkn get` and `ghtkn auth` refresh the token automatically instead of running the device flow.
When no valid refresh token exists, they run the device flow as before.
A refresh token is valid for six months.

## refresh-token-ttl: automatically remove unused refresh tokens from the backend

Even though they are encrypted, holding on to long-lived refresh tokens carries some risk.
For an access token you use infrequently, authenticating with the device flow each time is good enough without a refresh token.
So the agent periodically (every 24 hours) deletes access tokens and refresh tokens that have gone unused for a certain period, removing the whole file from the backend.
The period before deletion defaults to three days, and can be changed with the `--refresh-token-ttl` option of `ghtkn agent unlock`.
The value is a number followed by a `d` (day), `w` (week), or `m` (30-day month) suffix, e.g. `7d`, `4w`, `2m`.
Only these three units are accepted: a TTL is naturally measured in days, weeks, or months, so `m` always means a 30-day month here, and units that `time.ParseDuration` would otherwise accept, such as `h`/`m`/`s` (e.g. `720h` or `1m30s`), are rejected rather than silently reinterpreted (`m` as a minute).
Because a refresh token is valid for six months, you cannot specify a longer period.

`--refresh-token-ttl` applies only together with `--enable-refresh`. An unlock without `--enable-refresh` keeps no refresh token for the TTL to bound, so passing it there is an error rather than a silent no-op.

e.g. set it to four weeks

```sh
ghtkn agent unlock --enable-refresh --refresh-token-ttl=4w
```

When a refresh token that is still within its expiration fails to refresh, the response carries an incident warning (a possible-leak signal) that the client surfaces to the user.
