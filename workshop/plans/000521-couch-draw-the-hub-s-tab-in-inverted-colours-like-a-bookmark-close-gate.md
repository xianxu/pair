---
gate: boundary-review
issue: 521
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T22:29:28-07:00"
      dispatched: "2026-10-12T22:28:00-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Done when does not cover the expanded rendering contract
          detail: workshop/issues/000521-couch-draw-the-hub-s-tab-in-inverted-colours-like-a-bookmark.md:53 still names only tab-bar acceptance. Enumerated gaps are switcher main-row inversion, selected-hub arrow/bold and blocked-red preservation, unchanged attention-detail styling, and mandatory live approval superseded by the recorded waiver. Update Done when to cover all these contract changes and record the synchronization in Revisions.
          family: acceptance-criteria-track-contract
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T22:32:44-07:00"
      dispatched: "2026-10-12T22:31:36-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The issue's Done when at lines 53–55 now covers both rendering surfaces, selected arrow/bold and blocked red, unchanged attention details, containment, and the recorded smoke waiver. Revisions at lines 101–104 records this synchronization. These criteria match the approved composition at lines 32–49 and the implementation in reserve.go:366 and menu_render.go:618; this is a prose-only correction.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 23fb1c94d4f03d07b2844bd359be561d472a288f
      blocked: false
---

# Gate ledger — pair#521 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T22:29:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `acceptance-criteria-track-contract` Done when does not cover the expanded rendering contract
  workshop/issues/000521-couch-draw-the-hub-s-tab-in-inverted-colours-like-a-bookmark.md:53 still names only tab-bar acceptance. Enumerated gaps are switcher main-row inversion, selected-hub arrow/bold and blocked-red preservation, unchanged attention-detail styling, and mandatory live approval superseded by the recorded waiver. Update Done when to cover all these contract changes and record the synchronization in Revisions.

## Round 2 — 2026-10-12T22:32:44-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — The issue's Done when at lines 53–55 now covers both rendering surfaces, selected arrow/bold and blocked red, unchanged attention details, containment, and the recorded smoke waiver. Revisions at lines 101–104 records this synchronization. These criteria match the approved composition at lines 32–49 and the implementation in reserve.go:366 and menu_render.go:618; this is a prose-only correction.

## Open findings

(none — every finding has been disposed)
