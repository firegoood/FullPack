#!/usr/bin/env bash
# Run only synthetic accounts on generated loopback ports. No production restart.
set -euo pipefail
if [[ $# -ne 3 || $EUID -ne 0 ]]; then
  echo "Usage (root): bash $0 /path/auto-load-audit /path/xray /path/new-results.json" >&2
  exit 2
fi
audit_binary=$(realpath -- "$1")
xray_binary=$(realpath -- "$2")
result=$(realpath -m -- "$3")
[[ -f $audit_binary && -x $audit_binary && -f $xray_binary && -x $xray_binary ]]
if [[ -e $result || -e $result.log ]]; then
  echo "Refusing to overwrite an existing report/log." >&2
  exit 2
fi
fixture=$(mktemp -d /tmp/fullpack-synthetic.XXXXXX)
unit="fullpack-synthetic-${fixture##*.}"
cleanup() {
  # Stop only a unit whose executable is inside the directory we just created.
  if [[ $(systemctl show "$unit.service" -p ExecStart --value 2>/dev/null) == *"$fixture/auto-load-audit"* ]]; then
    systemctl stop "$unit.service" >/dev/null 2>&1 || true
  fi
  case "$fixture" in
    /tmp/fullpack-synthetic.??????) rm -rf -- "$fixture" ;;
  esac
}
trap cleanup EXIT
umask 077
chown nobody:nogroup "$fixture"
chmod 0750 "$fixture"
install -m 0755 -- "$audit_binary" "$fixture/auto-load-audit"
install -m 0755 -- "$xray_binary" "$fixture/xray-fixture"
set +e
systemd-run --unit="$unit" --wait --pipe --collect \
  -p User=nobody -p Group=nogroup -p PrivateNetwork=yes \
  -p NoNewPrivileges=yes -p CPUQuota=100% -p CPUWeight=10 \
  -p MemoryMax=512M -p MemorySwapMax=0 -p TasksMax=128 \
  -p LimitNOFILE=16384 -p RuntimeMaxSec=900 -p Nice=10 -p UMask=0077 \
  -p ProtectSystem=strict -p ProtectHome=yes \
  -p "ReadWritePaths=$fixture" \
  -p 'InaccessiblePaths=/etc/fullpack /var/lib/firenode /opt/firenode' \
  --setenv=GOMAXPROCS=1 --setenv="TMPDIR=$fixture" \
  "$fixture/auto-load-audit" --run --users=500,800 \
  --xray="$fixture/xray-fixture" --output="$fixture/results.json" \
  2>&1 | tee "$result.log"
status=${PIPESTATUS[0]}
set -e
if [[ -f $fixture/results.json ]]; then
  cp -- "$fixture/results.json" "$result"
  chmod 0600 "$result"
fi
exit "$status"
