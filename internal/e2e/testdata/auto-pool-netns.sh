#!/bin/bash
# Actual datagram pool/TUN acceptance on an isolated disposable CI runner.
set -euo pipefail
test "${FULLPACK_AUTO_NETNS:-}" = 1
test "$(id -u)" = 0
work=$(mktemp -d); suffix=$$
iran=fpp-i-$suffix; kharej=fpp-k-$suffix
spid=; cpid=; bpid=
cleanup(){
 result=$?
 if test "$result" != 0;then cat "$work/iran.log" "$work/kharej.log" 2>/dev/null||true;fi
 for pid in "$spid" "$cpid" "$bpid";do if test -n "$pid";then kill "$pid" 2>/dev/null||true;wait "$pid" 2>/dev/null||true;fi;done
 ip netns del "$iran" 2>/dev/null||true;ip netns del "$kharej" 2>/dev/null||true
 rm -rf -- "$work"
}
trap cleanup EXIT
go build -o "$work/fullpack" .
go build -o "$work/fixture" internal/e2e/testdata/waticmpfixture.go
go build -o "$work/config" internal/e2e/testdata/auto-pool-config.go
"$work/config" "$work"
ip netns add "$iran";ip netns add "$kharej"
ip link add fpp-i-$suffix type veth peer name fpp-k-$suffix
ip link set fpp-i-$suffix netns "$iran";ip link set fpp-k-$suffix netns "$kharej"
ip -n "$iran" link set fpp-i-$suffix name eth0;ip -n "$kharej" link set fpp-k-$suffix name eth0
ip -n "$iran" addr add 10.79.0.1/24 dev eth0;ip -n "$kharej" addr add 10.79.0.2/24 dev eth0
for ns in "$iran" "$kharej";do
 ip -n "$ns" link set lo up;ip -n "$ns" link set eth0 up
 ip netns exec "$ns" sysctl -qw net.ipv4.conf.all.rp_filter=0 net.ipv4.conf.eth0.rp_filter=0
done
ip netns exec "$iran" "$work/fullpack" -c "$work/iran.toml" > "$work/iran.log" 2>&1 & spid=$!
ip netns exec "$kharej" "$work/fullpack" -c "$work/kharej.toml" > "$work/kharej.log" 2>&1 & cpid=$!
status(){ "$work/fullpack" auto status "$1" --json; }
wait_four(){
 for attempt in $(seq 1 180);do
  status iran > "$work/i.json" 2>/dev/null||true;status kharej > "$work/k.json" 2>/dev/null||true
  if python3 - "$work" <<'PY'
import json,sys
try:
 a,b=[json.load(open(sys.argv[1]+'/'+x+'.json')) for x in ('i','k')]
 assert len(a['candidates']) == len(b['candidates']) == 12
 assert a['selection']['healthy'] == b['selection']['healthy'] == 4
 assert not a['selection']['scanning']
 assert a['selection']['selected'] == b['selection']['selected']
 assert len(a['selection']['selected']) == 4
 assert a['active'] and a['active'] == b['active']
 assert a['generation'] == b['generation']
except (AssertionError,KeyError,ValueError):sys.exit(1)
PY
  then return;fi
  sleep 1
 done
 cat "$work/i.json" "$work/k.json";return 1
}
traffic(){
 ip netns exec "$iran" "$work/fixture" -mode client -address 172.29.89.2:22080 -streams 4 -bytes 32768 -timeout 20s
 ip netns exec "$iran" ping -I fpauto -c 2 -W 5 -M do -s 1300 172.29.89.2
 ip netns exec "$kharej" ping -I fpauto -c 2 -W 5 -M do -s 1300 172.29.89.1
}
wait_four
ip netns exec "$kharej" "$work/fixture" -mode backend -address 172.29.89.2:22080 & bpid=$!
sleep 0.2
traffic
ip -n "$iran" -j link show fpauto > "$work/tun-before.json"
cp "$work/i.json" "$work/before.json"
mapfile -t failed < <(python3 - "$work/before.json" <<'PY'
import json,sys
s=json.load(open(sys.argv[1]))
print('\n'.join([x for x in s['selection']['selected'] if x!=s['active']][:2]))
PY
)
test "${#failed[@]}" = 2
# Explicitly retire two real selected datagram paths at the Iran authority. A separate
# loopback E2E test faults their network; here the accepted runtime exclusions
# must trigger fresh measurement and preserve the one stable TUN.
for id in "${failed[@]}";do
 "$work/fullpack" auto disable-carrier iran "$id"
done
wait_four
traffic
ip -n "$iran" -j link show fpauto > "$work/tun-after.json"
python3 - "$work" "${failed[@]}" <<'PY'
import json,sys
d=sys.argv[1]
before=json.load(open(d+'/before.json'));after=json.load(open(d+'/i.json'))
assert before['active']==after['active'] and before['generation']==after['generation']
assert all(x not in after['selection']['selected'] for x in sys.argv[2:])
assert json.load(open(d+'/tun-before.json'))[0]['ifindex']==json.load(open(d+'/tun-after.json'))[0]['ifindex']
print('PASS: twelve enrolled datagrams, 4 -> 2 -> 4, same active generation and TUN, bidirectional IP integrity')
PY
