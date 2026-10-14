# FullPack v1.9.1 release verification

Date: 2026-10-14. Status: **PREPARATION; FINAL CI/PUBLIC INSTALL PENDING**.

The fork originally had no release assets or signing secret. Its independent
Ed25519 public key is recorded in `tools/customize/fullpack.json`, the
application and installer. The private half was created in memory and encrypted
into the repository's Actions secret; it is absent from source and artifacts.

Publication requires the existing complete CI workflow. Missing/mismatched
signing keys stop publication. The installer resolves one tag, verifies the
tag-bound signature with OpenSSL 3.0 or newer and checks the archive hash before
extraction. Offline installation requires the same signed metadata and tag.

## Failures retained during preparation

- The initial Go 1.26.9 formatter disagreed with the audited Go 1.27.2 formatter
  on two map entries. CI/release builds now use patched Go 1.27.2 consistently;
  the source minimum and installer rejection of unpatched Go remain intact.
- The ordinary loopback Connection Test omitted Wat peer addresses but checked
  privileges first, making its expected SKIPPED result depend on the CI user.
  Missing input is now reported before availability; privileged raw tests remain.
- The signer now rejects keys that differ from the compiled pin. Its existing
  tag-replay test must compile its child signer with its generated test public
  key too. This fixture does not change the production pin or verification.
- Documentation gained explicit compatibility-review stamps and missing
  Persian summaries. Original measurement dates and unpassed gates remain.
- The public install check rejected the first signature because the manual
  workflow's reserved `GITHUB_REF_NAME` stayed `main`. The release was withdrawn
  to draft. A narrowly scoped metadata repair checks the immutable tested source,
  all eleven successful CI jobs and seven original archive hashes before signing
  the unchanged checksum list explicitly for `v1.9.1`. Only its signature and
  tag metadata may be replaced; binaries, checksums and source tag stay unchanged.
- [Run 38516691971](https://github.com/firegoood/FullPack/actions/runs/38516691971)
  reached agreed Wat-to-Noise recovery in six seconds, then its unpaced
  100-transaction burst reported **21 resets out of 100** under MTU 700,
  0.5% loss, duplication/reordering and delay. Its precise rejection cause was
  not recorded; no claim is made that this burst passed or was repaired.

The recovery fixture now explicitly holds **100 admitted application sockets**
before releasing bulk traffic. A one-byte integrity echo proves the enrolled
peer and backend admitted each socket. It retains the same packet impairments,
100-connection workload, application limits and production deadlines. This is
a held-socket workload distinct from the retained default unpaced-burst fixture; passing
it does not resolve burst admission or establish sustained WAN/user capacity.
Failure paths also capture both peers' status and safe admission debug errors.

## Remaining gates

The final CI and public signed-install results will be recorded after completion.
No previous published FullPack tag exists, so the compatibility job's
previous-release leg is explicitly SKIPPED for this first fork release.
The existing WAN reports retain their QUIC/Wat route failures and partial
acceptance. Burst admission under impairment, current helper security,
24-hour soak and 1000-user capacity remain **NOT VERIFIED**.

<div dir="rtl">

## خلاصهٔ فارسی

این نسخه برای انتشار با کلید مستقل FullPack آماده می‌شود. نصب‌کننده پیش از
استخراج، امضای همان نسخه و چک‌سام آرشیو را با OpenSSL 3 یا جدیدتر بررسی می‌کند.
انتشار به موفقیت CI وابسته است. یک اجرای burst صدتایی پس از بازیابی Noise،
۲۱ اتصال ناموفق داشت؛ آن نتیجه حذف یا موفق اعلام نشده است. تست بازیابیِ
اتصال‌های از قبل پذیرفته‌شده با burst متفاوت است و ظرفیت واقعی کاربران یا
پایداری ۲۴ساعته را ثابت نمی‌کند. پوشهٔ `11/` و اطلاعات خصوصی منتشر نمی‌شوند.

</div>

*Last verified against FullPack v1.9.1 (documentation scope review, 2026-10-14; final release/install gates remain pending).*
