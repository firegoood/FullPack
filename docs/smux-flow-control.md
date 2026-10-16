# smux review and Auto slow-reader isolation (2026-10-16)

<div dir="rtl">

## خلاصهٔ فارسی

آخرین تگ رسمی کتابخانه، `v1.5.57` است و FullPack همین نسخه را دارد. تغییر
بعدی شاخهٔ اصلی فقط README است؛ نسخهٔ جدیدتری برای ارتقا پیدا نشد. شمارهٔ
نسخهٔ کتابخانه با پروتکل smux نسخهٔ ۱ یا ۲ تفاوت دارد. Auto قبلاً پروتکل ۱
را استفاده می‌کرد؛ در آن حالت یک مصرف‌کنندهٔ کند می‌تواند بافر مشترک را پر
کند و کنترل همهٔ کاربران را معطل کند. اصلاح حاضر توافق امن روی پروتکل ۲ و
بافر محدود مستقل هر اتصال را اضافه می‌کند. نتیجهٔ آزمایش محلی یا تست کوتاه
جایگزین آزمون طولانی روی WAN نیست؛ وضعیت انتشار و نصب جداگانه ثبت می‌شود.

</div>

## Upstream review

On 2026-10-16, the official repository's tags and `go list -m -json
github.com/xtaci/smux@latest` both resolve to **v1.5.57**, commit
`3b4ec04d359934256b3adea7133374e3c93a0622`, dated 2026-03-13. Comparing that
tag with master `ae956bb8d67bab37a312869b1d38ee3f52a7397a` finds one README-only
commit. There is no newer implementation to adopt. The older `/v2` module name
is not a reason to replace this maintained dependency.

Sources: [official tags](https://github.com/xtaci/smux/tags),
[exact comparison](https://github.com/xtaci/smux/compare/v1.5.57...ae956bb8d67bab37a312869b1d38ee3f52a7397a),
[configuration](https://github.com/xtaci/smux/blob/v1.5.57/mux.go),
[receive loop](https://github.com/xtaci/smux/blob/v1.5.57/session.go).

## Reproduced mechanism and adaptation

Upstream defaults to wire version 1. Its `MaxStreamBuffer` does not limit v1
streams. The receive loop stops before reading the next frame whenever its
shared byte bucket is empty. This blocks control data, window updates and
unrelated streams as well. Auto's authenticated STATUS request has an eight
second exchange deadline, so that congestion can cancel an otherwise usable
carrier and interrupt forwarded users. The separate heartbeat worker fixes
scan scheduling but cannot read through an exhausted receive bucket.

Simply selecting v2 does not eliminate aggregate starvation: enough individually
bounded slow streams can still fill the shared bucket. Increasing that bucket
only moves the threshold. Auto therefore uses the MIT-licensed source adaptation
in `internal/tunnel/peerauto/flowmux`, based on v1.5.57, with original source
hashes in `ORIGIN.json`. Manual transports retain the upstream Go dependency.

New Auto peers negotiate standard v2 within the existing PSK-authenticated
Noise transcript or the TLS-exporter-bound WSS handshake. A v2 WSS choice must
include a valid server proof, including on first contact. Missing capabilities
retain v1. No framing, cryptographic domain, manifest hash or enrollment policy
is changed; an existing pair does not need new fencing state or enrollment.
Both peers need the updated implementation for slow-reader isolation. A legacy
peer remains compatible with the bounded shared bucket and its limitation.

Until full production identity, manifest, challenges and incarnation checks
pass, the finite 4 MiB session bucket remains enforced. Probe conversations
retain that budget. After production enrollment, v2's finite individual windows
replace the shared wait. Receive-credit validation applies before and after
enrollment and rejects over-credit frames as `flow_control_error`. The maximum
queued receive payload is 256 KiB per stream (the upstream initial window);
normal updates advertise the configured 128 KiB. A frame already copied toward
the application and OS socket buffers are separate bounded/inherited resources.
Modulo byte counters keep their bound through uint32 wrap. Native Wat/QUIC
flow control and bounded transient handshake work remain unchanged.

The local candidate status exposes `mux.version`, `mux.isolated`,
`mux.maxBufferedStream` and `mux.bufferedReceive`. Queued bytes describe transport
backpressure, not account counts or proof of end-to-end application delivery.
Memory scales with accepted streams and their finite windows; unlimited
enrollment does not guarantee unlimited OS memory, descriptors or bandwidth.
There is no new fixed application connection ceiling or enormous global bucket.

## Verification status

The deterministic legacy test fills a 256 KiB shared receive budget and observes
control timeout. The isolated v2 test keeps 64 non-reading streams while both
control and an unrelated application stream continue, in each direction.
Compatibility tests exchange more than 4 MiB each way with **unmodified upstream
smux v1.5.57**, for wire versions 1/2 and both client roles. Additional cases check
excess receive credit, uint32 wrapping, old/new Noise combinations, wrong tokens,
manual WSS offers and forged/missing TLS-bound server proofs.

These lower-level tests passed locally with Go 1.27.2 and on isolated Linux.
All six Linux full-tunnel cases passed, with 5,316,608–5,808,128 queued receive
bytes, unchanged generations and continuing unrelated application data/control.
The manifest-rejected/locally-disabled enrollment cases retained their shared
budget; accepted production enabled isolation. The first fixture attempt had
invalid short test identities and failed before traffic; those fixture IDs were
corrected without relaxing validation. Race/full CI, signed deployment and
extended post-repair WAN delivery are **PENDING**. The full-tunnel regression
requires 64 slow readers to queue more
than the former 4 MiB budget while another application and multiple heartbeats
continue with unchanged generation and no interrupted/rejected flows, using
TCP, WSS and Stealth in Reverse and Direct. A skipped or failed workload is not
a pass. The earlier beta.4 timeout recurrence is retained in
`auto-control-liveness.md`.

**NOT VERIFIED:** real-hour/24-hour stability, all carriers under sustained load,
or guaranteed capacity for 1000 real users.

*Last verified against FullPack v1.9.4-beta.5 (source review, 2026-10-16; pending runtime gates are explicit above).*
