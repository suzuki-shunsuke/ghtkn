# Git Credential Helper

`version >= v0.1.2`

You can use ghtkn as a [Git Credential Helper](https://git-scm.com/book/en/v2/Git-Tools-Credential-Storage):

```sh
git config --global credential.helper '!ghtkn git-credential'
```

```ini
[credential]
	helper =
	helper = !ghtkn git-credential
```

> [!IMPORTANT]
> `helper =` is necessary to disable other helpers.
> https://git-scm.com/docs/gitcredentials#_configuration_options
> > If credential.helper is configured to the empty string, this resets the helper list to empty
> > (so you may override a helper set by a lower-priority config file by configuring the empty-string helper, followed by whatever set of helpers you would like).

## Switching GitHub Apps by repository owner

If you want to switch GitHub Apps by repository owner,

1. Set `.apps[].git_owner` in a configuration file
1. Configure Git `git config credential.useHttpPath true`

```sh
git config --global credential.useHttpPath true
```

```yaml
apps:
  - name: suzuki-shunsuke/write
    client_id: xxx
    git_owner: suzuki-shunsuke # Using this app if the repository owner is suzuki-shunsuke
```

> [!WARNING]
> `git_owner` must be unique.
> Please set `git_owner` to only one app per repository owner (organization and user).
> For instance, if you use a read-only app and a write app for a repository owner and you want to push commits, you should set `git_owner` to the write app.
>
> ```yaml
> apps:
>   - name: suzuki-shunsuke/write
>     client_id: xxx
>     git_owner: suzuki-shunsuke # Using this app if the repository owner is suzuki-shunsuke
>   - name: suzuki-shunsuke/read-only
>     client_id: xxx
>     # git_owner: suzuki-shunsuke # Don't set `git_owner` to read-only app to push commits
> ```

### Switching GitHub Apps to access fork repositories

Unfortunately, `.apps[].git_owner` doesn't match when accessing fork repositories.
For instance, when you checkout a pull request from a fork repository by [gh pr checkout](https://cli.github.com/manual/gh_pr_checkout) command and push commits to the fork repository, `.apps[].git_owner` doesn't work unless you configure fork repositories in `ghtkn.yaml`.

As of ghtkn v0.2.6, the environment variable `GHTKN_GIT_APP` is useful.
`GHTKN_GIT_APP` is similar to `GHTKN_APP` (see the multiple apps reference) but it's used for Git Credential Helper.

e.g.

```sh
export GHTKN_GIT_APP=suzuki-shunsuke/git
```

The priority of the app used for Git Credential Helper is as follows:

1. `.apps[].git_owner` if git credential helper's username matches
1. `GHTKN_GIT_APP`
1. `GHTKN_APP` if `GHTKN_GIT_APP` is not set
1. The default app

## :warning: Troubleshooting of Git Credential Helper on macOS

If Git Credential Helper doesn't work on macOS, please check if osxkeychain is used.

You can check the trace log of Git by `GIT_TRACE=1 GIT_CURL_VERBOSE=1`.

```sh
GIT_TRACE=1 GIT_CURL_VERBOSE=1 git push origin
```

If git outputs the following log, Git uses `git-credential-osxkeychain`, not ghtkn.

```
09:25:49.373133 git.c:750               trace: exec: git-credential-osxkeychain get
09:25:49.373152 run-command.c:655       trace: run_command: git-credential-osxkeychain get
```

Please check the git config.

```sh
git config --get-all --show-origin credential.helper
```

The following output shows osxkeychain is used by the system setting `/Library/Developer/CommandLineTools/usr/share/git-core/gitconfig`.

```
file:/Library/Developer/CommandLineTools/usr/share/git-core/gitconfig   osxkeychain
file:/Users/shunsukesuzuki/.gitconfig   !ghtkn git-credential
```

To solve the problem, please set credential.helper to the empty string.

```ini
[credential]
	helper =
	helper = !ghtkn git-credential
```

https://git-scm.com/docs/gitcredentials#_configuration_options

> If credential.helper is configured to the empty string, this resets the helper list to empty
> (so you may override a helper set by a lower-priority config file by configuring the empty-string helper, followed by whatever set of helpers you would like).
