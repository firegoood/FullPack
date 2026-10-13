# FullPack: bounded Iran/Kharej WAN verification

Date: 2026-10-13. Status: **PARTIAL WAN ACCEPTANCE**.
24-hour soak: **NOT VERIFIED**. These measurements do not establish that every
carrier works on these routes or that the implementation is free of bugs.

## Scope and evidence

The operator supplied one Iran and two Kharej VPS for bounded protocol tests.
The published source baseline is
`3d29613b8c49ebb4e164035f8a3752e9a69d79e6`. Tests reused the locally verified
Linux amd64 FullPack artifact, SHA256
`846b40cc419cdbe79d8f9fbd9d0dfe1e5b000e699863b17a1cc819ab4ca91fe3`.
It is an audit build, not a signed release. No product source, dependency,
enrollment validation, TLS verification or generation fence was changed to
obtain a passing result. `11/` was not used or published.

Raw evidence and the external harness are retained outside Git at
`E:/P_OX/fullpack-waticmp-audit/wan-20261013`. Server addresses and credentials
are omitted from this report. The SSH password was supplied to a masked prompt
and held in process memory, not written to a configuration or report. Host keys
were pinned. The temporary localhost SSH bridge and its capability tokens were
removed after testing.

Each machine ran Ubuntu 24.04 amd64. Iran and Kharej2 had eth0 MTU1500; Kharej1
had MTU1400 and existing upstream PCK/L3 services. Tests used a separate private
directory, unused carrier ports, loopback application endpoints, random pair
secrets and fresh enrollment/state paths. Auto was explicitly enabled only in
these disposable configurations. Existing services, installations, routes,
MTUs and default-off policy were preserved.

The Go test binary used for cover verification was built with Go 1.27.2.
Installer-pinned Naive 154.0.8037.49-4, sing-box 1.14.2 and Xray 26.3.27 archives
were checksum-verified. Xray reports a Go 1.26.1 build; the patched FullPack
toolchain must not be presented as evidence that all third-party helpers were
built with that toolchain or passed a new vulnerability scan.

## Acceptance meaning

The matrix covered all 18 stream carriers and all six L3 carriers in Reverse
and Direct initiation on each pair: 96 distinct cases, plus selected retries.
The three matrix JSONL files contain 112 valid records including four grouped
harness/setup-error records; those four are excluded from carrier results.

**P** requires the selected carrier to be active on both peers at the same
generation, actual TCP/UDP integrity, a fresh isolated integrity probe, and
unchanged active route/generation after that probe. Traffic consisted of eight
64-byte interactive TCP echoes, four concurrent 32 KiB TCP exchanges and UDP
payloads of 1, 64, 1400 and 8000 bytes. L3 additionally required real TUN traffic
and bidirectional 9000-byte IP pings. This is bounded functional traffic, not a
maximum-bandwidth or loaded-latency benchmark.

**U** means the carrier did not become an agreed usable route within the
bounded test. It is not a code-defect diagnosis and not a pass. Some endpoints
authenticated temporarily without completing health probes and activation.
Generic `peer identity/version/policy rejected` status can also represent an
authentication exchange timeout; its underlying cause remains unresolved for
these cases.

**B** means a local prerequisite was blocked by the existing raw-tunnel owner
or port reservation. The ownership guard correctly refused to take over
Kharej1's existing upstream packet sockets. Those cases cannot establish WAN
carrier availability. No existing tunnel was stopped and no guard was bypassed.

Where a retry passed, the table records that successful configuration; the
initial failed attempt remains in the raw evidence.

## Stream results

| Carrier | Kharej1 Reverse | Kharej1 Direct | Kharej2 Reverse | Kharej2 Direct |
| --- | --- | --- | --- | --- |
| Wat ICMP | P | P | U | U |
| Stealth | P | P | P | U |
| TCP | P | P | P | U |
| TCPMUX | P | P | P | U |
| WS | P | U | P | U |
| WSMUX | P | U | P | U |
| WSS | P | P | P | U |
| WSSMUX | P | P | P | U |
| KCP | P | P | U | U |
| UDP | P | P | U | U |
| xDi | P | P | U | U |
| PCK | B | B | P | U |
| QUIC | U | U | U | U |
| SPOOF, UDP profile | P | P | U | U |
| SNI | B | B | P | U |
| Naive/HTTP2 | P | P | P | U |
| XHTTP/TLS | P | P | P | U |
| REALITY/Vision | P | P | P | U |

Successful counts are 15/18, 13/18, 12/18 and 0/18 respectively, with the B
cases distinguished from U. Direct here means Iran initiates the enrolled
carrier; Iran still owns the application entry port and switching authority.

Native Wat tests used reverse Echo direction and a 512-byte payload limit in
both initiation modes. Normal Echo WAN behavior remains NOT VERIFIED. SPOOF
used the UDP profile with each machine's real source address. These results
do not verify forged-source forwarding or the other five SPOOF profiles on WAN;
their namespace evidence remains separate.

