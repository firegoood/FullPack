# Controlled Auto admission deployment (2026-10-15)

<div dir="rtl">

## خلاصهٔ فارسی

کاربر نصب نسخهٔ اصلاح‌شده و تست دوباره را تأیید کرد. نسخهٔ آزمایشی مشخص،
امضای معتبر فورک، امکان بازگشت باینری و بررسی واقعی داده لازم‌اند. تنظیمات
و تاریخچهٔ نسل‌ها حفظ می‌شوند؛ ۱۰۰۰ حساب با ۱۰۰۰ کاربر هم‌زمان یکسان نیست.

</div>

## Current gate

**PENDING / NOT VERIFIED:** v1.9.4-beta.1 full CI/signature, installation on all
three hosts, actual application/WAN data and sustained WAN capacity. The isolated extended identity
run below has completed; it does not pass those deployment/WAN gates. Earlier results remain historical in `auto-admission-repair.md` and
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
or restarted by this isolated test. Actual WAN testing remains pending.

*Last verified against FullPack v1.9.4-beta.1 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*
