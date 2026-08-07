#!/usr/bin/env bash
set -euo pipefail

mode="${1:-}"
demo_url="${STEAM_PROFIT_DEMO_URL:-}"
demo_username="${STEAM_PROFIT_DEMO_USERNAME:-}"
demo_password="${STEAM_PROFIT_DEMO_PASSWORD:-}"

if [[ "${mode}" != "prepare" && "${mode}" != "verify" ]]; then
  echo "usage: hosted_persistence_smoke.sh prepare|verify" >&2
  exit 2
fi
if [[ "${demo_url}" != https://* ]]; then
  echo "hosted persistence smoke requires STEAM_PROFIT_DEMO_URL with HTTPS" >&2
  exit 2
fi
if [[ -z "${demo_username}" || -z "${demo_password}" ]]; then
  echo "hosted persistence smoke requires demo username and password" >&2
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

demo_url="${demo_url%/}"
records_url="${demo_url}/api/v1/records?item=Railway%20Hosted%20Persistence%20v0.6"
curl_common=(--fail --silent --show-error --max-time 15)

curl_with_demo_access() {
  local credential="${demo_username}:${demo_password}"
  credential="${credential//\\/\\\\}"
  credential="${credential//\"/\\\"}"
  printf 'user = "%s"\n' "${credential}" | curl "${curl_common[@]}" --config - "$@"
}

record_payload="$(curl_with_demo_access "${records_url}")"
if [[ "${mode}" == "prepare" && "${record_payload}" == *'"total":0'* ]]; then
  curl_with_demo_access \
    --request POST \
    --header 'Content-Type: application/json' \
    --data '{"item_name":"Railway Hosted Persistence v0.6","buy_price":0.48,"quantity":1,"multiplier":0.837,"sell_prices":[0.8]}' \
    "${demo_url}/api/v1/calculations" >/dev/null
  record_payload="$(curl_with_demo_access "${records_url}")"
fi

if [[ "${record_payload}" != *'"total":1'* ]]; then
  echo "expected exactly one hosted persistence marker" >&2
  exit 1
fi

echo "hosted persistence ${mode} passed"
