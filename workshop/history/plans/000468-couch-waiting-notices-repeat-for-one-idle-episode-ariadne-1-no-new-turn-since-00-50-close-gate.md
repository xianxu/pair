---
gate: boundary-review
issue: 468
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T02:15:31-07:00"
      dispatched: "2026-10-12T02:14:17-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: A not-waiting verdict overwrites the notified episode, so a brief not-waiting state re-arms the notice
          detail: 'wait_notice.go:333 replaces seen[b] with a judgedNot entry, dropping notified and judged. A slot quiet past the threshold that briefly reads as not waiting (a message in flight, Pending=true), then flips back with the same screen and no turn, sends a second notice. This is the only place in the window that resets the episode. Fix: keep notified and judged on the judgedNot entry.'
          family: episode-identity-lost-on-rejudge
          round: 1
        - id: BR-2
          severity: Minor
          title: The 468 test pushes LastActivity only and leaves out the wrapper's unsettle and re-settle around each burst
          detail: In the real cause, each update check also flips Settled false, then true 3 s later. The test keeps Settled true, so the flip is not pinned. It is harmless today because quiet is under the threshold during the flip.
          family: test-models-partial-signal
          round: 1
        - id: BR-3
          severity: Minor
          title: atlas/couch.md Episode bullet has one overlong reflowed line
          detail: The line starting "pending), so an idle slot costs one tail per output burst. The map is pruned..." is much longer than the lines around it.
          family: doc-reflow
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: 9fa31ab46ccae193bce10d2d7f354a8e69e458bd
      blocked: false
---

# Gate ledger — pair#468 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T02:15:31-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `episode-identity-lost-on-rejudge` A not-waiting verdict overwrites the notified episode, so a brief not-waiting state re-arms the notice
  wait_notice.go:333 replaces seen[b] with a judgedNot entry, dropping notified and judged. A slot quiet past the threshold that briefly reads as not waiting (a message in flight, Pending=true), then flips back with the same screen and no turn, sends a second notice. This is the only place in the window that resets the episode. Fix: keep notified and judged on the judgedNot entry.
- **BR-2** [Minor] `test-models-partial-signal` The 468 test pushes LastActivity only and leaves out the wrapper's unsettle and re-settle around each burst
  In the real cause, each update check also flips Settled false, then true 3 s later. The test keeps Settled true, so the flip is not pinned. It is harmless today because quiet is under the threshold during the flip.
- **BR-3** [Minor] `doc-reflow` atlas/couch.md Episode bullet has one overlong reflowed line
  The line starting "pending), so an idle slot costs one tail per output burst. The map is pruned..." is much longer than the lines around it.

## Open findings

- **BR-1** [Minor] `episode-identity-lost-on-rejudge` A not-waiting verdict overwrites the notified episode, so a brief not-waiting state re-arms the notice
- **BR-2** [Minor] `test-models-partial-signal` The 468 test pushes LastActivity only and leaves out the wrapper's unsettle and re-settle around each burst
- **BR-3** [Minor] `doc-reflow` atlas/couch.md Episode bullet has one overlong reflowed line
