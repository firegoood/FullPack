# Auto without a compulsory connection ceiling (2026-10-15)

<div dir="rtl">

## خلاصهٔ فارسی

کاربر حذف سقف اجباری اتصال را خواست. در تنظیمات جدید Auto مقدار صریح
`connection_limit = 0` یعنی بدون سقف تعداد اتصال؛ مقدار مثبت، محدودیت اختیاری
مدیر است و سقف ۱۹۲ ندارد. شمارش اتصال از صف موقت احراز هویت جدا می‌شود.
تنظیمات قدیمی تا ارتقای هماهنگ دو طرف حفظ می‌شوند؛ نبود سقف نرم‌افزاری
به معنی حافظه، سرعت یا ظرفیت نامحدود سرور نیست. نتایج آزمون و نصب باید
جداگانه ثبت می‌شوند. آزمون محلی هزار اتصال و بار ترکیبی پاس شده‌اند؛
نسخهٔ امضاشده روی سه سرور نصب و هزار اتصال هم‌زمان در هر جفت روی WAN
تأیید شد. این آزمون کوتاه، تضمین هزار کاربر واقعی یا پایداری ۲۴ساعته نیست.

</div>

## Policy and compatibility

- New Quick/advanced CLI and web setup explicitly enroll `connection_limit = 0`:
  no application connection ceiling. Positive values are optional operator caps
  without a 192 maximum. Active TCP/UDP accounting grows with actual flows,
  without preallocating storage proportional to a configuration value.
- The optional JSON `connectionLimit` field is absent from legacy canonical
  manifests. Legacy `max_connections = 0` or omission keeps the old default 64;
  explicit legacy caps retain their meaning and manifest hashes. A new policy
  changes the authenticated manifest; an old/mismatched peer must reject it.
  Existing state is never erased to accept the changed manifest.
- A separate bound applies to unfinished stream/backend handshakes. Permits
  return before forwarding application traffic, so they are not an active-flow
  ceiling. Local listeners wait before accepting additional handshakes; active
  flows are not evicted to make room. Cancellation joins waiting owners.
  New policies prefetch 128 streams on the active path and four on standby;
  a commit wakes the bounded cache without waiting for user traffic. This
  removes the previous four-new-streams-per-round-trip admission bottleneck
  while established flows return their handshake permits.
- Optional positive caps retain FIFO fairness, a two-second queue deadline and
  UDP's quarter share. Unlimited mode has no application queue/count gate or
  UDP association-count share. Per-source datagram queues, idle expiry and
  actual carrier/OS/server resource constraints still apply.
  Unfinished UDP handshakes have their own transient budget and release it
  before forwarding; they cannot occupy TCP handshake permits. Exhausted OS
  file/socket resources are typed `local_resource` and accept retries with
  backoff, rather than permanently abandoning the forwarded listener.
- Wat Auto uses an explicit stream-limit override, including unlimited zero;
  manual native defaults remain unchanged. Nonces, replay windows, tombstones,
  per-stream send/receive windows and retransmission budgets remain bounded.
  Auto retirement uses a permanent contiguous seen-identity frontier plus at
  most 4096 out-of-order identities. Closing thousands of streams does not
  fill a tombstone table or permit old stream identities to resurrect. Manual
  native tombstone behavior remains unchanged.
  QUIC uses the same inherited 65536-stream credit window as manual forwarding,
  with credits returned on close. Advertising 2^60 would let an anonymous TLS
  peer force enormous skipped-stream allocation before Auto authentication.
  This native transport constraint remains; removing Auto's admission gate
  does not claim an infinite transport or server. Ordinary multiplexed streams
  retain bounded byte buffers.
- Application diagnostics and UI distinguish unlimited zero from saturation.
  CPU/RSS probe-pressure checks remain; unlimited mode does not infer pressure
  merely because some number of application connections are open.

## Verification

Local Linux checks passed for the affected peer policy/admission, legacy hash,
Quick setup and native engine packages. Nine held-load scenarios passed: 1000
application connections on TCP/WSS/Stealth/QUIC in both Reverse and Direct, plus
1000 TCP with 256 UDP associations. Every held TCP stream completed two
integrity exchanges, both peers agreed on the generation, count rejection stayed
zero and joined shutdown released the active flows. Native Wat separately passed
1000 authenticated held engine streams over a bounded in-memory packet transport,
12000 sequential retirements and both disjoint peer namespaces. Raw ICMP was
**SKIPPED** in the unprivileged package run; these are not filtered-ICMP WAN tests.

