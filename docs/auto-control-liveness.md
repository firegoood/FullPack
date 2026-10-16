# Auto control liveness and the NL EOF incident (2026-10-16)

<div dir="rtl">

## خلاصهٔ فارسی

پس از چند ساعت ترافیک واقعی، NL دچار قطع اتصال و EOF شد. لاگ‌ها تعویض‌های
مکرر حامل و بازه‌های بدون مسیر را نشان می‌دهند. اسکن کند مسیرهای ذخیره می‌توانست
بررسی زنده‌بودن حامل فعال را معطل کند و مهلت کنترل را تمام کند. بررسی کنترل از
اسکن جدا شده است؛ مالک تغییر مسیر، احراز هویت و تاریخچهٔ نسل‌ها حفظ می‌شوند.
نتیجهٔ تست کوتاه، اثبات پایداری ۲۴ساعته یا رفع تمام علت‌های احتمالی EOF نیست.

</div>

## Captured before any restart

The installed release was signed **v1.9.4-beta.2**. Both NL processes had run
since 2026-10-15 21:32:42 UTC with **zero systemd restarts**. The first successful
TCP Direct commit was at 21:34:55 UTC. Recovery commits started at 04:40:27 UTC
on 2026-10-16; by the 12:33:37 UTC snapshot both peers agreed on `tcpmux-d`,
generation **63**. These are generation/connection counts, not user counts.

Iran's cumulative NL counters included 29,654 interrupted flows, 244,250 rejected
TCP admissions, 619,451 `no_route` data errors and 257,187 UDP queue drops.
These overlapping counters do not count distinct people or application EOFs.
The most recent recovery interval produced repeated `no_route` warnings between
12:19 and 12:33 UTC. The 12:33 snapshot showed traffic again; that does not make
the preceding outage a successful test.

NL FullPack RSS was about 73 MiB with 209 file descriptors; Iran's NL process
was about 66 MiB with 117 descriptors. Their NOFILE limit was 1,048,576. Xray
PID 34168 and FireNode PID 34097 remained unchanged. Historical kernel OOM
records belonged to the explicitly isolated synthetic lab on the previous day,
not these production processes. No current memory/descriptor ceiling was
established as the cause. Raw private receipts retain journals and snapshots;
operator accounts and secrets are not published.

## Defect and repair

The authority previously polled all control streams inside the same serial
leader loop that performs reserve and active probes. Each isolated measurement
has bounded setup/exchange/drain/abort work. Several slow measurements in one
iteration can together exceed the follower's **90-second control idle deadline**.
An otherwise usable production carrier can then be closed and fenced because
its heartbeat was delayed by a different reserve. Repeated local cancellations
can consume the existing emergency switch budget and prolong `NO_ROUTE`.

The repair runs liveness in a separately joined worker. Per-candidate locks
still serialize authenticated frames. A busy transaction is skipped without
queuing; heartbeats on other candidates progress independently. A heartbeat
does not select a carrier, prepare or commit generations, modify enrollment,
or spend/replenish the durable recovery budget. Actual control failures and
authenticated live-route disagreements still cancel and fence the carrier;
typed `CONTROL_LOST` events make later incidents easier to distinguish.
Heartbeat batches and the bilateral prepare/commit boundary are mutually
exclusive: the follower can install a new route just before its ACK reaches
the authority. Checking that transient difference would prematurely cancel
the previous carrier instead of preserving its bounded drain. Busy commits
skip heartbeat ticks; they do not queue more liveness workers.

Concurrent heartbeats must distinguish a prepared fencing epoch from the
previous live route. `STATUS` reports the runtime committed route and its
generation; `STATE_SYNC`/`RECOVER` retain the durable high-water generation.
A fenced route is never advertised as live. Authentication, frame sequencing,
manifest binding, probe budgets and the one-hour replay cache are unchanged.

The scheduling defect matches the observed long-running recovery churn, but the
old binary did not record every control closure cause. It cannot establish that
every historical EOF or unavailable reserve had this single cause. Existing
WS Direct probe timeouts, missing HTTPS prerequisites and WAN filtering must
remain separate observations.

## Verification

Linux Go 1.27.2 control regressions passed in an isolated, unprivileged
network namespace. The server has no C compiler, so race validation is
explicitly delegated to the full CI gate. All peerauto package tests also passed in that isolated Linux run. Release CI,
signed deployment and post-repair application delivery remain pending. The regression uses a compressed follower
idle deadline while holding the authority/reserve transaction busy; it is not a
real-hour soak. It also checks generation disagreements, prepared versus live
epochs, and joined cancellation. No new application connection ceiling is added.

**NOT VERIFIED:** continuous real-hour/24-hour operation, sustained bandwidth,
every carrier under load and guaranteed production capacity for 1000 real users.

*Last verified against FullPack v1.9.4-beta.3 (source compatibility review, 2026-10-16; pending runtime gates are explicit above).*
