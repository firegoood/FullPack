# Updates & rollback

The **Update** menu detects a newer GitHub release and installs the prebuilt
`fullpack_linux_<arch>.tar.gz`.

## How it downloads

**Direct from GitHub, or through a tunnel peer** when this server cannot reach
GitHub itself. Both paths terminate TLS at GitHub, so the download is verified
against the release's published **SHA-256** and the pinned FullPack Ed25519
signature over that release's tag and checksum list.

Older source-built FullPack installations may still trust the inherited upstream
key. Reinstall once with the verified fork installer before using their updater.

**An archive that cannot be verified is refused, not installed.** A server that
can reach neither path installs offline instead — see the offline install
section in the main README.

## Safety net

Before touching anything, Update saves a **restore point**. After installing it
health-checks the result and **rolls back by itself** if anything fails to come
back up. You can also roll back on demand from **Update → Restore points**.

## Channels

Follow **stable** (default) or **beta** under **Update → Release channel**.

## Upgrading a very old install

From a clone-based install (≤ v1.2.0): run Update once; after that it is
release-based.

---

<div dir="rtl">

## خلاصهٔ فارسی

منوی **Update** ریلیز جدیدتر گیت‌هاب را پیدا و نصب می‌کند.

**چطور دانلود می‌کند:** مستقیم از گیت‌هاب، یا **از طریق یک تونل** وقتی خود این
سرور به گیت‌هاب دسترسی ندارد. هر دو مسیر TLS را در گیت‌هاب terminate می‌کنند، پس
دانلود با **SHA-256** منتشرشدهٔ ریلیز تأیید می‌شود.
**آرشیوی که تأیید نشود نصب نمی‌شود، رد می‌شود.**

**تور ایمنی:** قبل از دست زدن به چیزی یک **restore point** ذخیره می‌کند، بعد از
نصب نتیجه را health-check می‌کند و **اگر چیزی بالا نیامد خودش برمی‌گردد عقب**.
دستی هم می‌شود از `Update → Restore points` برگشت.

**کانال:** پیش‌فرض **stable**؛ اگر بخواهی pre-release‌ها را هم بگیری، از
`Update → Release channel` روی **beta** بگذار.

سروری که به هیچ‌کدام دسترسی ندارد، [آفلاین نصب می‌کند](install.md).

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against FullPack v1.9.0.*

*Last verified against FullPack v1.9.2 (documentation compatibility review, 2026-10-14; historical measurements and pending gates retain their recorded scope).*

*Last verified against FullPack v1.9.3 (source/documentation compatibility review, 2026-10-15; original measurements, earlier release results and pending acceptance gates retain their recorded dates and scope).*

## Admission release compatibility review (2026-10-15)

An explicitly selected prerelease is identified by its complete suffix. For the controlled admission deployment, the installer uses `BP_RELEASE_TAG=v1.9.4-beta.1` and verifies that tag-bound signature. Stable/latest selection remains independent. Keep current enrollment/fencing and a private binary rollback; never restore an older generation snapshot.

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*
