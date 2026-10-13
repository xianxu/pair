---
gate: plan-quality
issue: 506
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T17:18:43-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Per-link viewer counts (Viewers, couch --broadcast-list) don't say where pointer-link streams go
          detail: events picks viewViewers or driveViewers by a drive bool (server.go:152-156), and Server.Viewers returns (view, drive) for couch --broadcast-list (console_drive.go:253). The plan passes link to events but doesn't say whether pointer streams count as view or get a third counter, which would change Viewers and the list output.
          family: link-model-consumer-unplaced
          round: 1
        - id: PQ-2
          severity: Minor
          title: The plan lists test cases in prose instead of one strategy line per risky function
          detail: Tasks 2, 4, 6 and 8 spell out individual cases and notice strings. Shrink each to one strategy line, e.g. readPalette gets a table test seeded with malformed and legacy shapes, and boardRoute's ordering is proved by a 403-not-413 over-limit body.
          family: test-prose-enumeration
          round: 1
        - id: PQ-3
          severity: Minor
          title: castLink sits beside the terminal cast's own view/pointer classification (broadcast/server.go:87-96)
          detail: Say why the webcast doesn't share broadcast's pointer-link type, or put the grant enum in broadcast so both casts read one model (ARCH-DRY).
          family: reuse-existing-link-classifier
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-10-12T17:20:57-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Revisions: pointer streams count as view; Viewers() and broadcast-list unchanged; Task 1 asserts it.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: One strategy line per risky function added; the case lists are marked as examples of those strategies.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'The reason for not sharing the type is stated (the terminal''s on/off pointer phase, no drive level), with #407 as the trigger to move it.'
          round: 2
      blocked: false
content_hash: 181d16b0b7b8dcc0dcf98f0d313f5c306d8716b7afe7a088be5741360c03ccf1
---

# Gate ledger — pair#506 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T17:18:43-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `link-model-consumer-unplaced` Per-link viewer counts (Viewers, couch --broadcast-list) don't say where pointer-link streams go
  events picks viewViewers or driveViewers by a drive bool (server.go:152-156), and Server.Viewers returns (view, drive) for couch --broadcast-list (console_drive.go:253). The plan passes link to events but doesn't say whether pointer streams count as view or get a third counter, which would change Viewers and the list output.
- **PQ-2** [Minor] `test-prose-enumeration` The plan lists test cases in prose instead of one strategy line per risky function
  Tasks 2, 4, 6 and 8 spell out individual cases and notice strings. Shrink each to one strategy line, e.g. readPalette gets a table test seeded with malformed and legacy shapes, and boardRoute's ordering is proved by a 403-not-413 over-limit body.
- **PQ-3** [Minor] `reuse-existing-link-classifier` castLink sits beside the terminal cast's own view/pointer classification (broadcast/server.go:87-96)
  Say why the webcast doesn't share broadcast's pointer-link type, or put the grant enum in broadcast so both casts read one model (ARCH-DRY).

## Round 2 — 2026-10-12T17:20:57-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Revisions: pointer streams count as view; Viewers() and broadcast-list unchanged; Task 1 asserts it.
- PQ-2 — addressed — One strategy line per risky function added; the case lists are marked as examples of those strategies.
- PQ-3 — addressed — The reason for not sharing the type is stated (the terminal's on/off pointer phase, no drive level), with #407 as the trigger to move it.

## Open findings

(none — every finding has been disposed)
