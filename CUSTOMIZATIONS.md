# FullPack customization ledger

## Approved preferences

| Date | Preference | Scope |
| --- | --- | --- |
| 2026-10-12 | Project name: **FullPack** (two `l` characters) | CLI, web panel/PWA, Telegram messages, current documentation, installation and release packaging |
| 2026-10-12 | Preserve customizations through upstream updates | Repository instructions, versioned reapply manifest/tool and verification |
| 2026-10-12 | Add CLI tunnel methods missing from the web panel | Reverse Naive/HTTP2, XHTTP/TLS, REALITY/Vision; direct IP and SNI spoofing |
| 2026-10-13 | Execute audited Wat ICMP / peer coordination / intelligent Auto prompt | Opt-in native `waticmp`, preservation of xDi/manual/fallback defaults, immutable local WatTunnel reference, licensed source attribution and evidence gates |
| 2026-10-13 | Publish to the user-created `firegoood/FullPack` repository | Git destination, application source/update URLs, installer, documentation, panel links and persistent reapply guards |
| 2026-10-13 | Never upload the entire `11/` directory | Root Git ignore rule, persistent update guard and source/release publication exclusion; the local reference remains read-only |
| 2026-10-13 | Audit the additions and complete Auto for Reverse **and Direct/L3** | Shared carrier registry, explicit connection direction, encrypted legacy adapters, managed HTTPS identities and a stable TUN with authenticated datagram forwarding; verification status is recorded separately |
| 2026-10-13 | Test the supplied Iran and two Kharej servers | Bounded protocol and Auto recovery tests; preserve existing services and security checks, record partial WAN evidence and clean up audit resources |
| 2026-10-14 | Review Auto availability, WAN failures, shutdown and capacity criticisms | Reproduce consecutive carrier failures, separate bounded emergency recovery from optional changes, classify diagnostics, preserve cleanup checks and record measured limits |
| 2026-10-14 | Make one-command installation work by publishing a FullPack release | Independent fork signing key, signed immutable-tag online/offline installation, patched release toolchain and unchanged CI acceptance gates |
| 2026-10-14 | Inspect occupied 2052/8880 ports on Iran/Kharej2 and skipped HTTPS CLI tests | Preserve unrelated port owners, reject occupied Auto/Wat user listeners before CLI host changes, retain explicit-address and helper prerequisite checks |
| 2026-10-14 | Inspect the operator's offline Auto and add approved Reverse/Direct endpoints | Re-enroll a fresh four-candidate pair, preserve retired configuration/fencing and verify TCP/WSS passes separately from Direct/Wat failures |
| 2026-10-14 | Simplify Auto setup and publish an updated release | Shared CLI/web Quick Auto builder, explicit endpoint review, private peer link, advanced setup preservation and signed v1.9.2 source/tag/assets |
| 2026-10-15 | Include every protocol in easy Auto and replenish selected routes | Explicit complete stream/L3 pools, four measured stable paths, authenticated reserve replenishment at two healthy paths, unchanged generation and bounded recovery policy |
| 2026-10-15 | Diagnose EOF with 500/800 synthetic users | Opt-in loopback load audit with fresh Xray identities, matched direct/Auto workloads, connection-per-user accounting, bounded resources and UDP/TCP shared-capacity evidence; no production capacity/enrollment change |
| 2026-10-15 | Fix the diagnosed Auto admission and misleading-health problems | Bound UDP occupancy, queue short TCP bursts with a bounded deadline, preserve enrollment totals and generation fencing, and expose capacity/backend data warnings in CLI/web independently of carrier health |
| 2026-10-15 | Install the repaired build and test 700–1000 simulated users | Identifiable verified fork build with rollback, coordinated tunnel/panel restarts, preserved manifests/fencing and scoped synthetic/WAN traffic checks; do not claim simultaneous-user capacity above enrolled limits |
| 2026-10-15 | Remove compulsory Auto connection ceilings, as in manual forwarding | Explicit unlimited new enrollment, optional operator cap without a 192 maximum, dynamic active-flow accounting, independent bounded handshake work and carrier stream-policy parity; preserve legacy enrollment/fencing and measure actual concurrency |
| 2026-10-16 | Investigate and fix NL EOF after hours of traffic | Independent authenticated control liveness during slow reserve scans, live-route status distinct from prepared durable epochs, bounded joined cancellation and retained WAN incident evidence |
| 2026-10-16 | Review and update smux if a better version exists | Official latest tag is already v1.5.57; review upstream changes, preserve manual compatibility and test Auto's authenticated v2 flow isolation against original smux |
| 2026-10-17 | Change conflicting NL ports and prevent occupied ports in new tunnels | Coordinated fresh enrollment preserving retired configs/fencing; shared local socket/config checks in CLI/web, bounded Quick Auto free-range suggestions, visible endpoint review and receiver-local Direct validation |

## Listener preflight and free Auto ports (2026-10-17)

The user authorized replacing conflicting NL listener ports and choosing free
defaults for new tunnels. Preserve other services and TR. Existing shared Auto
endpoints change only through coordinated fresh enrollment with retired
configuration/fencing retained; never silently patch one peer's manifest.
New CLI/web creation checks all local Auto listeners, raw TCP reservations and
private HTTPS helper ports, including stopped configured tunnels. Foreign
backends and the other side's listeners are not local port conflicts. ICMP role
markers do not reserve TCP/UDP sockets. Quick Auto suggests a bounded free local
range, shows exact endpoints and rechecks on save; Kharej validates Direct ports
on trusted-link import. A suggestion is neither a held reservation nor proof of
remote availability. Keep manual choices and ignore stale suggestion responses.
Do not stop unrelated owners, scan remote addresses or rewrite an approved link
when a receiver detects a conflict.
Beta.8 run `39001411364` failed the VERSION/app identity check and was not
released or installed. Its superseding beta.9 must pass all original gates.
The earlier `46b8c53` diagnostic CI run `39000287122` failed the existing
Direct-spread fixture's first-listener startup race. Preserve its failure and
the bounded initial-ECONNREFUSED wait, exact eight-flow spread assertions and
strict treatment of failures after a connection is accepted.
Beta.9 run `39001491738` failed document-version stamps and the missing API
contract sample for `suggestAutoPorts`; no beta.9 binary was released/installed.
Preserve stream/L3 API contract samples and scoped documentation reviews in
beta.10. Never update old measurement dates or turn pending gates into passes
when recording compatibility with the new listener preflight.
Run the existing document/panel contract tests early in the normal CI job,
before the full suite, while retaining every normal/race/coverage/raw/load gate.

## Auto stream isolation and smux review (2026-10-16)

