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
and retains the complete release gate. Production port migration and signed-release gates are
**PENDING** until their actual receipts are recorded. This feature is not an
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

*Last verified against FullPack v1.9.4-beta.10 (source/documentation compatibility review of local listener preflight and status attribution, 2026-10-17; original measurements and failed or pending WAN/release/deployment/soak gates retain their separately recorded dates and scope).*
