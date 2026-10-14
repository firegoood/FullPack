# FullPack v1.9.1 release verification

Date: 2026-10-14. Status: **v1.9.1 PUBLISHED; SIGNED PUBLIC INSTALL VERIFIED**.

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
  tag metadata were replaced; binaries, checksums and source tag stayed unchanged.
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

## Completed release gates

[v1.9.1](https://github.com/firegoood/FullPack/releases/tag/v1.9.1) retains source
commit `5abcbfab71a6e860cf68706a7b21d4c5364eba0a`. All eleven CI verification jobs
passed in [run 38558483463](https://github.com/firegoood/FullPack/actions/runs/38558483463),
including whole-project normal/race tests, 58.0% coverage with all fourteen
unchanged package floors, static/security checks, native/L3/helper fixtures,
bounded soak/fuzz and held 100/192-connection admission. The corrected release's
public installer subsequently passed and both operator servers installed it.

An independent download verified all twelve public assets: the tag-bound
Ed25519 signature, seven architecture checksums/ELF identities, exact binary and
legal archive members, exclusion of `11/`, the compiled fork key, installer
equality with the immutable tag and the Go 1.27.2 SBOM. No private signing key
or operator configuration is part of those assets.

## Remaining acceptance gates

No previous published FullPack tag existed, so the compatibility job's
previous-release leg was explicitly SKIPPED for this first fork release.
The existing WAN reports retain their QUIC/Wat route failures and partial
acceptance. Burst admission under impairment, current helper security,
24-hour soak and 1000-user capacity remain **NOT VERIFIED**.

<div dir="rtl">

## خلاصهٔ فارسی

نسخهٔ ۱.۹.۱ با کلید مستقل FullPack منتشر و نصب عمومی آن بررسی شد. نصب‌کننده پیش از
استخراج، امضای همان نسخه و چک‌سام آرشیو را با OpenSSL 3 یا جدیدتر بررسی می‌کند.
هر یازده بخش CI و بررسی مستقل هفت آرشیو موفق بودند. یک اجرای burst صدتایی پس از بازیابی Noise،
۲۱ اتصال ناموفق داشت؛ آن نتیجه حذف یا موفق اعلام نشده است. تست بازیابیِ
اتصال‌های از قبل پذیرفته‌شده با burst متفاوت است و ظرفیت واقعی کاربران یا
پایداری ۲۴ساعته را ثابت نمی‌کند. پوشهٔ `11/` و اطلاعات خصوصی منتشر نمی‌شوند.

</div>

*Last verified against FullPack v1.9.2 (documentation scope review, 2026-10-14; final release/install gates remain pending).*
