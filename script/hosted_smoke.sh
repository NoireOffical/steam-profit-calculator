#!/usr/bin/env bash
set -euo pipefail

demo_url="${STEAM_PROFIT_DEMO_URL:-${1:-}}"
demo_username="${STEAM_PROFIT_DEMO_USERNAME:-}"
demo_password="${STEAM_PROFIT_DEMO_PASSWORD:-}"

if [[ -z "${demo_url}" ]]; then
  echo "STEAM_PROFIT_DEMO_URL or the first URL argument is required" >&2
  exit 2
fi
demo_url="${demo_url%/}"
if [[ "${demo_url}" != https://* && "${STEAM_PROFIT_ALLOW_HTTP_SMOKE:-false}" != "true" ]]; then
  echo "hosted smoke requires HTTPS; set STEAM_PROFIT_ALLOW_HTTP_SMOKE=true only for local testing" >&2
  exit 2
fi
if [[ -n "${demo_username}" && -z "${demo_password}" ]] || [[ -z "${demo_username}" && -n "${demo_password}" ]]; then
  echo "hosted smoke requires both demo username and password when authentication is enabled" >&2
  exit 2
fi
if [[ "${demo_username}" == *:* || "${demo_username}" == *$'\r'* || "${demo_username}" == *$'\n'* ]]; then
  echo "demo username must not contain a colon or line break" >&2
  exit 2
fi
if [[ "${demo_password}" == *$'\r'* || "${demo_password}" == *$'\n'* ]]; then
  echo "demo password must not contain a line break" >&2
  exit 2
fi

curl_common=(--fail --silent --show-error --max-time 15)
health_payload="$(curl "${curl_common[@]}" "${demo_url}/api/health")"
if [[ "${health_payload}" != *'"status":"ok"'* ]]; then
  echo "dependency health response did not report ok" >&2
  exit 1
fi

curl_with_demo_access() {
  if [[ -z "${demo_username}" ]]; then
    curl "${curl_common[@]}" "$@"
    return
  fi
  local credential="${demo_username}:${demo_password}"
  credential="${credential//\\/\\\\}"
  credential="${credential//\"/\\\"}"
  printf 'user = "%s"\n' "${credential}" | curl "${curl_common[@]}" --config - "$@"
}

if [[ -n "${demo_username}" ]]; then
  unauthenticated_status="$(curl --silent --show-error --max-time 15 --output /dev/null --write-out '%{http_code}' "${demo_url}/")"
  if [[ "${unauthenticated_status}" != "401" ]]; then
    echo "protected demo did not reject an unauthenticated browser request" >&2
    exit 1
  fi
fi

browser_payload="$(curl_with_demo_access "${demo_url}/")"
if [[ "${browser_payload}" != *"Steam Profit Lab"* ]]; then
  echo "browser application marker is missing" >&2
  exit 1
fi
assistant_payload="$(curl_with_demo_access "${demo_url}/api/v1/assistant/status")"
if [[ "${assistant_payload}" != *'"prompt_version"'* || "${assistant_payload}" != *'"available"'* ]]; then
  echo "assistant status response is missing safe capability metadata" >&2
  exit 1
fi

echo "hosted smoke passed: dependency health, access boundary, browser, and assistant status"
