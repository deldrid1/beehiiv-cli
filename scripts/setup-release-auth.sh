#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  setup-release-auth.sh [--repo <owner/repo>] [--tap-repo <owner/repo>] [--tap-branch <branch>] [--tap-formula-path <path>]

Required environment:
  HOMEBREW_TAP_TOKEN     Fine-grained PAT for the Homebrew tap repository
  WINGET_PUBLISH_TOKEN   Fine-grained PAT for the winget-pkgs fork

Examples:
  HOMEBREW_TAP_TOKEN=... WINGET_PUBLISH_TOKEN=... \
    ./scripts/setup-release-auth.sh

  HOMEBREW_TAP_TOKEN=... WINGET_PUBLISH_TOKEN=... \
    ./scripts/setup-release-auth.sh \
      --repo deldrid1/beehiiv-cli \
      --tap-repo deldrid1/homebrew-tap
EOF
}

repo="deldrid1/beehiiv-cli"
tap_repo="deldrid1/homebrew-tap"
tap_branch="main"
tap_formula_path="Formula/beehiiv.rb"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      repo="${2:-}"
      shift 2
      ;;
    --tap-repo)
      tap_repo="${2:-}"
      shift 2
      ;;
    --tap-branch)
      tap_branch="${2:-}"
      shift 2
      ;;
    --tap-formula-path)
      tap_formula_path="${2:-}"
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

[[ -n "${HOMEBREW_TAP_TOKEN:-}" ]] || {
  echo "HOMEBREW_TAP_TOKEN is required" >&2
  exit 1
}

[[ -n "${WINGET_PUBLISH_TOKEN:-}" ]] || {
  echo "WINGET_PUBLISH_TOKEN is required" >&2
  exit 1
}

command -v gh >/dev/null 2>&1 || {
  echo "gh CLI is required" >&2
  exit 1
}

gh auth status >/dev/null 2>&1 || {
  echo "gh CLI must already be authenticated" >&2
  exit 1
}

gh variable set HOMEBREW_TAP_REPOSITORY --repo "${repo}" --body "${tap_repo}"
gh variable set HOMEBREW_TAP_BRANCH --repo "${repo}" --body "${tap_branch}"
gh variable set HOMEBREW_TAP_FORMULA_PATH --repo "${repo}" --body "${tap_formula_path}"

printf '%s' "${HOMEBREW_TAP_TOKEN}" | gh secret set --app actions HOMEBREW_TAP_TOKEN --repo "${repo}"
printf '%s' "${WINGET_PUBLISH_TOKEN}" | gh secret set --app actions WINGET_PUBLISH_TOKEN --repo "${repo}"

echo "Updated GitHub Actions variables and secrets for ${repo}."