The dependency is already the newest official tag, `github.com/xtaci/smux
v1.5.57` (`3b4ec04d359934256b3adea7133374e3c93a0622`). The only subsequent
master commit `ae956bb8d67bab37a312869b1d38ee3f52a7397a` changes README files;
it contains no implementation update to import. Library release numbers and
smux wire protocol versions 1/2 are separate concepts.

Preserve the Auto-only licensed adaptation in `internal/tunnel/peerauto/flowmux`:
negotiate standard v2 inside the existing Noise PSK transcript or TLS-bound WSS
proof, retain legacy v1 fallback and unchanged manual transport behavior, and
enable receive isolation only after complete production enrollment/incarnation
checks. Before enrollment keep the finite session receive bucket. Afterward,
finite per-stream windows replace the shared bucket that can block all control
behind slow applications. Reject excess receive credit, retain bounded transient
handshakes and never replace unlimited admission with a hidden stream-count cap
or an enormous shared buffer. Native Wat/QUIC flow control remains separate.
Keep the original MIT license/source hashes in binaries and release archives,
old/new negotiation and upstream wire compatibility tests, slow-reader/control
regressions in both directions and honest local/WAN/soak limits. Existing pair
manifests, crypto domains, authorization and fencing state are not rewritten.
`docs/auto-control-liveness.md` records beta.4's repeated WAN timeouts after its
short successful delivery tests; never report that deployment as a lasting fix.

Receive isolation alone does not prevent control or new-stream handshakes
waiting behind queued bulk writers. Preserve a bounded eight-frame lane for
the one enrolled control stream and a bounded 32-frame lane for SYN/initial
data-handshake writes. Only an enrolled v2 stream's first 256 small bytes have
handshake priority; a larger first write retires that allowance. Keep one
ordinary queued frame after at most sixteen priority frames, standard receive
credit and framing, legacy/manual behavior and native Wat/QUIC. Priority cannot
preempt bytes already in the underlying connection or guarantee WAN bandwidth.

Coalesce tiny received frames into bounded 4 KiB allocations so receive-ring
metadata scales with buffered payload rather than a one-byte frame count.
Preserve byte order, partial-read offsets, allocator ownership and receive
credit. Keep regressions for tiny frames and partial heads. Slow-reader fixtures
must retain their unread Stream wrappers so their Close finalizers cannot turn
the intended blocked-reader workload into discarded traffic.

Beta.5 (`7a115437372a7de8d1a5104c6ec42ca6453b6863`) was not released or
installed: CI `38942638366` failed the shared-process pool recovery checks in
native/race runs and subsequent race bootstraps. Preserve that failed evidence.
Each of the six new slow-reader cases must run in a fresh instance of the same
normal/race/coverage test executable, with an exact selected workload marker.
Forward Go's coverage directory so its child counters remain in the profile.
Keep resource diagnostics and all cases; do not raise the 256 MiB production
pressure threshold or convert pre-bootstrap pressure skips into successful work.
Beta.6 run `38952186871` found SA4000 in a stateful test expression and was
cancelled as superseded without release/installation. Keep explicit sequential
allowance checks and retain the complete gate on its beta.7 replacement.

Beta.7 source `9d49231a1c205f13651d11eb3bc3723e489f257d` passed all eleven
verification jobs plus signing in run `38952846950`. All nine explicit held-load
workloads actually ran. Seven signed architecture archives were independently
verified before installation on Iran/NL. Keep the unchanged enrollment/config,
durable fencing and private beta.4 rollback; TR and operator backend processes
were not restarted. Subsequent WAN continuity, UDP drops and remaining soak
limits are recorded in `docs/smux-flow-control.md`; local CI is not WAN acceptance.
The beta.7 thirty-minute fresh/persistent WAN checks failed after about 23 minutes
while TCP still reported HEALTHY. Preserve their failed evidence in
`docs/verification/auto-smux-priority-wan-failure-20261016.json`; subsequent WS
recovery does not establish a lasting EOF fix.

On 2026-10-17, the continuing NL investigation found a live WS session with
`ready=false` after an extra isolated handshake failed. Keep attempt diagnostics
scoped: a failed extra accept/authentication must not overwrite the readiness of
an existing enrolled session. Check ownership and publish under Manager.mu;
retain typed failures when no live session remains and keep private debug logs
for failed attempts. This status repair does not establish an EOF/network fix.

## Auto control liveness repair approved on 2026-10-16

Slow isolated reserve probes must never starve production control heartbeats.
Keep one switching authority and candidate-local transaction serialization;
the independent joined heartbeat loop may only check liveness and fence an
authenticated generation disagreement. Busy streams are skipped without
queuing requests or blocking heartbeats for other candidates. STATUS reports
the currently live committed route; STATE_SYNC/RECOVER retain the durable
high-water fencing epoch, including an uncommitted preparation. Never restore
an active route from disk, erase generations, relax authentication or remove
the five-second/twelve-commit recovery gate to conceal repeated EOF. Preserve
the incident's bounded evidence and remaining soak gates in
`docs/auto-control-liveness.md`; old short load tests remain historical.
Keep all eight 1000-held-connection carrier/direction cases and the mixed
TCP/UDP case in the release gate. Run each in a fresh race-test process within
the existing twelve-minute total audit bound; log fixture RSS/heap/goroutines.
Do not remove a case, raise production pressure limits or treat a pressure
skip before bootstrap as a passing application workload.

## Unlimited Auto connection policy approved on 2026-10-15

The operator explicitly rejected a compulsory fixed connection ceiling. This
supersedes the earlier repair's 64/192 restriction for newly enrolled policy;
old measurements remain historical. New Quick and advanced CLI/web setup use
`connection_limit = 0`, with positive optional caps and no artificial maximum.
The optional field distinguishes this deliberate choice from legacy omitted
`max_connections`, whose historical zero/default hash still means 64. Changing
an installed pair's shared policy requires coordinated enrollment, not a local
edit, ignored manifest mismatch or deleted generation history.

Active-flow counts must not allocate channels/arrays sized by a configured
maximum. Established connections do not retain transient handshake permits.
New enrolled policies prefetch a bounded 128 streams on the active carrier
(four on standby), awakened by commit. This is not an application-count cap
or four-new-streams-per-WAN-round-trip bottleneck. Keep transport reserve credit
consistent with that cache, and retain legacy four-stream prefetch semantics.
Bound unfinished handshakes, caches, datagrams and per-flow byte windows;
backpressure and actual resource/OS exhaustion remain possible. Keep optional
positive caps' UDP share/FIFO and typed diagnostics, and do not classify an
unlimited zero as saturated. Wat Auto must honor the enrolled stream override
without its legacy 256 stream cap; retain manual native defaults and all nonce,
replay, tombstone and byte-window protections. Unlimited QUIC retains the manual
engine's transport credit window (65536 streams), not unsafe 2^60 credit: sparse
stream IDs could otherwise allocate huge numbers before application auth. This
inherited transport constraint must remain visible in the evidence and is not
a new Auto admission cap. Pressure skips must retain production
and must not turn into carrier failures. Record actual concurrency and remaining
WAN/soak gates in `docs/auto-unlimited-connections.md`.
Auto Wat retirement must compact sequential identities into a permanent prefix
plus at most 4096 out-of-order identities, retaining lifetime non-resurrection
without filling a churn-sized tombstone table. Manual native behavior remains
unchanged. OS file/socket exhaustion must be classified as `local_resource`;
forwarded listeners retry with bounded backoff and keep established flows.

