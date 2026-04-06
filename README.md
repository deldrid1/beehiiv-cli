# Beehiiv CLI for Go

Pure Go CLI for the Beehiiv API with no CGO or non-Go system dependencies at build time or runtime. The binary is designed to compile cleanly for macOS and Windows and to work well for both interactive use and JSON-first scripting.

## Project Status

This project is ready for a public pre-v1 release. The CLI has a Cobra-based command surface, generated docs and completions, secure default auth storage, and GitHub Release automation with Homebrew and winget distribution workflows.

## Features

- Self-documenting command tree with `--help` on every command
- `auth login` plus `login` alias for secure API key setup
- `auth oauth login` for Beehiiv OAuth authorization-code flow with PKCE
- `auth status` to inspect the active session without printing live credentials
- Structured `config.json` settings plus OS keychain or keyring secret storage
- Curated workflow aliases like `subs`, `show`, `find`, `stats`, and `ping`
- Full command coverage for the curated Beehiiv OpenAPI surface in this repo
- Cursor-aware and offset-aware pagination with `--all`
- Internal rate limiting plus Beehiiv header-aware backoff for `429` responses
- JSON-first output by default, with `--compact`, `--raw`, and `--output table`
- `--verbose` request and response tracing for troubleshooting

## Requirements

- Go 1.26 or newer to build from source
- A Beehiiv API key
- A Beehiiv publication ID for publication-scoped commands

