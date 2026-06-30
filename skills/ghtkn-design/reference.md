# How ghtkn Works and Design

## How does ghtkn work?

ghtkn gets and outputs an access token in the following way:

1. Read command line options and environment variables
2. Read a configuration file. It has pairs of app name and client id
3. Determine the GitHub App (see the multiple apps reference)
4. Get the client id from the configuration file
5. Get the access token by client id from the backend
6. If the access token isn't found in the backend or the access token expires, [creating a new access token through Device Flow. A user need to input the verification code and approve the request](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app#using-the-device-flow-to-generate-a-user-access-token)
7. Get the authenticated user login by GitHub API for Git Credential Helper
8. Store the access token, expiration date, and authenticated user login in the backend
9. Output the access token

## Comparison between GitHub App User Access Token and other access tokens

### GitHub CLI OAuth App access token

https://cli.github.com/manual/gh_auth_token

This can be easily generated with `gh auth login`, `gh auth token` in GitHub CLI.
You don't need to generate Personal Access Tokens, and it's convenient.
Also, when scopes across Users or Organizations are needed, it's difficult with non-Public GitHub Apps, but installing GitHub CLI OAuth App across multiple Users or Organizations solves such problems.

However, this access token is not very good from a security perspective.
While you can restrict the scope (permission) and target Organizations, these tend to be quite broad for convenience.
Also, it's basically indefinite.
Therefore, the risk when this token is leaked is very high.

So, a more secure mechanism is needed.

### fine-grained Personal Access Token

https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens

We'll ignore Legacy PAT as it's almost the same as OAuth App tokens.

Fine-grained access tokens have the following disadvantages compared to User Access Tokens:

- Regular rotation is cumbersome
- Management is cumbersome
- High risk when leaked
  - While the validity period is not indefinite, it tends to be quite long
    - Since short periods make rotation cumbersome, it tends to be 1 year or 6 months
    - Not on the order of a few hours

### GitHub App installation access token

https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation

- Pros
  - Can change permissions, repositories, and validity period when generating tokens
- Cons
  - Cannot operate as a User
    - e.g., PR creator becomes the App
  - Private Key management is cumbersome
  - High risk when Private Key is leaked

## API rate limit

https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#primary-rate-limit-for-github-app-installations

> Primary rate limits for GitHub App user access tokens (as opposed to installation access tokens) are dictated by the primary rate limits for the authenticated user.
> This rate limit is combined with any requests that another GitHub App or OAuth app makes on that user's behalf and any requests that the user makes with a personal access token.
> For more information, see Rate limits for the REST API.

The rate limit for authenticated users is 5,000 per hour, so it should be fine for normal use.

> All of these requests count towards your personal rate limit of 5,000 requests per hour.
