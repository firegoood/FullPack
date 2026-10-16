# Controlled Auto admission deployment (2026-10-15)

<div dir="rtl">

## خلاصهٔ فارسی

نسخهٔ امضاشدهٔ آزمایشی روی هر سه سرور نصب شد؛ انتقال واقعی از چهار پورت
موفق بود و تنظیمات و تاریخچهٔ نسل‌ها حفظ شدند. تست‌های ۷۰۰، ۸۰۰ و ۱۰۰۰
حساب با حداکثر ۱۶ اتصال هم‌زمان موفق بودند. در تست ۱۰۰۰ اتصال باز، سقف
۶۴ اتصال همچنان درخواست‌های اضافی را رد کرد. این محدودیتِ Auto است؛
اصلاح صف و سهم UDP، ظرفیت ۱۰۰۰ کاربر هم‌زمان را فراهم نکرده است.

</div>

## Current gate

**PASSED within the recorded scope:** v1.9.4-beta.1 full CI/signature,
installation on all three hosts, four forwarded-port data checks, and bounded
application/WAN transactions on the active TCP and Stealth routes. The
[deployment/WAN report](verification/auto-admission-wan-20261015.json) retains
both before-upgrade comparisons and post-upgrade results, including rejected
clients. **NOT VERIFIED:** 700–1000 simultaneous users, sustained mixed-application
bandwidth, 24-hour operation and WAN L3 failover for this deployment. The held
1000-attempt test did not serve all clients. Earlier results remain historical
in `auto-admission-repair.md` and `verification/auto-admission-20261015.json`;
their pre-deployment status is not rewritten.

## Deployment invariants

- Keep each existing enrolled ceiling (currently 64), candidate endpoints,
  secret, tunnel identity, backend mapping and helper enrollment unchanged.
- Preserve current fencing through upgrade and binary rollback. Never restore
  an old generation snapshot over newer state or delete it to force activation.
- Retain a root-only previous binary and record hashes before replacement.
  Verify the fork's tag-bound release signature without changing trust pins.
- Restart only FullPack pair members and active FullPack UI/monitor services;
  keep operator FireNode/Xray services and unrelated listeners intact.
- Use bounded CPU/memory/time/connection limits for tests; synthetic credentials
  never enter Git, reports or operator persistence. Record failed/limited tests
  and clean up precisely scoped fixtures and any temporary test accounts.
- Report actual admitted connections, queue waits/rejections and continued data
  integrity. A bounded stress result cannot prove unlimited or 24-hour uptime.

## Extended isolated identity test (before deployment)

The source at `59f4add099823eace5c2c548150117f813c97649` completed 30 phases
with 700, 800 and 1000 fresh Xray identities on 2026-10-15, 14:50–14:54 UTC.
The [raw report](verification/auto-admission-extended-20261015.json) records
all short transactions verified, including 2000/2000 at 1000 identities with
two connections each and bounded attempted concurrency 128/384 at enrolled
limits 64/192. External pacing used 32 outstanding connections. Deliberately
held bursts still admitted only 64/192 and rejected overflow; those rejected
clients were not served. There was no accepted-stream corruption, leaked
admission/queue or generation change. UDP's 16-slot share did not block TCP.

The private Linux network namespace used one CPU, 768 MiB memory, no swap,
128 tasks and a 900-second watchdog with patched Go 1.27.2. Runtime was
230.093 seconds and CPU time 94.993 seconds. Sampled per-process RSS is in
the report; the systemd high-water output was inconsistent with these samples
and is not used as a memory-peak claim. Production services were not deployed
or restarted by this isolated test. The separate installation/WAN evidence is
recorded below.

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

## Release gate interruption

Run `38758569067` passed ten verification jobs, including normal, coverage,
native recovery and managed HTTPS, but its race job failed the L3 handover
bootstrap. Every probe was skipped by the existing process pressure guard;
the old failure output did not distinguish RSS from CPU. Race instrumentation
and preceding stress tests share that process. The L3 fixture now explicitly
budgets 2048 MiB and reports measured RSS/CPU on future bootstrap failure.
The production default remains 256 MiB; CPU/admission guards and all CI gates
remain enabled. No release or server deployment is accepted by that failed run.

## Installed v1.9.3 comparison before upgrade

Both actual stream pairs were checked with fresh runtime-only Shadowsocks
credentials and two exact owned HTTP 404 responses per accepted connection.
Each pair verified 700/700, 800/800 and 1000/1000 short connections, followed
by 2880/2880 transactions over 180 seconds. Attempted concurrency was 16 and
starts were paced at 16 per second; each cycle intentionally held 250 ms.
These are small HTTP transactions through encryption, not a bandwidth or
700–1000-simultaneous-user result. Direct-Xray prerequisite checks sampled
three credentials; they were not a full direct-Xray WAN capacity benchmark.

In a separate 1000-attempt held burst, TR admitted 64 and rejected 936. NL
admitted 60 synthetic connections while the sampled total reached 64 with
other live traffic; 940 synthetic attempts failed. No accepted synthetic
session lost its second verified response. Post-load direct and tunneled NL
checks passed. Both pairs retained their active route/generation during these
comparisons. Each set of 1000 temporary users was removed using scoped API
cleanup; operator accounts, configuration and FireNode/Xray were preserved.

