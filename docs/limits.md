# Per-tunnel limits

**Edit → Limits** puts two caps on a single tunnel:

- **Simultaneous connections** — the most forwarded connections it will carry at
  once.
- **Throughput** — a ceiling on its total bandwidth.

Both are **off by default**. They are useful when several services or customers
share one link and you want to stop any one of them from taking it all.

These are the manual tunnel limits. New Auto setup explicitly enrolls
`connection_limit = 0`, with no application connection ceiling. A positive cap
is optional, without a 192 maximum. Legacy enrollment remains unchanged until
coordinated replacement. See [unlimited Auto](auto-unlimited-connections.md).
Sockets and short loopback tests do not establish sustained WAN user capacity.

---

<div dir="rtl">

## خلاصهٔ فارسی

از `Edit → Limits` می‌شود دو سقف روی یک تونل گذاشت: **حداکثر اتصال هم‌زمان** و
**سقف پهنای باند**. هر دو پیش‌فرض **خاموش** (صفر = بی‌نهایت) هستند.

وقتی چند سرویس یا چند مشتری روی یک لینک مشترک‌اند به کار می‌آید تا هیچ‌کدام کل
لینک را نگیرد.

اگر سقف اتصال را زیر ۱۰ بگذاری هشدار می‌دهد — یک مرورگر به‌تنهایی بیشتر از این
اتصال باز می‌کند.

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against FullPack v1.9.0.*

*Last verified against FullPack v1.9.2 (documentation compatibility review, 2026-10-14; historical measurements and pending gates retain their recorded scope).*

*Last verified against FullPack v1.9.3 (source/documentation compatibility review, 2026-10-15; original measurements, earlier release results and pending acceptance gates retain their recorded dates and scope).*

## Admission release compatibility review (2026-10-15)

The earlier capped admission repair retained the enrolled total (default 64, maximum 192). That historical policy is recorded in [admission repair](auto-admission-repair.md). Optional positive caps still use the UDP quarter share and bounded two-second TCP FIFO. New unlimited enrollment has no connection-count gate; physical resources and native transport constraints remain.

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

*Last verified against FullPack v1.9.4-beta.5 (compatibility review of authenticated Auto smux isolation, 2026-10-16; historical measurements keep their original scope; deployment and soak require separate evidence).*
