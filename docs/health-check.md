# Health Check

**Manage → Health Check** runs a full check of the server and every tunnel, and
prints a concrete fix under each problem it finds. It verifies:

- kernel tuning is applied,
- the web panel is running,
- every tunnel's state,
- real **TCP reachability** (not just whether systemd is happy),
- whether the path can carry a full-sized packet — see
  [TCP MSS clamp](mss-clamp.md) for the setting it names when it cannot,
- TLS certificate expiry, and
- token strength.

It also reports whether the [monitor service](monitor-service.md) is running,
and says plainly that dropped tunnels will **not** be restarted if it is down.

## File Locations

Right next to it, **Manage → File Locations** lists where every config, service
and backup lives on the machine. See [Server layout](server-layout.md) for the
full map.

---

<div dir="rtl">

## خلاصهٔ فارسی

**Manage → Health Check** کل سرور و همهٔ تونل‌ها را بررسی می‌کند و **زیر هر
مشکل، راه‌حل مشخصش را می‌نویسد**. وقتی چیزی درست کار نمی‌کند، از اینجا شروع کن.

چه چیزهایی را چک می‌کند: اعمال شدن تنظیمات کرنل، اجرای پنل وب، وضعیت هر تونل،
**دسترسی واقعی TCP** (نه فقط راضی بودن systemd)، توانایی مسیر در عبور دادن پکت
کامل (اگر نتواند، مقدار دقیق [MSS clamp](mss-clamp.md) را می‌گوید)، انقضای گواهی
TLS و قدرت توکن.

همچنین می‌گوید [سرویس مانیتور](monitor-service.md) در حال اجراست یا نه — و اگر
نباشد، صریح می‌گوید که تونل‌های افتاده **ری‌استارت نخواهند شد**.

کنارش **Manage → File Locations** محل تک‌تک فایل‌های پیکربندی، سرویس و پشتیبان را
نشان می‌دهد.

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against FullPack v1.9.0.*

*Last verified against FullPack v1.9.2 (documentation compatibility review, 2026-10-14; historical measurements and pending gates retain their recorded scope).*

*Last verified against FullPack v1.9.3 (source/documentation compatibility review, 2026-10-15; original measurements, earlier release results and pending acceptance gates retain their recorded dates and scope).*

## Admission release compatibility review (2026-10-15)

For peer Auto, carrier state and `DataWarning`/`DataCode` are independent. Current capacity or recent admission/backend errors remain visible even with a healthy carrier; the recent warning window is 30 seconds. Available admission is not an application-delivery check. See [repair and limits](auto-admission-repair.md).

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

*Last verified against FullPack v1.9.4-beta.7 (source review of bounded Auto receive/transmit isolation, 2026-10-16; historical measurements keep their scope; CI/deployment and soak require separate evidence).*
