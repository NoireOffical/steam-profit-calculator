#!/bin/sh
set -eu

if [ "$(id -u)" -eq 0 ]; then
  chown steam-profit:steam-profit /data
  exec su-exec steam-profit:steam-profit /app/steam-profit-server "$@"
fi

exec /app/steam-profit-server "$@"
