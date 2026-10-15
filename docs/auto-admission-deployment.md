# Controlled Auto admission deployment (2026-10-15)

## خلاصهٔ فارسی

کاربر نصب نسخهٔ اصلاح‌شده و تست دوباره را تأیید کرد. نسخهٔ آزمایشی مشخص،
امضای معتبر فورک، امکان بازگشت باینری و بررسی واقعی داده لازم‌اند. تنظیمات
و تاریخچهٔ نسل‌ها حفظ می‌شوند؛ ۱۰۰۰ حساب با ۱۰۰۰ کاربر هم‌زمان یکسان نیست.

## Current gate

**PENDING / NOT VERIFIED:** v1.9.4-beta.1 full CI/signature, installation on all
three hosts, actual application/WAN data and the extended 700/800/1000-identity
load run. Earlier results remain historical in `auto-admission-repair.md` and
`verification/auto-admission-20261015.json`; their pre-deployment status is not
rewritten. No 24-hour or 1000-concurrent-user acceptance is claimed.

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
