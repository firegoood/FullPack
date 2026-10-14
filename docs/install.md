# Installing FullPack

## The normal way

One command as root on the VPS:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/firegoood/FullPack/main/install.sh)
```

It downloads the prebuilt release archive for your architecture (amd64/arm64)
into `/root/FullPack`, **verifies its checksum and the tag-bound FullPack
Ed25519 signature**, installs the binary, and opens the menu when it finishes.
Release installation needs OpenSSL 3.0 or newer, without Go. Online downloads
resolve the tag once and fetch all files from that release.
The older OpenSSL 1.1.1 `pkeyutl` cannot verify Ed25519; see the
[OpenSSL command documentation](https://docs.openssl.org/3.0/man1/openssl-pkeyutl/).

Reopen the menu any time with:

```bash
sudo fullpack
```

For a new Auto pair, choose **Setup Iran → Reverse → Auto → Quick TCP/UDP
Setup**, enter both server IPs and the forwarded ports, and apply the private
setup link on Kharej. The [Quick Auto guide](quick-auto-setup.md) lists its four
endpoints and the corresponding web form.

When this installer updates an existing installation, it reloads an active
FullPack panel/monitor service so the new embedded form is available. Tunnel
processes keep running; upgrading their engine still requires their restart.

Everything lands in a tidy layout — the release bundle in `/root/FullPack`,
backups in `/root/FullPack/backups`, tunnel configs in `/etc/fullpack`. See
[server layout](server-layout.md).

> **Building from source** still works as a fallback: clone the repo and run
> `sudo bash install.sh` inside it. If the release download fails it builds with
> Go, fetching modules **directly first** and via Iran-friendly mirrors
> (RunFlare, goproxy.cn) only when direct access fails.

Source builds need Go 1.26.9 or a patched later branch. On the Go 1.27 branch,
use 1.27.2 or newer: the installer refuses 1.27.0/1.27.1, which lack the security
fixes already present in the 1.26.9 baseline. The same requirement applies when
building manually. See the [official Go release history](https://go.dev/doc/devel/release).

---

## Offline install (the server cannot reach GitHub)

Download the release on any machine **with** internet, copy it to the server, and
install it there. Nothing is fetched from the VPS.

![Offline install](../img/offline-install.gif)

From the [releases page](https://github.com/firegoood/FullPack/releases/latest),
download the archive for the server's architecture — run `uname -m` on it:
`x86_64` → `fullpack_linux_amd64.tar.gz`, `aarch64` → `fullpack_linux_arm64.tar.gz`.

### With the installer (recommended)

It also records the layout for the uninstaller. Download `install.sh`,
`SHA256SUMS`, `SHA256SUMS.sig` and `RELEASE_TAG` alongside the archive from the
same release. Put all five in the **same private folder**, then run the script.
It verifies the tag-bound signature and archive checksum before extraction
without touching the network. Missing verification files are refused:

```bash
scp install.sh SHA256SUMS SHA256SUMS.sig RELEASE_TAG fullpack_linux_amd64.tar.gz root@SERVER_IP:/root/
ssh root@SERVER_IP "cd /root && sudo bash install.sh"
```

The tag can also be supplied explicitly, for example
`BP_RELEASE_TAG=v1.9.1 bash install.sh`. It must match the signed checksums.
Use the installer for offline extraction so signature checks and installation
metadata are both retained.

### Updating offline

The same way: repeat the steps with the newer archive. `install` replaces the
binary in place, and your tunnels in `/etc/fullpack` are untouched. Restart them
afterwards with `sudo fullpack` → **Manage → Restart ALL**.

---

## Updating online

**Main menu → 8) Update.** It downloads the release, verifies it against the
published SHA-256 and pinned signature, installs it, and **rolls back automatically** if a tunnel does
not come back up. Anything that cannot be verified is refused rather than
installed. [More](updates.md).

Older source-built FullPack installations may still carry the upstream public
release key. Run this fork's verified installer once to enroll the independent
FullPack release key before using their automatic updater. This affects release
authentication, not tunnel enrollment or configuration.

## Uninstalling

**Main menu → 9) Uninstall** removes everything FullPack installed.

---

<div dir="rtl">

## خلاصهٔ فارسی

**نصب عادی:** یک دستور با کاربر root روی سرور — آرشیو ریلیز مخصوص معماری سرور را
دانلود می‌کند، چک‌سام و امضای مخصوص FullPack را **تأیید** می‌کند، نصب می‌کند و خودش منو را باز
می‌کند. بعداً با `sudo fullpack` منو را باز کن.

**نصب آفلاین (سروری که به گیت‌هاب دسترسی ندارد):** فایل ریلیز را روی یک ماشین با
اینترنت دانلود کن و به سرور کپی کن. با `uname -m` معماری را ببین: `x86_64` یعنی
amd64 و `aarch64` یعنی arm64. فایل‌های `install.sh`، `SHA256SUMS`،
`SHA256SUMS.sig` و `RELEASE_TAG` را از همان ریلیز کنار آرشیو در **یک پوشهٔ خصوصی**
بگذار و اسکریپت را اجرا کن — فایل محلی و امضای نسخه را تأیید می‌کند و به شبکه
دست نمی‌زند. فایل یا امضای نامعتبر نصب نمی‌شود.

**آپدیت:** از منوی اصلی گزینهٔ ۸ — با تأیید SHA-256 و **بازگشت خودکار** اگر تونل
بالا نیامد. آپدیت آفلاین هم همان مراحل نصب با آرشیو جدید است و کانفیگ‌های
`/etc/fullpack` دست‌نخورده می‌مانند.

</div>

---
[← Back to the docs index](README.md)

---

*Last verified against FullPack v1.9.0.*

*Last verified against FullPack v1.9.2 (documentation compatibility review, 2026-10-14; historical measurements and pending gates retain their recorded scope).*
