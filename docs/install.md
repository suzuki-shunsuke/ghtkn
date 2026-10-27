---
description: Install the ghtkn CLI and verify downloaded release assets. Use when installing ghtkn via Homebrew, Scoop, aqua, go install, or GitHub Releases, or verifying assets with gh / Cosign / packslip / slsa-verifier.
---

# Install

ghtkn is written in Go. So you only have to install a binary in your `PATH`.

There are some ways to install ghtkn.

1. [Homebrew](#homebrew)
1. [Scoop](#scoop)
1. [aqua](#aqua)
1. [GitHub Releases](#github-releases)
1. [Build an executable binary from source code yourself using Go](#build-an-executable-binary-from-source-code-yourself-using-go)

## Homebrew

You can install ghtkn using [Homebrew](https://brew.sh/).

```sh
brew install suzuki-shunsuke/ghtkn/ghtkn --cask
```

## Scoop

You can install ghtkn using [Scoop](https://scoop.sh/).

```sh
scoop bucket add suzuki-shunsuke https://github.com/suzuki-shunsuke/scoop-bucket
scoop install ghtkn
```

## aqua

[aqua-registry >= v4.407.0 is required](https://github.com/aquaproj/aqua-registry/releases/tag/v4.407.0).

You can install ghtkn using [aqua](https://aquaproj.github.io/).

```sh
aqua g -i suzuki-shunsuke/ghtkn
```

## Build an executable binary from source code yourself using Go

```sh
go install github.com/suzuki-shunsuke/ghtkn/cmd/ghtkn@latest
```

## GitHub Releases

You can download an asset from [GitHub Releases](https://github.com/suzuki-shunsuke/ghtkn/releases).
Please unarchive it and install a pre built binary into `$PATH`. 

### Verify downloaded assets from GitHub Releases

You can verify downloaded assets using some tools.

1. [GitHub CLI](https://cli.github.com/)
1. [Cosign](https://github.com/sigstore/cosign)
1. [Packslip](https://packslip.dev/) `ghtkn >= v0.4.3`
1. [slsa-verifier](https://github.com/slsa-framework/slsa-verifier) `ghtkn <= v0.4.1`

### 1. GitHub CLI

You can install GitHub CLI by aqua.

```sh
aqua g -i cli/cli
```

```sh
version=v0.4.3
asset=ghtkn_darwin_arm64.tar.gz
gh release download -R suzuki-shunsuke/ghtkn "$version" -p "$asset"
gh attestation verify "$asset" \
  -R suzuki-shunsuke/ghtkn \
  --signer-workflow suzuki-shunsuke/go-release-workflow/.github/workflows/release.yaml
```

### 2. Cosign

You can install Cosign by aqua.

```sh
aqua g -i sigstore/cosign
```

From ghtkn v0.2.6, the bundle file is `ghtkn_checksums.txt.sigstore.json`.
Cosign v2.4.2 or later is required.

```sh
version=v0.4.3
checksum_file="ghtkn_checksums.txt"
asset=ghtkn_darwin_arm64.tar.gz
gh release download "$version" \
  -R suzuki-shunsuke/ghtkn \
  -p "$asset" \
  -p "$checksum_file" \
  -p "${checksum_file}.sigstore.json"
cosign verify-blob \
  "$checksum_file" \
  --bundle "${checksum_file}.sigstore.json" \
  --certificate-identity-regexp 'https://github\.com/suzuki-shunsuke/go-release-workflow/\.github/workflows/release\.yaml@.*' \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
cat "$checksum_file" | sha256sum -c --ignore-missing -
```

For ghtkn v0.2.5 or older, `*.pem` and `*.sig` are available instead.

```sh
version=v0.2.5
checksum_file="ghtkn_checksums.txt"
asset=ghtkn_darwin_arm64.tar.gz
gh release download "$version" \
  -R suzuki-shunsuke/ghtkn \
  -p "$asset" \
  -p "$checksum_file" \
  -p "${checksum_file}.pem" \
  -p "${checksum_file}.sig"
cosign verify-blob \
  --signature "${checksum_file}.sig" \
  --certificate "${checksum_file}.pem" \
  --certificate-identity-regexp 'https://github\.com/suzuki-shunsuke/go-release-workflow/\.github/workflows/release\.yaml@.*' \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  "$checksum_file"
cat "$checksum_file" | sha256sum -c --ignore-missing -
```

### 3. Packslip

`ghtkn >= v0.4.3`

You can install packslip by aqua.

```sh
aqua g -i jdx/packslip
```

```sh
version=v0.4.3
asset=ghtkn_darwin_arm64.tar.gz
gh release download -R suzuki-shunsuke/ghtkn "$version" -p "$asset" -p packslip.sigstore.json
packslip verify packslip.sigstore.json \
    --pin ps1_5hcnzb5qr3f35iia3sbixht6u4 \
    --identity-prefix https://github.com/suzuki-shunsuke/go-release-workflow/.github/workflows/release.yaml@ \
    --artifact "$asset"
```

- `--pin` requires the release to be signed from the repository `suzuki-shunsuke/ghtkn`, identified by its repository ID rather than its name
- `--identity-prefix` requires the release to be signed by the release workflow of [suzuki-shunsuke/go-release-workflow](https://github.com/suzuki-shunsuke/go-release-workflow)

On success, packslip prints a line starting with `ok:`.
Please make sure it shows the project and version you meant, e.g. `github.com/suzuki-shunsuke/ghtkn 0.4.3`.

### 4. slsa-verifier

> [!WARNING]
> As of ghtkn v0.4.2, `multiple.intoto.jsonl` hasn't been published anymore.

You can install slsa-verifier by aqua.

```sh
aqua g -i slsa-framework/slsa-verifier
```

```sh
version=v0.4.1
asset=ghtkn_darwin_arm64.tar.gz
gh release download -R suzuki-shunsuke/ghtkn "$version" -p "$asset" -p multiple.intoto.jsonl
slsa-verifier verify-artifact "$asset" \
  --provenance-path multiple.intoto.jsonl \
  --source-uri github.com/suzuki-shunsuke/ghtkn \
  --source-tag "$version"
```
