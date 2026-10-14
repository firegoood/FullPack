# Auto policy and WAN criticism review

Date: 2026-10-14 (Asia/Tehran). Status: **BOUNDED REVIEW PASSED WITH OPEN GATES**.
The original 2026-10-13 WAN matrix remains historical evidence. This review
does not imply a signed release, permanent deployment or a 24-hour soak.

## Findings and decisions

| Criticism | Finding and resulting scope |
| --- | --- |
| Consecutive outages can inherit the 300-second cooldown | Confirmed by a real three-carrier regression. Emergency recovery now has a separate bounded gate; optional quality switching keeps its original policy. |
| Timeout can be reported as identity/policy rejection | Confirmed. Status now carries `errorCode` and `errorStage`, with separate timeout, rejection, closed/refused connection, bind, local denial and resource-owner categories. |
| QUIC cannot be blamed only on Kharej2 or MTU | Correct. A valid exclusive-port retry and manual QUIC on Kharej1 failed, while ordinary authenticated UDP/DF on the same port passed. The filtering device/cause remains unidentified. |
| Wat's superiority on the hardest route is unproved | Correct. A smaller normal-Echo retry on Kharej2 also failed to qualify. No comparative superiority claim is made. |
| WAN L3 automatic recovery is untested | Now tested for UDP to xDi on Kharej1 with the same real TUN and agreed generation. Automatic PCK/SNI recovery on Kharej2 remains NOT VERIFIED. |
| Forced-stop leftovers do not prove graceful cleanup | Correct. The original harness's one-second active stop and later group signaling confounded that claim. Lock contention and unjoined cancellation callbacks were also reviewed and repaired. Direct engine TERM is recorded separately. |
| 192 connections is not 192 independent users | Correct. Default admission remains 64, maximum 192. A bounded local 100/192-socket test passed; 1000-user production capacity and sustained WAN load remain NOT VERIFIED. |

## Consecutive failure regression and policy

`TestPeerAutoConsecutiveFailuresKeepDefaultsAndReachThirdCarrier` runs actual
TCP, WSS and WSSMUX carriers through separate fault proxies, with both enrolled
managers alive. All three authenticate; standby carriers also pass three
independent integrity probes. The test retains the real defaults, 300 seconds
and two changes per hour, then closes only TCP's network connections, followed
shortly by WSS's connections. It requires agreed, increasing generations and
actual TCP/UDP integrity on the third carrier.

An isolated source copy restoring only the old policy gate reproduced the
failure with this exact warmed-standby test. After TCP to WSS, WSSMUX remained
authenticated and had passing probes, but the authority stayed NO_ROUTE with
eligibility five minutes later. It failed the 45-second recovery bound. The
current policy passed both transitions; local measurements were about 8 and
10 seconds. These local timings are not WAN timings.

Emergency recovery is permitted only from the existing debounced no-route
branch. It uses a five-second minimum between cross-carrier commits and an
absolute twelve-commit rolling-hour ceiling. The original durable timestamp
history is shared by all commits: a restart cannot replenish the allowance,
and an emergency commit also extends the full optional cooldown. Optional
quality changes retain configured Cooldown, MinimumDwell and
MaxSwitchesPerHour. Same-carrier reconnection consumes no cross-carrier slot.

There is no new enrollment/state schema or wire protocol. Observe-only, enrolled
and runtime locks, disabled candidates, pressure SKIPPED results, quarantine,
three-sample integrity checks, exclusive authority, durable generation fencing
and uncertain-commit handling remain enforced. A genuinely dead route can
recover sooner; a reachable route with a small score advantage cannot use the
emergency gate. Twelve outages within an hour can still exhaust the emergency
ceiling and leave NO_ROUTE; the policy is bounded rather than unlimited.

Both supplied WAN pairs passed TCP to WSS to WSSMUX using the default policy,
pretested standbys and separate peer/port-specific fault rules. Measured from
fault insertion until observation of an agreed commit:

| Pair | TCP to WSS | WSS to WSSMUX |
| --- | ---: | ---: |
| Kharej1 | 18.922 s | 18.907 s |
| Kharej2 | 17.656 s | 17.625 s |

The Iran application listener inode stayed unchanged. Actual TCP and UDP
traffic passed after each transition. These are single bounded observations,
not an SLA or a claim that existing TCP sessions migrated without interruption.

## Diagnostics

The old control-authentication failure branch gave the same rejection label
for read timeouts and actual identity/protocol failures. Typed internal
rejection/protocol errors now keep these separate. Public status contains
fixed summaries and categories rather than arbitrary frame/helper error text;
tests verify that an unknown error containing a secret does not export it.
Configured role-specific debug level and JSON format are now honored by Auto.

Timeout does not establish filtering. EPERM/EACCES can arise from local firewall
policy as well as missing privilege, so their category is `local_denied`, not a
claim that root/CAP_NET_RAW is absent. Packet traces and prerequisites are still
needed to establish a cause. The WAN binary predates this final cosmetic
category/summary correction and subsequent documentation/test edits; policy
and shutdown behavior are the tested code.

## Independent QUIC and Wat checks

The initial Kharej1 UDP443 QUIC retry in the old report was invalid: two pairs
shared Iran's UDP443 listener concurrently and its local bind failed. The
historical raw record is retained and the report corrected.

