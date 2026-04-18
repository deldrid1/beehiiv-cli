# Beehiiv Claude Code Plugin

This plugin gives Claude Code a plain-language workflow for `beehiiv-cli`.

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
- `.claude-plugin/plugin.json` — the Claude Code plugin manifest

## Prerequisites

- `beehiiv` must be installed locally or available through `BEEHIIV_CLI_BIN`
- Beehiiv auth should be configured with `beehiiv auth login`

## Install in Claude Code

This repo includes a repo marketplace file at `.claude-plugin/marketplace.json`.

### Option 1: Install from this repo (recommended for contributors)

From any Claude Code session:

```text
/plugin marketplace add /absolute/path/to/beehiiv-cli
/plugin install beehiiv-claude@beehiiv-local-plugins
```

Then restart Claude Code so the skill is loaded, and ask for an outcome in plain language.

### Option 2: Install from GitHub

```text
/plugin marketplace add deldrid1/beehiiv-cli
/plugin install beehiiv-claude@beehiiv-local-plugins
```

### Option 3: Use the skill without installing the plugin

You can drop just the skill into your personal Claude skills directory:

```bash
cp -r plugins/beehiiv-claude/skills/beehiiv-reporting-assistant \
  ~/.claude/skills/
```

Claude will discover it at the next session start.

## Suggested prompts

- "Use Beehiiv to summarize the last 7 days of engagement."
- "Use Beehiiv to chart unique opens over the last 30 days."
- "Use Beehiiv to export subscribers to a CSV in the exports folder."
- "Use Beehiiv to check whether my local setup is ready."

## Why a skill instead of an MCP connector

`beehiiv-cli` is already the tool — it handles auth, pagination, retries, and rate limiting. A skill teaches Claude when and how to invoke the CLI without duplicating that surface as an MCP server. If you want conversational read access without a local CLI install, Beehiiv's [official MCP server](https://www.beehiiv.com/features/mcp) is a separate option; see the CLI-vs-MCP comparison in the root [README.md](../../README.md#cli-vs-beehiiv-mcp-server).