The signed `v1.9.4-beta.2` source
`42164d93523771853ec98e3af90008e1ed0723a2` passed all eleven CI verification jobs
and exact-tag public installation in run `38806904756`. All three supplied hosts
now run that verified binary. Fresh `TR-unlimited` / `NL-unlimited` pairs retain
the same 38 approved candidates and existing forwarded ports; original pair
configurations/fencing and private binary rollback remain intact. Preserve
`docs/verification/auto-unlimited-local-20261016.json` and
`docs/verification/auto-unlimited-wan-20261016.json`: 2000 held local connections
with 1000 Xray identities passed, and installed TCP Reverse/Direct WAN each
passed 1000 held connections/accounts with sampled concurrency 1000, no new
rejections, unchanged generations and independently verified account cleanup.
Setup concurrency was 128; this is not simultaneous completion of 1000 new
handshakes. The 4000-connection lab OOM and the direct-Xray 1000-start timeout
remain failures, and 24-hour/production acceptance remains unverified. Never
replace these bounded statements with an unconditional capacity guarantee.

## Controlled admission deployment approved on 2026-10-15

The operator explicitly requested installation and further load tests. Prepare
the identifiable `v1.9.4-beta.1` fork prerelease with the existing signature and
full CI gates, then retain a private binary rollback on each enrolled server.
Keep manifests, keys and evolving generation/history intact through upgrade and
rollback; a saved fencing snapshot is forensic evidence, not state to restore.
Restart only the relevant FullPack services, leave FireNode/Xray running, verify
data and distinguish the 700/800/1000-identity test from admitted connections.
Extend only the diagnostic identity ceiling to 1000; production remains at its
actual enrolled ceiling. Preserve previous raw evidence and record deployment
and WAN outcomes separately in `docs/auto-admission-deployment.md`.
Prerelease version extraction must retain the full suffix in Make/CI. The
release must compare the complete tag to the binary's declared version and
verify public installation using that exact tag, including for prereleases;
keep `TestReleaseToolingKeepsPrereleaseIdentity` and the full CI dependency.
The L3 handover test must budget its shared race/stress instrumentation RSS
explicitly (2048 MiB) and report measured RSS/CPU on bootstrap failure. Keep
the production 256 MiB default and all real pressure/CPU/admission guards;
this fixture allowance is not a production-capacity increase or a skipped gate.

The signed `v1.9.4-beta.1` source `4e0bc91fa5b332662da9258ac2d982ad63300629`
passed all eleven CI verification jobs and exact-tag public installation in
run `38761783514`, then was installed on all three supplied hosts. Preserve
`docs/verification/auto-admission-wan-20261015.json`: each stream pair passed
700/800/1000-identity phases and 2880 short WAN transactions with concurrency
16. A separate 1000-attempt held burst served only 64 TR / 61 NL synthetic
connections; the enrolled total remained 64, including other live traffic.
These rejected clients are not successful users. Runtime test accounts and
fixtures were independently verified removed; operator config hashes and
FireNode/Xray processes were retained. NL cold start exceeded the initial
180-second observer and activated after about 215 seconds; preserve that
timing failure separately from its subsequent data success.
The user's manual-BackPack comparison exposes an unresolved Auto capacity
gap: manual forwarding permits 0 = unlimited, while Auto uses its own 64/192
policy. Queue/UDP repairs do not establish 700–1000 simultaneous-user support.
Retain this distinction and the pending sustained-bandwidth/24-hour gates;
do not silently raise existing enrollment or label overflow as acceptance.

## Auto admission repair approved on 2026-10-15

- Keep the original total connection ceiling (default 64, maximum 192) and
  existing enrollment hashes. A larger capacity requires a separately verified
  design and fresh enrollment; this repair never changes an operator's manifest.
- UDP may occupy at most one quarter of the total, rounded down (minimum one
  when the total exceeds one). This retains TCP capacity despite UDP's existing
  60-second idle retention. A one-slot enrollment reserves that slot for TCP.
  Active UDP is not evicted to conceal overload. TCP can use the entire total
  when UDP is absent; overload of either protocol remains explicitly bounded.
- TCP uses a FIFO of at most twice the enrolled total, with a two-second wait.
  Reserve queue admission before launching a worker. Timeout, cancellation and
  close must free exactly one ticket and preserve active sessions. A waiter
  cannot extend an old generation's authorization after a carrier commit.
- Export current live/UDP/queued counts, limits and bounded typed error counters.
  Report recent failures for 30 seconds independently of carrier status. Do not
  switch a healthy carrier or restart the engine because of local capacity or
  backend failure. An available slot is not proof of application delivery.
- Preserve the before-fix raw load measurements and distinguish them from
  after-fix tests. Record validation and remaining limits in
  `docs/auto-admission-repair.md`; no release/deployment/24-hour claim is implied.

## Synthetic-user load diagnosis approved on 2026-10-15

- Preserve the opt-in `tools/auto-load-audit` fixture and its reproducible results
  in `docs/auto-synthetic-load-audit.md`. All addresses are generated loopback
  listeners. No production accounts, certificates, endpoints or configuration
  are loaded by the test program.
- The explicit receiver-local Xray binary is an isolated diagnostic fixture,
  not an Auto helper installation or runtime enrollment. Record its version
  and SHA-256; never copy the operator's credentials into a fixture/report.
- Distinguish synthetic identities, connections per identity, attempted and
  authenticated/integrity-verified connections. Overflow is measured without
  bypassing the existing 64-default/192-maximum admission policy or its hash.
- Compare the same valid Shadowsocks traffic directly with Xray and through
  Auto, simultaneous bursts with paced admission, and UDP occupancy with TCP
  recovery. Keep active-generation, cleanup and accepted-stream integrity checks.
- A private Linux network namespace and resource limits keep the load separate
  from production traffic. Short loopback results are not WAN/24-hour or
  500/800-real-user production acceptance. Production changes require their
  own verified implementation and coordinated enrollment, not a test flag.