Create [an API key here](https://app.beehiiv.com/settings/workspace/api) as described in the [REST API documentation](https://developers.beehiiv.com/welcome/create-an-api-key)

## Build from Source

From the Go module root:

```bash
make build
```

Cross-platform examples:

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./dist/beehiiv-darwin-arm64 ./cmd/beehiiv
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./dist/beehiiv-darwin-amd64 ./cmd/beehiiv
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./dist/beehiiv-windows-amd64.exe ./cmd/beehiiv
```

## Install from Built Binaries

### macOS

1. Download the correct binary for your machine, for example `beehiiv-darwin-arm64` on Apple Silicon or `beehiiv-darwin-amd64` on Intel.
2. Make it executable:

```bash
chmod +x ./beehiiv-darwin-arm64
```

3. Move it somewhere on your `PATH`, for example:

```bash
mv ./beehiiv-darwin-arm64 /usr/local/bin/beehiiv
```

4. Run:

```bash
beehiiv auth login
```

The CLI stores its config at:

```text
~/Library/Application Support/beehiiv-cli/config.json
```

Secrets are stored in the macOS Keychain by default.

### Windows

1. Download `beehiiv-windows-amd64.exe`.
2. Place it in a stable folder such as `C:\Tools\beehiiv\`.
3. Add that folder to your `PATH`.
4. Run:

```powershell
beehiiv.exe auth login
```

The CLI stores its config at:

```text
%AppData%\beehiiv-cli\config.json
```

Secrets are stored in the Windows Credential Manager by default.

## Package Managers

Package-manager installs:

```bash
brew tap <owner>/<tap>
brew install beehiiv
```

```powershell
winget install Deldrid1.BeehiivCLI
```

Maintainer setup and publication notes live in [packaging/winget/README.md](/Users/austineldridge/GitRepos/beehiiv-cli/packaging/winget/README.md).
Homebrew maintainer setup lives in [packaging/homebrew/README.md](/Users/austineldridge/GitRepos/beehiiv-cli/packaging/homebrew/README.md).

## Authentication

Run:

```bash
beehiiv auth login
```

Or:

```bash
beehiiv login
```

You can also provide values directly:

```bash
beehiiv auth login --api-key YOUR_API_KEY --publication-id pub_123
```

If `--publication-id` is omitted, the CLI calls `GET /publications`. If your API key sees exactly one publication, it is selected automatically. Otherwise the CLI prompts you to choose from the returned `pub_...` IDs.

OAuth login is available for Beehiiv OAuth apps:

```bash
beehiiv auth oauth login --client-id YOUR_CLIENT_ID --scope all
```

The default redirect URI is `http://localhost:3008/callback`, which must exactly match one of your Beehiiv OAuth app redirect URIs. The CLI uses PKCE for public-client flows and can fall back to manual callback pasting when needed:

```bash
beehiiv auth oauth login --client-id YOUR_CLIENT_ID --manual --no-browser
```

Useful auth commands:

```bash
beehiiv auth status
beehiiv auth path
beehiiv auth logout
```

## Config Precedence

The CLI resolves configuration in this order:

1. Command-line flags
2. Environment variables
3. Stored session secrets in the OS keychain or keyring plus `config.json` settings
4. Built-in defaults

Supported environment variables:

```text
BEEHIIV_API_KEY
BEEHIIV_BEARER_TOKEN
BEEHIIV_PUBLICATION_ID
BEEHIIV_BASE_URL
BEEHIIV_RATE_LIMIT_RPM
BEEHIIV_OAUTH_CLIENT_ID
BEEHIIV_OAUTH_CLIENT_SECRET
BEEHIIV_OAUTH_REDIRECT_URI
BEEHIIV_OAUTH_SCOPES
```

The CLI stores non-secret settings in `config.json`. Secrets are not written to `config.json`.

Auth status example:

```bash
beehiiv auth status
```

## Usage

See the root help:

```bash
beehiiv
beehiiv --help
```

List publications:

```bash
beehiiv publications list
beehiiv pubs list
```

List all subscriptions using cursor pagination when available:

```bash
beehiiv subscriptions list --all --query limit=100
```

Use repeatable query flags for arrays:

```bash
beehiiv subscriptions list \
  --query expand=stats,custom_fields \
  --query status=active
```

Fetch a subscription by ID:

```bash
beehiiv subscriptions show sub_123
```

Look up a subscription by email:

```bash
beehiiv subscriptions find person@example.com
```

Create a custom field from inline JSON:

```bash
beehiiv custom-fields create --body '{"kind":"string","display":"Favorite Airport"}'
```

Create a webhook from a file:

```bash
beehiiv webhooks create --body @webhook.json
beehiiv hooks ping endpoint_123
```

Inspect the current auth session safely:

```bash
beehiiv auth status
```

Print a table instead of JSON:

```bash
beehiiv subscriptions list --output table
```

Print the raw API response body:

```bash
beehiiv subscriptions get sub_123 --raw
```

Print request and response details to `stderr` for troubleshooting:

```bash
beehiiv subscriptions get sub_123 --verbose
```

## Pagination

- List commands return a single page by default.
- Add `--all` to exhaust every page.
- For hybrid endpoints like subscriptions, the CLI prefers cursor pagination unless you explicitly pass `--query page=...`.
- Aggregated `--all` output looks like:

```json
{
  "data": [],
  "pagination": {
    "mode": "cursor",
    "pages_fetched": 3,
    "items_returned": 250,
    "has_more": false,
    "next_cursor": null
  }
}
```

## Rate Limiting

The CLI defaults to an internal limit of `150` requests per minute and also honors Beehiiv rate-limit headers. When Beehiiv responds with `429`, the client waits for the reset window and retries automatically.

## Development

Useful local commands:

```bash
make build
make docs
make test
make fmt
make fmt-check
make lint
```

Generated CLI reference docs land in `docs/reference/cli/`, generated manpages land in `share/man/man1/`, and generated shell completions land in `share/completions/`.

To enable the local pre-commit hook:

```bash
git config core.hooksPath .githooks
```

See `CONTRIBUTING.md` for the contributor workflow and `SECURITY.md` for responsible disclosure guidance.

Override the internal limiter if needed:

```bash
beehiiv subscriptions list --rate-limit-rpm 120
```

## Testing

Run the local unit and integration suite:

```bash
go test ./...
```

Run the gated live Beehiiv suite:

```bash
BEEHIIV_LIVE_TESTS=1 \
BEEHIIV_API_KEY=your_api_key \
BEEHIIV_PUBLICATION_ID=pub_your_publication \
go test ./... -run Live
```

The live tests look for `BEEHIIV_API_KEY` first, then `BEEHIIV_BEARER_TOKEN`. They also require `BEEHIIV_PUBLICATION_ID`.
