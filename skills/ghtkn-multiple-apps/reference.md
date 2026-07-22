# Using Multiple Apps

You can configure multiple GitHub Apps in the `apps` section of the configuration file and create and use different Apps for each Organization or User.
By default, the first App in `apps` is used.

Each app must have a unique `name`, `client_id`, and `git_owner`.
`client_id` identifies the GitHub App everywhere below the configuration: the stored access token, its refresh, and its revocation are all keyed by it.
Two apps sharing one client id would therefore be two names for a single access token, where revoking or minting for one silently does it for the other, so ghtkn rejects that configuration.
One app entry is enough to reach every account the App is installed on; to use it for several repository owners, select it with `GHTKN_APP` or `GHTKN_GIT_APP` rather than adding a second entry with the same `client_id`.

You can specify the App by command line argument:

```sh
ghtkn get suzuki-shunsuke/write
```

The value is the app name defined in the configuration file.
Alternatively, you can specify it with the environment variable `GHTKN_APP`.
For example, it might be convenient to switch `GHTKN_APP` for each directory using a tool like [direnv](https://direnv.net/).

I check out my repositories from [https://github.com/suzuki-shunsuke](https://github.com/suzuki-shunsuke) into the `~/repos/src/github.com/suzuki-shunsuke` directory.
I then place a `.envrc` file in that directory with the following content:

```sh
source_up

export GHTKN_APP=suzuki-shunsuke/write
```

Similarly, I place a `.envrc` file in `~/repos/src/github.com/aquaproj` as well:

```sh
source_up

export GHTKN_APP=aquaproj/write
```

I've also set up a default App that has no permissions.
While some might think an access token with no permissions is useless, it can still be used to read public repositories and helps you avoid hitting API rate limits compared to not using an access token at all.
So, it's quite useful.

```yaml
apps:
  - name: suzuki-shunsuke/none
    client_id: xxx
```

With this setup, the access token is transparently switched depending on the working directory. What's written in the `.envrc` is the `GHTKN_APP`, not the access token itself, which is safe because it's not a secret.
