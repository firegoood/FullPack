#!/bin/sh
# Race instrumentation retains process-level allocation metadata between cases.
# Give every carrier/direction the same fresh-process resource budget. Keep all
# nine held-connection cases, their assertions and production pressure guards.
set -eu
test "${FULLPACK_AUTO_UNLIMITED_LOAD:-}" = 1
for carrier in tcp wss stealth quic; do
  for direction in reverse direct; do
    go test ./internal/e2e -race -v -count=1 -timeout=3m \
      -run "^TestPeerAutoUnlimitedApplicationConnections$/${carrier}$/${direction}$"
  done
done
go test ./internal/e2e -race -v -count=1 -timeout=3m \
  -run '^TestPeerAutoUnlimitedApplicationConnections$/mixed-tcp-udp$'
