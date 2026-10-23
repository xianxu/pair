# #759: tsnet decision for the operator

**Decision required; no enrollment or product dependency added.**

**Recommendation: still A, conditionally.** A dedicated tsnet node gives Couch
its own HTTPS listener and connection-based WhoIs without a root daemon or
spoofable local proxy headers. Accept its shared dependency risk only after
qualification; this experiment does **not** clear the default build for release.
If that cost is unacceptable, return to the root-daemon option rather than
weaken identity. Public Cloudflare casts remain unchanged.

| Measured: Go1.27.3, Darwin/arm64 | Existing Couch | tsnet linkage probe |
| --- | ---: | ---: |
| Binary bytes | 37,566,322 | 62,699,650 |
| Nonstandard compiled packages | 129 | 461 |
| Compiled module identities | 27 | 54 |

Added25,133,328 bytes (**23.97MiB /66.9%**). Neither binary ran; this is selected
API linkage, not the final adapter. RSS, startup and four-session load remain
unmeasured until authorized enrollment. Cached builds were0.613s/0.610s;
first-use builds had unequal cache conditions and do not establish cold cost.

**What the ten shared upgrades broke:** no upgrade-caused regression established.
The Go1.27.3-only recorded suite passed in10m52s. The upgrade-only full suite
(no tsnet) completed **exit2**, four failed tests,133 passing Go packages:
two failures were my scratch PATH omitting Pair/helper; both pass with only
private same-source prerequisites supplied. One was our missing pure-file
inventory entries, now audited/fixed with the complete artifactpath package
passing. The fourth is a symlink-TMPDIR fixture failure reproduced with original
modules (#831). These controls classify failures; they do not replace the failed
full run with a green status.

Native direct/wrapped Zellij, Claude/Codex PTY, nvim and scrolling passed, as did
the required xterm oracle and complete terminal package (186 top-level passes,
eight skips). Seven extra native-history cases remain unqualified: six failures
and one equality-only pass on shell output; original-module control also fails
(#828). Shared upgrades affect ultraviolet, ansi, colorprofile, runewidth,
terminfo, go-colorful, x/net, oauth2, sync and term. A clean combined qualification
is still required before integration.

**Mitigations proposed for approval:**

- **Watch/identity:** pin1.104.2; require initial snapshot, terminal-error/EOF and
  signaled-overflow handling, expiry and exact login policy. WhoIs uses only
  captured accepted-connection addresses. No general sequence-gap detector or
  instantaneous revocation promise during partitions.
- **ListenTLS:** avoid its background-context convenience path. Serialize Start,
  then context-bound readiness; use tailnet-only Listen with owned TLS wrapping,
  exact assigned-host SNI and CertPair bounded by epoch and handshake deadline.
  Up(ctx) cannot cancel synchronous Start. Retain uncertain opening and suppress
  late readiness. This composition needs review and fake/native qualification.
- **Close:** revoke first; join one owned Close worker after Start. Five seconds
  bounds UI waiting, not library cleanup. Timeout stays blocked, with no reset or
  replacement. Confirmed prior-process termination permits explicit recovery
  under the persisted latch; restart alone grants no permission or logout proof.
- **Privacy/cost:** qualify ts_omit_logtail and ts_omit_webclient, then remeasure.
  They omit logtail-backed debug/network uploads and the management-web import;
  verify remaining telemetry and license obligations. They cannot undo module
  selection or shared upgrades. Logf suppression is insufficient; pre-Start
  logtail.Disable() is an alternative process-wide, irreversible upload opt-out.

The ten upgraded root licenses are unchanged; embedded web/font notices remain
uncleared. A reduced build needs its own closure and notice review. Operator
setup remains one console-approved Couch node, actual assigned hostname,
owned0700 state/0600 files; existing desktop Tailscale stays. Kill/revoke,
four-session cap, physical-host reset and audit/transcript provenance remain.

[Exact measurements and failure summary](../../evidence/000759/tsnet-dependency-20261022/shared-upgrades-suite-summary.json) ·
[Pinned API review](../../evidence/000759/tsnet-dependency-20261022/review.md) ·
[Cost graph](../../evidence/000759/tsnet-dependency-20261022/closure-summary.json) ·
[License inventory](../../evidence/000759/tsnet-dependency-20261022/licenses-summary.md)
