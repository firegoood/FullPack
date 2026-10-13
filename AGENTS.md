# FullPack customization rules

Read `CUSTOMIZATIONS.md` and `tools/customize/fullpack.json` before editing or
updating this repository. These files record the user's persistent preferences.

- Preserve the authorized native Wat ICMP and peer Auto changes described in
  `docs/waticmp-dual-server-design.md` and their evidence/limits in
  `docs/waticmp-verification.md`. Keep `11/WatTunnel` read-only. Exclude the
  entire `11/` directory from Git/publication, formatting, tests and release
  archives. Preserve the adapted engine's MIT attribution and its nonce,
  resource, ACK and stream repairs.
  Auto remains opt-in; never weaken enrollment, generation fencing or security
  floors to resolve an upstream merge, and never report pending gates as passed.

- Preserve the approved complete Auto carrier registry shared by CLI, web and
  engine. Stream Auto supports Reverse and Direct initiation; L3 Auto uses the
  six datagram carriers and owns one stable TUN across agreed generations.
  Keep IP outside reliable KCP/TCP streams, authenticated bounded reassembly,
  replay/nonce protections and datagram probes. Keep SPOOF production/probe
  conversations on the same enrolled endpoint with bounded authenticated
  demultiplexing. Never export local helper private keys, paths or interfaces;
  retain receiver-local enrollment forms and matching public trust checks.

- The final project name is **FullPack**, with two `l` characters. The command
  and installation namespace are `fullpack`. Do not use `FulPack`.
- Keep web tunnel choices in parity with the CLI: include managed HTTPS
  (Naive/HTTP2, XHTTP/TLS, REALITY/Vision) and direct IP/SNI spoofing. Preserve
  their setup/edit forms, role-specific helper configurations, public/private
  endpoints, secret-preserving edits and peer setup links. Reuse CLI/engine
  validators and the authenticated REALITY cover probe; keep TLS verification.
  Clearing a non-secret HTTPS identity/host field must offer a fresh peer link.
  Node rebuilds must preserve compatible local helper paths and loopback ports;
  Naive dial addresses and exported links must match the server certificate.
  HTTPS dashboard cards and peer/latency probes must use the public helper
  endpoint; keep engine health on the private reverse listener.
  Filter KCP-only profiles out of HTTPS edit choices. Transport switches must
  retire incompatible profile labels and executable paths while preserving
  manual tuning and the shared Xray path for XHTTP/REALITY.
- The naming reapply tool does not recreate feature code. Preserve the web
  transport additions through source merges and run their tests on Linux.
  The manifest's required feature guards must pass before accepting an update.
- Preserve every recorded customization when integrating a newer BackPack
  release. Do not replace this checkout with an upstream archive or reset it to
  upstream. Fetch and merge/rebase the upstream source, resolve conflicts with
  the customizations intact, then reapply and verify the customization layer.
- Run `python tools/customize/reapply.py --apply` after integrating upstream,
  followed by `python tools/customize/reapply.py --check`. Review the resulting
  diff, build, and run the affected existing tests before declaring it ready.
- Preserve `AGENTS.md`, `CUSTOMIZATIONS.md`, and `tools/customize/` through merges.
  Record future user-approved changes in the ledger and maintain their reapply
  logic/checks in the same task. Do not invent additional customizations.
- Product names, command examples, installation paths, systemd units and release
  asset names must agree. Keep release URLs pointed at the actual fork
  `firegoood/FullPack`, the new repository explicitly supplied by the user.
  Preserve this destination until the user supplies a different address.
- Never install official upstream binaries over FullPack: incorporate upstream
  **source**, reapply the customizations, and build FullPack release assets.
- Keep upstream attribution, LICENSE, NOTICE and TRADEMARK.md intact. Keep the
  Go module path and wire-protocol identifiers, cryptographic domain labels,
  existing environment variables and backup metadata compatible with upstream.
  Naming is not permission to change transport behavior or weaken verification.
- Do not disable signature verification to make fork releases install. A fork
  needs its own signing key and matching installer/public key before publishing
  signed releases; never put a private signing key in the repository.
- Build and verify with a patched Go toolchain. Keep the installer rejection
  of Go 1.27.0/1.27.1: their higher minor version does not include the security
  fixes in the project's Go 1.26.9 baseline. Go 1.27 needs patch 2 or newer.
- This workflow reduces merge risk; it does not guarantee arbitrary upstream
  changes are compatible. Review new upstream behavior, especially installation,
  update, service, panel and protocol changes. Extend the reapply layer when a
  changed upstream structure no longer fits it, and report unresolved failures.
