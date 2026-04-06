#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

cd "${repo_root}"

make docs

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Skipping generated-artifact drift check outside a git worktree."
  exit 0
fi

generated_paths=(
  "docs/reference/cli"
  "share/man/man1"
  "share/completions"
)

status="$(git status --porcelain --untracked-files=all -- "${generated_paths[@]}")"
if [[ -n "${status}" ]]; then
  echo "Generated artifacts are out of date. Run 'make docs' and commit the results."
  echo
  echo "${status}"
  exit 1
fi
