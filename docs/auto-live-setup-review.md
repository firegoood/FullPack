# Live Auto setup review — 2026-10-14

## Initial state

The operator's Iran `TR` and Kharej2 `TR-kharej` configs matched in identities,
secret and backend maps `8880,2052`, with UDP enabled, observe-only disabled and
no manual lock. They contained exactly one candidate: Wat ICMP Normal. Both
units were running; Auto reported NO_ROUTE/RECOVERING because native handshake
timed out. Other protocols could not be probed: none had been enrolled.

The ordinary `fullpack check` address layer also incorrectly rejected port zero
in their native/Auto role markers. The engine had already accepted the actual
peer configuration. Static checks now allow only the appropriate opted-in
markers and optional Auto L3 maps; legacy port-zero rejection, malformed peer
addresses, user-port validation and full engine enrollment checks remain.
All seven Linux spec tests, including thirteen new marker cases, passed as an
unprivileged user with Go 1.27.2.

## Authorized mixed re-enrollment

The operator approved additional endpoints and asked for Reverse plus Direct.
At the time, each pair was limited to four candidates, so a fresh `TR-auto` /
`TR-auto-kharej` pair enrolled:

| Candidate | Direction | Listener | Outcome in this bounded window |
| --- | --- | --- | --- |
| tcp-rev | Reverse | Iran TCP24443 | authenticated, 32 KiB integrity PASS, active |
| wss-rev | Reverse | Iran TCP24444 | authenticated, 32 KiB integrity PASS, standby |
| noise-dir | Direct | Kharej2 TCP24445 | initial authentication, control/probe failure |
| retained Wat | Normal Echo, Reverse initiation | enrolled ICMP peers | handshake timeout |

Both peers agreed on TCP at generation 2 and reported HEALTHY. TCP and WSS
manual probes passed with approximately 69/64 ms RTT in the sampled exchange.
Their paced probe rates are not application throughput or speed-test results.
The original config/unit/state were backed up privately; original configs were
retired outside the active TOML glob and their fencing state retained in place.
Original units were disabled/stopped. No generation history was erased to make
the new manifest fit. User ports stayed 2052/8880 on Iran; firenode/Xray on
Kharej2 stayed active with the original backend listeners.

Direct Noise returned `probe offer: EOF` and `probe start: timeout` in two
bounded attempts. Metadata-only capture observed short connections and peer
FIN/RST. A peer warning reported invalid probe start; these observations do not
identify the responsible network device or establish filtering as the cause.
Direct transfer remains NOT VERIFIED in this window. Wat Normal also remains
unavailable; a ten-second ICMP baseline measured Iran-to-Kharej2 0/40 and the
opposite direction 6/40. The exact `-c`/deadline combination sent forty probes;
these are the observed counts, not an eight-packet claim.

The existing Xray backends did not answer a generic unauthenticated HTTP request
either locally or through the forwarded ports. Auto admitted/completed the
bounded forwarding attempts, but successful VPN/application exchange was not
established. The separate carrier integrity probes are the passing data gate.
No production fault injection, heavy load, 24-hour soak or user-capacity test
was performed here. Historical WAN results remain separately bounded.

## Subsequent quick setup request

A later read at 19:41 UTC found Iran NO_ROUTE at the same generation 2 while
Kharej still reported TCP HEALTHY. Kharej's journal identified the exact local
cause: `probe replay budget exhausted`. The old 256-entry one-hour cache cannot
accommodate a healthy route's ten-second data checks (up to 360 per hour), plus
full/standby probes. Earlier TCP/WSS passes were bounded samples, not continuous
acceptance. They remain recorded above, with this later failure retained.

The v1.9.2 source keeps the full one-hour replay window and a fixed 4096-entry
cap, sized for the global one-second probe spacing (3600 starts per hour) plus
bounded switch transactions. Genuine replay still fails; cache/concurrency/
pressure exhaustion is typed SKIPPED, leaves the production control usable and
does not increment carrier/data failures or quarantine. The cap also applies
to preparation transactions. This fixes the reproduced local cause without
changing enrollment, generations, endpoints or cryptographic verification.
The modelled-hour regression does not establish a real-hour/24-hour soak.
The production servers still ran v1.9.1 during this observation.

A subsequent read found both peers HEALTHY on TCP at generation 3 after old
replay entries expired, without a restart or state reset. That recovery does
not repair the old cache limit: the two installed engines remained v1.9.1 and
need the v1.9.2 update on both sides plus a tunnel restart. The corrected source's
Linux budget/control regressions and isolated carrier/recovery tests passed;
their temporary executables/directories were removed from both servers. No
production fault injection or unrelated service change occurred in this phase.

The operator then requested a simpler setup and a new signed release. Quick
setup selects built-in TCP/WSS/Noise Reverse plus Noise Direct and retains the
advanced Wat/HTTPS/L3 path; it does not rewrite this deployed pair. See
`quick-auto-setup.md`. Published release verification is recorded separately.
The added isolated Linux Noise Reverse/Direct tests passed actual 32 KiB TCP,
8000-byte UDP, independent probes and handover with a held user stream. They
do not supersede the unavailable WAN Direct observation above.

<div dir="rtl">

## خلاصهٔ فارسی

کانفیگ اول درست جفت شده بود ولی فقط Wat Normal داشت؛ به همین دلیل حامل
دیگری آزموده نمی‌شد. پس از تأیید کاربر، جفت جدید با چهار حامل و با حفظ
وضعیت امنیتی قبلی ساخته شد. هر دو سمت روی TCP سالم شدند و آزمون مستقل
WSS هم گذشت. Direct Noise و Wat در این بازه تأیید نشدند و نتیجهٔ آزمون
حامل با عملکرد واقعی کلاینت VPN یا پذیرش Production یکسان نیست. بررسی بعدی
پر شدن حافظهٔ محدود Probe و قطع منطقی مسیر ایران را نشان داد؛ اصلاح این علت
در سورس ۱.۹.۲ و تست اختصاصی ثبت شده است، نه به‌عنوان پایداری ۲۴ساعته.

</div>

*Last verified against FullPack v1.9.2 (setup/documentation compatibility review, 2026-10-14; original measurements and pending acceptance gates unchanged).*

*Last verified against FullPack v1.9.3 (source/documentation compatibility review, 2026-10-15; original measurements, earlier release results and pending acceptance gates retain their recorded dates and scope).*

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

*Last verified against FullPack v1.9.4-beta.4 (compatibility review of the control-liveness repair, 2026-10-16; historical evidence keeps its original scope and runtime/soak gates require separate receipts).*
