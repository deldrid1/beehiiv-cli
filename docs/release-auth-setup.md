# Release Auth Setup

This file is the repo-native setup guide for rotating the release-publishing credentials used by GitHub Actions.

It is written to be easy for humans and agents to follow. Codex, Claude, OpenClaw, and similar agents can use this together with `gh` and [`scripts/setup-release-auth.sh`](../scripts/setup-release-auth.sh).

## What to rotate

Replace these two GitHub Actions secrets in `deldrid1/beehiiv-cli`:

- `HOMEBREW_TAP_TOKEN`
- `WINGET_PUBLISH_TOKEN`

Use separate fine-grained personal access tokens instead of one broad token.
GitHub personal access tokens are created in the GitHub web UI, then stored into this repository's Actions secrets with `gh`.

## Token 1: Homebrew

Use a fine-grained PAT with:

- Resource owner: `deldrid1`
- Repository access: `Only select repositories`
  Select `deldrid1/homebrew-tap`
- Repository permissions:
  `Contents: Read and write`

Suggested name:

- `beehiiv-cli-homebrew-tap`

Suggested creation link:

- [Create Homebrew PAT](https://github.com/settings/personal-access-tokens/new?name=beehiiv-cli-homebrew-tap&description=Publish+Homebrew+formula+updates+for+beehiiv-cli&target_name=deldrid1&contents=write)

## Token 2: winget

Use a fine-grained PAT with:

- Resource owner: `deldrid1`
- Repository access: `Only select repositories`
  Select `deldrid1/winget-pkgs`
- Repository permissions:
  `Contents: Read and write`
  `Pull requests: Read and write`

Suggested name:

- `beehiiv-cli-winget-publish`

Suggested creation link:

- [Create winget PAT](https://github.com/settings/personal-access-tokens/new?name=beehiiv-cli-winget-publish&description=Push+winget+manifests+and+open+PRs+for+beehiiv-cli&target_name=deldrid1&contents=write&pull_requests=write)

## Apply the tokens

Once you have both token values, run:

```bash
HOMEBREW_TAP_TOKEN=YOUR_NEW_HOMEBREW_TOKEN \
WINGET_PUBLISH_TOKEN=YOUR_NEW_WINGET_TOKEN \
./scripts/setup-release-auth.sh
```

That script updates:

- Actions variables for the Homebrew tap
- Actions secret `HOMEBREW_TAP_TOKEN`
- Actions secret `WINGET_PUBLISH_TOKEN`

## Verify

After rotating the tokens:

1. Trigger `Homebrew Bump` for the latest tag and confirm it succeeds.
2. Trigger `Package Install` for the latest tag and confirm both jobs succeed.
3. On the next release tag, confirm:
   - the GitHub Release succeeds
   - the Homebrew tap updates
   - a branch is pushed to `deldrid1/winget-pkgs`
   - a PR opens against `microsoft/winget-pkgs`

## Why this file exists

This repo uses GitHub Actions, `gh`, and release automation that multiple agents can drive. A plain markdown guide plus a shell script is more portable than a tool-specific skill.

If this process becomes useful across several repos, then it would make sense to extract it into a reusable agent skill or publish it to an agent marketplace.

## References

- GitHub fine-grained PAT docs: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token
- GitHub PAT permissions reference: https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens
