# Quick Auto setup

Choose **Setup Iran -> Reverse -> Auto -> Quick TCP/UDP Setup**, or the panel's
**Auto -> Quick: all protocols** form. Enter the two server IPv4 addresses,
a name and forwarded ports. Choose port forwarding (stream) or an IP network
(L3, where forwarded ports are optional). Review the collapsed endpoint list
and then transfer the private version-3 peer link to Kharej:

```bash
sudo fullpack link apply '<private setup link>'
```

Both peers need the pool feature version. Existing pairs are unchanged; the
four-endpoint v1.9.2 configuration is not silently rewritten by an update.

## Approved pool and selected paths

Stream setup enrolls all eighteen registry carriers in Reverse and Direct
initiation, with both normal/reverse Wat Echo directions: 38 endpoints.
L3 setup separately enrolls all six datagram carriers in both directions: 12.
Routed IP never enters the reliable stream adapters. The one stable TUN remains
owned across agreed route generations.

The immutable approved pool is distinct from the **four selected** paths.
Authentication alone is not stability: a path needs three successful isolated
full data-integrity probes separated by at least ten seconds. Ranking uses the
median of bounded useful-throughput/RTT samples; these paced measurements are
not a line-rate speed test. The active route is retained while choosing backups.

When only two selected paths remain confirmed healthy, Auto starts a reserve
sweep and agrees on replacements with Kharej. It keeps at most four selected
paths, stores that subset privately and preserves the route generation/switch
history. A pressure or concurrency/replay-budget SKIPPED result is not a failed
carrier. Repeated shortfalls retry at a bounded interval (default 60 seconds).
If only one or two paths work on the network, the panel reports that shortfall;
it cannot create additional working paths by software alone.

Use `fullpack auto rescan NAME` or **Rescan approved reserve pool** for a manual
sweep. Observe-only, manual locks, optional-change cooldown/dwell/hourly limits,
and bounded outage recovery stay in effect. New setup enrolls
`connection_limit = 0` with no application connection ceiling. Optional positive
caps are available in advanced setup without a 192 maximum. Existing capped
pairs need coordinated new enrollment; see [unlimited Auto](auto-unlimited-connections.md).
A socket is not an independent user or a production capacity result.

## Ports and prerequisites

Default public data base ports are **24443 on Iran** and **24475 on Kharej**.
The registry's reviewed offsets use up to the next seventeen ports; Wat uses
ICMP rather than a TCP/UDP socket. Helper engine targets use separate enrolled
loopback ports at the relevant data port plus 128. Review the exact list and
permit the displayed socket families on their listener host. A wildcard local
bind may be selected on NAT hosts without altering the public enrollment.

Raw carriers need their Linux privileges and an eligible route. Naive needs
pinned Sing-Box/NaiveProxy and XHTTP/REALITY need pinned Xray on both nodes;
missing prerequisites are explicitly SKIPPED. Quick creation does not download
helpers or borrow unrelated binaries. Install optional helpers explicitly with
the verified installer (`BP_HELPERS=naive,xray`). REALITY retains the successful
Apple cover setting and must pass the actual authenticated cover check locally.

New pool helper identities are derived with purpose-separated HKDF from the
fresh shared pair secret. Both trusted enrolled peers can compute those new
credentials; they are not independent secrets from the counterpart. Existing
operator keys are never adopted. Export carries matching public CA/REALITY trust
and account settings; temporary private keys, executable paths and interfaces
remain local. Generated TLS identities are valid for one year from creation;
re-enroll before expiry. TLS hostname, certificate/public-key and authenticated carrier
checks stay enabled. Keep the enrollment link private and use a trusted channel.

## Verification scope

The signed v1.9.3 release passed all eleven CI verification jobs and public
installation. Whole-project normal/race/coverage, generated HTTPS in both
directions and real raw/L3 matrices passed; coverage is 58.2% with all unchanged
package floors. Strict stream network faults confirmed four-to-two-to-four
without breaking a held user connection or changing the active generation.
Actual twelve-candidate L3 replenishment preserved the same TUN and generation.
The two-physical-route shortfall case recovered without claiming extra paths.
See [release verification](fullpack-release-verification.md) for the exact
source, run, artifacts and bounded evidence. These are isolated acceptance
results; WAN availability, 24-hour soak and 1000-user capacity remain NOT VERIFIED.

<div dir="rtl">

## خلاصهٔ فارسی

در ساخت ساده، همهٔ پروتکل‌های سازگار با حالت انتخابی و هر دو جهت از ابتدا
ثبت می‌شوند. حالت فوروارد ۳۸ گزینه و L3 دوازده گزینه دارد؛ چهار مسیر پس از
آزمون واقعی داده گلچین می‌شوند. با رسیدن تعداد مسیرهای سالم به دو، بررسی
ذخیره‌ها تکرار می‌شود و مسیر فعال هنگام تعویض ذخیره‌ها حفظ می‌شود. نبود ابزار
یا مجوز با SKIPPED مشخص می‌شود. اگر شبکه فقط دو مسیر سالم بدهد، نرم‌افزار
کمبود را گزارش می‌کند و نمی‌تواند سلامت بیشتر را تضمین کند. کانفیگ و تاریخچهٔ
امنیتی جفت قبلی بازنویسی نمی‌شود؛ لینک خصوصی را روی خارج اعمال کنید.

</div>

*Last verified against FullPack v1.9.3 (pool implementation and signed release review, 2026-10-15; expanded pool CI/signed release passed; WAN/24-hour/1000-user gates remain open).*

## Admission release compatibility review (2026-10-15)

Quick Auto preserves its enrolled connection policy. Carrier availability and data admission are reported separately: an online route can be at capacity. See [admission repair](auto-admission-repair.md) for queue/UDP limits and [deployment evidence](auto-admission-deployment.md) for the measured scope.

## Listener-port review (2026-10-17)

Quick CLI/web creation now suggests a free local listener block, including
private helper targets and stopped configured reservations. Review the displayed
endpoints; manual ports remain available. Creation rechecks TCP/UDP and bound
raw TCP sockets before saving. Iran does not prove remote availability: Kharej
checks its own Direct listeners during trusted-link import and refuses an
occupied endpoint. Generate a fresh reviewed link if that shared port must
change. See [listener-port checks](auto-listener-ports.md).

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

*Last verified against FullPack v1.9.4-beta.7 (source review of bounded Auto receive/transmit isolation, 2026-10-16; historical measurements keep their scope; CI/deployment and soak require separate evidence).*

*Last verified against FullPack v1.9.4-beta.12 (source/documentation compatibility review of local listener preflight, status attribution and explicit backend readiness in the slow-reader fixture, 2026-10-17; original measurements and failed or pending WAN/release/deployment/soak gates retain their separately recorded dates and scope).*
