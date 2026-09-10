---
gate: plan-quality
issue: 221
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-10T10:43:33-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Stay arm (current actor is newest pager) is a visible no-op; its rationale "badge clears" is falsified by the design's own reserve.go:124 finding
          detail: RenderStatusRow never draws a bell on the active actor, so acknowledge-and-stay changes nothing the operator can see, which is the dropped-keypress look the nothing-paging arm adds a notice to avoid. Reachable only via the focusedAtDelivery snapshot race (console.go:276 vs 1181), hence Minor; either add a brief notice or correct the Spec's rationale.
          family: silent-noop-arm
          round: 1
        - id: PQ-2
          severity: Minor
          title: Panel arm hard-codes onMenuKey(KeyEnter), restating decodeCSIu's current mapping (panelkeys.go:196) instead of forwarding the chord
          detail: The Spec says ctrl+return is unclaimed in the panel and should stay forwarded. Feed the chord bytes through onMenuInput so the panel decoder remains the single source (ARCH-DRY), or pin DecodePanelKeys of the new sequence equal to KeyEnter in a test so the two cannot drift.
          family: derive-dont-restate
          round: 1
        - id: PQ-3
          severity: Minor
          title: README sentences that enumerate couch's intercepted chords go stale beyond the one-key-plus-Enter line the plan names
          detail: README.md:378-381 (Ctrl-Space and Ctrl-Backspace belong to couch, intercepted in both encodings), 384-385 (every other chord passes through untouched), 419 (third intercepted chord). Also state in README, not only keys.go, that Ctrl-Return is recognised only under the Kitty protocol.
          family: doc-sweep-enumeration
          round: 1
        - id: PQ-4
          severity: Minor
          title: Test steps enumerate cases in prose; compress to function plus strategy line and fix the negative-neighbour class
          detail: TestInterceptorRecognisesEverySequenceAtEverySplit (keys_test.go:249) and TestEveryInterceptedChordHasAHandler already walk knownSequences, so splits and handler coverage come free. Negative neighbours should be every codepoint-13 encoding Pair consumes, which adds the missing 13;1u (wrap.go:1322) and 13;4u (shortcut.go:375). Seed FuzzInterceptorFeed (keys_test.go:165) with the row and its neighbours.
          family: test-prose-enumeration
          round: 1
      blocked: false
content_hash: 85f3c8a5191b157fd28bfe307a3b67ef515e4b1ac72ed3f1b1f7ad54c8ec235e
---

# Gate ledger — pair#221 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T10:43:33-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `silent-noop-arm` Stay arm (current actor is newest pager) is a visible no-op; its rationale "badge clears" is falsified by the design's own reserve.go:124 finding
  RenderStatusRow never draws a bell on the active actor, so acknowledge-and-stay changes nothing the operator can see, which is the dropped-keypress look the nothing-paging arm adds a notice to avoid. Reachable only via the focusedAtDelivery snapshot race (console.go:276 vs 1181), hence Minor; either add a brief notice or correct the Spec's rationale.
- **PQ-2** [Minor] `derive-dont-restate` Panel arm hard-codes onMenuKey(KeyEnter), restating decodeCSIu's current mapping (panelkeys.go:196) instead of forwarding the chord
  The Spec says ctrl+return is unclaimed in the panel and should stay forwarded. Feed the chord bytes through onMenuInput so the panel decoder remains the single source (ARCH-DRY), or pin DecodePanelKeys of the new sequence equal to KeyEnter in a test so the two cannot drift.
- **PQ-3** [Minor] `doc-sweep-enumeration` README sentences that enumerate couch's intercepted chords go stale beyond the one-key-plus-Enter line the plan names
  README.md:378-381 (Ctrl-Space and Ctrl-Backspace belong to couch, intercepted in both encodings), 384-385 (every other chord passes through untouched), 419 (third intercepted chord). Also state in README, not only keys.go, that Ctrl-Return is recognised only under the Kitty protocol.
- **PQ-4** [Minor] `test-prose-enumeration` Test steps enumerate cases in prose; compress to function plus strategy line and fix the negative-neighbour class
  TestInterceptorRecognisesEverySequenceAtEverySplit (keys_test.go:249) and TestEveryInterceptedChordHasAHandler already walk knownSequences, so splits and handler coverage come free. Negative neighbours should be every codepoint-13 encoding Pair consumes, which adds the missing 13;1u (wrap.go:1322) and 13;4u (shortcut.go:375). Seed FuzzInterceptorFeed (keys_test.go:165) with the row and its neighbours.

## Open findings

- **PQ-1** [Minor] `silent-noop-arm` Stay arm (current actor is newest pager) is a visible no-op; its rationale "badge clears" is falsified by the design's own reserve.go:124 finding
- **PQ-2** [Minor] `derive-dont-restate` Panel arm hard-codes onMenuKey(KeyEnter), restating decodeCSIu's current mapping (panelkeys.go:196) instead of forwarding the chord
- **PQ-3** [Minor] `doc-sweep-enumeration` README sentences that enumerate couch's intercepted chords go stale beyond the one-key-plus-Enter line the plan names
- **PQ-4** [Minor] `test-prose-enumeration` Test steps enumerate cases in prose; compress to function plus strategy line and fix the negative-neighbour class