- The 2026-10-15 measured run completed sixteen 500/800-identity cases: direct
  Xray passed up to 1600 simultaneous connections, Auto bursts admitted exactly
  64/192, and externally paced 32-at-a-time admission passed all 1000/1600.
  Sixty-four valid UDP associations denied new TCP until approximately 60 seconds
  after UDP client closure, while status remained HEALTHY. Preserve the raw
  evidence and its short-loopback limits; no production capacity fix is implied.

## Full protocol pool and replenishment approved on 2026-10-15

The operator asked the easy setup to evaluate every protocol/direction, select
four stable routes and rescan/replenish when only two remain healthy. This
supersedes the four-endpoint Quick preset below, without changing existing
four-candidate pairs or granting runtime authority to invent new endpoints.

- A fresh explicitly approved stream pool includes the eighteen registry
  carriers in both initiation directions, plus both Wat Echo directions (38
  candidates). A separate quick L3 mode includes all six datagrams in both
  directions (12); never route IP through reliable stream carriers.
- At most four paths are selected from a bounded pool of at most 48. Require
  three separated successful full integrity probes, retain the healthy active
  route during reserve selection and use the authenticated peer acknowledgment
  and private durable state without deleting generation/switch history.
- Reaching two healthy selected paths triggers a reserve sweep. Genuine data
  failures/disconnection reduce availability; budget/pressure skips do not.
  Repeated shortfalls use bounded retry/backoff and report insufficient routes;
  the network may provide fewer than three or four usable paths.
- Generated HTTPS credentials are new pair-scoped identities derived with
  purpose-separated HKDF from the fresh shared secret. They are computable by
  both enrolled peers, who already share switching/authentication authority;
  they are not independent secrets from the counterpart. Existing operator
  keys are untouched. Public CA/REALITY keys must match the local material,
  private values/paths never enter peer exports, TLS verification stays on and
  REALITY still needs the actual authenticated cover probe. Missing pinned
  helper executables/raw privileges are SKIPPED, never silently downloaded or
  borrowed from unrelated applications.
- Pool TLS uses Chrome-compatible P-256 certificates and the standard Go
  RFC 6979 signer for reproducible public trust. Do not reintroduce the
  Ed25519 certificate handshake mismatch or disable certificate verification.
- Preserve registry parity, CLI/web pool counts and rescan actions, bounded
  handshake/control polling, existing manual/observe/lock policy, connection
  bounds and joined cleanup. No arbitrary upstream binary or signing change.
- Signed v1.9.3 source `057f69d2a9d418f95196c1a9d54759e348e945d4` passed all
  eleven full-CI gates and public signed installation in run `38679834616`.
  Strict four-to-two-to-four stream faults, real L3/TUN replenishment, generated
  HTTPS in both directions, race and 58.2% coverage passed. Preserve the bounded
  evidence in `docs/fullpack-release-verification.md`; this does not imply
  WAN-wide, real-hour/24-hour or 1000-user acceptance.

## Historical four-endpoint Quick implementation (2026-10-14)

- Preserve `docs/auto-live-setup-review.md`: the original pair had only Wat
  Normal; adding arbitrary protocols was never implicit runtime enrollment.
  The authorized fresh mixed pair preserved old fencing and reached HEALTHY
  on TCP, with a passing WSS probe. Direct Noise/Wat WAN outcomes remain distinct
  from passing isolated Linux Direct tests and from application acceptance.
- Preserve the shared `QuickAutoCarriers`/`expandQuickAuto` builder. CLI and web
  quick setup review four built-in TCP/WSS/Noise Reverse and Noise Direct
  endpoints, generate fresh identities/secrets and use the same engine/peer-link
  validators. Quick mode is explicitly selected Auto, with failover enabled;
  advanced observe-only, all carriers/helpers/Wat and L3 remain available.
- Keep quick setup restricted to new Iran/server stream pairs. Never use it to
  overwrite an existing manifest or erase durable state. Preserve bounded
  connections/probes, cooldown/dwell/recovery policy, TLS/authenticated carrier
  binding and receiver-local import forms. Do not download helpers implicitly.
- Static config checks recognize native/Auto port-zero role markers and optional
  Auto L3 maps without relaxing legacy socket/user-port checks. The CLI still
  applies the full engine enrollment verdict after those syntax checks.
- The operator requested the simplified setup in an updated release, authorizing
  v1.9.2 commit/push/tag/signed assets to `firegoood/FullPack`. The existing fork
  signing key and all full-CI release gates remain required. The local `11/`
  reference and server credentials/configs are never included in publication.
- On installer updates, refresh only active FullPack panel/monitor processes to
  load the new embedded setup form; leave tunnel sessions/generation ownership
  running. Existing source/release signature checks still precede this refresh.
- The live pair later exposed a real uptime regression: 256 one-hour probe
  replay entries exhausted under the ten-second health cadence, causing false
  carrier failures and Iran NO_ROUTE while Kharej retained its agreed route.
  Preserve the full one-hour replay protection with the bounded 4096-entry
  cache; exhausted budgets are typed SKIPPED and cannot cancel production or
  increment data failures/quarantine. Preserve the paced-hour, hard-cap,
  replay/expiry and subsequent-control regression tests. A clock-model test is
  distinct from real-hour/24-hour acceptance.
- v1.9.2 was published at immutable source `28bd07f6445b958107fc8f310c8a40de20185104`
  after all eleven CI verification jobs and the public signed-install check
  passed in run `38626831074`. Seven public archives, exact legal contents,
  tag-bound signature and binary Go 1.27.2 build information were independently
  verified. Keep these release results separate from the remaining WAN/24-hour/
  1000-user limits in `docs/fullpack-release-verification.md`.

## Peer setup port ownership review on 2026-10-14

- Peer CLI setup uses the panel's forwarded-listener refusal before network
  optimization, config writes or service creation. An occupied listener can
  belong to another application even when FullPack's tunnel list is empty.
- Missing approved peer addresses must be rejected during spec validation;
  preserve those enrollment checks and the same-port foreign backend allowance.
- HTTPS Connection Test requires the exact isolated pinned helper paths on both
  peers. Absence is SKIPPED with a local reason; never relabel it a network
  failure or borrow an unrelated proxy's executable/configuration implicitly.
- Preserve unrelated server services. The supplied servers' occupied ports were
  traced to existing anonymous-tunnel/firenode services, not orphan FullPack
  controllers. Evidence and bounded regression results are recorded in
  `docs/peer-setup-port-review.md`.
- The operator subsequently authorized stopping `anonymous-tunnel-ir01` on
  Iran to release 2052/8880. That one unit was stopped and its autostart
  disabled, retaining its config/unit files. Both TCP/UDP ports were verified
  free; Kharej2's firenode backend stayed active. This is a scoped operational
  authorization, not permission to stop other port owners in future setups.
