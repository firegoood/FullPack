# Local listener checks and Quick Auto port suggestions

<div dir="rtl">

## خلاصهٔ فارسی

در ایران، پورت‌های ۲۴۴۵۴ و ۲۴۴۵۷ تونل NL با رزرو PCK و SNI تونل TR
تداخل داشتند. یک پورت رزروشدهٔ Raw TCP الزاماً در فهرست listenerهای عادی
دیده نمی‌شود. هنگام ساخت، CLI و وب همهٔ پورت‌های محلی Auto و helperهای خصوصی
را با سوکت واقعی و تنظیمات تونل‌های موجود بررسی می‌کنند. Quick Auto یک بازهٔ
آزاد پیشنهاد می‌دهد و انتخاب دستی را حفظ می‌کند. آزادبودن پورت خارج، در همان
سرور هنگام واردکردن لینک بررسی می‌شود. تغییر جفت موجود باید هماهنگ باشد و
تنظیمات و تاریخچهٔ قبلی نگه داشته شوند.
در بررسی کامل‌تر، سه رزرو helper دیگر هم شناسایی شد و مجموعاً پنج پورت
NL به ۲۴۶۵۴، ۲۴۶۵۷، ۲۴۶۵۸، ۲۴۶۵۹ و ۲۴۶۶۰ منتقل شدند. روی نسخهٔ امضاشدهٔ
بتا ۱۲، آزمون اتصال ماندگار و اتصال‌های تازه هرکدام ۳۰ دقیقه انتقال واقعی
را بدون خطا گذراندند؛ این نتیجه، تأیید پایداری ۲۴ساعته یا هزار کاربر واقعی نیست.

</div>

The shared creation preflight runs before saving a new configuration/service or
CLI host tuning. It covers local Reverse/Direct Auto carrier listeners, TCP/UDP
forwarded sockets, private helper targets and bound-only raw TCP reservations.
It also respects stopped tunnel configurations. Foreign backend ports, the
other peer's listener, ICMP identifiers and portless raw profiles are distinct.
Errors identify the local address, protocol and candidate/configured owner;
the check never stops another service or deletes firewall rules.

Quick CLI and web setup search at most 128 local blocks, including the complete
selected traffic plane's listener and private-helper ports. The exact suggested
endpoints remain visible for enrollment review. Manual edits invalidate pending
web suggestions; an explicit button requests a fresh suggestion. The receiver
rechecks its Direct listeners on trusted-link import and refuses conflicts;
Iran cannot prove remote availability and never scans arbitrary remote IPs or
silently edits a signed/enrolled peer link.

A suggestion does not hold the ports. Creation checks again, and another process
can still bind during the interval before actual service startup. Runtime start
status remains separate from successful configuration creation. Existing Auto
manifests/fencing are never rewritten by a suggested default. The operator's
authorized port replacement uses a fresh coordinated enrollment and preserves
the retiring pair's private config/fencing for forensics, not active-state reuse.

Passing isolated Linux Go 1.27.2 regression tests cover occupied TCP/UDP, bound TCP without listen, stopped
configured reservations, helper collisions, role-specific checks, bounded search
and creation refusal before a config write. All 53 panel tests, including stale
web responses, and the ten customization-tool tests passed locally. Beta.8's
release run `39001411364` rejected a VERSION/app version mismatch before cross
builds; it was not released or installed. Beta.9 synchronizes both version files
and retains the complete release gate. At this earlier review, production port
migration and signed-release gates were **PENDING**; the completed beta.12
deployment is recorded below. This feature is not an
acceptance of the previous TCP WAN failures or a 24-hour stability claim.

The earlier diagnostic-only source `46b8c53` failed race run `39000287122`:
the pre-existing Direct spread fixture dialed its first forwarded port after
session authentication but before that independently started listener bound.
The fixture now waits at most three seconds, only for that first connection's
ECONNREFUSED. It retains exactly eight accepted flows, payload integrity and
session/backend spread assertions; accepted-flow failures are never retried.
The failed run is preserved and is not a passing release gate.
The repaired fixture passed 25 isolated Linux repeats; its full race/CI result
remains separate from that local result. This is a test startup repair; it
does not change the production Direct engine.

Beta.9 run `39001491738` failed normal/coverage documentation-stamp checks and
the panel API contract's missing `suggestAutoPorts` sample. It was not released
or installed. The replacement records both stream/L3 requests in that contract
and adds scoped compatibility-review stamps while retaining historical dates,
failed WAN evidence and pending acceptance gates. All original tests remain
mandatory; beta.10 is a new immutable candidate.

