---
description: Run a command with ghtkn access tokens in environment variables using ghtkn exec. Use to keep tokens out of your output instead of ghtkn get, to give one command tokens of several apps, or to understand its exit codes and signal handling.
---

# Running Commands with `ghtkn exec`

`ghtkn exec` runs an external command with GitHub App User Access Tokens in its environment.

```sh
ghtkn exec [-config <config file>] [-log-level <level>] [-device-flow] [-min-expiration <duration>] [-continue-on-error] [-e <env name>[:<app name>]...] -- <command> [<args>...]
```

```sh
ghtkn exec -- gh pr view
ghtkn exec -e GH_TOKEN -- gh pr view
ghtkn exec -e PINACT_GITHUB_TOKEN:suzuki-shunsuke/read -e GH_TOKEN:suzuki-shunsuke/write -- bash foo.sh
```

## Why not `ghtkn get`

`ghtkn get` writes the token to stdout, from where it reaches terminal transcripts, logs, and chat messages. Coding agents have leaked tokens exactly that way. With `ghtkn exec`, ghtkn prints it nowhere: the token only exists in the environment of the command.

This prevents accidents rather than making the token secret. The command can still read the variable, `ghtkn exec -- env` prints it, and on Linux another process of the same user can read `/proc/<pid>/environ`. The exposure is the same as `GH_TOKEN=$(ghtkn get) gh ...`, minus the chance of the value landing in your output.

For git there is something better still: the [Git credential helper](git-credential-helper.md) lets git ask ghtkn for the token itself, so no token passes through your shell at all.

## `--` is required

The `--` between the flags of `ghtkn exec` and the command is mandatory. It is what makes the command's own flags reach the command:

```console
$ ghtkn exec -- gh pr view -R suzuki-shunsuke/ghtkn   # -R belongs to gh
$ ghtkn exec gh pr view                               # error: '--' is required
```

A `--` inside the command's own arguments is passed through as is, because only the first one separates the two:

```sh
ghtkn exec -- git log -- README.md
```

## Choosing environment variables and apps

Without `-e`, the access token of the app ghtkn selects automatically is set to `GITHUB_TOKEN`. App selection follows the same rules as `ghtkn get` and `ghtkn auth`: the `GHTKN_APP` environment variable, otherwise the first app in the configuration file. See [Using Multiple Apps](multiple-apps.md).

`-e` replaces that default entirely, so `GITHUB_TOKEN` is no longer set once any `-e` is given:

| Form | Meaning |
| --- | --- |
| `-e GH_TOKEN` | `GH_TOKEN` gets the token of the app ghtkn selects automatically |
| `-e GH_TOKEN:suzuki-shunsuke/write` | `GH_TOKEN` gets the token of the app `suzuki-shunsuke/write` |

`-e` is repeatable, which lets one command hold tokens of different apps, for instance a read-only token for one tool and a writable one for another. The name is cut at the first `:`, so an app name containing a colon still works. Note that `-e` takes an app name, not a value: `-e GH_TOKEN=xxx` is an error, and it tells you to use `:`.

An access token is created or read once per app, however many environment variables are bound to it, so several `-e` of one app never run the device flow twice. Tokens are acquired one at a time, in the order the `-e` were given.

An environment variable inherited from ghtkn's own environment is replaced rather than duplicated. Everything else is inherited unchanged: the rest of the environment, the working directory, and stdin, stdout and stderr. Nothing ghtkn does writes to stdout, not even while running the device flow, so the command's stdout carries the command's output alone.

## The device flow

`ghtkn exec` behaves like `ghtkn get`: the device flow is disabled by default, so an app whose cached token is missing or expiring fails instead of prompting. Enable it with `-device-flow` or `GHTKN_ENABLE_DEVICE_FLOW=true`, or simply run `ghtkn auth` beforehand. `-min-expiration` works as in `ghtkn get` too.

Note that the token is acquired once, before the command starts. A command running longer than the token's remaining lifetime will see it expire; `ghtkn exec` doesn't renew it in the meantime. Use `-min-expiration` to require a token that lives long enough for what you are about to run.

## When a token can't be acquired

By default the command is not run at all and ghtkn exits `111`.

With `-continue-on-error` the command runs anyway and a warning is logged. The environment variables whose tokens were acquired are still set; the others keep whatever value they had in ghtkn's environment. That last part matters: if `GITHUB_TOKEN` was already set to another credential, the command silently uses it. The warning says so.

`ghtkn info` reports `GH_TOKEN` and `GITHUB_TOKEN` (redacted) whenever they are set, so run it to find out whether such a credential is waiting to be inherited.

## Exit codes

| Code | Meaning |
| --- | --- |
| The command's own | The command ran |
| 128 + signal number | The command was killed by a signal, as in a shell. 130 is Ctrl-C, 143 SIGTERM, 137 SIGKILL |
| 111 | No access token could be acquired and the command was not run |
| 126 | The command exists but could not be executed |
| 127 | The command was not found |
| 1 | ghtkn itself failed, for instance on an invalid flag or configuration |

Nothing distinguishes a `111`, `126` or `127` produced by ghtkn from the same code returned by the command itself. The codes follow the shell's convention so that the usual scripts around them keep working.

## Signals and the process

On Linux and macOS, ghtkn does not stay around while the command runs. Once the tokens are in place it replaces its own process with the command through `execve(2)`, so there is no wrapper process at all:

```console
$ ghtkn exec -- sleep 30 &
$ ps -o comm= -p $!
sleep
```

The command therefore keeps ghtkn's process id, process group, session and terminal. Its exit code, the signals it receives and its job control are the ones it would have if you had run it directly, because it *is* the process you started. Nothing has to be forwarded or translated: Ctrl-C, Ctrl-Z, `kill`, `wait` and `$!` all behave as usual, and no wrapper can be left holding a command that ignores a signal.

Windows has no `execve(2)`, so there the command runs as a child of ghtkn and ghtkn waits for it. The signals ghtkn receives are forwarded to the command so that it is terminated the way it would be otherwise, and receiving the same signal a second time kills the command rather than leaving ghtkn waiting for one that ignores it. Ctrl-C is excluded from that escalation, since interactive commands such as `python` and `node` treat it as "cancel the current line" and keep running.

## Windows

Beyond running the command as a child rather than replacing ghtkn with it, three things differ on Windows, because the operating system does:

- Signals aren't really forwarded. The console delivers Ctrl-C and Ctrl-Break to the command itself, and `os.Process.Signal` accepts nothing but a kill there. The escalation doesn't apply either, because Ctrl-C is excluded from it and Windows delivers nothing else to ghtkn, so a command that ignores Ctrl-C has to be killed by other means.
- A command killed that way doesn't report a signal, so there is no 128 + signal number; the exit code Windows reports is used as is.
- There is no execute permission bit, so 126 doesn't occur.

## Wrapping a command

A wrapper keeps the token out of every invocation of a tool, not just the ones you remember:

```sh
gh() {
    ghtkn exec -e GH_TOKEN -- gh "$@"
}
```

This doesn't loop: `ghtkn exec` runs the `gh` executable found in `PATH`, and a shell function isn't visible to it. A wrapper *script* named `gh` in `PATH` is a different matter, because ghtkn would find the script again - give the real executable's absolute path there, as [the README](https://github.com/suzuki-shunsuke/ghtkn#wrapping-commands) does.
