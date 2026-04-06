#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  install-test.sh homebrew [--tag <tag>] [--release-repository <owner/repo>]

Environment:
  RELEASE_TAG           Git tag to validate, e.g. v1.2.3
  RELEASE_REPOSITORY    Release repository, e.g. owner/repo

Examples:
  RELEASE_TAG=v1.2.3 RELEASE_REPOSITORY=deldrid1/beehiiv-cli \
    ./scripts/install-test.sh homebrew
EOF
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

command_name="${1:-}"
if [[ -z "${command_name}" ]]; then
  usage >&2
  exit 1
fi
shift

release_tag="${RELEASE_TAG:-}"
release_repository="${RELEASE_REPOSITORY:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag)
      release_tag="${2:-}"
      shift 2
      ;;
    --release-repository)
      release_repository="${2:-}"
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

[[ -n "${release_tag}" ]] || { echo "RELEASE_TAG or --tag is required" >&2; exit 1; }
[[ -n "${release_repository}" ]] || { echo "RELEASE_REPOSITORY or --release-repository is required" >&2; exit 1; }

case "${command_name}" in
  homebrew)
    [[ "$(uname -s)" == "Darwin" ]] || {
      echo "homebrew install validation currently requires macOS" >&2
      exit 1
    }
    command -v brew >/dev/null 2>&1 || {
      echo "brew is required for homebrew install validation" >&2
      exit 1
    }
    if brew list --versions beehiiv >/dev/null 2>&1; then
      echo "beehiiv is already installed via Homebrew; remove it before running this validator" >&2
      exit 1
    fi

    tap_name="local/beehiiv-cli-test"
    tap_dir="$(mktemp -d "${TMPDIR:-/tmp}/beehiiv-homebrew-tap.XXXXXX")"
    formula_path="${tap_dir}/Formula/beehiiv.rb"
    cleanup() {
      brew uninstall --formula beehiiv >/dev/null 2>&1 || true
      brew untap --force "${tap_name}" >/dev/null 2>&1 || true
      rm -rf "${tap_dir}"
    }
    trap cleanup EXIT

    "${repo_root}/scripts/update-homebrew-tap.sh" \
      --tag "${release_tag}" \
      --release-repository "${release_repository}" \
      --render-only "${formula_path}"

    git init -q "${tap_dir}"
    git -C "${tap_dir}" config user.name "install-test"
    git -C "${tap_dir}" config user.email "install-test@example.com"
    git -C "${tap_dir}" add "Formula/beehiiv.rb"
    git -C "${tap_dir}" commit -q -m "Add beehiiv formula"

    HOMEBREW_NO_AUTO_UPDATE=1 brew tap --custom-remote "${tap_name}" "${tap_dir}"
    HOMEBREW_NO_AUTO_UPDATE=1 brew install --formula "${tap_name}/beehiiv"
    beehiiv version
    beehiiv completion bash >/dev/null
    HOMEBREW_NO_AUTO_UPDATE=1 brew test "${tap_name}/beehiiv"
    ;;
  *)
    echo "unknown install-test command: ${command_name}" >&2
    usage >&2
    exit 1
    ;;
esac
