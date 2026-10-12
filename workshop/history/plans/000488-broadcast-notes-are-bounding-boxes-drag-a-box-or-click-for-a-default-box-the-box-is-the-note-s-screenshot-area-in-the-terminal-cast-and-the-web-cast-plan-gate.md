---
gate: plan-quality
issue: 488
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T11:21:54-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: 'M3 pane-rect cache: 300 ms timeout is below list-panes'' measured 590 ms, and it misses layout changes inside the session'
          detail: layoutcmd/layoutcmd.go:55 and termcmd/run.go:678 record that list-panes --json costs 590 ms on zellij 0.44.3, so a 300 ms timeout always degrades the anchor to screen and spec 4b is never met. Refreshing only on slot switch or resize misses review-pane, terminal-split and draft-minimize changes inside the session, so stale rects silently put the wrong pane name on notes; the geometry epoch catches only outer resizes. The plan also names layoutcmd's pair-side Runtime as the seam, but couchtty calls no zellij actions today. Set the budget from the measured cost, name a freshness check or trigger, and name Couch's actual seam.
          family: unbacked-external-cost-assumption
          round: 1
        - id: PQ-2
          severity: Minor
          title: The plan states no non-goals, such as not building the 455/483 submission and not persisting terminal anchors
          family: missing-non-goals
          round: 1
        - id: PQ-3
          severity: Minor
          title: No adversarial test strategy for the stricter box parsers and the view.json reader
          detail: 'Add one line per parser: fuzz it, seeded with truncated, negative, oversized and off-grid boxes and with malformed view.json.'
          family: test-strategy-untrusted-parsers
          round: 1
        - id: PQ-4
          severity: Minor
          title: A derived top-right pin can land in guarded cells (marks.go:62), which refuses an otherwise valid box
          detail: Say the pin moves out of the guard instead of the request being refused.
          family: derived-value-validation-refuses-input
          round: 1
        - id: PQ-5
          severity: Minor
          title: The list-panes fake is static JSON fixtures, with no conformance check on the geometry shape
          family: stateful-fake-missing
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T11:23:18-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: No cache; async single-flight fresh resolve, 2 s budget from the measured 590 ms, epoch and generation checks at apply, injected PaneRects seam in couchcmd (verified shortcut_focus.go:54). The issue's M3 checklist row still says rect cache; fix the wording.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: "Non-goals: no 455/483 submission, no persisted terminal anchors, no boxes in the \U0001F4ADN menu."
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Fuzz lines for broadcast.ParseNote, board.ParseNote and the view.json reader, plus error-not-fabrication.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: A pin on a guarded cell moves left, then down; refused only with no unguarded spot; table-tested.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Stateful layout fake behind the PaneRects function type, plus a live conformance check skipped without zellij.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T11:24:30-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
content_hash: edcbc32350506ccb025afc34db71afb859e572ae4cc423216addd08e666ed4b6
---

# Gate ledger — pair#488 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T11:21:54-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unbacked-external-cost-assumption` M3 pane-rect cache: 300 ms timeout is below list-panes' measured 590 ms, and it misses layout changes inside the session
  layoutcmd/layoutcmd.go:55 and termcmd/run.go:678 record that list-panes --json costs 590 ms on zellij 0.44.3, so a 300 ms timeout always degrades the anchor to screen and spec 4b is never met. Refreshing only on slot switch or resize misses review-pane, terminal-split and draft-minimize changes inside the session, so stale rects silently put the wrong pane name on notes; the geometry epoch catches only outer resizes. The plan also names layoutcmd's pair-side Runtime as the seam, but couchtty calls no zellij actions today. Set the budget from the measured cost, name a freshness check or trigger, and name Couch's actual seam.
- **PQ-2** [Minor] `missing-non-goals` The plan states no non-goals, such as not building the 455/483 submission and not persisting terminal anchors
- **PQ-3** [Minor] `test-strategy-untrusted-parsers` No adversarial test strategy for the stricter box parsers and the view.json reader
  Add one line per parser: fuzz it, seeded with truncated, negative, oversized and off-grid boxes and with malformed view.json.
- **PQ-4** [Minor] `derived-value-validation-refuses-input` A derived top-right pin can land in guarded cells (marks.go:62), which refuses an otherwise valid box
  Say the pin moves out of the guard instead of the request being refused.
- **PQ-5** [Minor] `stateful-fake-missing` The list-panes fake is static JSON fixtures, with no conformance check on the geometry shape

## Round 2 — 2026-10-12T11:23:18-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — No cache; async single-flight fresh resolve, 2 s budget from the measured 590 ms, epoch and generation checks at apply, injected PaneRects seam in couchcmd (verified shortcut_focus.go:54). The issue's M3 checklist row still says rect cache; fix the wording.
- PQ-2 — addressed — Non-goals: no 455/483 submission, no persisted terminal anchors, no boxes in the 💭N menu.
- PQ-3 — addressed — Fuzz lines for broadcast.ParseNote, board.ParseNote and the view.json reader, plus error-not-fabrication.
- PQ-4 — addressed — A pin on a guarded cell moves left, then down; refused only with no unguarded spot; table-tested.
- PQ-5 — addressed — Stateful layout fake behind the PaneRects function type, plus a live conformance check skipped without zellij.

## Round 3 — 2026-10-12T11:24:30-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)