## L3 results

| Datagram carrier | Kharej1 Reverse | Kharej1 Direct | Kharej2 Reverse | Kharej2 Direct |
| --- | --- | --- | --- | --- |
| UDP | P | P | U | U |
| QUIC | U | U | U | U |
| xDi | P | P | U | U |
| PCK | B | B | P | U |
| SPOOF, UDP profile | P | P | U | U |
| SNI | B | B | P | U |

Kharej2's Reverse PCK and SNI carried the real 9000-MTU TUN exchanges, optional
enrolled TCP/UDP forwards and independent datagram probes. This is not evidence
of automatic L3 failure recovery on WAN; that gate remains NOT VERIFIED.

## Diagnoses and retries

ICMP baseline measured 20/20 responses on Kharej1, approximately 83 ms average,
and 6/20 on Kharej2, approximately 61 ms for received packets. Additional
600/1200/1372-byte ICMP payloads passed on Kharej1 and received no responses on
Kharej2. This is ICMP behavior during this window, not a general TCP-loss rate.

An independent, authenticated and bounded UDP echo test on port24699 sent three
random datagrams per size in each direction. Kharej1 returned all three at
96/512/1024/1199/1200/1280/1372/1472 bytes. Kharej2 returned all three at 96 and
512 bytes and none at 1024 bytes or larger. A network size filter/blackhole is
consistent with these measurements, but the exact cutoff and device are not
identified. QUIC's mandatory 1200-byte Initial cannot qualify on the measured
Kharej2 path. QUIC also failed on Kharej1; its cause remains unresolved. Sole
Reverse QUIC retries on UDP443 failed on both pairs.

The original Microsoft REALITY cover failed on all three machines despite
ordinary TLS reachability. We ran the repository's exact opt-in
`TestRealityAutomaticCoverScanWithInstalledXray` on each server: actual
authenticated Chrome/Vision traffic failed for Microsoft; Apple passed on all
three; Bing passed on the two Kharej machines and failed from Iran. Ordinary
TLS success was never counted as a passing REALITY cover.

Using the verified Apple cover, REALITY passed Reverse on both pairs at
Iran TCP443 and Direct on Kharej1 at TCP8443. Kharej1's existing TCP443 listener
was left untouched. REALITY Direct on Kharej2 TCP443 failed. Stealth Direct
on that same available port also failed. Direct WS/WSMUX on Kharej1 were retried
on fresh ports and still failed to qualify. Direct PCK/SNI on Kharej2 were
retried on fresh ports after initial port-reuse conflicts; they still failed.
The original HTTPS test harness's public-key field parsing and XHTTP DNS-name
mistakes were corrected before those cases were rerun; those setup errors are
not counted as product protocol failures.

## Controlled Auto recovery

Each pair enrolled Reverse TCP and WSS on dedicated ports. Both peers initially
agreed on TCP generation3. A temporary peer-specific OUTPUT rule dropped only
the experimental TCP carrier port. Auto selected WSS generation4 on both peers
in 24.344 s for Kharej1 and 24.250 s for Kharej2, measured from fault insertion
to observation of the agreed commit. These are single observations, not an SLA.

The Iran application listener inode stayed unchanged. TCP/UDP traffic passed
while faulted and throughout a further three-minute observation after the rule
was removed. Each pair had 11 successful traffic samples across the test;
healthy WSS did not automatically switch back to TCP. This does not claim that
old TCP sessions migrated without interruption or that every carrier pair has
been tested for recovery.

## Teardown and remaining gates

The external supervisor allowed five seconds after TERM before forcing KILL.
Some interrupted raw/L3 teardown left 14 scoped firewall rules in total. This
short supervisor grace is not evidence that native graceful shutdown always
finishes within five seconds. After confirming no audit processes remained,
only the exact audit-added rules were removed; original rules were preserved.

Final evidence (`*-final-cleanup.json`) confirms the private audit directories,
test keys/configurations, helpers, processes, forwarding listeners and TUNs
were removed from all three machines. Original listener PIDs and firewall
rule bodies match the initial inventory. Default routes and Kharej1's existing
`bp0` route remain present. No permanent service, sysctl, MTU or package change
was made. Empty kernel firewall tables can remain loaded after temporary rules;
their ACCEPT policies do not add a filtering rule.

Full acceptance remains pending for the unavailable cases, actual L3 automatic
failure recovery on WAN, normal Wat Echo, forged-source/profile combinations,
arbitrary NAT/firewall behavior, production load, current helper security and
a 24-hour soak. During WAN testing, no signed release, Git push or permanent
deployment was performed. These bounded results supplement, not replace, the local
race/security/namespace evidence in `waticmp-verification.md`.