- A follow-up inventory found that anonymous's enabled watchdog timer had
  restarted the stopped unit. The associated timer/watch service were then
  stopped and timer autostart disabled before stopping the main unit again.
  The initial port-release evidence was temporary; the updated report retains
  that limitation. Uninstall files were left for the operator's requested command.

## Approved Wat ICMP and peer Auto implementation

- Preserve `docs/waticmp-dual-server-design.md`, `docs/waticmp-verification.md`,
  the native engine's MIT LICENSE/ORIGIN.json and its security/resource repairs
  through upstream source merges. Do not rename xDi or replace its wire.
- The entire `11/` directory is excluded from Git/publication, formatting,
  clean builds and releases. `11/WatTunnel` remains the authoritative read-only
  reference. Native source is adapted into
  `internal/tunnel/waticmp`, without a local module replacement or helper process.
- Auto is disabled by default. Preserve typed peer authentication, explicit
  enrollment, stable Iran port ownership, fresh cryptographic incarnations,
  persisted generation fencing and exclusive per-tunnel controller ownership.
- This ledger records authorized scope; the verification ledger records which
  parts actually pass. Do not turn pending acceptance into a production claim.
- Native normal/reverse stream forwarding uses the attributed Wat engine's
  actual Session/Stream, not xDi/KCP, smux or a helper executable. Preserve fresh
  bilateral X25519/PSK keys, directional nonces, bounded packet/stream/window
  resources, ACK arithmetic, stream deadlines, FIN drain and joined shutdown.
  Preserve authenticated DF/PMTU checks and the delayed-path repairs: initial
  RTT timeout, bounded retired-packet timing, and full paced-header accounting.
- Auto owns stable Iran TCP/UDP ports and is the sole switching authority for
  that tunnel. Preserve the strict typed mutually authenticated protocol,
  explicit candidate/backend enrollment, fresh isolated probe sessions, durable
  fsync/rename generations, exclusive no-follow ownership locks, uncertain-
  commit fencing and recovery through only the pre-enrolled directions. Peer failures remain
  independent; a slow/offline application backend is not a carrier failure.
- Preserve the original authenticated Wat ICMP, Noise/stealth, QUIC, WSS and
  WSSMUX adapters. The approved Auto expansion covers every Reverse carrier,
  explicit Direct initiation and the six existing L3 datagram carriers. Legacy
  TCP/WebSocket/KCP/raw wrappers require Auto's authenticated encryption; never
  enable plaintext fallback. Wat's native stream engine is not an L3 carrier.
- The carrier registry must drive engine validation, CLI and web choices. The
  pair explicitly enrolls at most four candidates; Iran remains the sole
  authority even when Iran initiates a Direct connection. L3 owns its TUN once,
  keeps IP traffic outside reliable stream retransmission, authenticates fresh
  directional datagrams, bounds replay/reassembly and rejects retired generations.
- Managed Auto candidates keep TLS/REALITY verification, restricted loopback
  helper targets, role-specific executables and identities. Enrollment links
  omit local paths, interfaces and private keys. Preserve existing fencing,
  probe budgets and secret-preserving edit restrictions through source updates.
- Preserve default-off Auto and default-off severe-degradation switching,
  observe-only, manual lock/disable/test actions, pressure SKIPPED results,
  global/per-pair probe budgets, jitter, debounce, quarantine, cooldown, dwell
  and persisted hourly switch limits. Never change a healthy path for a small
  score advantage. Old TCP may drain or be interrupted; no migration claim.
- Preserve terminal and web setup/edit/status/action parity, secret-preserving
  edits, version-3 credential links and trusted Kharej enrollment import.
  Native forms hide incompatible legacy tuning; switching back restores the
  managed HTTPS forms. An imported client is reported as imported/waiting,
  not as an already-connected peer requiring another enrollment handoff.
- Connection Test option 0 includes actual normal/reverse native data tests
  when both explicit IPs are supplied; report missing prerequisites as SKIPPED.
  Its existing TCP/UDP test coordinator is distinct from Auto's pre-enrolled
  bootstrap and from TCP route Link Test diagnostics.
- Preserve `tools/package-release.sh`, embedded `fullpack licenses` attribution,
  Linux raw/network-namespace fixtures and the final evidence in
  `docs/waticmp-verification.md` / `docs/waticmp-benchmark.md`. Naming reapply
  cannot recreate these features. Merge source and pass feature guards/tests.
  Preserve actual authenticated QUIC/WSS/WSSMUX Auto setup, independent probe
  and user-stream handover/drain regression tests alongside native coverage.
- Uninstall must first stop/delete tunnel controllers safely, preserve files
  when that fails, and remove the default installer-owned helper tree after
  services stop. Never recursively follow a symlinked/public helper root or
  remove operator-supplied executable paths outside that dedicated namespace.

## Auto completion approved on 2026-10-13

- The shared eighteen-carrier registry covers Reverse and explicit Direct
  initiation. L3 uses only UDP, QUIC, xDi, PCK, SPOOF and SNI with a stable TUN.
  See `docs/auto-carrier-completion.md` for identities, packet security and
  prerequisites; `docs/waticmp-verification.md` records acceptance separately.
- Preserve SPOOF's authenticated bounded conversation demultiplexing on the
  same enrolled raw receive endpoint, the concurrent bounded secure KCP accept
  workers, and native Wat's optional Direct bidirectional stream namespaces.
  Do not enable that optional namespace in the legacy manual adapter.
- CLI/web imports collect destination-local helper identity and raw/TUN binds;
  compatible Node rebuilds and CLI link updates retain that machine's private
  identity. Never carry a redacted private key across the fleet API or inherit
  it into an incompatible role, secret or enrollment. Blank redacted edits
  preserve secrets; replacement keys/certificates must match public enrollment.
- Keep QUIC unavailable on outer paths below its 1200-byte initial requirement.
  A failed selection must retain the working route/generation. Preserve the
  real small-path and full-MTU fixtures and the pinned HTTPS tests in CI.
- This completion does not change installer signature policy, dependency
  versions, manual wire identifiers, default-off Auto or WAN/release gates.
- Expansion acceptance: whole-project Linux race passed at 58.4% total with
  all fourteen existing floors unchanged; later CLI rebuild edits passed the
  exact affected-package race suite. Final exact-source formatting, vet,
  staticcheck and static amd64/arm64 builds passed. All 46 panel and nine
  customization tests passed. Actual Reverse/Direct raw/helper and real-TUN
  matrices, six SPOOF stream profiles, small-path rejection and optional L3
  forwards passed. The online vulnerability DB returned HTTP403; the official
  2026-10-08 snapshot reported no reached/imported-package vulnerabilities.
  WAN, arbitrary NAT/anti-spoofing policy and the expansion's 24-hour soak
  remain NOT VERIFIED. Preserve this evidence boundary through updates.

