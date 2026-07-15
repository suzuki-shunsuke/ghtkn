# Sandbox Configuration

## Claude Code Settings

[Claude Code](https://code.claude.com/docs/en/sandboxing) can run Bash commands inside an OS-level sandbox that restricts filesystem and network access.
ghtkn doesn't work inside it with the default settings.
This page describes the minimum settings ghtkn needs, per backend.

> [!NOTE]
> This page is about the sandbox built into Claude Code, which is enabled by `sandbox.enabled` in `settings.json`.
> Sandbox settings are read when Claude Code starts, so restart it after changing them. Editing `settings.json` in a running session has no effect.

What you need depends on the backend, and the difference is large.
The `agent` backend needs the least: the agent process runs outside the sandbox and does the network access and the file writes on the sandboxed client's behalf.

### agent Backend

Allow the agent socket.
On ghtkn v0.3.4 or later, that is all you need; older versions need more when a token has to be minted (see [Older versions](#older-versions-of-ghtkn)).

On macOS:

```json
{
  "sandbox": {
    "enabled": true,
    "network": {
      "allowUnixSockets": ["~/.cache/ghtkn/agent.sock"]
    }
  }
}
```

Without this, every command that talks to the agent fails:

```console
$ ghtkn agent status
ERR ghtkn failed error="query the agent status: connect to the ghtkn agent: dial unix /Users/foo/.cache/ghtkn/agent.sock: connect: operation not permitted"
```

Allow the path the agent actually uses (see [Socket path](../ghtkn-backend/reference.md#socket-path)); `~/` is expanded.
If you set `GHTKN_AGENT_SOCKET` or `XDG_RUNTIME_DIR`, allow that path instead.

On Linux, allow Unix sockets entirely:

```json
{
  "sandbox": {
    "enabled": true,
    "network": {
      "allowAllUnixSockets": true
    }
  }
}
```

`allowUnixSockets` is macOS-only. On Linux the sandbox blocks Unix sockets with a seccomp filter that rejects `socket(AF_UNIX, ...)`, and seccomp can't read the socket path, so the allowlist can't be per-path: it's all or nothing.

Weigh that before turning it on. Claude Code's documentation warns that allowing Unix sockets can hand a sandboxed command powerful system services, `/var/run/docker.sock` being the example it gives, since reaching the Docker socket effectively means reaching the host. On macOS you allow one socket that only serves ghtkn tokens; on Linux you allow every socket on the machine.

The Linux seccomp filter is also optional: it ships with `@anthropic-ai/sandbox-runtime`, and the Dependencies tab of `/sandbox` shows whether it is present. Without it Unix sockets aren't blocked at all, so the agent backend works with no settings, but nothing else is blocked either.

#### Why the agent backend needs nothing else

`>= 0.3.4`

`ghtkn get` and `ghtkn auth` don't reach GitHub from inside the sandbox.
The agent owns the token lifecycle: it runs the device flow and the token refresh itself, and the client only asks it to begin and then polls, over the socket.
The encryption key and the encrypted token files are read and written by the agent process too.

So the sandboxed side needs only the configuration file, which is readable by default, and the socket.
No allowed domains, no write access, and none of the settings that weaken the sandbox.

This assumes the agent is already running and unlocked outside the sandbox.
`ghtkn agent unlock` needs a terminal, so run it in your own shell rather than through a coding agent.

#### Older versions of ghtkn

Before v0.3.4 the client, not the agent, ran the device flow and then stored the minted token in the agent over the socket.
There was no token refresh either, so a new token had to be minted through the device flow every 8 hours.

Reading a cached token still needs only the socket, so a coding agent that just uses a token you minted works with the settings above.
But minting a token from inside the sandbox reaches GitHub from the client, so it additionally needs [`network.allowedDomains`](#network) and, on macOS, [the trust service workaround](#tls-verification-fails-on-macos).

The simpler answer is the same as for the other backends: run `ghtkn auth` in your own terminal, and let the sandboxed commands read what it cached.
Upgrading to v0.3.4 or later removes the question entirely.

### keyring Backend

The keyring backend reaches the OS keyring in a completely different way on each platform, so what the sandbox allows differs too.

#### macOS

Reading a cached token works with no settings.

That is more surprising than it sounds, so it's worth saying why. macOS enforces the sandbox with Seatbelt, and the profile Claude Code generates doesn't restrict which programs run: it allows `process-exec` unconditionally, so the `security` command that the keyring library shells out to starts normally. It restricts writes and network instead. The profile also allows the Mach lookups that reach the keychain (`com.apple.SecurityServer` and `com.apple.securityd.xpc`) out of the box, and the keychain file itself is readable under the default read policy described in the text backend section below.

Storing a token doesn't work, because the keychain lives under `~/Library/Keychains`, which the sandbox doesn't allow writing:

```console
$ security add-generic-password -s "github.com/suzuki-shunsuke/ghtkn" -a "$CLIENT_ID" -w "$TOKEN"
security: SecKeychainItemCreateFromContent (<default>): UNIX[Operation not permitted]
```

In practice this rarely matters. Storing a token means minting one, which means the device flow, which is interactive and disabled by default.
Run `ghtkn auth` in your own terminal and let the sandboxed commands read the token it caches.

To let a sandboxed command store tokens, allow the keychain directory:

```json
{
  "sandbox": {
    "filesystem": {
      "allowWrite": ["~/Library/Keychains"]
    }
  }
}
```

This lets every sandboxed command modify your login keychain, not just ghtkn, so prefer running `ghtkn auth` outside the sandbox.

#### Linux

Nothing works without settings. The keyring backend talks to the Secret Service over the D-Bus session bus, which is a Unix socket, and the sandbox blocks those, so even reading a cached token fails. Allowing it means `network.allowAllUnixSockets: true`, with the caveats described in the agent backend section above.

In the environments where the Linux keyring is a problem to begin with (containers, microVMs), use the `agent` or `text` backend instead.

### text Backend

Reading a cached token works with no settings, on both macOS and Linux. The sandbox is asymmetric: writes are an allowlist that starts at the working directory and `$TMPDIR`, while reads are a denylist that starts at the whole machine. The token directory isn't on the default deny list, so it's readable.

That holds only for the default read policy, though. Reads are denied by `filesystem.denyRead`, `credentials.files`, and `Read` deny permission rules, which all merge into one list, so a setup that denies `~/` or a pattern the token path happens to match needs the directory re-allowed with `filesystem.allowRead`.

Writing needs the token directory:

```json
{
  "sandbox": {
    "filesystem": {
      "allowWrite": ["~/.cache/ghtkn/tokens"]
    }
  }
}
```

Allow the directory, not the file.
The text backend writes a token by creating a temporary file next to it and renaming it into place, so allowing only `<dir>/<client-id>` isn't enough.

Allow the path the backend actually resolves (see [text Backend](../ghtkn-backend/reference.md#text-backend)).
If you set `GHTKN_TEXT_BACKEND_DIR` or `XDG_CACHE_HOME`, allow that path instead.

As with the keyring backend, writes only happen when a token is minted, so read-only use needs no settings.

### Network

The `agent` backend on v0.3.4 or later needs no allowed domains, as explained above.

Every other case needs them whenever the client itself reaches GitHub: minting a token with the device flow (`keyring`, `text`, and `agent` before v0.3.4), and `ghtkn revoke` on any backend.

| Host | Purpose |
| --- | --- |
| `github.com` | Device flow and token refresh |
| `api.github.com` | `ghtkn revoke` |

```json
{
  "sandbox": {
    "network": {
      "allowedDomains": ["github.com", "api.github.com"]
    }
  }
}
```

No domains are allowed up front, but a host you haven't listed isn't simply refused: Claude Code prompts for approval the first time a command needs it, and approving it allows the host for the rest of the session. Listing them here just avoids the prompt. If an administrator sets `network.allowManagedDomainsOnly` in managed settings, the list becomes a hard allowlist and non-allowed hosts are blocked instead of prompted.

On macOS, allowing the hosts isn't enough. See below.

Note that ghtkn working in the sandbox doesn't make `git push` or `gh` work: they reach GitHub themselves and need these hosts regardless of the backend.
`gh` is a Go program, so it hits the same macOS problem described next.

### TLS verification fails on macOS

This section is macOS-only, and applies only when the client itself reaches GitHub, so the `agent` backend on v0.3.4 or later never hits it.

```
Post "https://api.github.com/credentials/revoke": tls: failed to verify certificate: x509: OSStatus -26276
```

This isn't a problem with `allowedDomains`.
On macOS, Go doesn't verify certificates itself: it calls the platform verifier, which talks to the system trust service (`com.apple.trustd.agent`) over Mach IPC, and the sandbox blocks that lookup.
It affects every Go-based CLI. Claude Code's [Troubleshooting](https://code.claude.com/docs/en/sandboxing#troubleshooting) lists the same symptom for `gh`, `gcloud`, and `terraform`.

Note the contrast with the keyring backend above: the sandbox allows the keychain's Mach services by default, and `com.apple.trustd.agent` is the one it deliberately withholds until you ask for it.

Claude Code's documentation recommends running the affected tool outside the sandbox:

```json
{
  "sandbox": {
    "excludedCommands": ["ghtkn *"]
  }
}
```

This is the simplest option, and for ghtkn it works better than it does for some other tools, because ghtkn is normally invoked under its own name. Two caveats. The pattern is matched against the command string, so it doesn't cover ghtkn started by something else: `git push` runs sandboxed and so does the `ghtkn git-credential` that Git spawns from it. And the exclusion applies to the whole command, so `ghtkn revoke ... && rm -rf ~/.config` would run entirely outside the sandbox.

The other option is to let the sandbox reach the trust service:

```json
{
  "sandbox": {
    "enableWeakerNetworkIsolation": true
  }
}
```

`network.allowMachLookup: ["com.apple.trustd.agent"]` is equivalent. Claude Code's documentation presents `enableWeakerNetworkIsolation` for the case where you use `httpProxyPort` with a MITM proxy and a custom CA, but it fixes the plain case too: it grants exactly the lookup that is missing.

As the name says, it weakens the sandbox: the trust service fetches OCSP/CRL data over the network without going through the sandbox's proxy, so that traffic isn't covered by `allowedDomains`. Unlike `excludedCommands`, it applies to every command rather than the ones you name.

`GODEBUG=x509usefallbackroots=1` is not an alternative.
It only works for programs that embed a copy of the root certificates, as [aqua](https://aquaproj.github.io/docs/guides/claude-code-sandbox) does since v2.62.0, and ghtkn doesn't: per [`x509.SetFallbackRoots`](https://pkg.go.dev/crypto/x509#SetFallbackRoots), "Setting x509usefallbackroots=1 without calling SetFallbackRoots has no effect".
`SSL_CERT_FILE` is ignored on macOS for the same platform-verifier reason.

Using the `agent` backend on v0.3.4 or later avoids this entirely, which is one more reason to prefer it inside a sandbox.
