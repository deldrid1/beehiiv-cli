---
name: beehiiv-reporting-assistant
description: Translate plain-language Beehiiv requests into CLI workflows for stats, reports, charts, CSV exports, and setup help. Use when someone wants newsletter analytics or Beehiiv data without knowing the beehiiv-cli command syntax.
metadata:
  short-description: Plain-language Beehiiv reports, charts, and exports
---

# Beehiiv Reporting Assistant

## Overview

Use this skill when the user wants Beehiiv outcomes in plain language, not raw command help.

This skill turns requests like "show me open rates for the last two weeks" or "export all subscribers to CSV" into concrete `beehiiv` CLI steps, runs the right commands, and explains the results in business language.

## Quick Start

1. Run `bash <path-to-skill>/scripts/diagnose_beehiiv.sh` first.
2. If the CLI is unavailable, ask the user to install `beehiiv-cli` or set `BEEHIIV_CLI_BIN`.
3. If auth is not configured, help the user run `beehiiv auth login`.
4. Prefer the curated reporting workflows:
   - `beehiiv reports summary`
   - `beehiiv reports chart --metric ... --days ...`
   - `beehiiv reports export subscriptions|posts|engagements --file ...`
5. Explain the results in plain language and always mention file paths for exports.

## Workflow

### 1) Understand the request

Classify the user ask into one of these buckets:

- summary or report
- chart or trend line
- CSV export
- setup or troubleshooting
- lower-level lookup

If the user does not give a time window:

- default to `7` days for quick summaries or charts
- default to `30` days for exports and broader performance questions

If the user does not give an export path:

- default to `./exports/beehiiv-<dataset>-YYYY-MM-DD.csv`

### 2) Check the local Beehiiv environment

Use the diagnostic script first.

Interpret the result this way:

- `available=false`: the Beehiiv CLI is not reachable
- `configured=false`: the CLI is installed but auth is not set up
- empty `publication_id`: the user may need to select or provide a publication

If setup is incomplete, switch into setup-helper mode:

- ask the user for the smallest missing piece
- keep the guidance concrete
- prefer `beehiiv auth login` over manual environment instructions unless the user asks for automation

### 3) Prefer the friendly reports surface

Use these commands first because they are best for non-technical users:

- publication summary:
  - `beehiiv reports summary`
  - add `--days <n>` when the time window matters
- engagement chart:
  - `beehiiv reports chart --metric <metric> --days <n>`
- CSV export:
  - `beehiiv reports export subscriptions --file <path>`
  - `beehiiv reports export posts --file <path>`
  - `beehiiv reports export engagements --file <path>`

### 4) Fall back cleanly if `reports` is unavailable

If `beehiiv reports --help` fails, use the lower-level commands in `references/intent-map.md`.

Common fallbacks:

- publication overview:
  - `beehiiv publications get --query expand=stats`
- post rollup:
  - `beehiiv posts aggregate-stats`
- recent post performance:
  - `beehiiv posts list --query expand=stats --query limit=10`
- engagement trend:
  - `beehiiv engagements list --query start_date=YYYY-MM-DD --query number_of_days=14 --query granularity=day`
- raw subscriber export:
  - `beehiiv subscriptions list --all --query expand=stats,custom_fields`

When using fallbacks, do the formatting work for the user instead of exposing raw JSON unless they asked for it.

### 5) Explain results for a non-technical person

Translate the output into plain English:

- lead with the answer, not the command
- mention exact dates like `April 1, 2026` instead of relative time if there is any ambiguity
- call out notable changes, strong performers, and possible follow-up questions
- if you exported a file, state the exact file path

Good explanation style:

- "Unique opens averaged 330 per day across March 31, 2026 through April 1, 2026."
- "Launch Week is the strongest recent post here, with a 61.2% email open rate."

### 6) Safety and operating rules

- Never print live API keys or tokens.
- Use `beehiiv auth status` when you need to check auth.
- Prefer `--output table` when the user wants something readable in-thread.
- Prefer `--output json` only when you need to parse or transform the result.
- If you create an export file, do not overwrite a user-named file silently if it already exists; either confirm or choose a dated filename.

## Common Requests

- "How is the newsletter doing?"
  - run `beehiiv reports summary`

- "Chart opens for the last 14 days"
  - run `beehiiv reports chart --metric unique_opens --days 14`

- "Export all subscribers to CSV"
  - run `beehiiv reports export subscriptions --file ./exports/beehiiv-subscriptions-YYYY-MM-DD.csv`

- "Give me the top recent posts"
  - run `beehiiv reports summary --recent-posts 10` if available
  - otherwise use `beehiiv posts list --query expand=stats --query limit=10`

- "Help me get this working"
  - run the diagnostic script
  - then guide setup with the troubleshooting reference

## References

- For request-to-command mapping: [references/intent-map.md](references/intent-map.md)
- For install and auth issues: [references/troubleshooting.md](references/troubleshooting.md)
