---
name: ghtkn-git-credential-helper
description: Configure ghtkn as a Git credential helper and switch GitHub Apps by repository owner. Use when setting up git authentication via ghtkn, handling fork repos, or troubleshooting the credential helper on macOS.
---

ghtkn can act as a [Git Credential Helper](https://git-scm.com/book/en/v2/Git-Tools-Credential-Storage):

```sh
git config --global credential.helper '!ghtkn git-credential'
```

Set an empty `helper =` first to disable other helpers. You can switch GitHub Apps per repository owner with `apps[].git_owner` (plus `git config credential.useHttpPath true`), or with the `GHTKN_GIT_APP` env var for fork repositories. On macOS, `git-credential-osxkeychain` may take precedence.

If this overview is enough, you don't need to read further.

## Reference

For details (owner-based switching, fork repositories, the app priority order, and macOS troubleshooting), read [reference.md](reference.md) in this skill directory.