## Bounded WAN verification on 2026-10-13

- Preserve `docs/fullpack-wan-verification.md` and section 9 of the verification
  ledger through source updates. They supplement the historical local gates;
  they do not convert failed/unavailable carriers or a 24-hour soak into passes.
- The 96-case stream/L3 matrix and selected retries passed 15/18 Reverse and
  13/18 Direct stream routes on Kharej1, and 12/18 Reverse on Kharej2. Direct
  Kharej2 did not qualify in the bounded tests. Existing BackPack raw ownership
  blocked Kharej1 PCK/SNI; the guard was not weakened and services were retained.
- The exact authenticated REALITY cover test rejected Microsoft on all three
  machines; Apple passed and enabled the successful WAN retries. An ordinary
  TLS response must not replace that authenticated test.
- Controlled TCP failure recovered to enrolled WSS on both pairs in about
  24 seconds with agreed generations, the same Iran listener and three minutes
  of subsequent traffic. Normal Wat Echo, forged sources, automatic L3 WAN
  recovery, production load and 24-hour acceptance remain NOT VERIFIED.
- Audit binaries/configurations, processes, test interfaces and exact added
  firewall rules were removed. Original rules, listener PIDs and routes were
  verified. Passwords were kept in memory; no product source/security changes,
  Git push, signed release or permanent installation occurred during WAN testing.

## Auto criticism review approved on 2026-10-14

- Preserve `docs/auto-policy-review.md` and its distinction between successful
  bounded tests and pending 24-hour/1000-user acceptance. No carrier or core
  architecture was added or replaced.
- Confirmed debounced no-route recovery uses a five-second minimum and the
  existing absolute twelve-commit rolling-hour ceiling. All commits share the
  original durable timestamp history and continue to constrain optional
  Cooldown/MinimumDwell/MaxSwitchesPerHour. Preserve locks, observe-only,
  integrity/pressure gates and uncertain-commit generation fencing.
- Preserve the actual three-carrier/two-failure regression and restart/flapping
  checks. Both supplied WAN pairs passed the consecutive recovery scenario;
  UDP to xDi passed actual automatic WAN L3 recovery on Kharej1 with one TUN.
- Keep safe typed `errorCode`/`errorStage` diagnostics, role-specific debug
  logging and distinct timeout/rejection/closed/bind/local-denial categories.
  A timeout must not be called bad enrollment or proof of filtering.
- Keep bounded xtables-lock waits for scoped ICMP/MSS operations and joined
  Auto cancellation cleanup. Test direct engine TERM separately from supervisor
  KILL/group forwarding; never report interrupted cleanup as passed.
- Default connection admission stays 64, maximum 192. Preserve the opt-in
  100/192-socket admission/integrity test and CI invocation. These are application
  connections, not independent users or proven WAN/1000-user capacity.
- Preserve the corrected historical QUIC bind-conflict and stop-grace evidence.
  Valid QUIC Auto/manual retries and smaller normal-Echo Wat on Kharej2 still
  did not qualify; transport security was not weakened to turn them into passes.

## Original native/Auto verification recorded on 2026-10-13

- Whole-project Linux race/coverage, formatting, vet and staticcheck passed
  with Go 1.27.2; final total coverage was 57.3% and all fourteen existing
  package floors passed without being lowered. All 43 panel JavaScript tests
  and the eight customization-tool tests passed.
- Actual isolated two-node normal/reverse native traffic, restricted MTUs,
  cross-carrier recovery, reachable severe degradation, replay/identity
  isolation and legacy local-HEAD interoperability passed. Existing CI soak
  and bounded parser fuzz runs passed; measured xDi benchmark failures remain
  documented rather than removed.
  QUIC/WSS/WSSMUX Auto also passed actual sole-candidate bootstrap, isolated
  probes and coordinated handover/drain to Noise on loopback with race.
- Disposable Linux source installation, systemd traffic, rename, upgrade,
  compatible earlier-local-build rollback, deletion and uninstall passed.
  Installer-owned helper cleanup preserves external operator tools.
- Seven standalone Linux architecture archives include legal attribution and
  exclude the unchanged read-only WatTunnel tree. Two same-source builds were
  byte-identical. No signing key, publication or live deployment was performed.
- Evidence and exact limitations are in `docs/waticmp-verification.md`.
  Actual filtered two-VPS WAN, NAT/address changes, a published earlier-release
  binary, loaded interactive latency and a 24-hour soak remain NOT VERIFIED.
  Local acceptance must not become a production guarantee during an update.

## Source publication authorization on 2026-10-13

- After local acceptance, the operator requested committing and pushing the
  customized FullPack source to the existing `firegoood/BackPack` fork.
- The operator then supplied the new repository
  `https://github.com/firegoood/FullPack.git` and requested publication there.
  Current source/install/update URLs use `firegoood/FullPack`; the earlier
  push to `firegoood/BackPack` is a historical event, not the active destination.
- Repository routing was checked with all nine customization tests, affected
  Linux app/CLI/TUI/update/panel tests, 43 panel JavaScript tests and standalone
  Linux amd64/arm64 builds using Go 1.27.2. The built version command prints
  `https://github.com/firegoood/FullPack`; local archive version parsing passed.
- This authorization covers this source publication. It does not authorize
  release tags, unsigned binary publication, live deployment or automatic
  publication of later work. The entire `11/` directory and local audit
  artifacts remain excluded from Git. Neither tracked source nor the published
  commit history contains files from `11/`.
- The operator subsequently requested publication of the bounded WAN report,
  its ledger and preservation guards. This separate request authorizes the
  current documentation/rules push to the same `firegoood/FullPack` repository.
- On 2026-10-14, the operator separately requested publication of the Auto
  policy, diagnostic and shutdown fixes, their regression/admission tests and
  bounded audit report to the same repository. This authorizes this source
  push; the remaining acceptance gates and publication exclusions still apply.

## Signed fork installation and release approved on 2026-10-14

- The operator requested a release or installer repair after the normal command
  could not download an asset. The fork had no releases or signing secret.
  This request authorizes the v1.9.1 source/tag/release publication to FullPack.
- The fork now has an independent Ed25519 key. Only its public half is pinned
  in `tools/customize/fullpack.json`, the installer and application. The private
  half was generated in memory and encrypted into the repository's
  `RELEASE_SIGNING_KEY` Actions secret; it was never written into the checkout
  or a local private-key file. Preserve this trust anchor through source merges.
