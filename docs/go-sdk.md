---
description: Use the ghtkn Go SDK, which lets tools such as aqua, pinact, ghir, and ghaperf reuse ghtkn tokens without a PAT. Use when a tool's ghtkn integration doesn't kick in, when deciding what enables it (GHTKN_ENABLE, the config file), or when embedding ghtkn in your own Go CLI.
---

# Go SDK

[ghtkn Go SDK](https://pkg.go.dev/github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn) lets a Go application create and reuse GitHub App User Access Tokens the same way the ghtkn CLI does.
ghtkn itself is built on it, so a tool using the SDK reads the same configuration file, the same [backend](backend.md), and the same cached tokens as the `ghtkn` command.

There are two sides to this document: using a tool that already embeds the SDK, and embedding it in your own CLI.

## Tools that already use the SDK

[aqua](https://aquaproj.github.io/), [pinact](https://github.com/suzuki-shunsuke/pinact), [ghir](https://github.com/suzuki-shunsuke/ghir), and [ghaperf](https://github.com/suzuki-shunsuke/ghaperf) embed the SDK.
They obtain a ghtkn access token themselves, so you don't have to set `GITHUB_TOKEN` for them or wrap them in [`ghtkn exec`](exec.md).
Nothing has to be wired up: install ghtkn, run `ghtkn auth` once, and these tools authenticate as you with a short-lived token.

## When the integration is enabled

A tool asks the SDK whether the ghtkn integration should be enabled (`ghtkn.Enabled`), and the answer is resolved in this order:

1. the first environment variable the tool itself nominates that is set, if it defines one (for instance a tool-specific `<TOOL>_GHTKN_ENABLE` switch - check the tool's own documentation),
1. the `GHTKN_ENABLE` environment variable,
1. whether the [ghtkn configuration file](configuration.md) exists.

For the first two, the value must be a boolean (`true`, `false`, `1`, `0`); anything else is an error rather than a silent "disabled".
The third step is why the integration usually needs no setup at all: once `ghtkn init` has created `ghtkn.yaml`, it turns on by itself.
It is also how you turn it off without deleting anything - `GHTKN_ENABLE=false` disables ghtkn for every tool that uses the SDK.

Being enabled is not the same as running the device flow.
The device flow is disabled by default in the SDK just as it is in `ghtkn get`, so a tool fails rather than prompting when there is no usable token.
Run [`ghtkn auth`](token-management.md) yourself beforehand; that is the interactive part, and it belongs in your terminal, not inside another tool's run.

## When the integration doesn't work

Almost always a version is old, on one side or the other. Update both the tool that embeds the SDK and the ghtkn CLI to their latest versions.
The SDK is a separate module from the CLI, so a tool keeps whatever SDK version it was built with until it is rebuilt and released - a current `ghtkn` binary does not upgrade the SDK inside `aqua` or `pinact`.

Version boundaries worth knowing:

- SDK `>= v0.3.0` - `ghtkn.Enabled` exists, so the integration turns on automatically when the configuration file exists. Older SDKs required the tool's own switch.
- SDK `>= v0.3.0` - the [agent backend](backend.md) is supported. Older SDKs cannot read tokens from it.
- SDK `>= v0.4.0` - the backend set in the configuration file is respected. Older SDKs ignore it, so a tool may look in the keyring while ghtkn stores tokens in the agent.

If the tool sees no token while `ghtkn get` works, compare what each one uses: `ghtkn info` prints the resolved configuration and every `GHTKN_*` variable ghtkn reads. See [Troubleshooting](troubleshooting.md).

## Embedding the SDK in your own CLI

The API reference is the [SDK's Go documentation](https://pkg.go.dev/github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn), and runnable examples live in [the SDK repository](https://github.com/suzuki-shunsuke/ghtkn-go-sdk/tree/main/examples).
Both ship with the SDK, so they describe the version you actually depend on; this document deliberately doesn't restate them.

Two things are worth knowing before you read them, because they are decisions ghtkn made rather than API details:

- The device flow is disabled unless you enable it, so a token that has to be created makes `Get` fail instead of prompting. Tell the user to run `ghtkn auth` rather than starting an interactive flow from inside your tool.
- The access token you get back is a live secret. Don't print it, log it, or include it in an error message. See [Token Management](token-management.md).
