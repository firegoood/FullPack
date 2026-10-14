# FullPack Auto: Reverse, Direct and L3

Approved scope: complete the existing Reverse carriers, explicit Direct
initiation and datagram L3. Auto remains disabled unless explicitly enrolled.
This extension follows `waticmp-dual-server-design.md`; measured acceptance and
remaining environment gates belong in `waticmp-verification.md`.

## Carriers and packet planes

`config.AutoCarriers()` is the shared registry used by CLI, panel and engine
validation. An enrollment selects **one to four** approved candidates, not all
eighteen simultaneously. Every candidate supports Reverse (Kharej initiates)
or Direct (Iran initiates). Iran remains the sole switching authority in both.

| Registry choices | Auto session protection | L3 IP datagrams |
| --- | --- | --- |
| Wat ICMP | Native Wat streams, fresh bilateral X25519/PSK enrollment | No |
| Stealth | Noise and smux | No |
| TCP, TCPMUX | Noise and smux | No |
| WS, WSMUX | WebSocket, Noise and smux | No |
| WSS, WSSMUX | Existing authenticated TLS/WebSocket adapter | No |
| KCP | Encrypted KCP, Noise and smux | No |
| UDP | Encrypted KCP control, Noise and smux | Yes, separate unreliable packets |
| xDi, PCK | Existing raw packet layer, encrypted KCP control and Noise | Yes, separate unreliable packets |
| QUIC | Existing authenticated QUIC adapter | Yes, QUIC DATAGRAM |
| IP spoofing, SNI spoofing | Raw packet layer, encrypted KCP control and Noise | Yes, separate unreliable packets |
| Naive/HTTP2, XHTTP/TLS, REALITY/Vision | Existing verified HTTPS helper and private Noise session | No |

Manual transports retain their existing wire behavior. Auto wrappers require
both upgraded peers; adding authenticated Auto to a legacy plaintext carrier
does not turn it into a compatible manual peer. Plaintext fallback is refused.

Native Wat Direct optionally enables disjoint odd/even stream identities so
Iran can open control/data streams while Kharej accepts them. The default
manual engine keeps its previous stream admission behavior. Geographic
cryptographic roles and directional nonces remain unchanged. Production Wat
handshake accepts are serialized; probe enrollment remains separately keyed.

SPOOF Auto owns one raw provider per candidate. Production and fresh probes
use independent random conversation IDs authenticated before allocation, on
the same explicitly enrolled receive port. This fixes the legacy forged-source
KCP session collision without changing manual spoof framing or opening an
additional probe port. All six existing profiles remain available: UDP, TCP,
ICMP, ICMPv6, IPIP and GRE. Spoofed addresses must be permitted by the actual
network; a local passing fixture cannot establish that a VPS permits spoofing.

## Setup, identities and local edits

Create the Iran enrollment in CLI Setup or the panel Native/Auto family.
Choose stream forwarding or L3, then each candidate's carrier, connection
direction and approved public listener. The local listener bind can differ
from its public/NAT address but must retain the enrolled port. Wat uses its
explicit peer IPv4s and a separate Echo Request direction.

Transfer the version-3 credential link through a trusted private channel.
CLI Kharej setup and web enrollment preview collect receiver-local bindings,
interfaces and helper identities. A direct HTTPS listener on Kharej needs its
matching certificate/key or the REALITY private key corresponding to the
enrolled public key. Generate/provision that identity before enrollment; do
not generate a different identity independently on the receiving endpoint.
Generic `fullpack link apply` is suitable only when required local material
is already available; use the setup wizard or panel import to provide it.

Naive dial endpoints and certificate names must match. XHTTP checks its TLS
name and enrolled CA; REALITY retains the authenticated cover probe. Helpers
bridge only their restricted loopback engine port. Public endpoints, account
credentials, public CA and public keys are paired; local executable paths,
CA files, certificates, private keys, interface names and local binds are not
exported. Settings redact passwords/private keys and blank secret edits keep
the current secret. A replacement local identity must still verify against
the paired public identity.

The panel can edit local bindings, raw/TUN interfaces, helper paths and local
identity files. Shared endpoints, account identities, port allowlists and
policy remain fenced: changing them requires a new enrolled pair. Node
rebuilds and compatible repeated CLI link application restore local material
from the receiving machine's private config before strict validation, without
round-tripping private keys through settings responses. Incompatible role,
secret or shared manifest changes are refused before inheriting local keys.

## L3 and switching limits

Only UDP, QUIC, xDi, PCK, SPOOF and SNI may carry L3. The coordinator opens one
TUN with distinct enrolled addresses in the same subnet and retains it across
handover. Forwarded ports are optional. IPv6 requires inner MTU at least 1280;
inner packets may be up to 9000 bytes. Default interface names are pair-specific.

