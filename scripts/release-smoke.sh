#!/usr/bin/env bash

set -euo pipefail

fail() {
  echo "release-smoke: $*" >&2
  exit 1
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
dist_dir="${1:-${repo_root}/dist}"

[[ -d "${dist_dir}" ]] || fail "dist directory not found: ${dist_dir}"
[[ -s "${dist_dir}/checksums.txt" ]] || fail "missing checksums.txt"
[[ -s "${dist_dir}/artifacts.json" ]] || fail "missing artifacts.json"

for generated_file in \
  "${repo_root}/share/man/man1/beehiiv.1" \
  "${repo_root}/share/completions/beehiiv.bash" \
  "${repo_root}/share/completions/beehiiv.fish" \
  "${repo_root}/share/completions/beehiiv.ps1" \
  "${repo_root}/share/completions/_beehiiv"; do
  [[ -s "${generated_file}" ]] || fail "missing generated asset: ${generated_file}"
done

archives=()
while IFS= read -r archive; do
  archives+=("${archive}")
done < <(find "${dist_dir}" -maxdepth 1 -type f \( -name 'beehiiv_*.tar.gz' -o -name 'beehiiv_*.zip' \) | sort)
[[ "${#archives[@]}" -gt 0 ]] || fail "no release archives found"

plugin_archive="$(find "${dist_dir}" -maxdepth 1 -type f -name 'beehiiv-codex-plugin_*.zip' | sort | head -n1)"
[[ -n "${plugin_archive}" ]] || fail "missing Codex plugin archive"

required_patterns=(
  'darwin_arm64'
  'darwin_x86_64'
  'linux_arm64'
  'linux_x86_64'
  'windows_arm64'
  'windows_x86_64'
)

archive_names="$(printf '%s\n' "${archives[@]##*/}")"
for pattern in "${required_patterns[@]}"; do
  if ! grep -q "${pattern}" <<<"${archive_names}"; then
    fail "expected release archive containing ${pattern}"
  fi
done

while IFS= read -r line; do
  [[ -n "${line}" ]] || continue
  artifact="${line##* }"
  [[ -f "${dist_dir}/${artifact}" ]] || fail "checksums.txt references missing artifact ${artifact}"
done < "${dist_dir}/checksums.txt"

if find "${dist_dir}" -type f -name '*.installer.yaml' | grep -q .; then
  :
else
  fail "winget manifest output not found in dist"
fi

first_tarball="${archives[0]}"
if [[ "${first_tarball}" != *.tar.gz ]]; then
  for candidate in "${archives[@]}"; do
    if [[ "${candidate}" == *.tar.gz ]]; then
      first_tarball="${candidate}"
      break
    fi
  done
fi
[[ "${first_tarball}" == *.tar.gz ]] || fail "no tar.gz archive found"

tar_listing="$(tar -tzf "${first_tarball}")"
grep -q 'README.md$' <<<"${tar_listing}" || fail "archive missing README.md"
grep -q 'LICENSE$' <<<"${tar_listing}" || fail "archive missing LICENSE"
grep -q 'share/man/man1/beehiiv.1$' <<<"${tar_listing}" || fail "archive missing manpage"
grep -q 'share/completions/beehiiv.bash$' <<<"${tar_listing}" || fail "archive missing bash completion"

first_zip=""
for candidate in "${archives[@]}"; do
  if [[ "${candidate}" == *.zip ]]; then
    first_zip="${candidate}"
    break
  fi
done
[[ -n "${first_zip}" ]] || fail "no zip archive found"
if command -v unzip >/dev/null 2>&1; then
  zip_listing="$(unzip -Z1 "${first_zip}")"
  grep -q 'README.md$' <<<"${zip_listing}" || fail "zip archive missing README.md"
  grep -q 'LICENSE$' <<<"${zip_listing}" || fail "zip archive missing LICENSE"
  grep -q 'share/completions/beehiiv.ps1$' <<<"${zip_listing}" || fail "zip archive missing PowerShell completion"

  plugin_listing="$(unzip -Z1 "${plugin_archive}")"
  grep -q '^beehiiv-codex/.codex-plugin/plugin.json$' <<<"${plugin_listing}" || fail "plugin archive missing plugin manifest"
  grep -q '^beehiiv-codex/skills/beehiiv-reporting-assistant/SKILL.md$' <<<"${plugin_listing}" || fail "plugin archive missing skill"
fi

host_os="$(uname -s)"
host_arch="$(uname -m)"

case "${host_os}" in
  Darwin)
    runnable_os="darwin"
    ;;
  Linux)
    runnable_os="linux"
    ;;
  *)
    fail "unsupported host OS for executable smoke test: ${host_os}"
    ;;
esac

case "${host_arch}" in
  x86_64|amd64)
    runnable_arch="x86_64"
    ;;
  arm64|aarch64)
    runnable_arch="arm64"
    ;;
  *)
    fail "unsupported host architecture for executable smoke test: ${host_arch}"
    ;;
esac

runnable_tarball=""
for candidate in "${archives[@]}"; do
  if [[ "${candidate}" == *"${runnable_os}_${runnable_arch}"*.tar.gz ]]; then
    runnable_tarball="${candidate}"
    break
  fi
done
[[ -n "${runnable_tarball}" ]] || fail "missing ${runnable_os}_${runnable_arch} archive"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

tar -xzf "${runnable_tarball}" -C "${tmp_dir}"
runnable_binary="$(find "${tmp_dir}" -type f -name 'beehiiv' | head -n1)"
[[ -x "${runnable_binary}" ]] || fail "archive does not contain an executable beehiiv binary"

"${runnable_binary}" --help >/dev/null
"${runnable_binary}" completion bash >/dev/null
version_output="$("${runnable_binary}" version)"
grep -q 'beehiiv version ' <<<"${version_output}" || fail "release binary returned unexpected version output"
