# Tunnel Metrics

**Manage → Tunnel Metrics** shows traffic and connection counts per tunnel, and
for **KCP** the numbers that actually explain a slow link:

- retransmits,
- lost and duplicated segments, and
- how many packets **forward error correction (FEC)** repaired.

That last one is the direct answer to "is KCP earning its overhead on my route?"

Traffic totals are counted on **every** transport and are kept across restarts,
so the numbers do not reset when a tunnel bounces (and they carry on after a
[backup restore](backup-restore.md)).

---

<div dir="rtl">

## خلاصهٔ فارسی

**Manage → Tunnel Metrics** ترافیک و تعداد اتصال هر تونل را نشان می‌دهد، و برای
**KCP** همان عددهایی که واقعاً توضیح می‌دهند چرا لینک کند است: ارسال‌های مجدد،
سگمنت‌های گم‌شده و تکراری، و اینکه **تصحیح خطا (FEC) چند پکت را ترمیم کرده**.

عدد آخری جواب مستقیم این سؤال است: «آیا KCP روی مسیر من ارزش سربارش را دارد؟»
اگر ترمیم‌های FEC تقریباً صفر باشد، بی‌دلیل داری پهنای باند parity می‌دهی.

آمار ترافیک روی **همهٔ** ترنسپورت‌ها شمرده می‌شود و بین ری‌استارت‌ها حفظ می‌شود،
پس با بالا و پایین شدن تونل صفر نمی‌شود (و بعد از
[بازگردانی پشتیبان](backup-restore.md) هم ادامه پیدا می‌کند).

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against FullPack v1.9.0.*

*Last verified against FullPack v1.9.2 (documentation compatibility review, 2026-10-14; historical measurements and pending gates retain their recorded scope).*

*Last verified against FullPack v1.9.3 (source/documentation compatibility review, 2026-10-15; original measurements, earlier release results and pending acceptance gates retain their recorded dates and scope).*

## Admission release compatibility review (2026-10-15)

Peer Auto flow status additionally reports `limit`, `liveUdp`, `udpLimit`, `queuedTcp` and `queueLimit`. `Data` reports a separate current/recent warning and bounded typed error counters. Queued/rejected connections are not successful users, and cumulative counters do not imply every later request failed.

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*

*Last verified against FullPack v1.9.4-beta.7 (source review of bounded Auto receive/transmit isolation, 2026-10-16; historical measurements keep their scope; CI/deployment and soak require separate evidence).*

*Last verified against FullPack v1.9.4-beta.10 (source/documentation compatibility review of local listener preflight and status attribution, 2026-10-17; original measurements and failed or pending WAN/release/deployment/soak gates retain their separately recorded dates and scope).*
