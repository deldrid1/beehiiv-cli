# Winget Publication Setup

This repo uses GoReleaser's `winget` publisher to generate manifests after a tagged release. When `WINGET_PUBLISH_TOKEN` is present, GoReleaser can push a branch to a fork of `microsoft/winget-pkgs` and open a PR back to `master`. When the token is not present, the manifests are still generated and kept local to `dist/` only.

## Required repository configuration

Set these repository secrets before enabling public publication:

- `WINGET_PUBLISH_TOKEN`
  Classic personal access token with the `public_repo` scope. This flow needs to push to your fork and then create a PR against `microsoft/winget-pkgs`, which is not reliably covered by a fork-scoped fine-grained PAT.

## Maintainer flow

1. Fork `microsoft/winget-pkgs`.
2. Configure the repository secret above.
3. Push a semver tag such as `v1.2.3`.
4. Let the `Release` workflow publish the GitHub Release and let GoReleaser generate the winget manifest PR.
5. Review the resulting PR in your fork and the PR opened against `microsoft/winget-pkgs`.
6. The release automation will close older open Beehiiv CLI winget PRs once the newer one is created successfully.
7. Run the `Package Install` workflow to validate the local-manifest install path against the published release assets if you need an extra manual confirmation.

The current package identifier is `Deldrid1.BeehiivCLI`. If you ever need to change it, update [`.goreleaser.yaml`](../../.goreleaser.yaml) and [`.github/workflows/package-install.yml`](../../.github/workflows/package-install.yml) together.

## Validation notes

The `Package Install` workflow validates the generated manifest with `winget validate`, then installs Beehiiv through a local manifest via `winget install --manifest <path>`, following Microsoft's local-manifest guidance. It enables `LocalManifestFiles` first because that feature is disabled by default on Windows runners.

The current release flow publishes Windows ZIP artifacts, and both the local validation workflow and the GoReleaser publisher treat them as ZIP-based portable manifests. Microsoft's current WinGet docs list both `portable` and `zip` as supported installer types, so this is a valid starting point for public publication. It is still worth dry-running against your `winget-pkgs` fork before the first public release so any schema or policy drift is caught early.

For token rotation and agent-friendly setup, see [docs/release-auth-setup.md](../../docs/release-auth-setup.md).

## References

- GoReleaser winget docs: https://goreleaser.com/customization/winget/
- WinGet manifest authoring: https://learn.microsoft.com/en-us/windows/package-manager/package/manifest
- WinGet local manifest installs: https://learn.microsoft.com/en-us/windows/package-manager/winget/install
- WinGet community repository submission: https://learn.microsoft.com/en-us/windows/package-manager/package/repository
