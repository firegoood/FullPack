# Quick Auto setup

Quick setup creates an explicitly enrolled port-forwarding pair. It is available
from **Setup Iran -> Reverse -> Auto -> Quick TCP/UDP Setup** in the CLI, and from the
panel's **Auto -> Quick TCP/UDP setup** form. Enter Iran/Kharej IPv4 addresses,
a tunnel name and forwarded ports such as `2052,8880`; UDP forwarding is optional.

The CLI generates the enrollment secret and pair identities. The panel's quick
payload uses the same server builder and validators. Review all four endpoints
before creating the tunnel:

| Carrier | Connection initiator | Listener |
| --- | --- | --- |
| TCP with Auto's authenticated encryption | Kharej (Reverse) | Iran:24443 |
| WSS with authenticated TLS binding | Kharej (Reverse) | Iran:24444 |
| Noise / Stealth | Kharej (Reverse) | Iran:24445 |
| Noise / Stealth | Iran (Direct) | Kharej:24445 |

The forwarded user ports remain on Iran. Backend applications listen on Kharej.
These data ports must be free and permitted on their respective listener hosts.
The panel's collapsed port section changes the Reverse base and Direct port;
Advanced CLI setup also offers arbitrary endpoints. Quick setup does not install
HTTPS helpers or require raw sockets.

After creating Iran, copy its private version-3 setup link to Kharej and run the
printed command, or use the panel's enrollment import form:

```bash
sudo fullpack link apply '<private setup link>'
```

On NAT hosts, the import form can select a receiver-local wildcard bind without
changing the enrolled public endpoint. Local bind overrides/private helper
settings do not travel inside the peer link.

Quick mode explicitly enables automatic failover. It retains the existing
64-connection default, 192 maximum, probe budgets and optional-change cooldown/
dwell/hourly policy; confirmed outage recovery keeps its separate bounded gate.
Full probes normally run about every ten minutes, with jitter, and active-route
liveness checks run more often. A manual **test-now** action measures an enrolled,
authenticated candidate without changing the active route. Auto does not scan
arbitrary protocols/endpoints outside its one-to-four candidate manifest.

Advanced setup retains all eighteen stream carriers, both initiation directions,
Wat ICMP, managed Naive/XHTTP/REALITY identities and the six datagram L3 carriers.
Choose it for a different selection or a stable TUN. Neither quick mode nor a
successful setup guarantees that every enrolled route will work on a given WAN.

Quick setup creates fresh pairs. It does not replace an existing enrolled
manifest or delete its generation fencing. Re-enrollment requires a new pair;
preserve a verified backup of the old config/state before retiring it.

## Verification

The shared builder/peer-link and invalid-input Linux tests and the panel payload
tests passed during development. The isolated browser form showed the same four
reviewed endpoints, preserved inputs across quick/advanced switching and omitted
advanced settings from the quick payload. All eleven full Linux CI gates and
the signed public installer passed for the immutable v1.9.2 release in
[run 38626831074](https://github.com/firegoood/FullPack/actions/runs/38626831074).
Independent downloads verified every release archive, tag-bound signature and
patched toolchain. See [release verification](fullpack-release-verification.md)
for the retained WAN and long-duration acceptance limits.

<div dir="rtl">

## خلاصهٔ فارسی

برای ساخت سریع، در CLI یا پنل Auto گزینهٔ Quick را انتخاب کنید، IP دو سرور
و پورت‌های فوروارد را وارد کنید و چهار endpoint نمایش‌داده‌شده را بررسی کنید.
سپس لینک خصوصی ایران را روی خارج اعمال کنید. ساخت پیشرفته برای Wat، HTTPS
و L3 محفوظ است. این مسیر سریع، کانفیگ و وضعیت امنیتی جفت موجود را بازنویسی
نمی‌کند و موفقیت همهٔ پروتکل‌ها روی هر مسیر اینترنت را تضمین نمی‌کند.

</div>

*Last verified against FullPack v1.9.2 (setup/documentation compatibility review, 2026-10-14; original measurements and pending acceptance gates unchanged).*
