---
name: ghtkn-install
description: Install the ghtkn CLI and verify downloaded release assets. Use when installing ghtkn via Homebrew, Scoop, aqua, go install, or GitHub Releases, or verifying assets with gh / slsa-verifier / Cosign.
---

ghtkn is a single Go binary; install it into your `PATH`. Common methods:

- Homebrew: `brew install suzuki-shunsuke/ghtkn/ghtkn --cask`
- Scoop: `scoop bucket add suzuki-shunsuke https://github.com/suzuki-shunsuke/scoop-bucket && scoop install ghtkn`
- aqua: `aqua g -i suzuki-shunsuke/ghtkn` (aqua-registry >= v4.407.0)
- go install: `go install github.com/suzuki-shunsuke/ghtkn/cmd/ghtkn@latest`
- GitHub Releases: download an asset, unarchive, and put the binary in `$PATH`.

Release assets can be verified with GitHub CLI (`gh attestation verify`), slsa-verifier, or Cosign.

If this overview is enough, you don't need to read further.

## Reference

For details on each method and asset verification, read [reference.md](reference.md) in this skill directory.