Beta.10 run `39003292518` passed ten verification jobs, but race failed the
slow-reader fixture's load precondition (5,170,176 bytes, below its unchanged
5 MiB requirement). The fast application/control assertions had not run for
that case. It was not released or installed. The fixture now requires delivery
of every slow-reader marker to the real backend; a successful local TCP write
does not substitute for admission. All 64 unread connections, the original
threshold, child instrumentation and cleanup, and fast/control assertions remain.
Beta.11 is a fresh candidate requiring the complete release gate.
The repaired fixture passed all six carrier/direction cases in 122.88 seconds
in an isolated Linux Go 1.27.2 process, with original application/control and
joined-cleanup assertions. That local run was not race-instrumented; the full
race and signed-release gates were separate and pending at that local-review
stage; subsequent beta.12 results are recorded below.

*Last verified against FullPack v1.9.4-beta.12 (source/documentation compatibility review of local listener preflight, status attribution and explicit backend readiness in the slow-reader fixture, 2026-10-17; original measurements and failed or pending WAN/release/deployment/soak gates retain their separately recorded dates and scope).*

## Signed beta.12 deployment (2026-10-17)

All eleven verification jobs and the release job in run `39025041926` passed,
including the whole race suite, all nine held-load workloads and five repeated
ownership/cancellation regressions. The seven archive hashes, tag-bound fork
signature, exact legal members, absence of `11/` and Go 1.27.2 build records
were independently verified. Signed beta.12 was installed on Iran and NL;
configurations and running engine PIDs were preserved during binary installation.

The actual fresh pair is `NL-clearports` / `NL-clearports-kharej`. Only these
carrier endpoints changed; its 38 approved endpoints, 8888/8889 forwards and
explicit unlimited policy remain intact:

| Carrier | Retiring Iran TCP port | New Iran TCP port |
| --- | --- | --- |
| WSS Reverse | 24454 | 24654 |
| TCPMux Reverse | 24457 | 24657 |
| WS Reverse | 24458 | 24658 |
| WSMux Reverse | 24459 | 24659 |
| WSSMux Reverse | 24460 | 24660 |

The first two conflicts were bound TR PCK/SNI reservations. The last three
were configured TR HTTPS-helper reservations, important even before a helper
starts. The five replacement sockets were free and all configured local claims
were compared on both peers before activation. All five actual listening
sockets were later verified as owned by the new signed engine. Each changed
carrier passed three full official bidirectional data probes, separated by at
least ten seconds. TR's config/PID and operator backends were preserved.

The retiring config, last fencing state and unit were retained privately; the
new pair uses fresh authenticated identities, not reset old-generation state.
The earlier two-port-only pending plan was never activated. The coordinated
transaction took about 197 seconds, including cold pool measurements and failed
Direct probes. This is not an instantaneous or interruption-free migration.
Both peers then agreed TCP Reverse generation 2 and four measured selected
paths. The previous TCP WAN failure remains a separate failed observation.

Both actual post-migration owned application checks on 8888 completed thirty
minutes, exceeding the earlier approximately 23-minute failure: one persistent
connection completed 900 verified exchanges, and 1,800 sequential fresh
connections completed 3,600 verified exchanges. Both final application results
were successful with matching response hashes. Independent peer telemetry
observed HEALTHY TCP Reverse generation 2 throughout; these checks did not
exercise carrier failover or 1,800 simultaneous connections. The 8889 smoke
check also passed. A 24-hour/1000-real-user acceptance remains **PENDING**.

Final telemetry also limits that result: the overlapping observation windows
recorded Iran UDP queue-drop increments of 18,805 and 18,661. They must not be
summed or converted into an end-to-end loss ratio. No flow interruption,
rejection or typed data-error counter increase was recorded in those windows.
The final pool contained only the active TCP path; reserves were not currently
confirmed stable because full probes were skipped under local resource
pressure. Readiness remained intact and automatic refill remained requested.
Pressure skips are neither failed carriers nor renewed data measurements;
four stable reserve paths are **NOT VERIFIED** at this final observation.
The initial four-path selection above is historical, not a final guarantee.

All temporary application identities were removed, independent Xray API queries
confirmed zero, and the bounded audit units and private credential inputs were
cleaned. Post-window smoke traffic passed and the final socket/history audit
again verified the signed engine, all five listener owners, retired fencing,
TR and operator backend preservation. Detailed receipts are in
`verification/nl-listener-ports-deployment-20261017.json`.

An earlier management-attached attempt lost its SSH management connection and
did not return a complete application result. It remains **INCONCLUSIVE**;
neither a carrier pass nor an application failure is inferred from the missing
receipt. Both temporary identities expired through their bounded cleanup leases,
and independent Xray API queries confirmed zero remaining audit identities.
The replacement tests retain private output files on the actual servers,
independently of management SSH, with bounded resource limits and joined
cleanup. No production engine or carrier is restarted for these measurements.
