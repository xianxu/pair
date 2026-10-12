---
gate: plan-quality
issue: 454
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-11T20:17:07-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Concurrent screen broadcast + web cast on a named tunnel collide (ErrTunnelBusy); plan never decides the policy
          detail: Cloudflared.Open claims key "named-"+Name exclusively (cloudflared.go:124-127; records.go:102-104 returns ErrTunnelBusy). Choose and state the design (mutually exclusive casts, separate webcast hostname, or one shared endpoint routing by token), since broadcast.Endpoint and the Recorder inherit it; add a test for the collision.
          family: shared-resource-contention-unstated
          round: 1
        - id: PQ-2
          severity: Minor
          title: '"reuse its tunnel builder" - broadcastSettings has no separate tunnel builder; Task 8 must extract one'
          detail: couchcmd/broadcast.go:30 does the tunnel choice inline; name the extraction in Task 8.
          family: unbacked-existing-code-claim
          round: 1
        - id: PQ-3
          severity: Minor
          title: Tasks 1/2/3/7/8 list test cases in prose; cut each to the function name plus one strategy line per risky function
          detail: 'e.g. splitFrames: fuzz seeded with spike captures and truncated or oversized frames; Tracker: generated input sequences against a stated active-tab invariant.'
          family: test-plan-enumerates-cases
          round: 1
        - id: PQ-4
          severity: Minor
          title: Never-reported pages count as active, so a background-opened uninjectable (chrome://) tab steals the active tab
          family: active-tab-rule-edge
          round: 1
        - id: PQ-5
          severity: Minor
          title: No explicit non-goals section; the deferrals (recording, Recorder accessor, page-forged toggle) are scattered
          family: missing-non-goals
          round: 1
        - id: PQ-6
          severity: Minor
          title: statusChrome collides with existing commitChrome/chrome() meaning status-row UI chrome (terminal.go:90-100)
          family: naming-collision
          round: 1
        - id: PQ-7
          severity: Minor
          title: LaunchOptions.Stderr destination is unspecified; if it is a file it needs an ARCH-FUNERAL line
          family: artifact-lifecycle-unnamed
          round: 1
        - id: PQ-8
          severity: Minor
          title: Live Chrome conformance check has no stated schedule (ARCH-MOCK)
          family: conformance-cadence-missing
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T20:18:19-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Shared ref-counted broadcast.Host with per-token mounts, tagged open/close states, invariant-based tests, and an ErrTunnelBusy collision regression.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: broadcastTunnel(getenv, store, guard) extraction named; webcastSettings takes the built Host.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Strategy lines added for splitFrames and Tracker, but the prose case lists were kept and the new Host section adds another one; carried to close review as Minor.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Accepted for MVP with rationale; recorded in atlas.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Explicit non-goals list added.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Renamed to WebcastCell / RenderedStatusRow.Webcast / statusWebcast.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Stderr goes to an 8 KiB in-memory ring, quoted only in the Launch error; nothing on disk.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Live check runs at each milestone close touching webcast/cdp and after a Chrome major bump; recorded in atlas.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-11T20:19:09-07:00"
      agent: claude
      dispose:
        - id: PQ-3
          disposition: not-addressed
          note: Strategy lines for splitFrames (fuzz) and Tracker (property test) were added, but Revisions keeps the enumerated prose in Tasks 1/2/3/7/8 as the case list instead of cutting it. Still Minor; it goes forward to the close review.
          round: 3
      blocked: false
content_hash: c5987cacad3d038846d970b7b8f1cba1bc9340c693a96cfc6c08ba3cd4e567b5
---

# Gate ledger — pair#454 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T20:17:07-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `shared-resource-contention-unstated` Concurrent screen broadcast + web cast on a named tunnel collide (ErrTunnelBusy); plan never decides the policy
  Cloudflared.Open claims key "named-"+Name exclusively (cloudflared.go:124-127; records.go:102-104 returns ErrTunnelBusy). Choose and state the design (mutually exclusive casts, separate webcast hostname, or one shared endpoint routing by token), since broadcast.Endpoint and the Recorder inherit it; add a test for the collision.
- **PQ-2** [Minor] `unbacked-existing-code-claim` "reuse its tunnel builder" - broadcastSettings has no separate tunnel builder; Task 8 must extract one
  couchcmd/broadcast.go:30 does the tunnel choice inline; name the extraction in Task 8.
- **PQ-3** [Minor] `test-plan-enumerates-cases` Tasks 1/2/3/7/8 list test cases in prose; cut each to the function name plus one strategy line per risky function
  e.g. splitFrames: fuzz seeded with spike captures and truncated or oversized frames; Tracker: generated input sequences against a stated active-tab invariant.
- **PQ-4** [Minor] `active-tab-rule-edge` Never-reported pages count as active, so a background-opened uninjectable (chrome://) tab steals the active tab
- **PQ-5** [Minor] `missing-non-goals` No explicit non-goals section; the deferrals (recording, Recorder accessor, page-forged toggle) are scattered
- **PQ-6** [Minor] `naming-collision` statusChrome collides with existing commitChrome/chrome() meaning status-row UI chrome (terminal.go:90-100)
- **PQ-7** [Minor] `artifact-lifecycle-unnamed` LaunchOptions.Stderr destination is unspecified; if it is a file it needs an ARCH-FUNERAL line
- **PQ-8** [Minor] `conformance-cadence-missing` Live Chrome conformance check has no stated schedule (ARCH-MOCK)

## Round 2 — 2026-10-11T20:18:19-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Shared ref-counted broadcast.Host with per-token mounts, tagged open/close states, invariant-based tests, and an ErrTunnelBusy collision regression.
- PQ-2 — addressed — broadcastTunnel(getenv, store, guard) extraction named; webcastSettings takes the built Host.
- PQ-3 — not-addressed — Strategy lines added for splitFrames and Tracker, but the prose case lists were kept and the new Host section adds another one; carried to close review as Minor.
- PQ-4 — addressed — Accepted for MVP with rationale; recorded in atlas.
- PQ-5 — addressed — Explicit non-goals list added.
- PQ-6 — addressed — Renamed to WebcastCell / RenderedStatusRow.Webcast / statusWebcast.
- PQ-7 — addressed — Stderr goes to an 8 KiB in-memory ring, quoted only in the Launch error; nothing on disk.
- PQ-8 — addressed — Live check runs at each milestone close touching webcast/cdp and after a Chrome major bump; recorded in atlas.

## Round 3 — 2026-10-11T20:19:09-07:00 (claude) — passed

### Disposed

- PQ-3 — not-addressed — Strategy lines for splitFrames (fuzz) and Tracker (property test) were added, but Revisions keeps the enumerated prose in Tasks 1/2/3/7/8 as the case list instead of cutting it. Still Minor; it goes forward to the close review.

## Open findings

- **PQ-3** [Minor] `test-plan-enumerates-cases` Tasks 1/2/3/7/8 list test cases in prose; cut each to the function name plus one strategy line per risky function
