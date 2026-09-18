---
gate: boundary-review
issue: 283
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-18T09:20:52-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Plan carries full test and implementation code instead of one strategy line per risky function
          detail: |-
            The code is accurate against the tree and follows the writing-plans template, but it goes stale once written. For the DECSCUSR handler the needed line is: untrusted params (absent, 0..6, >6, huge) -> table over the vt handler + endpoint replay at every byte split.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: plan-restates-diff
          round: 1
        - id: BR-2
          severity: Minor
          title: Acceptance test never releases the presenter, so its run goroutine outlives each case
          detail: |-
            NewPresenter spawns go p.run() (presenter.go:69) and TestChildCursorStyleReachesParentVerbatim never Releases it; e.Close is not deferred either. Snapshot parent.Bytes() first, then tear down in a defer.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: test-spawned-work-unbounded-extent
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-18T09:20:52-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Minor
          title: Operator-smoke checkbox claims a plain shell under couch was checked, but the Log says only the switcher caret was
          detail: The issue Plan ticks the plain-shell-shows-a-bar row and Done-when names the same check. The Log says the smoke covered the couch switcher panel caret, not a plain-shell pane. Either run the plain-shell check, or reword the checkbox and Done-when and add a Revisions entry.
          family: checkbox-claims-exceed-evidence
          round: 2
        - id: BR-4
          severity: Minor
          title: vt reports the default cursor with blink=true, while the frame requires Blink=false for it
          detail: The vt handler stores the default as Steady=false, so the CursorStyle callback and the qualifier raw observation report blink=true, while Frame.Validate refuses a blinking default. No pair code registers the callback today; a future consumer must apply the endpoint rule instead of trusting blink.
          family: default-blink-representation-per-layer
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#283 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T09:20:52-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-restates-diff` Plan carries full test and implementation code instead of one strategy line per risky function
  The code is accurate against the tree and follows the writing-plans template, but it goes stale once written. For the DECSCUSR handler the needed line is: untrusted params (absent, 0..6, >6, huge) -> table over the vt handler + endpoint replay at every byte split.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `test-spawned-work-unbounded-extent` Acceptance test never releases the presenter, so its run goroutine outlives each case
  NewPresenter spawns go p.run() (presenter.go:69) and TestChildCursorStyleReachesParentVerbatim never Releases it; e.Close is not deferred either. Snapshot parent.Bytes() first, then tear down in a defer.
  (carried from plan-quality PQ-2, deferred to the boundary review)

## Round 2 — 2026-09-18T09:20:52-07:00 (claude) — passed

### Raised

- **BR-3** [Minor] `checkbox-claims-exceed-evidence` Operator-smoke checkbox claims a plain shell under couch was checked, but the Log says only the switcher caret was
  The issue Plan ticks the plain-shell-shows-a-bar row and Done-when names the same check. The Log says the smoke covered the couch switcher panel caret, not a plain-shell pane. Either run the plain-shell check, or reword the checkbox and Done-when and add a Revisions entry.
- **BR-4** [Minor] `default-blink-representation-per-layer` vt reports the default cursor with blink=true, while the frame requires Blink=false for it
  The vt handler stores the default as Steady=false, so the CursorStyle callback and the qualifier raw observation report blink=true, while Frame.Validate refuses a blinking default. No pair code registers the callback today; a future consumer must apply the endpoint rule instead of trusting blink.

## Open findings

- **BR-1** [Minor] `plan-restates-diff` Plan carries full test and implementation code instead of one strategy line per risky function
- **BR-2** [Minor] `test-spawned-work-unbounded-extent` Acceptance test never releases the presenter, so its run goroutine outlives each case
- **BR-3** [Minor] `checkbox-claims-exceed-evidence` Operator-smoke checkbox claims a plain shell under couch was checked, but the Log says only the switcher caret was
- **BR-4** [Minor] `default-blink-representation-per-layer` vt reports the default cursor with blink=true, while the frame requires Blink=false for it