- Online installation resolves one tag before fetching the archive, checksum
  list and signature. Offline installation requires the same three artifacts
  plus `RELEASE_TAG` (or an explicit `BP_RELEASE_TAG`). Missing, altered or
  differently tagged signatures/checksums refuse installation; they never
  trigger an unverified fallback. Deliberate trusted local source builds remain
  available through the existing `BP_BUILD_FROM_SOURCE=1` option.
- Release signature verification requires OpenSSL 3.0 or newer and checks for
  the raw-message verification command. Unsupported versions are refused.
- The Release workflow reuses all existing CI jobs before publishing, rejects
  an absent/mismatched signing secret and verifies its own signed artifacts.
  CI and release builds pin Go 1.27.2, matching the audited build/format toolchain;
  the source minimum remains Go 1.26.9 with unpatched 1.27.0/1.27.1 rejected.
- Older source-built FullPack binaries carrying the inherited upstream key
  cannot authenticate this fork's new key. Reinstall once with the verified
  fork installer before using their automatic updater. No upstream key or
  transport/security identity was substituted, and `11/` remains excluded.
- Release availability is distinct from the bounded WAN and pending 24-hour /
  1000-user acceptance recorded in the existing verification reports.
- The loopback Connection Test has no explicit Wat peer addresses. Report that
  missing prerequisite before checking raw-socket availability, keeping its
  SKIPPED result consistent under both privileged and ordinary CI accounts.
  Actual enrolled/raw fixtures still require privileges and real packet tests.
- Preserve the stronger release checks for tag-bound test keys and dated,
  bilingual documentation. Version review stamps describe documentation
  compatibility; original measurement dates and pending gates remain intact.
- The Auto namespace recovery fixture holds 100 admitted application sockets
  before bulk transfer, retaining its packet impairments, connection count and
  existing production limits/deadlines. Legacy unpaced burst mode remains in
  the fixture and its observed reset failures are retained as a capacity limit,
  rather than being declared passed by the held-socket recovery test.
- Normal, race and coverage CI runs use separate matrix workers with fail-fast
  disabled. Every existing command and coverage floor remains required; this
  reduces release waiting time without dropping a check.
- Pass the resolved release tag explicitly to Make and both signing tools.
  GitHub's reserved `GITHUB_REF_NAME` cannot be overridden by a workflow env
  entry. The first v1.9.1 metadata was withdrawn when its install check rejected
  a signature for `main`; a narrowly scoped repair verifies the original source,
  all eleven passing CI jobs and all seven unchanged archive hashes before
  replacing only its signature and `RELEASE_TAG`. No source tag is retargeted.

## Web tunnel methods

- The reverse options API reads `cliTransportGroups()`; the direct options API
  exposes the complete `DirectCarriers()` list. The panel no longer maintains a
  separate filtered list of supported methods.
- HTTPS setup and edit offer the installed helper path, loopback reverse port,
  credentials, certificates/trust and protocol-specific fields. Server identities
  can be generated, while clients take matching values from a setup link.
  Naive needs sing-box on the server and naive on the client; XHTTP/REALITY need
  Xray. Blank helper paths use the existing CLI defaults, and missing helpers
  produce a validation error. The form does not download executables.
- Managed HTTPS uses the existing reverse TCP engine on loopback. The panel
  displays and edits the public HTTPS endpoint separately, refuses UDP and
  incompatible route/proxy/failover settings, and runs CLI/engine validation.
  REALITY server setup checks the selected cover with the existing bounded,
  authenticated TLS/HTTP2 and helper probe before saving.
- Dashboard transport names, public ports and peer/latency probes describe the
  HTTPS helper. Engine health still uses the underlying reverse TCP session;
  the helper's loopback connection is not reported as a remote peer.
- Edit preset menus filter the options API's `kcpOnly` flag and refresh when
  transport changes. Switching KCP to HTTPS retires a Throughput label while
  preserving manual tuning, or applies a newly selected compatible profile.
  Explicitly posting Throughput for HTTPS is refused. Switching between Naive
  and Xray clears the incompatible executable so the new helper's CLI default
  applies; XHTTP/REALITY retain their shared custom Xray executable.
- Settings responses omit passwords and REALITY private keys. Omitted edit
  fields keep their old values; empty passwords/private keys keep their secrets.
  Peer links contain the matching identity, public certificate or REALITY public
  key, never the server TLS/REALITY private key. Naive links infer the dial
  hostname from the certificate when no address was explicitly provided.
  Explicit Naive hostnames and link addresses must match that certificate.
  Clearing an optional HTTPS host or generated identity field still triggers
  the peer setup-link handoff; blank redacted secrets keep their existing value.
- The managed-node mirror carries HTTPS credentials, both ports and public CA
  trust in its JSON apply request. The destination validates and stores the
  public PEM locally and retires incompatible proxy/routing/simple-auth settings.
  Rebuilding the peer preserves its local helper path and loopback port when
  the helper is compatible (XHTTP/REALITY share Xray); peer credentials/trust
  and the public endpoint are replaced with the server's current values.
- IP spoofing uses the existing direct spoof drawer and peer-IP validation;
  SNI spoofing uses the existing domain field. The direct engine and wire
  protocol are unchanged. The legacy layer-4 engine is not a separate selectable
  CLI setup method in this baseline.
- Preserve `internal/manage/managedwebapi.go`, its tests, the shared
  `panel/js/ui/managed.js` helper and their integration in API/forms through
  updates. Required manifest snippets catch discarded integration points;
  the tests check both roles, validation, edit secrets/endpoints, generated
  identities, certificates, peer links and complete CLI/web choices.

## Naming and compatibility

- Display name: `FullPack`; executable: `fullpack`.
- Installation: `/root/FullPack`, `/etc/fullpack`, `/usr/local/bin/fullpack`.
- Services: `fullpack-*.service`; release assets: `fullpack_linux_<arch>.tar.gz`.
- Source/update repository: <https://github.com/firegoood/FullPack>, a separate
  repository created and supplied by the operator. The updater must not download official
  BackPack binaries, because they do not contain this customization layer.
- Upstream source: <https://github.com/AminMGMT/BackPack>.
- The Go module path, tunnel protocol labels, `backpack://` setup links, backup
  metadata, Prometheus metric identifiers and existing environment variables
  remain compatible. These internal identifiers are not display branding.
- The original licensing/attribution documents and historical changelog remain
  unchanged. README attribution stays visible and unaltered.
- This is a fresh FullPack installation namespace. Migrating an already running
  BackPack installation requires a separate, reviewed migration of its config
  and services; a naming change does not perform that migration.

## Integrating a later upstream release

1. Start from a clean, saved FullPack checkout. Preserve the customization files
   and review upstream release notes and the source diff.
2. Fetch the desired upstream tag/commit and merge or rebase its **source** into
   FullPack. Resolve conflicts without discarding customizations or new upstream
   fixes. Do not copy an upstream binary over the FullPack executable.
