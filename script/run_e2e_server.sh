#!/usr/bin/env bash
set -euo pipefail

e2e_runtime_dir="$(mktemp -d "${TMPDIR:-/tmp}/steam-profit-e2e.XXXXXX")"
e2e_server_bin="${e2e_runtime_dir}/steam-profit-server"

cleanup() {
  rm -rf "${e2e_runtime_dir}"
}
trap cleanup EXIT

build_flags=()
if [[ "$(uname -s)" == "Darwin" ]]; then
  build_flags=(-ldflags=-linkmode=external)
fi

go build "${build_flags[@]}" -trimpath -o "${e2e_server_bin}" ./cmd/steam-profit-server
if [[ "$(uname -s)" == "Darwin" ]]; then
  codesign --force --sign - "${e2e_server_bin}"
fi

STEAM_PROFIT_AI_PROVIDER=disabled \
  "${e2e_server_bin}" \
  --addr 127.0.0.1:18080 \
  --db "${e2e_runtime_dir}/records.db" \
  --import-csv demo/seed.csv
