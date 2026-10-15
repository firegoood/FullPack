# Auto without a compulsory connection ceiling (2026-10-15)

<div dir="rtl">

## خلاصهٔ فارسی

کاربر حذف سقف اجباری اتصال را خواست. در تنظیمات جدید Auto مقدار صریح
`connection_limit = 0` یعنی بدون سقف تعداد اتصال؛ مقدار مثبت، محدودیت اختیاری
مدیر است و سقف ۱۹۲ ندارد. شمارش اتصال از صف موقت احراز هویت جدا می‌شود.
تنظیمات قدیمی تا ارتقای هماهنگ دو طرف حفظ می‌شوند؛ نبود سقف نرم‌افزاری
به معنی حافظه، سرعت یا ظرفیت نامحدود سرور نیست. نتایج آزمون و نصب باید
جداگانه ثبت می‌شوند. آزمون محلی هزار اتصال و بار ترکیبی پاس شده‌اند؛
این نتیجه هنوز تأیید نصب روی سرورها، ظرفیت WAN یا پایداری ۲۴ساعته نیست.

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
1000 TCP with 256 UDP associations. Every held TCP stream exchanged two distinct
integrity challenges, both peers agreed on the generation, count rejection stayed
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
An explicit `--connections-per-user=1,2` supports a completed narrower workload;
the default `1,2,4` diagnostic keeps the larger challenge available. See
[the local evidence](verification/auto-unlimited-local-20261016.json).

**PENDING / NOT VERIFIED:** full CI/race acceptance, completion of the encrypted
application sweep, signed release, installed pair migration, WAN capacity and
24-hour operation. A target count or a passing socket open is not a successful
user/application transaction. Historical 64/192 rejection reports remain preserved.

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; new unlimited behavior requires the separately recorded tests above).*
