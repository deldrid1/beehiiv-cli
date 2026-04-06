#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  update-homebrew-tap.sh [--tag <tag>] [--release-repository <owner/repo>] [--render-only <output>]

Environment:
  RELEASE_TAG                  Git tag to publish, e.g. v1.2.3
  RELEASE_REPOSITORY           Release repository, e.g. owner/repo
  HOMEBREW_TAP_REPOSITORY      Tap repository, e.g. owner/homebrew-tap
  HOMEBREW_TAP_BRANCH          Tap branch, default: main
  HOMEBREW_TAP_FORMULA_PATH    Formula path in the tap, default: Formula/beehiiv.rb
  HOMEBREW_TAP_TOKEN           Personal access token with contents:write on the tap repo

Examples:
  RELEASE_TAG=v1.2.3 RELEASE_REPOSITORY=deldrid1/beehiiv-cli \
    ./scripts/update-homebrew-tap.sh --render-only /tmp/beehiiv.rb

  RELEASE_TAG=v1.2.3 RELEASE_REPOSITORY=deldrid1/beehiiv-cli \
    HOMEBREW_TAP_REPOSITORY=deldrid1/homebrew-tap HOMEBREW_TAP_TOKEN=... \
    ./scripts/update-homebrew-tap.sh
EOF
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

release_tag="${RELEASE_TAG:-}"
release_repository="${RELEASE_REPOSITORY:-}"
render_only=""

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
    --render-only)
      render_only="${2:-}"
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

release_version="${release_tag#v}"

template_path="${repo_root}/packaging/homebrew/beehiiv.rb.tmpl"
formula_path="${HOMEBREW_TAP_FORMULA_PATH:-Formula/beehiiv.rb}"
tap_repository="${HOMEBREW_TAP_REPOSITORY:-}"
tap_branch="${HOMEBREW_TAP_BRANCH:-main}"
tap_token="${HOMEBREW_TAP_TOKEN:-}"

api_url="https://api.github.com/repos/${release_repository}/releases/tags/${release_tag}"
if [[ -n "${tap_token}" ]]; then
  release_json="$(curl -fsSL -H "Authorization: Bearer ${tap_token}" -H "Accept: application/vnd.github+json" "${api_url}")"
else
  release_json="$(curl -fsSL -H "Accept: application/vnd.github+json" "${api_url}")"
fi

checksums_url="$(python3 - <<'PY' "${release_json}"
import json
import sys

release = json.loads(sys.argv[1])
for asset in release["assets"]:
    if asset["name"] == "checksums.txt":
        print(asset["browser_download_url"])
        break
PY
)"

[[ -n "${checksums_url}" ]] || { echo "checksums.txt asset not found for ${release_tag}" >&2; exit 1; }
checksums_file="$(mktemp)"
trap 'rm -f "${checksums_file}"' EXIT
curl -fsSL -o "${checksums_file}" "${checksums_url}"

asset_info="$(python3 - <<'PY' "${release_json}" "${release_version}"
import json
import sys

release = json.loads(sys.argv[1])
version = sys.argv[2]
targets = [
    ("darwin_amd64", f"beehiiv_{version}_darwin_x86_64.tar.gz"),
    ("darwin_arm64", f"beehiiv_{version}_darwin_arm64.tar.gz"),
    ("linux_amd64", f"beehiiv_{version}_linux_x86_64.tar.gz"),
    ("linux_arm64", f"beehiiv_{version}_linux_arm64.tar.gz"),
]

assets = {asset["name"]: asset["browser_download_url"] for asset in release["assets"]}
for key, name in targets:
    if name not in assets:
        raise SystemExit(f"missing asset {name}")
    print(f"{key}|{name}|{assets[name]}")
PY
)"

