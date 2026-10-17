# Auto admission and data-health repair (2026-10-15)

<div dir="rtl">

## خلاصهٔ فارسی

آزمون پیشین نشان داد که Auto پس از رسیدن به سقف ۶۴/۱۹۲ اتصال، درخواست‌های
اضافه را قطع می‌کند و نشست‌های UDP می‌توانند تمام ظرفیت TCP را تا ۶۰ ثانیه
نگه دارند. این اصلاح سهم UDP را محدود می‌کند و برای اتصال‌های TCP یک صف
محدود دوثانیه‌ای می‌گذارد. اشباع و خطای سرویس مقصد نیز مستقل از سلامت حامل
در CLI و پنل وب نمایش داده می‌شوند. سقف اتصال و هویت جفت‌های موجود تغییر
نمی‌کند؛ این اصلاح به‌معنی ظرفیت ۵۰۰/۸۰۰ کاربر هم‌زمان یا تأیید WAN نیست.

</div>

## Behavior and compatibility

This is the historical v1.9.4-beta.1 capped repair. The subsequent explicit
[unlimited policy](auto-unlimited-connections.md) supersedes its compulsory
64/192 restriction for new enrollment; the measurements below retain their scope.

- Keep the authenticated, enrolled total exactly as configured: default 64,
  maximum 192. Neither the default, schema, hash, generation nor durable history
  changes. Changing that total still requires fresh enrollment; increasing it
  beyond 192 requires additional implementation and capacity evidence.
- UDP uses at most one quarter of the total (16 at 64, 48 at 192). Totals 2–3
  permit one UDP association, while a total of one reserves its only slot for
  TCP. TCP can use the full total when UDP is absent. This favors TCP admission;
  UDP overflow is deliberately dropped and counted, not presented as success.
  Existing active UDP associations retain the original 60-second idle timeout.
- New TCP sockets have a FIFO with at most twice the enrolled total pending
  entries (128 or 384), and a two-second deadline. A queued socket has no live
  slot. Reserve the ticket before starting its worker; no unbounded application
  socket/goroutine queue, no unbounded payload buffering, no enrollment bypass.
  Releasing a slot promotes the oldest waiter. Cancellation, timeout and repeated
  close return resources exactly once. Revalidate the committed route after a
  peer-side capacity wait so a pending request cannot extend old authorization.
- Stream request headers, authentication domains and successful ACK=1 remain
  unchanged. Negative ACK=2 means peer admission capacity; ACK=0 still means
  backend unavailable. Older peers already reject any ACK other than 1, so a
  mixed pair retains successful framing but reports a generic backend error
  for the new negative value. Upgrade both peers for accurate typed diagnostics
  and symmetric admission fairness. No new identity/version acceptance is added.
- `auto status` keeps carrier `state` and reports `data` separately. Flow status
  includes total/UDP limits, live UDP, pending TCP and queue limit. Data counters
  distinguish queue full/timeout, UDP share, peer capacity, backend unavailable,
  no route, stream-pool timeout, handshake timeout and peer closure. The last
  error's code/stage/time is public; payloads and credentials are never logged.
  Warning logs are limited to one per ten seconds per manager.
- The panel shows **At capacity**, **Busy**, or **Data errors** with a warning
  sentence while carrier connectivity remains online. Recent data failures
  remain visible for 30 seconds; historical counters remain until engine restart.
  CLI tunnel detail/JSON and the Auto editor include the same distinction.
  Capacity does not consume carrier-failure/quarantine budgets, switch methods,
  restart production, change selection/fencing, or claim end-application health.

## Validation

Patched Go 1.27.2 builds and `go vet` passed. The 51 JavaScript panel tests,
ten customization tests, and preservation checks passed. Linux validation ran
as `nobody` in a private network namespace with no access to production config:

- The complete `peerauto` unit suite passed, including concurrent admission,
  timeout/cancellation accounting, negative peer ACK categories, legacy manifest
  hash preservation and rejection of a queued retired-generation request.
- Auto health and panel HTTPS/admission tests passed. The dashboard uses the
  enrolled Auto ceiling instead of the unused legacy server limit; Layer 3
  does not display the stream admission controls as its IP capacity.
