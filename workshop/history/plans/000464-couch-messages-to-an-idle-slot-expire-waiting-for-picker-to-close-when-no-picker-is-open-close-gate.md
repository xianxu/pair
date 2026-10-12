---
gate: boundary-review
issue: 464
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T00:39:10-07:00"
      dispatched: "2026-10-12T00:36:23-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Atlas still says dismissing an overlay without Enter leaves the latch set, and calls the two layers independent
          detail: 'atlas/architecture.md:743 "Known edge inherited from the one-shot overlay design: dismissing an overlay without Enter leaves the flag set until the next plain Enter" is now false for claude; atlas/architecture.md:739 "Two independent layers" no longer holds for claude because the overlay close now reads the composer recognizer. These are the only two instances in the window. Scope both to profiles without overlayHidesComposer.'
          family: doc-claim-drift
          round: 1
        - id: BR-2
          severity: Important
          title: The Spec's sender-actionable undeliverable reason was neither delivered nor descoped
          detail: peer_delivery.go:456 still reports "waiting for picker to close" with no next step for the sender, and every non-claude profile can still produce a stale latch. Either word it so the sender knows what to do (answer the picker in the pane) or add a plan Revisions entry descoping the clause.
          family: spec-clause-undelivered
          round: 1
        - id: BR-3
          severity: Minor
          title: Done-when live check deferred to the TL and unverified at close
          detail: The plan item is unchecked by design; make sure the TL's post-relaunch check is tracked somewhere so it is not lost when the issue closes.
          family: done-when-unverified-at-close
          round: 1
        - id: BR-4
          severity: Minor
          title: Screen close couples the overlay latch to the composer recognizer with no picker-with-text-input fixture
          detail: If claudeComposerActive reads a picker's free-text field (permission "No, and tell Claude…", AskUserQuestion "Type something") as a composer, the latch clears while the picker is open and both defensive layers fail together. Add such a fixture asserting the latch holds.
          family: layer-coupling-unfixtured
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T00:44:01-07:00"
      dispatched: "2026-10-12T00:41:06-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: atlas/architecture.md:739 now says the layers are independent except for Claude's overlay close and names the test; :743 limits the stale-flag edge to profiles without overlayHidesComposer. Both instances fixed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Descoped in the plan''s Revisions (2026-10-12, at close) to #465, whose Spec owns "blocked by a picker: <title>" receipts and the hub notice. Checked in workshop/issues/000465 lines 21-25.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: The Plan item records the owner (TL, ops:0), the trigger (next pair:0 build, Couch restart, slot relaunch) and the repro. It stays unchecked by design.
          round: 2
        - id: BR-4
          disposition: addressed
          note: TestPickerLatchHoldsOverQuestionPickerTextRow re-feeds the recorded question-picker frame after arming and asserts the latch holds and the composer reads Unknown; if the recognizer accepted that frame the test would fail. My split probe also found no premature clear.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: ae4e297f1eabb62c974344517709a4b0f27419b3
      blocked: false
---

# Gate ledger — pair#464 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T00:39:10-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `doc-claim-drift` Atlas still says dismissing an overlay without Enter leaves the latch set, and calls the two layers independent
  atlas/architecture.md:743 "Known edge inherited from the one-shot overlay design: dismissing an overlay without Enter leaves the flag set until the next plain Enter" is now false for claude; atlas/architecture.md:739 "Two independent layers" no longer holds for claude because the overlay close now reads the composer recognizer. These are the only two instances in the window. Scope both to profiles without overlayHidesComposer.
- **BR-2** [Important] `spec-clause-undelivered` The Spec's sender-actionable undeliverable reason was neither delivered nor descoped
  peer_delivery.go:456 still reports "waiting for picker to close" with no next step for the sender, and every non-claude profile can still produce a stale latch. Either word it so the sender knows what to do (answer the picker in the pane) or add a plan Revisions entry descoping the clause.
- **BR-3** [Minor] `done-when-unverified-at-close` Done-when live check deferred to the TL and unverified at close
  The plan item is unchecked by design; make sure the TL's post-relaunch check is tracked somewhere so it is not lost when the issue closes.
- **BR-4** [Minor] `layer-coupling-unfixtured` Screen close couples the overlay latch to the composer recognizer with no picker-with-text-input fixture
  If claudeComposerActive reads a picker's free-text field (permission "No, and tell Claude…", AskUserQuestion "Type something") as a composer, the latch clears while the picker is open and both defensive layers fail together. Add such a fixture asserting the latch holds.

## Round 2 — 2026-10-12T00:44:01-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — atlas/architecture.md:739 now says the layers are independent except for Claude's overlay close and names the test; :743 limits the stale-flag edge to profiles without overlayHidesComposer. Both instances fixed.
- BR-2 — addressed — Descoped in the plan's Revisions (2026-10-12, at close) to #465, whose Spec owns "blocked by a picker: <title>" receipts and the hub notice. Checked in workshop/issues/000465 lines 21-25.
- BR-3 — addressed — The Plan item records the owner (TL, ops:0), the trigger (next pair:0 build, Couch restart, slot relaunch) and the repro. It stays unchecked by design.
- BR-4 — addressed — TestPickerLatchHoldsOverQuestionPickerTextRow re-feeds the recorded question-picker frame after arming and asserts the latch holds and the composer reads Unknown; if the recognizer accepted that frame the test would fail. My split probe also found no premature clear.

## Open findings

(none — every finding has been disposed)
