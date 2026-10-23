# #759: tsnet decision for the operator

**Draft — upgrade-only full suite pending; extra native-history coverage blocked by #828. No enrollment.**

**Recommendation:** retain A as the preferred security boundary, but do not accept
this default dependency build yet. It gives Couch its own tailnet HTTPS listener
and connection-based WhoIs without a root daemon or a locally spoofable proxy.
Proceed only if the isolated shared-upgrade tests pass, the agreed telemetry policy is verified,
and the operator accepts a freshly measured, reduced build. If those conditions
cannot be met, return to the root-daemon option rather than weakening identity.

| Measured on Go1.27.3, Darwin/arm64 | Existing Couch | tsnet linkage probe |
| --- | ---: | ---: |
| Binary bytes | 37,566,322 | 62,699,650 |
| Nonstandard compiled packages | 129 | 461 |
| Compiled module identities | 27 | 54 |

The probe adds25,133,328bytes (23.97MiB,66.9%). It retains selected tsnet API
symbols but is not the final adapter; neither probe binary was run. Cached
repeat builds were0.613s/0.610s; first-use candidate build59.811s has different
cache conditions from the5.089s baseline and is not a cold-build comparison.
RSS, enrolled startup and four-session load are **unmeasured** until the operator
authorizes the one-device enrollment test.

**Shared-upgrade risk:** module selection forces ten existing upgrades, including
ultraviolet, ansi, colorprofile, runewidth, terminfo and go-colorful, plus
x/net, oauth2, sync and term. They affect all Couch rendering, not just remote.
Scratch tests use these upgrades without tsnet: native direct/wrapped Zellij,
Claude/Codex PTY layout, live Neovim and scrolling passed; cleanup confirmed.
Complete terminal package passed (186 top-level tests, eight skips). Seven extra
native-history cases are unqualified: six failed on shell output, while an
equality-only test passed on that wrong source. One matched original-module
control failed identically; #828 owns the fixture integrity gap. Both full suites
are queued. No upgrade-caused regression has been established; this is not full
rendering clearance. The ten upgraded
root license texts are unchanged. New embedded web/font asset notices are not
yet cleared; the compile-only build is not redistribution approval.

| Risk | Proposed mitigation before product implementation/enrollment |
| --- | --- |
| Watch schema is unstable; WhoIs is a local lookup, not fresh control-plane proof | Pin1.104.2 and its initial-status/peer-delta/removal schema; require the initial snapshot and fail closed on signaled overflow, terminal error/EOF, expiry or watch loss. There is no general sequence-based gap detector. WhoIs uses only the captured accepted connection address, exact login policy and current node evidence. Native revocation tests must measure propagation; no promise of instantaneous revocation during partitions. |
| ListenTLS calls Up with a background context; its certificate helper can wait one minute | Proposed: serialize Start, then context-bound the readiness wait, use tailnet-only Listen and owned TLS wrapping with exact assigned-host SNI. Bind CertPair to epoch cancellation and a handshake deadline. Up calls Start synchronously; its context cannot bound Start. Retain uncertain opening, suppress late readiness. This composition requires review and fake/native qualification. |
| Close has no whole-call deadline and discards subordinate errors | Revoke sessions/dispatch first, close HTTP listeners, and join exactly one owned Close worker only after Start settles. Five seconds bounds the UI wait, not an asserted library teardown guarantee. Timeout retains ownership in blocked state, no replacement node/reset; confirmed prior-process termination permits explicit recovery under the persisted latch. Restart alone proves neither graceful cleanup nor logout and grants no permission. Qualify this explicitly; do not claim an abandoned goroutine was joined. |
| Default logtail uploads and unnecessary embedded management UI | Prefer qualifying ts_omit_logtail and ts_omit_webclient build tags, to omit logtail-backed debug/network-log uploads and the management-web import path; verify the resulting asset closure and remaining obligations; measure that new build, do not reuse these size numbers. Tags cannot undo Go module selection or the ten shared upgrades. Alternative logtail.Disable() before any Start is an irreversible process-wide upload opt-out, not a per-node setting or a promise that all telemetry is off. Logf alone is insufficient. No global environment mutation. |

**Operator setup remains:** approve one Couch node through a console-only login
link; keep its keys in owned0700 state/0600 files; display its actual assigned
hostname. No auth key is required. The current desktop app stays installed.
Off/kill revoke first; link rotation, four-session cap, exact login allowlist,
capture refusal, audit/transcript provenance and physical-host-only reset remain.

Evidence: [measurements and exact module changes](../../evidence/000759/tsnet-dependency-20261022/closure-summary.json),
[pinned API review](../../evidence/000759/tsnet-dependency-20261022/review.md),
[license inventory](../../evidence/000759/tsnet-dependency-20261022/licenses-summary.md).
