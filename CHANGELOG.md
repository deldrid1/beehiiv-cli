# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2026-04-17

### Added

- Claude Code plugin at `plugins/beehiiv-claude/` with a `beehiiv-reporting-assistant` skill that turns plain-language requests into `beehiiv` CLI workflows.
- Repo-level Claude Code marketplace at `.claude-plugin/marketplace.json` so the plugin installs in one command (`/plugin marketplace add <repo>` then `/plugin install beehiiv-claude@beehiiv-local-plugins`).
- `CHANGELOG.md` documenting releases back to `v0.1.0`.

### Changed

- README gains a "Use with Claude Code" section and a link to the changelog.

## [0.2.0] - 2026-04-16

### Added

- OAuth-first authentication with PKCE; `beehiiv login` opens a browser and stores credentials in the OS keyring.
- Unified Cobra-based command execution; legacy CLI layer removed.
- `beehiiv auth oauth login` for custom OAuth clients.

### Changed

- Login command now respects full config precedence: flags override environment, which overrides stored config, which overrides defaults.
- `Options.Env` is populated from `os.Environ()` so `BEEHIIV_*` variables propagate consistently.
- README documents the live Windows winget install flow.
- CLI vs Beehiiv MCP server comparison added to the README.

### Fixed

- `GetTokenInfo` no longer panics with a nil pointer when no HTTP client is injected.
- `/oauth/token/info` responses parse correctly whether `scope` is returned as a string or as an array.
- `login` command honors `BEEHIIV_OAUTH_CLIENT_ID`, `BEEHIIV_OAUTH_CLIENT_SECRET`, `BEEHIIV_OAUTH_REDIRECT_URI`, and `BEEHIIV_OAUTH_SCOPES` environment variables.

## [0.1.6] - 2026-04-07

### Fixed

- Release workflow YAML is valid again after the plugin packaging change.

## [0.1.5] - 2026-04-07

### Fixed

- Release smoke test packages the Codex plugin archive as part of the release artifacts.

## [0.1.4] - 2026-04-07

### Added

- `beehiiv reports` command group: `summary`, `chart`, and `export` workflows for non-technical users (publication summaries, ASCII engagement charts, CSV exports of subscriptions / posts / engagements).
- Codex plugin at `plugins/beehiiv-codex/` with a `beehiiv-reporting-assistant` skill for plain-language Beehiiv workflows.

### Changed

- Release publishing automation hardened so plugin + binary artifacts ship together.

## [0.1.3] - 2026-04-06

### Added

- Expanded automation command coverage so journeys, steps, and stats are discoverable from the CLI.

### Changed

- README Quick Start rewritten to lead with `beehiiv login` and the most common workflows.
- Resource listings tuned for discoverability across subscriptions, posts, and publications.

### Fixed

- Broken links and typos in README cleaned up.

## [0.1.2] - 2026-04-06

### Fixed

- Winget publisher configuration simplified so the manifest validates on first submission.

## [0.1.1] - 2026-04-06

### Fixed

- Homebrew tap publishing now detects newly created formula files rather than only updated ones.
- Homebrew tap release path handles real (non-dry-run) releases.
- Package install validation (`winget`, `brew`) runs safely inside CI runners.
- Released package artifacts are validated against the correct checksums.

### Changed

- Bumped `github.com/spf13/pflag` 1.0.9 → 1.0.10.
- Bumped `actions/attest-build-provenance` 3 → 4.
- Bumped `actions/upload-artifact` 6 → 7.

## [0.1.0] - 2026-04-05

### Added

- Initial public release of `beehiiv-cli`.
- Full Beehiiv v2 API coverage across publications, subscriptions, posts, post content, authors, tiers, custom fields, polls, automations, automation journeys, segments, referral program, advertisements, webhooks, email blasts, engagements, newsletter lists, workspaces, post templates, and condition sets (71 operations across 28 resource groups).
- OAuth login and API key authentication, with secrets stored in the macOS Keychain or Windows Credential Manager.
- `--all` aggregation with cursor / page pagination and built-in retry plus rate-limit handling.
- Table, JSON, and raw output formats; `--verbose` request/response logging.
- Cross-platform distribution: Homebrew tap, winget package, and GitHub release binaries for macOS, Linux, and Windows.
- Generated CLI reference docs, manpages, and shell completions.

[Unreleased]: https://github.com/deldrid1/beehiiv-cli/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/deldrid1/beehiiv-cli/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/deldrid1/beehiiv-cli/compare/v0.1.6...v0.2.0
[0.1.6]: https://github.com/deldrid1/beehiiv-cli/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/deldrid1/beehiiv-cli/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/deldrid1/beehiiv-cli/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/deldrid1/beehiiv-cli/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/deldrid1/beehiiv-cli/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/deldrid1/beehiiv-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/deldrid1/beehiiv-cli/releases/tag/v0.1.0
