# Beehiiv Intent Map

Use this file when you need to translate natural-language requests into concrete Beehiiv CLI commands.

## Preferred reporting commands

These are the first choice when available.

| User intent | Command pattern |
| --- | --- |
| publication summary | `beehiiv reports summary --days <n>` |
| engagement chart | `beehiiv reports chart --metric <metric> --days <n>` |
| subscribers CSV | `beehiiv reports export subscriptions --file <path>` |
| posts CSV | `beehiiv reports export posts --file <path>` |
| engagements CSV | `beehiiv reports export engagements --file <path>` |

## Good chart metrics

- `total_opens`
- `unique_opens`
- `total_clicks`
- `unique_clicks`
- `total_verified_clicks`
- `unique_verified_clicks`

If the user says "open rate trend," start with `unique_opens` unless they ask for something else.

## Fallback commands

Use these when the `reports` group is not available.

### Publication and top-line performance

- publication stats:
  - `beehiiv publications get --query expand=stats`
- aggregate post stats:
  - `beehiiv posts aggregate-stats`
- recent posts with stats:
  - `beehiiv posts list --query expand=stats --query limit=10 --query order_by=publish_date --query direction=desc`

### Engagement trends

- `beehiiv engagements list --query start_date=YYYY-MM-DD --query number_of_days=<n> --query granularity=day --query direction=asc`

### Subscriber exports

- `beehiiv subscriptions list --all --query expand=stats,custom_fields`

### Post exports

- `beehiiv posts list --all --query expand=stats`

## Default export file names

If the user does not supply a path, use:

- `./exports/beehiiv-subscriptions-YYYY-MM-DD.csv`
- `./exports/beehiiv-posts-YYYY-MM-DD.csv`
- `./exports/beehiiv-engagements-YYYY-MM-DD.csv`

## Response style

After you run a command:

1. Explain what you found.
2. Mention the exact date window used.
3. Mention the export path if a file was written.
4. Suggest one next step only when it is clearly helpful.
