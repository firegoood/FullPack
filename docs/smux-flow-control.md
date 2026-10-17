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

Tiny frames share 4 KiB receive allocations instead of retaining a ring entry
and allocator object for every byte. Payload credit remains unchanged; partial
read offsets and original allocation capacities are preserved for recycling.
The regression queues 128 KiB of one-byte frames with at most 34 ring entries,
then checks exact byte order and zero remaining payload accounting. A second
case checks appending after a partial head read.

The transmit side also needs separation. With 128 bulk writers queued behind
synthetic frame pacing, the receive-only candidate's control write exceeded a
compressed 300 ms deadline. The same private fixture with control priority
completed in about 17 ms. These are local scheduler measurements, not WAN RTTs.
The adapted engine has an eight-frame lane for its single enrolled control
stream and a separate 32-frame lane for new stream SYN/initial handshake writes.
Only the first 256 small outgoing bytes per enrolled v2 stream can use handshake
priority; a large initial write retires the allowance. Existing receive credit,
FIFO framing within normal application writes and native Wat/QUIC are unchanged.
One queued bulk frame progresses after at most sixteen priority frames, and
control is chosen ahead of the handshake lane. This prevents both heartbeat and
new-connection admission from waiting behind megabytes of queued user payload.
It cannot preempt bytes already in an underlying socket/helper or create WAN
bandwidth. `mux.controlPriority` reports the local enrolled control lane.

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
corrected without relaxing validation. At that initial local record, race/full
CI, signed deployment and extended WAN delivery were pending; the subsequent
release/deployment status follows below. The full-tunnel regression
requires 64 slow readers to queue more
than the former 4 MiB budget while another application and multiple heartbeats
continue with unchanged generation and no interrupted/rejected flows, using
TCP, WSS and Stealth in Reverse and Direct. A skipped or failed workload is not
a pass. The earlier beta.4 timeout recurrence is retained in
`auto-control-liveness.md`.

The beta.6 isolated Linux rerun passed all six mandatory workloads, with
5,554,176–7,970,816 buffered receive bytes and a 17,024 KiB parent RSS after
joined children. The following pool recovery passed 4 -> 2 -> 4 at unchanged
generation 2. A separate instrumented parent/child check retained nine positive
profile entries in the mux enrollment file. Source hashes and bounded evidence
are recorded in `verification/auto-smux-priority-local-20261016.json`; full CI,
deployment and post-repair WAN delivery are still pending in that local record.

The receive-only beta.5 source `7a115437372a7de8d1a5104c6ec42ca6453b6863`
failed CI `38942638366`; it was **not released or installed**. Normal, coverage,
HTTPS, Direct/L3, static/security, fuzz, sustained and compatibility jobs passed,
but native/race shared-process pool bootstraps failed, followed by additional
race bootstraps. The earlier output did not record the pressure subtype; resource
retention is an inference. Raising production thresholds is not the repair.
All six slow-reader cases now run in fresh instances of the same instrumented
test executable. Each parent verifies its child's exact workload marker, and
forwards `test.gocoverdir` so Go includes child counter files in coverage. No case
is skipped by default, the race detector stays enabled when compiled in, and
the original 256 MiB production guard remains. Bootstrap failures now record
process resources and expanded pool diagnostics. The legacy receive baseline
waits for the receiver's actual bucket exhaustion before testing timeout, avoiding
a race where sender completion preceded receive accounting.
Unread receiver wrappers also remain referenced until cleanup: otherwise the
upstream Stream Close finalizer can discard the fixture's intended slow-reader
traffic. This is a fixture repair, not a change to production stream ownership.

**NOT VERIFIED:** real-hour/24-hour stability, all carriers under sustained load,
or guaranteed capacity for 1000 real users.

Beta.6 (`8bb95bc`) was not released or installed. Run `38952186871` found
Staticcheck SA4000 in a test expression that called the stateful `dataClass(128)`
twice within one `||` chain. Its remaining work was cancelled as superseded.
Beta.7 expresses each allowance transition as an explicit sequential test step;
production transport code is unchanged from the locally tested beta.6 candidate.
The replacement still requires every original CI and signed-deployment gate.

## Signed beta.7 deployment