The new Kharej1 retry reserved Iran UDP443 exclusively and used an actual Auto
QUIC listener. It still timed out before the QUIC handshake. A bounded header
capture saw 1232-byte Initial transmissions on Kharej1 and no delivered packet
on Iran during that capture window. A separate manual QUIC tunnel without Auto
also failed, with a confirmed bound listener and a live local echo backend.

After releasing UDP443, an independent authenticated UDP echo service on that
same Iran port returned 3/3 datagrams at 96, 512, 1199, 1200, 1232, 1280 and
1372 bytes from Kharej1. The client set IPv4 DF, avoiding a fragmentation
explanation for the 1232-byte contrast. Protocol/content-specific route
filtering is consistent with these observations; its location and mechanism
are not established. No ALPN, exporter, token, encryption or verification was
weakened to make QUIC pass. QUIC's actual loopback bootstrap, probe and handover
tests still pass; WAN availability remains NOT VERIFIED on both routes.

Wat on Kharej2 was retried with normal Echo direction, Reverse initiation and
a 256-byte payload cap. It did not qualify. The earlier reverse-Echo failures
and 6/20 ordinary ICMP baseline remain relevant, but neither proves an engine
defect or comparative superiority. Normal/reverse Wat on arbitrary filtered
paths remains an open gate.

## L3 and shutdown

With pretested UDP and xDi candidates on Kharej1, dropping only the experimental
UDP carrier port caused an agreed automatic xDi activation. The final clean
rerun reached xDi in about 19 seconds. Real 9000-byte bidirectional IP pings
and enrolled TCP/UDP forwards passed after recovery, and both TUN interface
indices remained unchanged. PCK/SNI on Kharej2 passed actual L3 traffic and
manual agreed handovers again, separately from automatic-failure recovery.

The original active-stop helper waited one second before KILL. Its five-second
setting applied to the separate lifetime timeout, not that active-stop path.
Later group TERM also made GNU timeout forward TERM again, potentially signaling
cleanup child commands. These harness mistakes are explicitly distinguished
from a direct engine SIGTERM. The final supervisor signals the owned engine
itself, waits for all live members of its process group for up to thirty seconds,
and reports forced KILL as failure.

Two command-level tests reproduced ICMP/MSS operations failing while another
firewall owner held the xtables lock. Their commands now wait up to five seconds
for the lock and have a ten-second command timeout, reusing PCK's bounded runner
for ICMP. Auto cancellation callbacks are joined before their owning workers
report completion; repeated/disarmed cleanup is tested. No hostwide rule,
sysctl or guard-ownership bypass was introduced.

Raw/L3 tests are checked against clean inventories. Direct TERM, process-group
join, remaining rule bodies and TUN removal are evidence separate from traffic
passing. An externally killed process or a firewall command that cannot succeed
can still leave resources; those cases must never be declared clean automatically.

The final clean-baseline L3/xDi rerun used direct engine TERM and joined both
process groups without KILL in approximately 0.19 s on Iran and 0.13 s on
Kharej1. The later inventory contained no audit rules on any of the three
servers. PCK/SNI direct-TERM teardown was also checked separately. Final
`*-review-final-cleanup.json` records removal of audit processes, helpers,
certificates, directories, listeners and TUNs, with original firewall rule
bodies, routes and service listeners preserved.

## Capacity and remaining acceptance

`FULLPACK_AUTO_LOAD=1` enables the bounded application-connection test, also
included explicitly in CI. It configures the existing 192 ceiling and holds
100, then 192 simultaneous application TCP sockets through an actual enrolled
loopback pair. Each socket carries 32 exchanges of 4096 integrity-checked
bytes. The 193rd connection is refused, and pressure-skipped probes retain the
working carrier/generation. The first measured process heap was approximately
49 MB at 100 sockets and 85 MB at 192, including both peers, the echo backend
and the race-enabled test process. These are not per-VPS memory estimates.

The test is a short bounded socket/admission check, not sustained traffic,
independent authenticated users, throughput capacity, WAN load or a 1000-user
target. Limits were not raised to claim that target. **24-hour soak: NOT VERIFIED**.
Published release/security gates, current third-party helper security, arbitrary
NAT and packet-source combinations also remain pending.

Evidence is retained privately at
`E:/P_OX/fullpack-waticmp-audit/wan-20261013` and the adjacent
`auto-review-*.log` files. The WAN artifact SHA256 is
`aa5786b2467a28b400d0178d74e54051e845ca47bb35dd6250de7842e49e3489`.
Server credentials, addresses, private keys and raw audit artifacts are excluded
from publication. This review supplements `fullpack-wan-verification.md` and
does not overwrite its original measurements.

## Source verification

With Go 1.27.2, affected config/CLI/peerauto/L3/network/manage/web/node race
suites passed. That broad run preceded the final cancellation/lock-wait edits;
their exact source then passed network, MSS, peerauto and L3 race suites plus
real QUIC/WSS/WSSMUX bootstrap/probe/handover and the consecutive-failure e2e
regression. The final diagnostic category wording passed the peerauto race
suite separately. All 46 panel and nine customization tests passed.
Final whole-project vet/staticcheck and standalone Linux amd64/arm64 builds
passed. Formatting passed in an isolated copy normalizing only the unchanged
Windows CRLF root files, as in earlier audits. The exact final warmed-standby
regression passed separately after removing an unused test context found by
staticcheck. Existing whole-project coverage/security results are historical evidence; their full
gates were not silently rerun or declared passed for this revision.
