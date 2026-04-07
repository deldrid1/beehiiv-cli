# Beehiiv Codex Plugin

This plugin gives Codex a plain-language workflow for `beehiiv-cli`.

It is designed for people who want to ask for outcomes like:

- "Give me a 14-day newsletter performance summary"
- "Chart unique opens for the last month"
- "Export all subscribers to CSV"
- "Help me connect Beehiiv and get set up"

instead of remembering Beehiiv CLI syntax.

## What the plugin includes

- `skills/beehiiv-reporting-assistant/`
  - one high-level skill that translates natural-language Beehiiv requests into CLI commands
  - prefers the curated `beehiiv reports ...` workflows when available
  - falls back to lower-level Beehiiv commands when needed

- `.codex-plugin/plugin.json`
  - the Codex plugin manifest

- `assets/`
  - lightweight icons for the plugin picker

## Prerequisites

- `beehiiv` must be installed locally or available through `BEEHIIV_CLI_BIN`
- Beehiiv auth should be configured with `beehiiv auth login`

## Install in Codex

This repo already includes a repo marketplace entry at `.agents/plugins/marketplace.json`.

1. Restart Codex after pulling these files.
2. Open the Plugins directory in the Codex app, or run `/plugins` in Codex CLI.
3. Choose the repo marketplace named `Beehiiv Local Plugins`.
4. Install `Beehiiv`.
5. Start a new thread and ask for the outcome you want.

## Team distribution options

### Option 1: Ship it with this repo

This is the best option for people already collaborating in `beehiiv-cli`.

1. Pull the latest main branch or a tagged release commit.
2. Restart Codex.
3. Install `Beehiiv` from the `Beehiiv Local Plugins` repo marketplace.

### Option 2: Install from a GitHub release asset

Each tagged release can publish `beehiiv-codex-plugin_VERSION.zip`.

1. Download that zip from the matching GitHub release.
2. Extract `beehiiv-codex` into `~/.codex/plugins/` on macOS/Linux or `%USERPROFILE%\\.codex\\plugins\\` on Windows.
3. Create `~/.agents/plugins/marketplace.json` with a personal marketplace entry that points at `./.codex/plugins/beehiiv-codex`.
4. Restart Codex and install `Beehiiv`.

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

## Suggested prompts

- "Use Beehiiv to summarize the last 7 days of engagement."
- "Use Beehiiv to chart unique opens over the last 30 days."
- "Use Beehiiv to export subscribers to a CSV in the exports folder."
- "Use Beehiiv to check whether my local setup is ready."
