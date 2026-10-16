# Peer setup port ownership review

## Observed server state on 2026-10-14

- Iran: TCP 2052/8880 belonged to PID 657,
  `/usr/local/lib/anonymous-tunnel/core`, in
  `anonymous-tunnel-ir01.service`. The enabled service had been running since
  2026-10-11 19:47:04 UTC, before this FullPack setup attempt.
- Kharej2: TCP/UDP 2052/8880 belonged to Xray PID 34682,
  `/var/lib/firenode/xray-core/xray`, in the enabled `firenode.service`.
- Both servers ran FullPack v1.9.1. `fullpack tunnel list --json` returned `[]`;
  neither had a tunnel TOML nor a FullPack tunnel unit. Only the FullPack panel
  and monitor services were present. There was no orphan FullPack user listener
  to delete. Existing port owners were preserved.
- A Reverse Iran user listener must be free locally. Kharej backend listeners
  must remain available as destinations; they are not duplicate Iran listeners.
  Adding an IP cannot reuse the Iran ports already held by a wildcard listener.

## Operator-authorized port release on 2026-10-14

After reviewing the identified owner, the operator explicitly requested stopping
`anonymous-tunnel-ir01` on Iran so FullPack could use 2052/8880. The unit's exact
path, executable and cgroup were checked again before `systemctl disable --now`
on that one unit. Its service/config files were retained; disabling autostart
prevents it reclaiming the ports after a reboot.

The unit was verified inactive/disabled with MainPID 0. Both TCP and UDP were
successfully bound simultaneously on 0.0.0.0:2052 and 0.0.0.0:8880 by a brief
verification process, then closed; `ss` showed no remaining listeners on those
ports. `firenode` on Kharej2 remained active/enabled with the same Xray PID 34682
and its TCP/UDP backend listeners intact. No new FullPack tunnel was created.

A later read-only uninstall inventory found a previously missed enabled
`anonymous-tunnel-watch-ir01.timer` and its oneshot watch service. That watchdog
had restarted the disabled main service (new core PID 182532), reclaiming both
ports. The initial port release was therefore temporary. Under the operator's
existing stop authorization, the timer was disabled/stopped first, its watch
service stopped, and the main service disabled/stopped again. All three units
were verified inactive and both TCP/UDP ports passed a fresh bind check. The
files were retained; a separate exact-path uninstall command was supplied for
the operator to execute. Future service replacement audits must inventory
associated timers before claiming ports will stay free after stopping a unit.

## Source review and repair

Explicit Auto listener hosts and both Wat IPv4 addresses are validated before
creation. Missing addresses therefore do not start a controller. The CLI peer
wizard did miss the panel's occupied-forward check at its common completion
step. It now calls `refuseBusyForwards` before optimization, config writes or
systemd creation, and returns cleanly on refusal. The shared error explains that
an unrelated application may own the ports even when the FullPack list is empty.

## HTTPS Connection Test prerequisites

Neither server had FullPack's isolated helper tree. Reverse Naive requires
sing-box v1.14.2 on Iran and naiveproxy v154.0.8037.49-4 on Kharej; XHTTP/REALITY
requires Xray v26.3.27 in FullPack's pinned tree. A separately installed Xray is
not automatically adopted. Missing helpers yield SKIPPED before traffic tests.
REALITY also needs its authenticated cover compatibility check to succeed.

The installer supports explicit `BP_HELPERS=naive,xray` on each server and
checks the official archive digests. Installing those prerequisites does not
establish that the corresponding WAN path will pass. No helper install or
production tunnel replacement was performed during this initial inspection.

## Verification status

Regression checks exercise two occupied TCP listeners with actual CLI
completion for Reverse Auto, Direct Auto and native Wat; rejection must occur
before host optimization, both occupied ports must appear in the diagnostic,
and the foreign client backend must remain allowed. Missing Iranian/foreign
Wat addresses and empty Reverse/Direct endpoint hosts are independently checked.
All fifteen selected Linux tests passed, including the occupied-listener and
missing-address regressions, existing enrollment/HTTPS secret preservation,
L3 forms, panel listener conflicts, address-specific binds and skipped-test
diagnostics. The test binary was built with Go 1.27.2 and hash-verified on Iran;
tests ran as `nobody` using disposable loopback listeners. The first incomplete
transfer to Kharej2 was rejected as a failed test artifact, not recorded as a
test pass. Temporary audit artifacts were removed afterward.

The Linux amd64 product build, affected-package `go vet`, all ten customization
tool tests, required reapply guards and whitespace checks passed. This scoped
setup repair has not been deployed/published; full release CI and live replacement
of the existing tunnel are outside these checks.

<div dir="rtl">

## خلاصهٔ فارسی

پورت‌های ۲۰۵۲ و ۸۸۸۰ متعلق به سرویس‌های قبلی دو سرور بودند و تونل یتیم
FullPack پیدا نشد. کنترل اشغال پورت در ساخت Auto/Wat از CLI قبل از هر تغییر
سیستم اضافه شد. سه آزمون HTTPS به دلیل نصب نبودن برنامه‌های کمکی در مسیرهای
اختصاصی FullPack اجرا نشده‌اند؛ این وضعیت اثبات شکست شبکه نیست.

</div>

*Last verified against FullPack v1.9.2 (setup/documentation compatibility review, 2026-10-14; original measurements and pending acceptance gates unchanged).*

*Last verified against FullPack v1.9.3 (source/documentation compatibility review, 2026-10-15; original measurements, earlier release results and pending acceptance gates retain their recorded dates and scope).*

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

*Last verified against FullPack v1.9.4-beta.5 (compatibility review of authenticated Auto smux isolation, 2026-10-16; historical measurements keep their original scope; deployment and soak require separate evidence).*
