# Winget Publication Setup

This repo uses GoReleaser's `winget` publisher to generate manifests after a tagged release. When `WINGET_PUBLISH_TOKEN` is present, GoReleaser can push a branch to a fork of `microsoft/winget-pkgs` and open a PR back to `master`. When the token is not present, the manifests are still generated and kept local to `dist/` only.

## Required repository configuration

Set these repository variables and secrets before enabling public publication:

- `WINGET_REPOSITORY_OWNER`
  Use the GitHub account or organization that owns your fork of `microsoft/winget-pkgs`.
- `WINGET_REPOSITORY_NAME`
  Usually `winget-pkgs`.
- `WINGET_REPOSITORY_BRANCH`
  Optional. If omitted, GoReleaser uses a version-specific branch name.
- `WINGET_PUBLISH_TOKEN`
  Personal access token with permission to push to the fork and open pull requests.

## Maintainer flow

1. Fork `microsoft/winget-pkgs`.
2. Configure the repository variables and secret above.
3. Push a semver tag such as `v1.2.3`.
4. Let the `Release` workflow publish the GitHub Release and let GoReleaser generate the winget manifest PR.
5. Review the resulting PR in your fork and the PR opened against `microsoft/winget-pkgs`.
6. Run the `Package Install` workflow to validate the local-manifest install path against the published release assets if you need an extra manual confirmation.

The current package identifier is `Deldrid1.BeehiivCLI`. If you ever need to change it, update [`.goreleaser.yaml`](/Users/austineldridge/GitRepos/beehiiv-cli/.goreleaser.yaml) and [`.github/workflows/package-install.yml`](/Users/austineldridge/GitRepos/beehiiv-cli/.github/workflows/package-install.yml) together.

## Validation notes

The `Package Install` workflow validates the generated manifest with `winget validate`, then installs Beehiiv through a local manifest via `winget install --manifest <path>`, following Microsoft's local-manifest guidance. It enables `LocalManifestFiles` first because that feature is disabled by default on Windows runners.

The current release flow publishes Windows ZIP artifacts, and both the local validation workflow and the GoReleaser publisher treat them as ZIP-based portable manifests. Microsoft's current WinGet docs list both `portable` and `zip` as supported installer types, so this is a valid starting point for public publication. It is still worth dry-running against your `winget-pkgs` fork before the first public release so any schema or policy drift is caught early.

## References

- GoReleaser winget docs: https://goreleaser.com/customization/winget/
- WinGet manifest authoring: https://learn.microsoft.com/en-us/windows/package-manager/package/manifest
- WinGet local manifest installs: https://learn.microsoft.com/en-us/windows/package-manager/winget/install
- WinGet community repository submission: https://learn.microsoft.com/en-us/windows/package-manager/package/repository
