#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  cleanup-winget-prs.sh --version <semver>

Environment:
  GH_TOKEN   Token with permission to list and close PRs on microsoft/winget-pkgs

Example:
  GH_TOKEN=... ./scripts/cleanup-winget-prs.sh --version 0.1.3
EOF
}

current_version=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      current_version="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

[[ -n "${current_version}" ]] || {
  echo "--version is required" >&2
  exit 1
}

[[ -n "${GH_TOKEN:-}" ]] || {
  echo "GH_TOKEN is required" >&2
  exit 1
}

command -v gh >/dev/null 2>&1 || {
  echo "gh CLI is required" >&2
  exit 1
}

command -v jq >/dev/null 2>&1 || {
  echo "jq is required" >&2
  exit 1
}

pr_json="$(gh pr list \
  --repo microsoft/winget-pkgs \
  --search 'Deldrid1.BeehiivCLI state:open' \
  --limit 20 \
  --json number,title,url)"

pr_numbers="$(printf '%s' "${pr_json}" | jq -r --arg current "New version: Deldrid1.BeehiivCLI ${current_version}" '
  .[]
  | select(.title != $current)
  | .number
')"

if [[ -z "${pr_numbers}" ]]; then
  echo "No superseded winget PRs to close."
  exit 0
fi

while IFS= read -r pr_number; do
  [[ -n "${pr_number}" ]] || continue
  gh pr close \
    --repo microsoft/winget-pkgs \
    "${pr_number}" \
    --comment "Closing as superseded by the newer Beehiiv CLI winget submission for ${current_version}."
done <<< "${pr_numbers}"

