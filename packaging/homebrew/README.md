# Homebrew Publication Setup

This repo updates Homebrew through a dedicated tap workflow instead of GoReleaser's deprecated formula publisher. The release source of truth is still the GitHub Release: the tap formula is rendered from release metadata, tarball URLs, and `checksums.txt`.

## Required repository configuration

Set these repository variables and secrets before enabling tap publication:

- `HOMEBREW_TAP_REPOSITORY`
  Tap repository, such as `deldrid1/homebrew-tap`.
- `HOMEBREW_TAP_BRANCH`
  Optional. Defaults to `main`.
- `HOMEBREW_TAP_FORMULA_PATH`
  Optional. Defaults to `Formula/beehiiv.rb`.
- `HOMEBREW_TAP_TOKEN`
  Personal access token with `contents:write` on the tap repository.

## Maintainer flow

1. Create or choose a tap repository.
2. Configure the variables and secret above in the CLI repository.
3. Push a semver tag such as `v1.2.3`.
4. Let the `Release` workflow publish archives and checksums.
5. Let the `Homebrew Bump` workflow render `Formula/beehiiv.rb` from the release metadata and push it to the tap repository.
6. Run the `Package Install` workflow, or on macOS run `./scripts/install-test.sh homebrew --tag v1.2.3 --release-repository deldrid1/beehiiv-cli`, to validate the formula against the published artifacts.

## Notes

The Homebrew formula template lives at `packaging/homebrew/beehiiv.rb.tmpl`, and the renderer/updater lives at `scripts/update-homebrew-tap.sh`.

The current workflow pushes directly to the configured tap branch. If you want a PR-based tap flow instead, keep the rendering logic and change the script or workflow to commit on a topic branch and open a pull request from there.