3. Run `python tools/customize/reapply.py --apply`. The tool performs local,
   repeatable naming changes only; it does not fetch, reset, push or publish.
4. Run `python tools/customize/reapply.py --check` and inspect the diff. A failed
   check means the update is not ready; update the overlay and feature integration
   for structural changes instead of ignoring its diagnostics. The tool reapplies
   naming and verifies feature guards; source merges must preserve feature code.
5. Run formatting checks, build and relevant existing tests. For a release, run
   the repository's CI checks, including Linux, race and wire-compatibility tests.
6. Build and sign **FullPack** assets from the customized source. Configure the
   fork's signing public key in both the application and installer and store the
   private key only in the release secret. Preserve the deliberately configured
   fork public key recorded in the customization manifest.

The reapply tool is idempotent. It handles the currently known naming surfaces;
upstream structural changes still require code review and test verification.

## Verification recorded on 2026-10-12

- FullPack naming check passed; repeating `--apply` changed zero files.
- Eight customization-tool tests passed, including repeatability, attribution,
  compatible cryptographic/protocol labels and fork release routing.
- Selected existing application naming/licensing/installer tests and terminal
  UI tests passed; all 35 JavaScript panel tests passed after the second review.
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath` succeeded.
- Linux short suites for `internal/manage` and `internal/webui` passed, including
  the web transport additions; these and `internal/node` also passed with the
  race detector. Windows CRLF checkout files were normalized only in an isolated
  Linux test copy for existing source-inspection tests.
- Browser checks used the real panel JS/templates with a local mock API: HTTPS
  setup, XHTTP edit, IP/SNI spoofing fields, method switching and the REALITY
  submitted payload were verified. Only applicable fields were sent.
  No production tunnel was created.
- Native Windows compilation remains unsupported by existing Linux socket
  code. The complete release CI, including whole-project race/coverage and
  security checks, remains required before publishing. No GitHub changes were
  published.

## Second review on 2026-10-12

- Fixed the missing peer handoff when an HTTPS host/identity field is cleared.
  Empty redacted passwords/private keys still preserve the old secret.
- Fixed node mirroring so compatible custom client helper executables and
  private loopback ports survive peer rebuilds; new peer identities still apply.
- Naive setup and exported links now reject certificate/address mismatches;
  node mirroring uses the certificate's concrete hostname when available.
- Fixed HTTPS dashboard summaries and latency/peer lookups to use the public
  helper protocol/address/port. Loopback helper peers are excluded from remote
  location fallback. Engine health and wire identifiers remain unchanged.
- Corrected Go formatting in the renamed Telegram translation table.
- All default Linux Go suites passed without `-short`, with Node 22 available
  for the panel API contract tests. `go vet ./...`, formatting and Linux build
  also passed. After the dashboard fix, the affected manage/webui/node suites
  were rerun with `-race -short`, and web vet/build passed on the final source.
- The pinned Naive, sing-box and Xray archives were checksum-verified and
  installed only in an isolated container. Real loopback traffic passed all
  13 tested reverse carriers, including Naive, XHTTP and REALITY (8/8 echoes
  each); authenticated REALITY cover/fallback/cancellation tests also passed
  with the race detector. The source installer succeeded in that container.
- All 35 JavaScript tests and 8 customization-tool tests passed. Repeating the
  naming reapply changed zero files, and the updated feature guards passed.
- Production systemd deployment and an actual two-VPS filtered-network test
  were not performed; loopback results do not measure those environments.

## Final review on 2026-10-12

- Added regression tests that reproduced two transport-switching mistakes:
  retaining a KCP-only Throughput profile on HTTPS and reusing Xray as the
  Naive executable (or sing-box as the client helper). Both were corrected.
- All Linux Go packages passed `go test ./... -race -count=1 -timeout 20m`
  with Node 22 and the checksum-verified pinned HTTPS helpers installed.
  Full formatting, `go vet ./...` and the Linux build passed as well.
  After the final profile/helper fixes, affected manage/webui race suites,
  their validators/API contract tests and the embedded-panel build were rerun.
- All 40 JavaScript tests and 8 customization-tool tests passed. Required
  update guards include the new profile/helper switching behavior. Repeating
  the naming reapply still changed zero files; whitespace checks passed.
- Real panel templates/JS were checked in the browser with a local mock API:
  KCP offered Throughput, switching to HTTPS selected a compatible profile,
  the HTTPS menu excluded Throughput, and XHTTP-to-Naive cleared the Xray path.
  No form was submitted to a production server.
- Changes outside mechanical naming were checked against the naming transform
  and matched the recorded web additions, tests and formatting. LICENSE,
  NOTICE, TRADEMARK.md, historical changelog, Go module and dependency files
  remain unchanged.
- An actual two-VPS/systemd deployment is still outside this verification.
  Release coverage/security gates must run before publishing a release.

## Independent final recheck on 2026-10-12

- A complete fresh Linux race run, build, formatting and vet passed before
  the additional security check. Staticcheck v0.8.2 reported no findings.
- The security scan found 12 reachable standard-library vulnerabilities in
  the audit image's Go 1.27.1. The project's Go 1.26.9 baseline contains the
  fixes; on the newer branch they require Go 1.27.2. See the
  [official release history](https://go.dev/doc/devel/release).
- Regression tests reproduced the installer accepting Go 1.27.0/1.27.1 just
  because their minor version is higher. The installer now rejects these
  versions, uses an available patched compiler or requests its checksum-verified
  Go 1.26.9 download. Tests cover both rejection and actual compiler selection.
  Repository rules and required update guards preserve this correction.
- Govulncheck v1.9.0 on Go 1.27.2 reported zero reachable vulnerabilities and
  zero vulnerable imported packages. It noted two module-level advisories on
  code the application does not call. Module requirements remain unchanged.
- The public database endpoint returned HTTP 403. The scan instead used all
  4,613 official OSV records from `golang/vulndb` snapshot
  `d5e0cfac5d260523df858cd4c0f849e248468fb6`, with Git-derived timestamps and
  the documented local database API. Full affected ranges were retained;
  optional fixed-version shortcuts were omitted so records were not skipped.
- On the final source with Go 1.27.2 and Node 22, all Linux packages passed
  `go test ./... -race -count=1 -timeout 20m`; the installer regressions also
  passed separately with the race detector. Formatting, `go vet ./...` and
  a static Linux build passed. All 40 panel JavaScript tests and 8 customization
  tests passed; reapplying naming changed zero files and whitespace checks
  were clean. No additional CLI/web parity or HTTPS edit/mirror issue was found.
- Actual two-VPS/systemd deployment and release coverage remain untested.
  No release was published and no user server was changed.
