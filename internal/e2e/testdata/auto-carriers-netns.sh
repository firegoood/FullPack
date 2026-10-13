#!/bin/bash
# Opt-in fixture for a disposable privileged Linux container, never a user VPS.
set -euo pipefail
test "${FULLPACK_AUTO_NETNS:-}" = 1
test "$(id -u)" = 0
work=$(mktemp -d); suffix=$$
iran=fpx-i-$suffix; kharej=fpx-k-$suffix
spid=; cpid=; bpid=
stop_pair(){
 for pid in "$spid" "$cpid" "$bpid";do if test -n "$pid";then kill "$pid" 2>/dev/null||true;wait "$pid" 2>/dev/null||true;fi;done
 spid=;cpid=;bpid=
}
cleanup(){
 result=$?
 if test "$result" != 0;then cat "$work/iran.log" "$work/kharej.log" 2>/dev/null||true;fi
 stop_pair
 ip netns del "$iran" 2>/dev/null||true;ip netns del "$kharej" 2>/dev/null||true
 rm -rf -- "$work"
}
trap cleanup EXIT
go build -o "$work/fullpack" .
go build -o "$work/fixture" internal/e2e/testdata/waticmpfixture.go
ip netns add "$iran";ip netns add "$kharej"
ip link add fpx-i-$suffix type veth peer name fpx-k-$suffix
ip link set fpx-i-$suffix netns "$iran";ip link set fpx-k-$suffix netns "$kharej"
ip -n "$iran" link set fpx-i-$suffix name eth0;ip -n "$kharej" link set fpx-k-$suffix name eth0
ip -n "$iran" addr add 10.79.0.1/24 dev eth0;ip -n "$kharej" addr add 10.79.0.2/24 dev eth0
for ns in "$iran" "$kharej";do
 ip -n "$ns" link set lo up;ip -n "$ns" link set eth0 up
 ip -n "$ns" link set eth0 mtu "${FULLPACK_AUTO_OUTER_MTU:-700}"
 ip netns exec "$ns" sysctl -qw net.ipv4.conf.all.rp_filter=0 net.ipv4.conf.eth0.rp_filter=0
done
status(){ "$work/fullpack" auto status "$1" --json; }
field(){ status "$1" | python3 -c "import json,sys;print(json.load(sys.stdin).get('$2',''))"; }
wait_route(){
 for attempt in $(seq 1 45);do
  if test "$(field "$iname" active 2>/dev/null||true)" = "$1" && test "$(field "$kname" active 2>/dev/null||true)" = "$1";then
   test "$(field "$iname" generation)" = "$(field "$kname" generation)";return
  fi
  sleep 1
 done
 status "$iname"||true;status "$kname"||true;return 1
}
traffic(){
 ip netns exec "$iran" "$work/fixture" -mode client -address "$entry:22080" -streams 4 -bytes 32768 -timeout 20s
 if test "$plane" = l3;then
  if test "${FULLPACK_AUTO_L3_FORWARD:-}" = 1;then
   ip netns exec "$iran" "$work/fixture" -mode client -address 127.0.0.1:23080 -streams 4 -bytes 32768 -timeout 20s
  fi
  ip netns exec "$iran" ping -I fpauto -c 2 -W 5 -M do -s 8972 172.29.89.2
  ip netns exec "$kharej" ping -I fpauto -c 2 -W 5 -M do -s 8972 172.29.89.1
 fi
}
n=0
for plane in ${FULLPACK_AUTO_PLANES:-stream l3};do
 carriers="${FULLPACK_AUTO_RAW_CARRIERS:-xdi pck spoof sni}"
 if test "$plane" = l3;then carriers="${FULLPACK_AUTO_L3_CARRIERS:-udp quic xdi pck spoof sni}";fi
 for carrier in $carriers;do
  if test "$plane" = l3 && test "$carrier" = quic && test "${FULLPACK_AUTO_OUTER_MTU:-700}" -lt 1200;then
   echo 'SKIPPED QUIC at outer MTU below its mandatory 1200-byte minimum; full-MTU QUIC is a separate matrix';continue
  fi
  for direction in ${FULLPACK_AUTO_DIRECTIONS:-reverse direct};do
   n=$((n+1));iname=auto-iran-$n;kname=auto-kharej-$n
   endpoint=10.79.0.1:24443;if test "$direction" = direct;then endpoint=10.79.0.2:24443;fi
   backup=stealth;entry=127.0.0.1;backend=127.0.0.1:21080
   if test "$plane" = l3;then
    backup=quic;entry=172.29.89.2;backend=172.29.89.2:22080
    if test "$carrier" = quic;then backup=udp;fi
    if test "${FULLPACK_AUTO_OUTER_MTU:-700}" -lt 1200;then backup=udp;if test "$carrier" = udp;then backup=xdi;fi;fi
   fi
   for role in iran kharej;do
    name=$iname;section=server;addrkey=bind_addr;addr=0.0.0.0:0
    if test "$role" = kharej;then name=$kname;section=client;addrkey=remote_addr;addr=10.79.0.1:24443;fi
    cat > "$work/$name.toml" <<EOF
