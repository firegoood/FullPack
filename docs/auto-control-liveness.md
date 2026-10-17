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

The `v1.9.4-beta.3` candidate at `67d06f3` was **not published or installed**.
CI run `38902957256` passed the full normal/race suites, coverage, native/raw
recovery, HTTPS, Direct/L3, security, fuzz, sustained and compatibility gates,
but the final explicitly enabled held-connection audit failed. Both Stealth
directions hit `SKIPPED: local CPU/memory/application pressure` before bootstrap:
zero application flows were admitted in those two cases. They are failures,
not 1000-connection passes. The old fixture did not record the pressure subtype.

The replacement `v1.9.4-beta.4` candidate runs each of the same eight
carrier/direction cases and the mixed TCP/UDP case in a fresh race-test process,
within the original twelve-minute aggregate bound. Each retains 1000 held TCP
connections, both integrity passes, zero rejection/unchanged-generation checks,
and the mixed case's 256 UDP associations. RSS, heap and goroutine diagnostics
are recorded. This separates accumulated instrumentation/fixture resources from
each carrier's measured load, without raising the 2048 MiB test budget or the
production 256 MiB default, bypassing pressure guards or omitting any scenario.

A private isolated comparison copied the original `ed42022` control batch
without changing its scheduling. Holding a reserve transaction blocked the next
active heartbeat and expired a compressed **six-second** follower idle deadline.
The repaired loop maintained four authenticated checks over **eight seconds**
with the same reserve held busy. Both tests passed their expected outcomes;
this demonstrates the scheduling defect, not a real-hour WAN reproduction.

Before updating production, a fresh synthetic Xray identity completed **300 new
connections over five minutes** through NL, with both HTTP integrity checks per
connection passing and generation **66** unchanged. A separate **five-minute
single persistent connection** passed **144** application exchanges. Both
accounts were removed and independently queried absent. The incident did not
repeat during these short pre-repair windows; they do not invalidate the earlier
outage or establish its only cause.

Linux Go 1.27.2 control regressions and all peerauto package tests passed in an
isolated, unprivileged network namespace. The compressed deadline is not a
real-hour soak. It also checks generation disagreements, prepared versus live
epochs, and joined cancellation. No new application connection ceiling is added.

All eleven verification jobs and the signed release job passed for beta.4 in
CI run `38906350352`, source `00d633a82b44c9677f3fe106fe9cdd521219fc38`.
All nine explicitly enabled held-load cases ran, including both Stealth
directions and mixed TCP/UDP. These are loopback connections, not WAN users.
The verified signed binary was installed on Iran and NL at 13:43:46 UTC, with
operator configs and fencing retained, TR engines and FireNode/Xray unchanged.
The pair reached TCP Reverse, generation 78, at 13:46:17 UTC. Short tests passed
300 new connections and a five-minute persistent application flow; both
forwarded ports and an approved-pool rescan were checked. A test-harness cleanup
RPC failed JSON parsing after the successful 300-connection workload; its
separate cleanup was recovered and all six test-account prefixes were queried
absent. The original harness failure receipt is preserved.

**The incident recurred after those short successes.** Iran recorded active
control timeouts on TCP Reverse at 13:58:34, WSSMUX Reverse at 13:59:58, WSSMUX
Direct at 14:06:24 and TCP Reverse again at 14:24:04 UTC. Automatic commits
advanced through generations 79–83 without a process restart. Final cleanup
stopped at its health assertion before deleting diagnostic resources. Beta.4
fixed the independently reproduced scheduling defect but did not eliminate the
observed WAN timeout churn. Its five-minute passes must never be represented
as a lasting EOF fix. The additional receive-bucket mechanism, compatibility
review and bounded Auto adaptation are documented in `smux-flow-control.md`.

A later explicit delivery check captured the distinction on beta.4: the direct
NL synthetic account returned the exact expected HTTP response, but Iran port
8889 reset after five seconds with `stream_pool_timeout` while both peers
reported `HEALTHY`, `spoof-d`, generation 107. An official manual switch to the
already enrolled SNI Reverse path, generation 108, did not restore the check.
A subsequent official TCP Reverse switch restored the short port-8889 check;
the generation/configuration receipts and independent test-account removal are
retained privately. This is temporary recovery with the old installed binary,
not acceptance of the pending receive/transmit repair.
TCP Reverse generation 109 subsequently timed out at 18:46:26 UTC, about
23 minutes after manual activation, and Auto committed SNI Reverse generation
110 at 18:46:49. This additional recurrence is retained with the short passes.

Beta.5's full release gate failed and its release job was skipped; no beta.5
binary was installed. The failed native/race pool bootstraps, subsequent race
failures and the isolated-fixture change remain recorded in `smux-flow-control.md`.

The receive/transmit repair's signed beta.7 passed all eleven verification jobs
and was installed on both NL endpoints without rewriting enrollment/fencing or
restarting operator backends. The cold scan took about 136 seconds before TCP
Reverse generation 120; requests during that scan were rejected as no-route.
Its separate post-install continuity tests failed after about 23 minutes,
despite HEALTHY TCP status; Auto subsequently recovered to WS Reverse generation
121. Exact failed workloads and limits are recorded in `smux-flow-control.md`.
Beta.7 is not accepted as a lasting WAN repair. Beta.4's earlier recurrences
remain historical failures.

**NOT VERIFIED:** continuous real-hour/24-hour operation, sustained bandwidth,
every carrier under load and guaranteed production capacity for 1000 real users.

*Last verified against FullPack v1.9.4-beta.7 (source review of bounded Auto receive/transmit isolation, 2026-10-16; historical measurements keep their scope; CI/deployment and soak require separate evidence).*

*Last verified against FullPack v1.9.4-beta.11 (source/documentation compatibility review of local listener preflight, status attribution and explicit backend readiness in the slow-reader fixture, 2026-10-17; original measurements and failed or pending WAN/release/deployment/soak gates retain their separately recorded dates and scope).*