An initial smoke-test launcher failed on Iran before executing the client
because an optional `/opt/firenode` isolation path did not exist. Its path was
marked optional and all four forwarded ports subsequently passed. That setup
failure is not recorded as a carrier/data-path failure. These comparisons
precede the repaired-binary deployment and do not pass its pending gates.

## Signed version and controlled installation

The signed prerelease [v1.9.4-beta.1](https://github.com/firegoood/FullPack/releases/tag/v1.9.4-beta.1)
is built from `4e0bc91fa5b332662da9258ac2d982ad63300629`.
[Run 38761783514](https://github.com/firegoood/FullPack/actions/runs/38761783514)
passed all eleven verification jobs and the public exact-tag install check.
An independent verification checked the fork signature, all seven archive
hashes, exact binary/legal members without `11/`, and every ELF's Go 1.27.2
build record. The prerelease is separate from stable `releases/latest`.

The exact version was installed on all three enrolled hosts with root-only
previous binaries and unchanged TOML hashes. Installer-side signature checks
also passed on each host. Active FullPack UI/monitor processes were refreshed;
FireNode/Xray were retained. Signed offline metadata was kept with the installed
bundle. When deliberately installing this prerelease, select its complete tag:

```bash
BP_RELEASE_TAG=v1.9.4-beta.1 bash <(curl -fsSL https://raw.githubusercontent.com/firegoood/FullPack/v1.9.4-beta.1/install.sh)
```

TR resumed on TCP at generation 4, approximately one minute after its restart.
NL exceeded the install observer's 180-second deadline while the complete pool
was still collecting three separated full data proofs. No fencing/config was
reset and the process was not restarted again. It committed Stealth at
generation 5 at 16:11:02 UTC, about 215 seconds after the observer began.
The initial timeout is retained as a failed timing check, followed by a
separate successful live observation. This is cold-start selection latency,
not evidence of five-second cold-start or zero-downtime upgrade acceptance.
A future improvement could prioritize the previously selected enrolled paths
at startup while retaining all three proofs, subset agreement and reserve scans.

## Installed application/WAN results

After installation, the TR TCP route and NL Stealth route each verified
700/700, 800/800 and 1000/1000 short encrypted Shadowsocks connections, then
2880/2880 over 180 seconds. These phases used at most 16 outstanding connections,
16 starts per second and an intentional 250 ms hold. Each accepted connection
verified the exact owned HTTP response twice. No synthetic application errors
or accepted-session losses occurred in these phases. Before-upgrade small
transaction phases also passed; this workload alone does not establish an
admission improvement, a throughput improvement or 1000-user capacity.

| Separate held burst | Attempts | Synthetic connections served | Rejected | Accepted sessions lost |
| --- | ---: | ---: | ---: | ---: |
| TR / TCP | 1000 | 64 | 936 (808 EOF, 128 reset) | 0 |
| NL / Stealth | 1000 | 61 | 939 (812 EOF, 127 reset) | 0 |

NL also carried other live traffic; its sampled total reached 64. The queue
was bounded at 128. Typed local `queue_full` and `queue_timeout` counters matched
these synthetic rejections; peer admission reported no failures. Capacity
warnings appeared separately from carrier health. No carrier/generation switch
was triggered by the load: TR stayed at generation 4 and NL at generation 5.
NL's cumulative `no_route` count was 38 before and after the workload, all from
the preceding startup; it was not a new load failure. The application and
direct-backend checks after overload passed. Ports 8880 and 8889 also passed
separate application smoke checks; the longer phases used 2052 and 8888.

Final inspection verified every running FullPack executable against the release
hash, unchanged TOML hashes and the same FireNode/Xray process IDs on both
outside servers. It independently queried Xray runtime users on-host: zero
fixture-owned users remained on all four inbounds. Operator user details never
left those processes. Scoped test directories, credentials, SDK/cache fixtures
and temporary units were removed; private binary rollback copies were retained.
Final selected-route health was three for TR and four for NL, with matching
peer generations. This does not establish availability of every enrolled
protocol or four healthy paths at all times.

## Capacity difference from manual forwarding

In the manual forwarding path, `config.Config.MaxConnections` uses **0 = unlimited**
and `internal/server/transport/limits.go` creates no connection limiter for that
setting. The installed v1.9.4-beta.1 Auto admission policy instead defaulted to 64
and rejected enrollment above 192. New unlimited policy is recorded separately
in `auto-unlimited-connections.md`; the measurements here describe the older
deployment. Both pairs in this recorded deployment had 64 shared across
their forwarded ports and TCP/UDP traffic. Those are connection slots, not
users: one Xray/V2Ray user can hold several connections.

The admission repair prevents retained UDP associations from occupying every
TCP slot and gives short TCP bursts a bounded wait. It deliberately retained
that total ceiling; it did not solve the requested 700–1000-user concurrency.
Above the ceiling, client EOF/reset is a real refusal to serve that request,
even while the carrier is healthy. It must not be described as a successful
1000-user test or an inherent capacity limit of the original manual engine or
of these servers.

A higher-capacity Auto design needs actual concurrent application and resource
tests, including carrier-internal stream/window budgets (the current Wat
profile is capped at 256 streams), followed by coordinated enrollment. Merely
changing the displayed/configuration maximum cannot prove capacity. Existing
manifests and fencing were not silently rewritten during this installation.

*Last verified against FullPack v1.9.4-beta.6 (source review of bounded Auto receive/transmit isolation, 2026-10-16; historical measurements keep their scope; CI/deployment and soak require separate evidence).*