[$section]
$addrkey = "$addr"
transport = "$carrier"
token = "disposable-auto-enrollment-secret-0123456789"
skip_optz = true
EOF
    if test "$role" = iran;then
     if test "$plane" = stream;then echo 'ports = ["22080=127.0.0.1:21080"]' >> "$work/$name.toml";elif test "${FULLPACK_AUTO_L3_FORWARD:-}" = 1;then echo 'ports = ["127.0.0.1:23080=172.29.89.2:22080"]' >> "$work/$name.toml";else echo 'ports = []' >> "$work/$name.toml";fi
     echo 'accept_udp = true' >> "$work/$name.toml"
    fi
    cat >> "$work/$name.toml" <<EOF
[auto]
enabled = true
observe_only = true
tunnel_id = "disposable-auto-$suffix-$n"
iran_id = "disposable-enrolled-iran"
kharej_id = "disposable-enrolled-kharej"
accept_udp = true
max_connections = 64
max_memory_mb = 384
cooldown = 5
minimum_dwell = 5
max_switches_per_hour = 12
probe_interval = 600
EOF
    if test "$plane" = stream;then echo 'ports = ["22080=127.0.0.1:21080"]' >> "$work/$name.toml";else
     cat >> "$work/$name.toml" <<EOF
ports = $(if test "${FULLPACK_AUTO_L3_FORWARD:-}" = 1;then printf '["127.0.0.1:23080=172.29.89.2:22080"]';else printf '[]';fi)
mode = "l3"
[auto.l3]
iran_ip = "172.29.89.1/30"
kharej_ip = "172.29.89.2/30"
mtu = 9000
interface = "fpauto"
EOF
    fi
    cat >> "$work/$name.toml" <<EOF
[[auto.candidates]]
id = "primary"
transport = "$carrier"
mode = "$direction"
endpoint = "$endpoint"
EOF
    if test "$carrier" = waticmp;then
     cat >> "$work/$name.toml" <<EOF
iran_ip = "10.79.0.1"
kharej_ip = "10.79.0.2"
direction = "${FULLPACK_AUTO_WAT_DIRECTION:-reverse}"
max_payload = 512
EOF
    elif test "$carrier" = spoof;then
     cat >> "$work/$name.toml" <<EOF
iran_ip = "10.79.0.1"
kharej_ip = "10.79.0.2"
[auto.candidates.packet]
spoof_profile = "${FULLPACK_AUTO_SPOOF_PROFILE:-udp}"
iran_source = "10.79.0.11"
kharej_source = "10.79.0.12"
interface = "eth0"
EOF
    elif test "$carrier" = pck || test "$carrier" = sni;then
     echo '[auto.candidates.packet]' >> "$work/$name.toml"
     echo 'interface = "eth0"' >> "$work/$name.toml"
     if test "$carrier" = sni;then echo 'sni_domain = "example.com"' >> "$work/$name.toml";fi
    fi
    cat >> "$work/$name.toml" <<EOF
[[auto.candidates]]
id = "backup"
transport = "$backup"
endpoint = "10.79.0.1:24444"
EOF
    if test "$plane" = l3 && test "${FULLPACK_AUTO_OUTER_MTU:-700}" -lt 1200;then
     cat >> "$work/$name.toml" <<EOF
[[auto.candidates]]
id = "large-mtu"
transport = "quic"
endpoint = "10.79.0.1:24445"
EOF
    fi
    chmod 600 "$work/$name.toml"
   done
   ip netns exec "$iran" "$work/fullpack" -c "$work/$iname.toml" > "$work/iran.log" 2>&1 & spid=$!
   ip netns exec "$kharej" "$work/fullpack" -c "$work/$kname.toml" > "$work/kharej.log" 2>&1 & cpid=$!
   wait_route primary
   ip netns exec "$kharej" "$work/fixture" -mode backend -address "$backend" & bpid=$!
   sleep 0.2
   traffic
   generation=$(field "$iname" generation)
   "$work/fullpack" auto test-now "$iname" primary
   test "$(field "$iname" generation)" = "$generation";traffic
   if test "$plane" = l3 && test "${FULLPACK_AUTO_OUTER_MTU:-700}" -lt 1200;then
    if "$work/fullpack" auto switch "$iname" large-mtu;then echo 'FAIL undersized QUIC path qualified';exit 1;fi
    test "$(field "$iname" generation)" = "$generation";wait_route primary
   fi
   "$work/fullpack" auto switch "$iname" backup
   wait_route backup;traffic
   echo "PASS $plane $carrier $direction: actual payloads, isolated probe, stable listener/TUN, agreed generation handover"
   stop_pair
  done
 done
done
