#!/usr/bin/env bash
set -euo pipefail

bin_path=""
if [[ -n "${BEEHIIV_CLI_BIN:-}" ]]; then
  bin_path="${BEEHIIV_CLI_BIN}"
elif command -v beehiiv >/dev/null 2>&1; then
  bin_path="$(command -v beehiiv)"
fi

if [[ -z "${bin_path}" ]]; then
  printf '{\n'
  printf '  "available": false,\n'
  printf '  "configured": false,\n'
  printf '  "publication_id": "",\n'
  printf '  "reports_available": false,\n'
  printf '  "beehiiv_bin": "",\n'
  printf '  "version": ""\n'
  printf '}\n'
  exit 0
fi

version_output="$("${bin_path}" version 2>/dev/null || true)"
auth_output="$("${bin_path}" auth status --output json 2>/dev/null || true)"
reports_available=false
if "${bin_path}" reports --help >/dev/null 2>&1; then
  reports_available=true
fi

configured=false
if printf '%s' "${auth_output}" | grep -q '"configured"[[:space:]]*:[[:space:]]*true'; then
  configured=true
fi

publication_id="$(printf '%s' "${auth_output}" | sed -n 's/.*"publication_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"

escaped_bin="$(printf '%s' "${bin_path}" | sed 's/\\/\\\\/g; s/"/\\"/g')"
escaped_version="$(printf '%s' "${version_output}" | tr '\n' ' ' | sed 's/\\/\\\\/g; s/"/\\"/g')"
escaped_publication_id="$(printf '%s' "${publication_id}" | sed 's/\\/\\\\/g; s/"/\\"/g')"

printf '{\n'
printf '  "available": true,\n'
printf '  "configured": %s,\n' "${configured}"
printf '  "publication_id": "%s",\n' "${escaped_publication_id}"
printf '  "reports_available": %s,\n' "${reports_available}"
printf '  "beehiiv_bin": "%s",\n' "${escaped_bin}"
printf '  "version": "%s"\n' "${escaped_version}"
printf '}\n'