- Real TCP/UDP integration passed: sixteen UDP associations leave forty-eight
  TCP slots, an extra UDP source cannot exceed its share, queued TCP proceeds
  after one slot releases, and active UDP still echoes during TCP pressure.
  A confirmed queued TCP socket and live UDP owners are joined on cancellation;
  both managers must finish with zero live and pending admissions.
- The opt-in 100/192-held-connection integration passed. The 193rd request
  times out in the bounded queue, reports capacity and leaves no queued ticket.
  The existing four-to-two-to-four reserve replenishment integration passed
  with the same active generation and held stream.

The repeated [raw synthetic-account measurements](verification/auto-admission-20261015.json)
completed twenty phases at 12:09:03–12:11:32 UTC. All of these are loopback tests,
not WAN load. At each capacity, every short connection sent and verified its
unique 1 KiB payload twice through real authenticated Xray Shadowsocks accounts:

| Enrolled total | Maximum concurrent outstanding short transactions | 500 identities × 2 transactions | 800 identities × 2 transactions | Observed peak waiting (500 / 800) |
| --- | ---: | ---: | ---: | ---: |
| 64 | 128 | 1000/1000 | 1600/1600 | 68 / 85 |
| 192 | 384 | 1000/1000 | 1600/1600 | 245 / 256 |

This workload uses a bounded client attempt window; it is not an unlimited
all-at-once burst. Queued admissions are measured in the engine, beyond the
separate 32-attempt externally paced comparison. There were no errors after
establishment or corrupt payloads. No phase exceeded its enrolled total or
twice-total queue, changed generation, or retained a TCP slot after closure.
Every deliberately held overflow case still admitted exactly 64/192, with
additional requests rejected after queue full/deadline as expected. Its data
status now says `capacity`, independently of carrier `HEALTHY`. Recent counters
from those cases remain visible for their documented 30-second window during
subsequent successful cases; they are cumulative, not new short-case failures.

Sixteen authenticated UDP associations filled the UDP share of a fresh limit-64
pair. A seventeenth source timed out at the share limit. New TCP passed while
the UDP share was full, immediately after closing the UDP clients, and after
UDP idle expiry. The remaining UDP slots retained their original idle behavior
(55.078 seconds measured after the five-second excess-source check); TCP no
longer waits for that expiry. Both peers remained on generation 2.

The repeated audit used one CPU core, 512 MiB memory, no swap, 128 tasks,
16,384 descriptors and a 900-second watchdog. It completed in 149.666 seconds,
used 34.993 CPU seconds and reported a cgroup peak of 433.1 MiB. Its Linux-built
Go 1.27.2 harness SHA-256 was
`3173e6cb0ea01f13011854ea3d4be013deab8b3a1f1f60573417a7d53db3e942`;
Xray's isolated fixture hash/version matches the before-fix run. Synthetic
secrets stayed separate from production, and production was not reconfigured.
All scoped test units/children exited and the private fixtures, temporary SDK
and build cache were removed. Production FullPack/FireNode PIDs and restart
counters on Iran/TR/NL matched their pre-test values.

The original [before-fix audit](auto-synthetic-load-audit.md) and
[raw measurements](verification/auto-load-20261015.json) remain historical.
After-fix short-transaction tests must be distinguished from indefinitely held
connections. A bounded queue cannot manufacture space when all live connections
remain open, and deliberate overflow remains a failure for that client.

## Remaining acceptance gates

**NOT VERIFIED:** sustained 500/800-real-user WAN capacity, bandwidth under mixed
application traffic, production deployment, complete race/CI release acceptance,
and a 24-hour soak. The two enrolled production pairs remain at their configured
limit until a coordinated, verified deployment; no manifest/fencing files are
rewritten by the diagnostic fixture. This source repair is not a signed release.

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

*Last verified against FullPack v1.9.4-beta.7 (source review of bounded Auto receive/transmit isolation, 2026-10-16; historical measurements keep their scope; CI/deployment and soak require separate evidence).*

*Last verified against FullPack v1.9.4-beta.12 (source/documentation compatibility review of local listener preflight, status attribution and explicit backend readiness in the slow-reader fixture, 2026-10-17; original measurements and failed or pending WAN/release/deployment/soak gates retain their separately recorded dates and scope).*