The independent Xray account audit initially passed 700 identities with 1/2/4
held connections each (up to 2800), then its isolated 1280 MiB cgroup ran out of
memory during the 800-identity sweep. Preserve this failure. The harness now waits for **both** peers' permitted half-close drain and releases
already completed fixture buffers between matched phases; production GC settings,
relay buffers and installed operator configuration are unchanged. The corrected sweep passed 800 identities ×4 (3200 held connections) and
1000 ×2 (2000), with both peers at zero live flows after closure. Its 1000 ×4
phase still hit the same 1280 MiB cgroup limit; **4000 connections are NOT
VERIFIED**. Direct Xray controls and held setup concurrency are reported separately.
A completed `--connections-per-user=1,2` sweep passed all 15 matched/plain/Auto
phases for 700/800/1000 identities, including 2000 held Auto connections with
1000 separately authenticated Xray accounts, no failures, unchanged generation
and zero live flows at both peers after closure. Setup was bounded at 128
concurrent handshakes; all established connections were held together. The
fixture shares both tunnel engines, clients and echo in one process; measured
RSS is not an installed single-engine capacity estimate.
The default `1,2,4` diagnostic keeps the larger challenge available. See
[the local evidence](verification/auto-unlimited-local-20261016.json).

The signed **v1.9.4-beta.2** source
`42164d93523771853ec98e3af90008e1ed0723a2` passed all eleven CI verification jobs,
including normal/race/coverage, both bounded/unlimited admission gates, native
ICMP recovery, Direct/L3, managed HTTPS, wire compatibility and static/security
checks in [release run 38806904756](https://github.com/firegoood/FullPack/actions/runs/38806904756).
Every published archive, exact-tag fork signature, legal members and Go 1.27.2
build record was independently verified before installation on all three hosts.

The prior TR/NL shared policy remained 64 until controlled fresh enrollment.
`TR-unlimited` / `TR-unlimited-kharej` and `NL-unlimited` /
`NL-unlimited-kharej` now explicitly enroll zero. All 38 approved candidates and
forwarded ports remain unchanged. Original configurations, original fencing files
and private binary rollbacks are retained; no forensic fencing snapshot was
restored. `max_connections = 64` may remain as the ignored legacy field beside
`connection_limit = 0`; the new field determines actual admission. CLI status
reports `flows.unlimited = true` and `flows.limit = 0` on both peers.

Installed WAN verification passed **1000 separate synthetic accounts and 1000
held application connections per pair**, through TCP Reverse on TR and TCP Direct
on NL. Every connection completed both HTTP response/body checks through Xray;
sampled active Auto flows reached 1000, generations stayed at 2 and new admission
rejections were zero. Setup was bounded to 128 unfinished handshakes and all
successful connections remained open together for five seconds before the second
check. This is a short held-capacity check, not 1000 handshakes completing at once.
The secondary forwarded ports 8880/8889 passed separate application smoke checks.
All runtime test accounts were independently verified absent, every running
FullPack process matched the signed binary, operator backend PIDs stayed unchanged
and owned temporary directories/active units were removed. See
[the installed WAN evidence](verification/auto-unlimited-wan-20261016.json).

A separate control burst of 1000 new handshakes directly into the installed TR
Xray, bypassing Auto, verified only five and timed out on 995. Its specific cause
was not established. Both backends subsequently passed 1000 held connections with
128 concurrent starts. Preserve this distinction; removing Auto's ceiling does
not prove unlimited backend authentication rate. Existing unavailable carrier
routes/prerequisites remain distinct from the active paths tested under load.

**NOT VERIFIED:** the 4000-connection workload that exceeded its isolated memory
budget, 1000 simultaneous cold Xray handshakes, every carrier under WAN load,
sustained application bandwidth and 24-hour operation. Historical rejection and
failed fixture/control reports remain preserved. These results are not a claim
of guaranteed production capacity for 1000 real users.

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-16; bounded evidence and remaining gates are recorded above).*