IP traffic uses directional ChaCha20-Poly1305 keys derived from the fresh
authenticated session, distinct production/probe purposes, generation-bound
associated data, monotonic nonces, a 4096-packet replay window and sixteen
bounded expiring reassembly slots. IP packets are not retransmitted through
KCP or smux. A candidate must pass its actual bidirectional datagram probe;
successful control authentication alone cannot qualify an L3 path. Retired
generations cannot deliver packets into the TUN after route replacement.

QUIC requires an outer path supporting its mandatory 1200-byte initial
datagram. On smaller paths it remains unavailable; an attempted selection
must retain the agreed working route and generation. The other tested raw
and UDP paths pass at outer MTU 700 with 9000-byte inner IP fragmentation.
This is not a claim that every raw profile supports every lower MTU.

Observe-only, enrollment locks, durable generations, uncertain-commit fencing,
failure debounce, probe pressure/busy skips, cooldown/dwell, hourly limits and
128 KB/s application probe pacing remain enforced. Existing TCP flows can
drain for up to thirty seconds or be interrupted; there is no promise of
seamless TCP migration. Stable L3 addresses similarly do not eliminate packet
loss during a failed route or switching interval.

Confirmed, debounced no-route recovery uses a separate five-second gate and
the absolute twelve-commit rolling-hour ceiling. All commits retain the shared
durable history and extend the full configured optional cooldown/hourly policy.
Healthy optional quality switching keeps Cooldown/MinimumDwell/MaxSwitchesPerHour;
locks, pressure skips and observe-only remain enforced. See
`auto-policy-review.md` for the consecutive-outage tests and limits.

## Repeatable checks

Use the patched toolchain and the existing unchanged coverage floors. Tests
and artifacts exclude `11/`. Fixtures require disposable Linux namespaces;
do not run them in host networking on a live tunnel server.

```sh
python3 tools/customize/reapply.py --apply
python3 tools/customize/reapply.py --check
node --test internal/webui/paneltest/*.test.js
sudo env "PATH=$PATH" WATICMP_LIVE=1 go test ./... -race -coverpkg=./... -coverprofile=coverage.out -timeout 25m
sudo env "PATH=$PATH" FULLPACK_AUTO_HELPERS=1 go test ./internal/e2e -run TestPeerAutoManagedHTTPSBothDirections -race -count=1
sudo env "PATH=$PATH" FULLPACK_AUTO_NETNS=1 FULLPACK_AUTO_RAW_CARRIERS='waticmp xdi pck spoof sni' bash internal/e2e/testdata/auto-carriers-netns.sh
sudo env "PATH=$PATH" FULLPACK_AUTO_NETNS=1 FULLPACK_AUTO_OUTER_MTU=1500 FULLPACK_AUTO_PLANES=l3 FULLPACK_AUTO_L3_CARRIERS=quic bash internal/e2e/testdata/auto-carriers-netns.sh
```

`FULLPACK_AUTO_L3_FORWARD=1` also exercises optional enrolled TCP/UDP forwarding
through the same stable TUN before and after handover.

The HTTPS test requires the checksum-pinned installer-managed Naive, sing-box
and Xray executables. Missing prerequisites are SKIPPED, not evidence of a
passing carrier. `FULLPACK_AUTO_SPOOF_PROFILE` selects an existing raw profile
for the namespace fixture. CI preserves the real helper and namespace checks.

At local acceptance, WAN/two live VPS were NOT VERIFIED. Subsequent bounded
operator-authorized tests are recorded in [the WAN report](fullpack-wan-verification.md);
they passed only the listed routes and a TCP-to-WSS recovery on both pairs.
Unavailable routes, arbitrary NAT/firewall combinations, production load and
a 24-hour soak remain **NOT VERIFIED**. No release or unattended default-on
policy is implied by either local or partial WAN acceptance.

<div dir="rtl">

## خلاصهٔ فارسی

Auto با ثبت و تأیید هر دو سرور، حداکثر چهار حامل از رجیستری مشترک انتخاب
می‌کند: ۱۸ حامل Stream با آغاز Reverse یا Direct و شش حامل دیتاگرامی L3.
L3 یک TUN ثابت دارد و بسته‌ها احراز هویت و محدودیت بازپخش/بازسازی دارند.
کلیدهای خصوصی و مسیرهای محلی در لینک ثبت سرور صادر نمی‌شوند. Auto پیش‌فرض
خاموش است؛ موفقیت محلی یا بعضی مسیرهای WAN به معنی تأیید همهٔ مسیرها،
ظرفیت ۱۰۰۰ کاربر یا آزمون ۲۴ساعته نیست.

</div>

*Last verified against FullPack v1.9.2 (documentation compatibility review, 2026-10-14; historical measurements and pending gates retain their recorded scope).*
