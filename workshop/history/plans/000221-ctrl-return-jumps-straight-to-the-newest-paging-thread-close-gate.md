---
gate: boundary-review
issue: 221
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-10T13:36:38-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Stay arm (current actor is newest pager) is a visible no-op; its rationale "badge clears" is falsified by the design's own reserve.go:124 finding
          detail: |-
            RenderStatusRow never draws a bell on the active actor, so acknowledge-and-stay changes nothing the operator can see, which is the dropped-keypress look the nothing-paging arm adds a notice to avoid. Reachable only via the focusedAtDelivery snapshot race (console.go:276 vs 1181), hence Minor; either add a brief notice or correct the Spec's rationale.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: silent-noop-arm
          round: 1
        - id: BR-2
          severity: Minor
          title: Panel arm hard-codes onMenuKey(KeyEnter), restating decodeCSIu's current mapping (panelkeys.go:196) instead of forwarding the chord
          detail: |-
            The Spec says ctrl+return is unclaimed in the panel and should stay forwarded. Feed the chord bytes through onMenuInput so the panel decoder remains the single source (ARCH-DRY), or pin DecodePanelKeys of the new sequence equal to KeyEnter in a test so the two cannot drift.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: derive-dont-restate
          round: 1
        - id: BR-3
          severity: Minor
          title: README sentences that enumerate couch's intercepted chords go stale beyond the one-key-plus-Enter line the plan names
          detail: |-
            README.md:378-381 (Ctrl-Space and Ctrl-Backspace belong to couch, intercepted in both encodings), 384-385 (every other chord passes through untouched), 419 (third intercepted chord). Also state in README, not only keys.go, that Ctrl-Return is recognised only under the Kitty protocol.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: doc-sweep-enumeration
          round: 1
        - id: BR-4
          severity: Minor
          title: Test steps enumerate cases in prose; compress to function plus strategy line and fix the negative-neighbour class
          detail: |-
            TestInterceptorRecognisesEverySequenceAtEverySplit (keys_test.go:249) and TestEveryInterceptedChordHasAHandler already walk knownSequences, so splits and handler coverage come free. Negative neighbours should be every codepoint-13 encoding Pair consumes, which adds the missing 13;1u (wrap.go:1322) and 13;4u (shortcut.go:375). Seed FuzzInterceptorFeed (keys_test.go:165) with the row and its neighbours.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: test-prose-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-10T13:36:38-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Minor
          title: arrivalNotification and SwitchTracker.Switch docs still define a notification hop as only ctrl-space + Return
          detail: console.go:447-448 and switchrule.go:30-31 state an exclusive definition that ctrl+return (console.go:1464) now breaks. The README chord listings were swept but these arrival-kind definitions were not. Say "ctrl-space + Return or ctrl+return", or "an arrival on an actor that was paging when chosen".
          family: doc-sweep-enumeration
          round: 2
        - id: BR-6
          severity: Minor
          title: The issue's ARCH-ORDER note says attention.Mark runs only on the Run goroutine, which is false
          detail: switchTo also runs on the operation goroutine via ExecuteConsoleOperation's switch (console.go:1940). It calls flushDeferredNotifications (console.go:504), which calls onChunk and then Mark (console.go:1271). The effect on ctrl+return is harmless (it lands on a thread that is still paging); correct the stated reason in a Revisions entry.
          family: false-rationale
          round: 2
        - id: BR-7
          severity: Minor
          title: The stay notice repeats switchTo's own "already" check under a separate lock
          detail: stay (console.go:1435) is computed before the lock is released, and switchTo recomputes already (console.go:474). A status-chip switch on the operation goroutine between the two can make the notice wrong or missing. Have switchTo report whether it landed or stayed (ARCH-DRY/ARCH-ORDER).
          family: single-authority-predicate
          round: 2
        - id: BR-8
          severity: Minor
          title: The one-line screen jump on a thread switch is "left for its own issue", but no issue exists
          detail: 'The Log attributes it to #209''s repaint nudge (RepaintSettle); #224 covers the single-writer door, not this. File it with sdlc issue new so it is not lost.'
          family: discovered-defect-untracked
          round: 2
      blocked: false
---