All eleven verification jobs and the signed release job passed in
[run 38952846950](https://github.com/firegoood/FullPack/actions/runs/38952846950),
source `9d49231a1c205f13651d11eb3bc3723e489f257d`. Each of the eight explicitly
enabled 1000-held-connection carrier/direction workloads and the mixed TCP/UDP
workload actually ran, with both integrity passes and joined cleanup. These are
loopback connections, not a guarantee of 1000 independent production users.
All seven release archives were independently checked against their hashes,
tag-bound fork signature, patched Go 1.27.2 build records and exact legal members.

The verified binary was installed on Iran and NL with unchanged configuration
hashes, enrollment and durable fencing. Private beta.4 binary rollback was kept;
its saved fencing is forensic only. TR engines and operator FireNode/Xray were
not restarted. Coordinated NL restart began at 20:46:35 UTC and committed TCP
Reverse generation 120 at 20:48:51 UTC, about 136 seconds later. Requests during
that cold route scan were rejected as `no_route`; they are not successful users
or zero-outage deployment. Both peers then exposed negotiated v2 receive
isolation and the enrolled control-priority lane.

The secondary port 8889 passed direct/WAN/after HTTP byte-integrity checks and
its temporary identity was independently queried absent. The thirty-minute fresh
and persistent application tests on port 8888 **FAILED** before completion. The
persistent connection timed out after 1417.92 seconds (701 exchanges); fresh
connections reset after 1395.31 seconds (1374 connections / 2746 exchanges).
Direct Xray prerequisites passed, and both temporary identities were removed
and independently queried absent. Both peers still reported HEALTHY TCP Reverse
generation 120 during the data failure; Iran recorded TCP control timeout at
21:13:11 UTC, then automatically committed WS Reverse generation 121 at
21:13:54 UTC. This candidate is not accepted as a lasting WAN EOF fix. Bounded
failure evidence is in `verification/auto-smux-priority-wan-failure-20261016.json`.
These tests use
owned synthetic Shadowsocks identities and the exact expected HTTP 404 body of
an owned missing-path endpoint; this checks delivery, not throughput or business
endpoint success. Increasing UDP queue-drop counters remain a separate
observation and must not be described as loss-free UDP acceptance.

On 2026-10-17 at 03:38 UTC, both processes still agreed on WS Reverse generation
121 without a restart since deployment. A subsequent direct/WAN port-8889 smoke
test passed with independently removed synthetic credentials. This later success
does not change the failed TCP tests. Iran also exposed `ready=false` for that
live WS session after an extra probe authentication failure. The diagnostic
repair now checks session ownership and publishes under the same manager lock:
an unsuccessful extra handshake cannot overwrite a live enrolled session's
readiness. Real closed-session/reconnect failures remain typed. This repairs
status attribution; it does not establish the cause of the earlier data outage.
The new regression failed against the previous implementation and passed against
the repair in an isolated Linux namespace with Go 1.27.2. The full peerauto
package and ten customization-tool tests passed locally. The diagnostic change
is not installed on production; its full CI remains a separate gate.

At 03:44:50–03:45:35 UTC, simultaneous TCP header summaries on both hosts showed
an established TCP Direct connection to the approved NL port 24485 repeatedly
losing forward progress after its initial exchange. For one captured connection,
Iran retransmitted sequence range 1857:2707 and acknowledged NL data through 3018;
NL continued acknowledging only 1857 and retransmitting 2767:3018. The matching
Iran data/ACK packets were absent from NL's capture. Both captures reported zero
kernel capture drops, and the subsequent qdisc/filter inspection showed no
qdisc drops or egress filter. This is evidence of packet loss outside the
application on that observed path, not proof of DPI or the cause of the earlier
active TCP Reverse failure. Packet bodies/credentials are not published.

Two other unavailable Iran endpoints have a separate local explanation:
WSS Reverse port 24454 and TCPMUX Reverse port 24457 conflict with raw-port
reservations already owned by the TR configuration. Their no-listener/SYN
failures must not be described as evidence of WAN filtering. The existing
TR process, its firewall rules and both immutable endpoint manifests are kept.

The subsequent WS Reverse generation-121 continuity checks both completed on
2026-10-17: one persistent connection passed **900** exact HTTP exchanges over
1800 seconds, and **1800** fresh connections passed **3600** exchanges over a
separate 1800-second interval. Direct prerequisites and post-WAN checks passed;
both owned identities were independently queried absent. Sampled counters showed
no new rejected/interrupted flows or typed data errors, while UDP queue drops
increased and remain separate. This is bounded evidence for the existing WS
route on installed beta.7, not acceptance of the earlier TCP failure, a real-hour
application soak or the pending port migration. The scoped record is
`verification/nl-ws-followup-20261017.json`.

*Last verified against FullPack v1.9.4-beta.7 (source review of bounded Auto receive/transmit isolation, 2026-10-16; historical measurements keep their scope; CI/deployment and soak require separate evidence).*

*Last verified against FullPack v1.9.4-beta.11 (source/documentation compatibility review of local listener preflight, status attribution and explicit backend readiness in the slow-reader fixture, 2026-10-17; original measurements and failed or pending WAN/release/deployment/soak gates retain their separately recorded dates and scope).*
