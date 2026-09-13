---
gate: boundary-review
issue: 227
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-13T09:50:06-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Task 4 table oracle says "iff !IsGlobalChord", contradicting the M-k exclusion
          detail: |-
            M-k is non-global yet must not be written to the child. The oracle should be RightTerminalChordPassesThrough itself, so the test cannot diverge from the exclusion set.
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: test-oracle-derives-from-classifier
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-13T09:50:06-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: pair keys "Terminal tabs" heading claims passthrough for Alt+k and Shift+Alt+←/→, which never pass through
          detail: catalog.go:10 heading applies to the whole group, which includes Alt+k (catalog.go:76, the PQ-1 exclusion) and the global Shift+Alt+←/→ (catalog.go:92-93). The M-k row's Help is unchanged, so nothing in the help says it survives. Derive a per-row note from RightTerminalChordPassesThrough in sections.go:49-53, or reword the heading to name the exceptions.
          family: help-derives-from-classifier
          round: 2
        - id: BR-3
          severity: Important
          title: README update appears missing for right-terminal chords becoming conditional under a full-screen app
          detail: README.md:15 and README.md:118-122 still document Alt+t/w/r, Alt+Shift+d, Alt+←/→ and Alt+Shift+Return as unconditional in the layout-3 terminal. Add one sentence to the layout-3 blurb and a scope note on those rows.
          family: readme-tracks-user-surface
          round: 2
        - id: BR-4
          severity: Important
          title: Live verification of the issue's purpose (parley M-t under nvim) is unticked and unlogged
          detail: Issue Plan's Manual row is unchecked and the Log has no 2026-09-13 entry; the escsmoke evidence exists only in commit 3c5a6691's body. Run the operator smoke test (M-t under nvim+parley, M-t at a shell), paste the escsmoke output into the Log, tick the row. Do not bypass plan-unchecked for this row.
          family: purpose-verified-live-and-logged
          round: 2
        - id: BR-5
          severity: Minor
          title: activeChildOwnsScreen repeats appMouseMode's lock/activeTabLocked/nil-check triple
          detail: run.go:1437 vs run.go:1424; an activeChild() helper would collapse both child-mode readers (ARCH-DRY).
          family: shared-accessor-helper
          round: 2
        - id: BR-6
          severity: Minor
          title: Durable plan's task checkboxes are all unticked while the issue Plan is ticked
          detail: workshop/plans/000227-...-plan.md steps remain "- [ ]"; tick them or note the issue Plan is authoritative before archival.
          family: plan-artifact-tracks-progress
          round: 2
      blocked: true
---

# Gate ledger — pair#227 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T09:50:06-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-oracle-derives-from-classifier` Task 4 table oracle says "iff !IsGlobalChord", contradicting the M-k exclusion
  M-k is non-global yet must not be written to the child. The oracle should be RightTerminalChordPassesThrough itself, so the test cannot diverge from the exclusion set.
  (carried from plan-quality PQ-5, deferred to the boundary review)

## Round 2 — 2026-09-13T09:50:06-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `help-derives-from-classifier` pair keys "Terminal tabs" heading claims passthrough for Alt+k and Shift+Alt+←/→, which never pass through
  catalog.go:10 heading applies to the whole group, which includes Alt+k (catalog.go:76, the PQ-1 exclusion) and the global Shift+Alt+←/→ (catalog.go:92-93). The M-k row's Help is unchanged, so nothing in the help says it survives. Derive a per-row note from RightTerminalChordPassesThrough in sections.go:49-53, or reword the heading to name the exceptions.
- **BR-3** [Important] `readme-tracks-user-surface` README update appears missing for right-terminal chords becoming conditional under a full-screen app
  README.md:15 and README.md:118-122 still document Alt+t/w/r, Alt+Shift+d, Alt+←/→ and Alt+Shift+Return as unconditional in the layout-3 terminal. Add one sentence to the layout-3 blurb and a scope note on those rows.
- **BR-4** [Important] `purpose-verified-live-and-logged` Live verification of the issue's purpose (parley M-t under nvim) is unticked and unlogged
  Issue Plan's Manual row is unchecked and the Log has no 2026-09-13 entry; the escsmoke evidence exists only in commit 3c5a6691's body. Run the operator smoke test (M-t under nvim+parley, M-t at a shell), paste the escsmoke output into the Log, tick the row. Do not bypass plan-unchecked for this row.
- **BR-5** [Minor] `shared-accessor-helper` activeChildOwnsScreen repeats appMouseMode's lock/activeTabLocked/nil-check triple
  run.go:1437 vs run.go:1424; an activeChild() helper would collapse both child-mode readers (ARCH-DRY).
- **BR-6** [Minor] `plan-artifact-tracks-progress` Durable plan's task checkboxes are all unticked while the issue Plan is ticked
  workshop/plans/000227-...-plan.md steps remain "- [ ]"; tick them or note the issue Plan is authoritative before archival.

## Open findings

- **BR-1** [Minor] `test-oracle-derives-from-classifier` Task 4 table oracle says "iff !IsGlobalChord", contradicting the M-k exclusion
- **BR-2** [Important] `help-derives-from-classifier` pair keys "Terminal tabs" heading claims passthrough for Alt+k and Shift+Alt+←/→, which never pass through
- **BR-3** [Important] `readme-tracks-user-surface` README update appears missing for right-terminal chords becoming conditional under a full-screen app
- **BR-4** [Important] `purpose-verified-live-and-logged` Live verification of the issue's purpose (parley M-t under nvim) is unticked and unlogged
- **BR-5** [Minor] `shared-accessor-helper` activeChildOwnsScreen repeats appMouseMode's lock/activeTabLocked/nil-check triple
- **BR-6** [Minor] `plan-artifact-tracks-progress` Durable plan's task checkboxes are all unticked while the issue Plan is ticked
