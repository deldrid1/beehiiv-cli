# Beehiiv CLI

A simple command-line app for interacting with the [Beehiiv V2 API](https://developers.beehiiv.com/welcome/getting-started). Use it (with AI! 🤖) to extract data and work with your publications, subscribers, posts, and webhooks from your terminal.

## Project Status

Ready for public use on macOS, Windows, and Linux. Install via Homebrew (macOS/Linux) or winget (Windows), or download a binary from the GitHub releases page. The Codex plugin ships both in-repo and as a release asset for team installs.

## Quick Start

1. Download the latest release from the [GitHub releases page](https://github.com/deldrid1/beehiiv-cli/releases) or install via a package manager:
```bash
brew tap deldrid1/homebrew-tap
brew install beehiiv
```
2. Sign in — your browser opens automatically:
```bash
beehiiv login
```
3. Try `beehiiv publications list` or `beehiiv --help`.

If you're setting up release publishing with an agent, start with [docs/release-auth-setup.md](docs/release-auth-setup.md). It includes the current token requirements, including the classic PAT requirement for winget publication.

## Use with Codex

The repo includes a Codex-installable plugin at `plugins/beehiiv-codex` for teammates who would rather ask for outcomes in plain language than memorize CLI commands.

If your team already works in this repo:

1. Pull the latest changes.
2. Restart Codex.
3. Open Plugins in Codex, then install `Beehiiv` from the `Beehiiv Local Plugins` marketplace.

If your team just wants the plugin:

1. Download `beehiiv-codex-plugin_VERSION.zip` from the matching [GitHub release](https://github.com/deldrid1/beehiiv-cli/releases).
2. Extract `beehiiv-codex` into `~/.codex/plugins/` on macOS/Linux or `%USERPROFILE%\\.codex\\plugins\\` on Windows.
3. Add a personal marketplace file at `~/.agents/plugins/marketplace.json` that points at `./.codex/plugins/beehiiv-codex`.
4. Restart Codex and install `Beehiiv` from that personal marketplace.

Example personal marketplace file:

```json
{
  "name": "beehiiv-personal-plugins",
  "interface": {
    "displayName": "Beehiiv Personal Plugins"
  },
  "plugins": [
    {
      "name": "beehiiv-codex",
      "source": {
        "source": "local",
        "path": "./.codex/plugins/beehiiv-codex"
      },
      "policy": {
        "installation": "AVAILABLE",
        "authentication": "ON_INSTALL"
      },
      "category": "Productivity"
    }
  ]
}
```

## Features

- Friendly `--help` output on every command
- One-command sign-in via OAuth — no API key setup required
- Secure credential storage in the macOS Keychain or Windows Credential Manager
- Handy shortcuts for common subscriber, publication, post, and webhook tasks
- Guided `reports` workflows for non-technical stats, charts, and CSV exports
- JSON output for scripts, with table output when you want something easier to read
- Automatic pagination with `--all`
- Built-in retry and rate-limit handling

## CLI vs Beehiiv MCP Server

Beehiiv ships an [official MCP server](https://www.beehiiv.com/features/mcp) for AI-assisted workflows. The table below compares it with this CLI so you can choose the right tool — or use both.

### Capability coverage

| Resource | CLI | MCP | Notes |
|---|---|---|---|
| Publications (list, get, stats) | ✅ | ✅ | |
| Posts (list, get, stats) | ✅ | ✅ | |
| Post content (free/paid audience views) | ✅ | ✅ | MCP adds audience-scoped rendering |
| Post ISP deliverability breakdown | ❌ | ✅ | MCP-only; per-domain metrics for 50+ subscriber domains |
| Post click tracking (per-link) | ❌ | ✅ | MCP-only; includes per-subscriber click lists |
| Subscriptions (list, get, filter) | ✅ | ✅ | |
| Authors | ✅ | ✅ | |
| Tiers | ✅ | ✅ | |
| Custom fields | ✅ | ✅ | |
| Content tags | ❌ | ✅ | MCP-only |
| Polls + poll responses | ✅ | ✅ | |
| Surveys + survey responses | ❌ | ✅ | MCP-only |
| Automations + journeys | ✅ | ✅ | MCP adds per-step subscriber counts |
| Automation email content | ❌ | ✅ | MCP-only; rendered email from a step |
| Segments (list, get, recalculate) | ✅ | ❌ | CLI-only |
| Segment members / results | ✅ | ❌ | CLI-only |
| Referral program | ✅ | ❌ | CLI-only |
| Advertisement opportunities | ✅ | ❌ | CLI-only |
| Webhooks (CRUD + test) | ✅ | ❌ | CLI-only |
| Email blasts | ✅ | ❌ | CLI-only |
| Engagements | ✅ | ❌ | CLI-only |
| Newsletter lists | ✅ | ❌ | CLI-only (Beta) |
| Workspaces | ✅ | ✅ | |
| Post templates | ✅ | ❌ | CLI-only |
| Condition sets | ✅ | ❌ | CLI-only |
| Beehiiv documentation search | ❌ | ✅ | MCP-only; search and read support articles |

### Write operations

| Operation | CLI | MCP |
|---|---|---|
| Create / update / delete posts | ✅ | ❌ |
| Create / update / delete subscriptions | ✅ | ❌ |
| Bulk import subscriptions | ✅ | ❌ |
| Bulk update subscriptions | ✅ | ❌ |
| Create / update / delete custom fields | ✅ | ❌ |
| Create / update / delete webhooks | ✅ | ❌ |
| Create / update tiers | ✅ | ❌ |
| Create automation journeys | ✅ | ❌ |
| Delete segments | ✅ | ❌ |
| Add subscription tags | ✅ | ❌ |

The MCP is **read-only (v1)** — Beehiiv has announced write support for a future v2.

### Cross-cutting features

| Feature | CLI | MCP |
|---|---|---|
| Full pagination (`--all` aggregation) | ✅ | Capped at 100/page |
| CSV export | ✅ | ❌ |
| ASCII charts in terminal | ✅ | ❌ |
| Publication summary reports | ✅ | ❌ |
| Table, JSON, and raw output | ✅ | JSON only |
| Rate-limit handling + retry | ✅ | Managed server-side |
| OAuth sign-in (no API key) | ✅ | ✅ (managed auth) |
| API key auth | ✅ | ❌ |
| Works on free Beehiiv plan | ✅ | ❌ (paid plans only) |
| Shell scripting / CI pipelines | ✅ | ❌ |
| AI agent integration (MCP) | ❌ | ✅ |
| Verbose request debugging | ✅ | ❌ |

### When to use which

- **CLI** — scripting, bulk operations, CI/CD, CSV exports, anything that writes data, free-tier accounts, or when you need the full API surface (71 operations across 28 resource groups).
- **MCP** — conversational AI workflows, asking questions about your newsletter in natural language, deliverability analysis (ISP breakdown), and survey data.
- **Both** — the CLI and MCP complement each other. Use the MCP for exploration and analysis, then use the CLI to act on what you find.

## Requirements

- For normal use: a Beehiiv account (OAuth sign-in is built in — no API key needed)
- To build from source: Go 1.26 or newer
- For publication-scoped commands: a Beehiiv publication ID

For CI/CD or programmatic use you can also pass an API key directly. See Beehiiv's [Create an API Key](https://developers.beehiiv.com/welcome/create-an-api-key) guide.

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

## Install from a Release

### macOS

1. Download the correct archive for your machine from the [latest release](https://github.com/deldrid1/beehiiv-cli/releases/latest):
   `beehiiv_VERSION_darwin_arm64.tar.gz` on Apple Silicon or `beehiiv_VERSION_darwin_x86_64.tar.gz` on Intel.
2. Extract it:

```bash
tar -xzf beehiiv_VERSION_darwin_arm64.tar.gz
```

3. Move `beehiiv` somewhere on your `PATH`, for example:

```bash
mv ./beehiiv /usr/local/bin/beehiiv
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

Install via [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/) (requires Windows 10 1809+ or Windows 11):

```powershell
winget install Deldrid1.BeehiivCLI
```

Then sign in:

```powershell
beehiiv login
```

If winget is not available, download the correct archive from the [latest release](https://github.com/deldrid1/beehiiv-cli/releases/latest) (e.g. `beehiiv_VERSION_windows_x86_64.zip`), extract it into a folder such as `C:\Tools\beehiiv\`, and add that folder to your `PATH`.

The CLI stores its config at `%AppData%\beehiiv-cli\config.json`. Secrets are stored in the Windows Credential Manager by default.

### Linux

1. Download the correct archive from the [latest release](https://github.com/deldrid1/beehiiv-cli/releases/latest), for example `beehiiv_VERSION_linux_x86_64.tar.gz`.
2. Extract it:

```bash
tar -xzf beehiiv_VERSION_linux_x86_64.tar.gz
```

3. Move `beehiiv` somewhere on your `PATH`, for example:

```bash
sudo mv ./beehiiv /usr/local/bin/beehiiv
```

## Package Managers

Homebrew:

```bash
brew tap deldrid1/homebrew-tap
brew install beehiiv
```

Winget (Windows 10 1809+ or Windows 11):

```powershell
winget install Deldrid1.BeehiivCLI
```

Maintainer setup and publication notes live in [packaging/winget/README.md](packaging/winget/README.md), [packaging/homebrew/README.md](packaging/homebrew/README.md), and [docs/release-auth-setup.md](docs/release-auth-setup.md).

## Authentication

Sign in with a single command — no API key required:

```bash
beehiiv login
```

Your browser opens the Beehiiv authorization page. After you approve, credentials are saved securely in the OS keyring. The OAuth client is pre-configured; nothing extra to set up.

If you prefer not to open a browser automatically:

```bash
beehiiv login --no-browser
```

### API key authentication (CI/CD)

For non-interactive environments, pass your API key directly:

```bash
beehiiv login --api-key YOUR_API_KEY
beehiiv login --api-key YOUR_API_KEY --publication-id pub_123
```

If `--publication-id` is omitted the CLI looks up your publications and selects automatically if there is only one, or prompts you to choose.

### Auth management

```bash
beehiiv auth status          # show masked credential state
beehiiv auth logout          # remove saved credentials (revokes OAuth token)
beehiiv auth path            # print the config file location
```

### Custom OAuth app (advanced)

If you are building your own Beehiiv OAuth integration:

```bash
beehiiv auth oauth login --client-id YOUR_CLIENT_ID
beehiiv auth oauth login --client-id YOUR_CLIENT_ID --scope all
beehiiv auth oauth login --client-id YOUR_CLIENT_ID --no-browser
```

The redirect URI defaults to `http://localhost:3008/callback` and must match your Beehiiv OAuth app settings. PKCE is used automatically.

## Configuration

The CLI checks configuration in this order:

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

The CLI stores non-secret settings in `config.json`. Secrets are kept out of that file.

Auth status example:

```bash
beehiiv auth status
```

## Common Commands

Start with help:

```bash
beehiiv
beehiiv --help
```

List publications:

```bash
beehiiv publications list
beehiiv pubs list
```

List all subscribers:

```bash
beehiiv subscriptions list --all --query limit=100
```

Use repeatable query flags when an endpoint accepts multiple values:

```bash
beehiiv subscriptions list \
  --query expand=stats,custom_fields \
  --query status=active
```

Show a subscriber by ID:

```bash
beehiiv subscriptions show sub_123
```

Inspect automations and their related email activity:

```bash
beehiiv automations list --query expand=stats
beehiiv automations show aut_123 --query expand=stats
beehiiv automations emails aut_123
```

Inspect segments and poll responses:

```bash
beehiiv segments members segment_123 --query expand=stats,custom_fields
beehiiv polls show poll_123 --query expand=stats
beehiiv polls responses poll_123
```

Look up a subscriber by email:

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

Check the current sign-in state safely:

```bash
beehiiv auth status
```

Show a table instead of JSON:

```bash
beehiiv subscriptions list --output table
```

Create a friendly publication summary, chart engagement trends, or export a CSV:

```bash
beehiiv reports summary
beehiiv reports chart --metric unique_opens --days 14
beehiiv reports export subscriptions --file subscriptions.csv
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

- List commands return one page by default.
- Add `--all` to fetch everything.
- For endpoints like subscriptions, the CLI prefers cursor pagination unless you explicitly pass `--query page=...`.
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

See [CONTRIBUTING.md](CONTRIBUTING.md) for the contributor workflow and [SECURITY.md](SECURITY.md) for responsible disclosure guidance.

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
