#!/usr/bin/env bash

set -euo pipefail

fail() {
  echo "package-codex-plugin: $*" >&2
  exit 1
}

if [[ $# -lt 1 ]]; then
  fail "usage: $0 <version-or-tag> [dist-dir]"
fi

version="${1#v}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
dist_dir="${2:-${repo_root}/dist}"
plugin_root="${repo_root}/plugins/beehiiv-codex"
archive_path="${dist_dir}/beehiiv-codex-plugin_${version}.zip"

[[ -d "${plugin_root}" ]] || fail "plugin directory not found: ${plugin_root}"
mkdir -p "${dist_dir}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

mkdir -p "${tmp_dir}/beehiiv-codex"
cp -R "${plugin_root}/." "${tmp_dir}/beehiiv-codex/"

(
  cd "${tmp_dir}"
  zip -rq "${archive_path}" "beehiiv-codex"
)

[[ -s "${archive_path}" ]] || fail "failed to create archive: ${archive_path}"
echo "${archive_path}"