# Gate ledger — pair#221 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T13:36:38-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `silent-noop-arm` Stay arm (current actor is newest pager) is a visible no-op; its rationale "badge clears" is falsified by the design's own reserve.go:124 finding
  RenderStatusRow never draws a bell on the active actor, so acknowledge-and-stay changes nothing the operator can see, which is the dropped-keypress look the nothing-paging arm adds a notice to avoid. Reachable only via the focusedAtDelivery snapshot race (console.go:276 vs 1181), hence Minor; either add a brief notice or correct the Spec's rationale.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `derive-dont-restate` Panel arm hard-codes onMenuKey(KeyEnter), restating decodeCSIu's current mapping (panelkeys.go:196) instead of forwarding the chord
  The Spec says ctrl+return is unclaimed in the panel and should stay forwarded. Feed the chord bytes through onMenuInput so the panel decoder remains the single source (ARCH-DRY), or pin DecodePanelKeys of the new sequence equal to KeyEnter in a test so the two cannot drift.
  (carried from plan-quality PQ-2, deferred to the boundary review)
- **BR-3** [Minor] `doc-sweep-enumeration` README sentences that enumerate couch's intercepted chords go stale beyond the one-key-plus-Enter line the plan names
  README.md:378-381 (Ctrl-Space and Ctrl-Backspace belong to couch, intercepted in both encodings), 384-385 (every other chord passes through untouched), 419 (third intercepted chord). Also state in README, not only keys.go, that Ctrl-Return is recognised only under the Kitty protocol.
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-4** [Minor] `test-prose-enumeration` Test steps enumerate cases in prose; compress to function plus strategy line and fix the negative-neighbour class
  TestInterceptorRecognisesEverySequenceAtEverySplit (keys_test.go:249) and TestEveryInterceptedChordHasAHandler already walk knownSequences, so splits and handler coverage come free. Negative neighbours should be every codepoint-13 encoding Pair consumes, which adds the missing 13;1u (wrap.go:1322) and 13;4u (shortcut.go:375). Seed FuzzInterceptorFeed (keys_test.go:165) with the row and its neighbours.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-09-10T13:36:38-07:00 (claude) — passed

### Raised

- **BR-5** [Minor] `doc-sweep-enumeration` arrivalNotification and SwitchTracker.Switch docs still define a notification hop as only ctrl-space + Return
  console.go:447-448 and switchrule.go:30-31 state an exclusive definition that ctrl+return (console.go:1464) now breaks. The README chord listings were swept but these arrival-kind definitions were not. Say "ctrl-space + Return or ctrl+return", or "an arrival on an actor that was paging when chosen".
- **BR-6** [Minor] `false-rationale` The issue's ARCH-ORDER note says attention.Mark runs only on the Run goroutine, which is false
  switchTo also runs on the operation goroutine via ExecuteConsoleOperation's switch (console.go:1940). It calls flushDeferredNotifications (console.go:504), which calls onChunk and then Mark (console.go:1271). The effect on ctrl+return is harmless (it lands on a thread that is still paging); correct the stated reason in a Revisions entry.
- **BR-7** [Minor] `single-authority-predicate` The stay notice repeats switchTo's own "already" check under a separate lock
  stay (console.go:1435) is computed before the lock is released, and switchTo recomputes already (console.go:474). A status-chip switch on the operation goroutine between the two can make the notice wrong or missing. Have switchTo report whether it landed or stayed (ARCH-DRY/ARCH-ORDER).
- **BR-8** [Minor] `discovered-defect-untracked` The one-line screen jump on a thread switch is "left for its own issue", but no issue exists
  The Log attributes it to #209's repaint nudge (RepaintSettle); #224 covers the single-writer door, not this. File it with sdlc issue new so it is not lost.

## Open findings

- **BR-1** [Minor] `silent-noop-arm` Stay arm (current actor is newest pager) is a visible no-op; its rationale "badge clears" is falsified by the design's own reserve.go:124 finding
- **BR-2** [Minor] `derive-dont-restate` Panel arm hard-codes onMenuKey(KeyEnter), restating decodeCSIu's current mapping (panelkeys.go:196) instead of forwarding the chord
- **BR-3** [Minor] `doc-sweep-enumeration` README sentences that enumerate couch's intercepted chords go stale beyond the one-key-plus-Enter line the plan names
- **BR-4** [Minor] `test-prose-enumeration` Test steps enumerate cases in prose; compress to function plus strategy line and fix the negative-neighbour class
- **BR-5** [Minor] `doc-sweep-enumeration` arrivalNotification and SwitchTracker.Switch docs still define a notification hop as only ctrl-space + Return
- **BR-6** [Minor] `false-rationale` The issue's ARCH-ORDER note says attention.Mark runs only on the Run goroutine, which is false
- **BR-7** [Minor] `single-authority-predicate` The stay notice repeats switchTo's own "already" check under a separate lock
- **BR-8** [Minor] `discovered-defect-untracked` The one-line screen jump on a thread switch is "left for its own issue", but no issue exists