darwin_amd64_name=""
darwin_amd64_url=""
darwin_arm64_name=""
darwin_arm64_url=""
linux_amd64_name=""
linux_amd64_url=""
linux_arm64_name=""
linux_arm64_url=""
while IFS='|' read -r key name url; do
  case "${key}" in
    darwin_amd64)
      darwin_amd64_name="${name}"
      darwin_amd64_url="${url}"
      ;;
    darwin_arm64)
      darwin_arm64_name="${name}"
      darwin_arm64_url="${url}"
      ;;
    linux_amd64)
      linux_amd64_name="${name}"
      linux_amd64_url="${url}"
      ;;
    linux_arm64)
      linux_arm64_name="${name}"
      linux_arm64_url="${url}"
      ;;
  esac
done <<< "${asset_info}"

checksum_for() {
  local filename="$1"
  local checksum
  checksum="$(awk -v file="${filename}" '$2 == file { print $1 }' "${checksums_file}")"
  [[ -n "${checksum}" ]] || {
    echo "missing checksum for ${filename}" >&2
    exit 1
  }
  printf '%s' "${checksum}"
}

render_formula() {
  local output_path="$1"
  python3 - <<'PY' "${template_path}" "${output_path}" "${release_tag}" \
    "${darwin_arm64_url}" "${darwin_amd64_url}" "${linux_arm64_url}" "${linux_amd64_url}" \
    "$(checksum_for "${darwin_arm64_name}")" "$(checksum_for "${darwin_amd64_name}")" "$(checksum_for "${linux_arm64_name}")" "$(checksum_for "${linux_amd64_name}")" \
    "${release_repository}"
from pathlib import Path
import sys

template_path = Path(sys.argv[1])
output_path = Path(sys.argv[2])
tag = sys.argv[3]
darwin_arm64_url = sys.argv[4]
darwin_amd64_url = sys.argv[5]
linux_arm64_url = sys.argv[6]
linux_amd64_url = sys.argv[7]
darwin_arm64_sha = sys.argv[8]
darwin_amd64_sha = sys.argv[9]
linux_arm64_sha = sys.argv[10]
linux_amd64_sha = sys.argv[11]
repository = sys.argv[12]

replacements = {
    "__CLASS_NAME__": "Beehiiv",
    "__DESC__": "Cross-platform Beehiiv API CLI",
    "__HOMEPAGE__": f"https://github.com/{repository}",
    "__VERSION__": tag[1:] if tag.startswith("v") else tag,
    "__LICENSE__": "MIT",
    "__URL_DARWIN_ARM64__": darwin_arm64_url,
    "__URL_DARWIN_AMD64__": darwin_amd64_url,
    "__URL_LINUX_ARM64__": linux_arm64_url,
    "__URL_LINUX_AMD64__": linux_amd64_url,
    "__SHA_DARWIN_ARM64__": darwin_arm64_sha,
    "__SHA_DARWIN_AMD64__": darwin_amd64_sha,
    "__SHA_LINUX_ARM64__": linux_arm64_sha,
    "__SHA_LINUX_AMD64__": linux_amd64_sha,
}

contents = template_path.read_text()
for key, value in replacements.items():
    contents = contents.replace(key, value)

output_path.parent.mkdir(parents=True, exist_ok=True)
output_path.write_text(contents)
PY
}

if [[ -n "${render_only}" ]]; then
  render_formula "${render_only}"
  exit 0
fi

[[ -n "${tap_repository}" ]] || { echo "HOMEBREW_TAP_REPOSITORY is required when not using --render-only" >&2; exit 1; }
[[ -n "${tap_token}" ]] || { echo "HOMEBREW_TAP_TOKEN is required when not using --render-only" >&2; exit 1; }

tap_dir="$(mktemp -d)"
trap 'rm -rf "${tap_dir}" "${checksums_file}"' EXIT

git clone --branch "${tap_branch}" "https://x-access-token:${tap_token}@github.com/${tap_repository}.git" "${tap_dir}" >/dev/null 2>&1
render_formula "${tap_dir}/${formula_path}"

cd "${tap_dir}"

if git diff --quiet -- "${formula_path}"; then
  echo "Homebrew formula already up to date."
  exit 0
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add "${formula_path}"
git commit -m "beehiiv ${release_tag}" >/dev/null
git push origin "${tap_branch}"
