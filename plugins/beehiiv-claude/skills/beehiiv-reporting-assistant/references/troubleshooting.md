# Beehiiv Troubleshooting

Use this file when the user wants setup help or a Beehiiv command fails before you can answer the business question.

## CLI not found

Symptoms:

- `available=false` from the diagnostic script
- `beehiiv version` fails

What to do:

1. Ask the user to install `beehiiv-cli` or point Claude at it with `BEEHIIV_CLI_BIN`.
2. If they are in the repo, `go run ./cmd/beehiiv --help` is a useful local fallback.
3. Re-run the diagnostic script after the binary is available.

## Auth not configured

Symptoms:

- `configured=false` from the diagnostic script
- API commands return auth errors

What to do:

1. Ask the user to run `beehiiv auth login`.
2. If they prefer environment variables, confirm `BEEHIIV_API_KEY` and `BEEHIIV_PUBLICATION_ID` are set.
3. Re-check with `beehiiv auth status`.

## Publication missing

Symptoms:

- `publication_id` is empty
- publication-scoped commands fail

What to do:

1. Ask the user to rerun `beehiiv auth login`, or
2. use `--publication-id <pub_id>` explicitly

## Reports group missing

Symptoms:

- `beehiiv reports --help` fails

What to do:

1. Fall back to the lower-level command map in `intent-map.md`.
2. If the user is working from this repo, suggest using the current build because the reporting workflows live in the newer CLI surface.

## Good setup check flow

Run these in order:

1. `bash <path-to-skill>/scripts/diagnose_beehiiv.sh`
2. `beehiiv version`
3. `beehiiv auth status`
4. `beehiiv reports --help`
