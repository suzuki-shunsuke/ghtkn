---
name: ghtkn-design
description: Understand how ghtkn works internally and how its tokens compare to alternatives. Use to explain the token workflow, compare with GitHub CLI OAuth / fine-grained PAT / installation tokens, or discuss API rate limits.
---

ghtkn determines the GitHub App, gets an access token from the backend, regenerates it via Device Flow when needed, and outputs it. Compared to other access tokens:

- GitHub CLI OAuth token: convenient but broad and effectively indefinite (high leak risk).
- fine-grained PAT: long-lived and cumbersome to rotate.
- installation access token: can't act as a user and needs Private Key management.

User access tokens count toward the authenticated user's 5,000 req/hour rate limit.

If this overview is enough, you don't need to read further.

## Reference

For the detailed workflow, the full comparison, and rate-limit notes, read [reference.md](reference.md) in this skill directory.
