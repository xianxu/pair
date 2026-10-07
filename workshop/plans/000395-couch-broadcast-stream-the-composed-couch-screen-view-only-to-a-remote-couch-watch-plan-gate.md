---
gate: plan-quality
issue: 395
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-07T15:14:00-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: 'Listener/tunnel seam is self-contradictory: TCP 127.0.0.1:0 vs unix socket, and Open(ctx, localURL) cannot express either choice'
          detail: Session says it listens on 127.0.0.1:0; Cloudflared says the Session listens on a unix socket in a 0700 dir. LocalOnly needs TCP (a browser can't reach a unix socket) and FakeTunnel returns the local URL. Decide in the plan who owns the listener kind, e.g. the Tunnel declares the listener it needs and the Session serves on it, before M2 freezes the seam that M4 must fit.
          family: seam-shape-undecided
          round: 1
        - id: PQ-2
          severity: Important
          title: Task 3.4 (M3) defaults COUCH_BROADCAST_TUNNEL to Cloudflared, which is only created in Task 4.1 (M4)
          detail: M3 either won't compile or will ship a default that doesn't exist. Have the M3 default refuse or use LocalOnly and switch it in M4, or move the Cloudflared wiring into Task 4.1.
          family: milestone-forward-dependency
          round: 1
        - id: PQ-3
          severity: Minor
          title: Test sections enumerate cases in prose instead of one strategy line per risky function
          detail: Compress to the risky functions (IndicatorShown, ViewerFrame, Stream, Hub, Server token/method gate, ReapOrphans, nextFontSize), each with its adversarial input class and mechanical guard. The hub should get seeded random interleavings checked against stated invariants.
          family: test-prose-enumeration
          round: 1
        - id: PQ-4
          severity: Minor
          title: No gated live conformance check for cloudflared's stderr URL banner and the --unix-socket flag
          detail: ARCH-MOCK wants the fake script's modeled behavior compared with the real binary on a cadence; the one-time M4 smoke doesn't cover drift in a cloudflared upgrade.
          family: missing-live-conformance
          round: 1
        - id: PQ-5
          severity: Minor
          title: Unix-socket directory has no lifetime entry; macOS 104-byte socket-path limit unaddressed
          detail: The Lifetimes section should name who removes the 0700 dir and socket (Stop; ReapOrphans after a crash). Tests should bind under a short path, because t.TempDir on macOS often exceeds sun_path.
          family: artifact-without-funeral
          round: 1
        - id: PQ-6
          severity: Minor
          title: live→toggle runs Stop (up to ~5s) on the Console input path, while start was moved off it
          family: blocking-work-on-input-path
          round: 1
        - id: PQ-7
          severity: Minor
          title: The OSC 52 link copy carries the token into the parent stream, which COUCH_CAPTURE_DIR records to disk
          family: credential-reaches-disk
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T15:15:29-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Tunnel.Listen() + Open(ctx, l); tunnel owns listener kind; LocalOnly TCP, Cloudflared unix socket with TCP fallback.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: M3 default is LocalOnly; Task 4.1 Step 5 switches unset to Cloudflared.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Test-strategy section gives one line per risky function; hub uses seeded interleavings against stated invariants.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: TestCloudflaredLive gated on BROADCAST_LIVE_CLOUDFLARED=1, rerun after cloudflared upgrades.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Socket dir lifetime named (Handle.Close, ReapOrphans); 100-byte fallback; tests bind under a short /tmp path.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Stop ends viewers synchronously and finishes teardown on a goroutine; stopping phase; test asserts off-input-path.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Explicitly accepted out of scope with reasoning (token dies with broadcast); atlas documents it.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-07T15:16:42-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
content_hash: ad6e94f68d0eb75b7dadee8aed1199261433512efa61aec4e3277efcd9d55ddd
---

# Gate ledger — pair#395 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T15:14:00-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `seam-shape-undecided` Listener/tunnel seam is self-contradictory: TCP 127.0.0.1:0 vs unix socket, and Open(ctx, localURL) cannot express either choice
  Session says it listens on 127.0.0.1:0; Cloudflared says the Session listens on a unix socket in a 0700 dir. LocalOnly needs TCP (a browser can't reach a unix socket) and FakeTunnel returns the local URL. Decide in the plan who owns the listener kind, e.g. the Tunnel declares the listener it needs and the Session serves on it, before M2 freezes the seam that M4 must fit.
- **PQ-2** [Important] `milestone-forward-dependency` Task 3.4 (M3) defaults COUCH_BROADCAST_TUNNEL to Cloudflared, which is only created in Task 4.1 (M4)
  M3 either won't compile or will ship a default that doesn't exist. Have the M3 default refuse or use LocalOnly and switch it in M4, or move the Cloudflared wiring into Task 4.1.
- **PQ-3** [Minor] `test-prose-enumeration` Test sections enumerate cases in prose instead of one strategy line per risky function
  Compress to the risky functions (IndicatorShown, ViewerFrame, Stream, Hub, Server token/method gate, ReapOrphans, nextFontSize), each with its adversarial input class and mechanical guard. The hub should get seeded random interleavings checked against stated invariants.
- **PQ-4** [Minor] `missing-live-conformance` No gated live conformance check for cloudflared's stderr URL banner and the --unix-socket flag
  ARCH-MOCK wants the fake script's modeled behavior compared with the real binary on a cadence; the one-time M4 smoke doesn't cover drift in a cloudflared upgrade.
- **PQ-5** [Minor] `artifact-without-funeral` Unix-socket directory has no lifetime entry; macOS 104-byte socket-path limit unaddressed
  The Lifetimes section should name who removes the 0700 dir and socket (Stop; ReapOrphans after a crash). Tests should bind under a short path, because t.TempDir on macOS often exceeds sun_path.
- **PQ-6** [Minor] `blocking-work-on-input-path` live→toggle runs Stop (up to ~5s) on the Console input path, while start was moved off it
- **PQ-7** [Minor] `credential-reaches-disk` The OSC 52 link copy carries the token into the parent stream, which COUCH_CAPTURE_DIR records to disk

## Round 2 — 2026-10-07T15:15:29-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Tunnel.Listen() + Open(ctx, l); tunnel owns listener kind; LocalOnly TCP, Cloudflared unix socket with TCP fallback.
- PQ-2 — addressed — M3 default is LocalOnly; Task 4.1 Step 5 switches unset to Cloudflared.
- PQ-3 — addressed — Test-strategy section gives one line per risky function; hub uses seeded interleavings against stated invariants.
- PQ-4 — addressed — TestCloudflaredLive gated on BROADCAST_LIVE_CLOUDFLARED=1, rerun after cloudflared upgrades.
- PQ-5 — addressed — Socket dir lifetime named (Handle.Close, ReapOrphans); 100-byte fallback; tests bind under a short /tmp path.
- PQ-6 — addressed — Stop ends viewers synchronously and finishes teardown on a goroutine; stopping phase; test asserts off-input-path.
- PQ-7 — addressed — Explicitly accepted out of scope with reasoning (token dies with broadcast); atlas documents it.

## Round 3 — 2026-10-07T15:16:42-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)
