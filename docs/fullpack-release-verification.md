# FullPack signed release verification

## v1.9.3 — full Quick Auto pools and measured reserve replenishment

Date: 2026-10-15. Status: **PUBLISHED; FULL CI AND SIGNED PUBLIC INSTALL PASSED**.

[v1.9.3](https://github.com/firegoood/FullPack/releases/tag/v1.9.3) is bound to
source `057f69d2a9d418f95196c1a9d54759e348e945d4`.
[Release run 38679834616](https://github.com/firegoood/FullPack/actions/runs/38679834616)
passed all eleven verification jobs and the signed public-install release job.

- Quick stream enrollment covers eighteen registry carriers in Reverse/Direct,
  plus both Wat Echo modes: 38 candidates. Quick L3 covers six datagrams in both
  directions: 12. At most four measured paths are selected; reaching two healthy
  selected paths triggers a fresh approved-pool sweep and paired acknowledgment.
- The strict six-carrier fault test waits for the initial sweep to finish,
  breaks two selected standbys, observes the refill threshold and returns from
  four to two to four. A held TCP stream, 32 KiB exchanges and 8000-byte UDP
  retain integrity; reserve replacement preserves the active generation. A
  later active failure with only two physical routes remaining recovers at a
  fenced generation and reports two, without fabricating extra availability.
  Two isolated unprivileged Linux runs passed in 85.00/85.47 seconds; final
  whole-project normal/race/coverage and privileged peer suites passed too.
- A real twelve-candidate L3 namespace fixture retires two selected datagrams
  at the authority, replenishes the subset, and proves the same active route,
  generation and TUN ifindex with bidirectional IP traffic, TCP and UDP. This
  fixture tests explicit path retirement; the separate stream test faults the
  network. Existing small/full-MTU raw/QUIC matrices also passed.
- Pinned Naive, XHTTP and REALITY integration passed in both directions for
  legacy and generated pool identities. Chrome-compatible P-256 certificates
  use standard RFC 6979 signing for reproducible enrolled public trust. TLS
  verification and the actual authenticated REALITY cover gate remain enabled.
- Full normal/race tests, 100/192 bounded connection-admission scenarios,
  native/raw recovery/degradation, fuzzing, two-minute soak, patched-toolchain
  security analysis and actual v1.9.2 wire compatibility passed. Coverage was
  **58.2%**, with all fourteen unchanged package floors satisfied. Socket
  admission counts do not establish independent user capacity.
- Independent public download verified all twelve assets, seven archive hashes,
  the fork's tag-bound Ed25519 signature, exact ELF/legal members, compiled key,
  raw-main/tag/release installer equality and Go 1.27.2 in every binary's build
  record. No `11/`, private enrollment/key material or server credentials were
  published. The existing fork signing pin was preserved.

Development failures for stale reserve reuse and unsupported generated TLS
certificates were repaired before this accepted source. The first failed CI
run is not acceptance evidence. Existing production pairs were not expanded
or silently re-enrolled; the wider pool requires explicit new enrollment on
matching feature versions at both peers. Network-dependent availability,
24-hour soak and 1000-user production capacity remain **NOT VERIFIED**.

## v1.9.2 — Quick Auto and probe uptime repair

Date: 2026-10-14. Status: **PUBLISHED; FULL CI AND SIGNED PUBLIC INSTALL PASSED**.

[v1.9.2](https://github.com/firegoood/FullPack/releases/tag/v1.9.2) is bound to
source commit `28bd07f6445b958107fc8f310c8a40de20185104`. All eleven verification
jobs and the release job passed in
[run 38626831074](https://github.com/firegoood/FullPack/actions/runs/38626831074).
The public raw `main/install.sh` matched the source installer, downloaded the
published signed release and installed the declared version successfully.

- Whole-project Linux normal/race tests, 58.0% total coverage and all fourteen
  unchanged package floors passed. The bounded 100/192-connection admission,
  native raw/namespace recovery, reachable degradation, Direct/L3, actual managed
  HTTPS, parser fuzz and existing short soak gates also passed.
- The online source vulnerability scan reported no vulnerabilities; static
  analysis, formatting, patched amd64/arm64 builds and actual interoperability
  with the previously published v1.9.1 passed. This is not a security audit of
  every third-party helper or proof of sustained WAN capacity.
- All 49 panel JavaScript and ten customization tests passed. Quick builder,
  trusted peer import, invalid input and occupied-listener preflight passed on
  Linux. The visible-form regression prevents cached Auto IPs from contaminating
  Wat setup links. The isolated browser form reviewed the four expected ports
  and retained values when changing to Advanced.
- A real v1.9.1 uptime failure was traced to its 256-entry hourly probe replay
  cache. v1.9.2 retains the full one-hour window with a bounded 4096-entry cap.
  Paced-hour, replay/expiry, hard-cap and typed-budget control regressions passed
  as an unprivileged Linux user. Real isolated Noise Reverse/Direct data/probe/
  held-stream handover and TCP → WSS → WSSMUX consecutive recovery also passed.
  The clock model is not a real-hour soak. See [live evidence](auto-live-setup-review.md).
- Independent public downloads verified all twelve assets, the tag-bound fork
  Ed25519 signature, seven archive checksums/ELF identities and exact legal/binary
  members, exclusion of `11/` and symlinks, the compiled public pin and installer
  equality with the immutable tag. Every binary's Go build information identifies
  Go 1.27.2 and the preserved module path; the SBOM agrees.

The Windows workstation's unsupported full filesystem/network test attempts
were not relabelled as passes or worked around in production code. Targeted
installer/docs tests ran there; the complete supported Linux suite passed in CI.
Published binaries do not include server credentials, private enrollment data
or the release signing key. No tag was moved after publication.

Direct Noise/Wat on the restricted operator WAN, the earlier unpaced impaired
burst with 21 resets, 24-hour soak and 1000-user acceptance remain **NOT VERIFIED**.
The source/runtime repairs do not turn those measurements into passing gates.

## Historical v1.9.1 release

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

نسخهٔ ۱.۹.۲ با ساخت سریع Auto و اصلاح پر شدن حافظهٔ Probe منتشر شد. هر یازده
بخش CI، نصب عمومی و بررسی مستقل امضا و هفت آرشیو موفق بودند. هر دو نوع
Reverse و Direct در تست ایزولهٔ Linux بررسی شدند؛ نتیجهٔ WAN محدود و تست
۲۴ساعته همچنان از این موفقیت‌ها جداست. سورس، تگ و کلید امضای FullPack حفظ شدند.

نسخهٔ ۱.۹.۱ با کلید مستقل FullPack منتشر و نصب عمومی آن بررسی شد. نصب‌کننده پیش از
استخراج، امضای همان نسخه و چک‌سام آرشیو را با OpenSSL 3 یا جدیدتر بررسی می‌کند.
هر یازده بخش CI و بررسی مستقل هفت آرشیو موفق بودند. یک اجرای burst صدتایی پس از بازیابی Noise،
۲۱ اتصال ناموفق داشت؛ آن نتیجه حذف یا موفق اعلام نشده است. تست بازیابیِ
اتصال‌های از قبل پذیرفته‌شده با burst متفاوت است و ظرفیت واقعی کاربران یا
پایداری ۲۴ساعته را ثابت نمی‌کند. پوشهٔ `11/` و اطلاعات خصوصی منتشر نمی‌شوند.

</div>

*Last verified against FullPack v1.9.2 (full CI, public signed installation and independent asset verification, 2026-10-14; historical measurements and remaining acceptance limits preserved).*

*Last verified against FullPack v1.9.3 (source/documentation compatibility review, 2026-10-15; original measurements, earlier release results and pending acceptance gates retain their recorded dates and scope).*

*Last verified against FullPack v1.9.4-beta.2 (source/documentation compatibility review, 2026-10-15; historical measurements retain their dates and scope; release, deployment, WAN and soak gates require their separately recorded evidence).*
