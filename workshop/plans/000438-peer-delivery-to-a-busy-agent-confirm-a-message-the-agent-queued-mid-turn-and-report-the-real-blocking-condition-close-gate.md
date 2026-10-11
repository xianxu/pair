---
gate: boundary-review
issue: 438
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T01:55:02-07:00"
      dispatched: "2026-10-11T01:52:18-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: peerConfirmEvidence doc says the last-seen reading; the expiry now reports a ranked history
          detail: 'peer_delivery.go:639 says the uncertain receipt reports the composer and turn as last seen. waitSummaryLocked replaces that with the longest waits, can drop the final reading (under 1s, or below third), and joins with the same separator the evidence uses internally. Other instances in this window: atlas/couch.md:355 does not mention the top-3 cap or the 1s floor.'
          family: doc-claim-matches-code
          round: 1
        - id: BR-2
          severity: Minor
          title: Atlas and Plan say Codex frames repaints and queues mid-turn input; only Claude Code was measured
          detail: atlas/couch.md:304 and the Plan's policy bullet both name Codex, but there is no Codex recording or fixture. Either measure it or scope the wording to Claude Code. The safe fallback (an unframed stream keeps the old rule) limits the impact.
          family: unmeasured-claim
          round: 1
        - id: BR-3
          severity: Minor
          title: RIS does not clear the framed and frameOpen flags
          detail: handleESC resets visibility on ESC c but leaves the frame state. After a reset, or a hand-off to a program that does not frame, output in flight still counts as complete. Low risk.
          family: terminal-reset-clears-mode-state
          round: 1
      recipe: small-diff-review
      reviewed: 9f2c7f0b3e1fcddd02cadb4b2c0102c5fd538def
      blocked: false
    - "n": 2
      timestamp: "2026-10-11T01:57:24-07:00"
      dispatched: "2026-10-11T01:57:03-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: peerConfirmEvidence doc now says expiry ranks it; waitSummaryLocked doc states top-3/1s floor/fallback; joiner is " | " (no collision with evidence's ";"), pinned by the updated test; atlas:357-358 mentions cap and floor.
          round: 2
        - id: BR-2
          disposition: addressed
          note: atlas/couch.md:304-309 and the Plan bullet now say Claude Code was measured and Codex's queueing was not.
          round: 2
        - id: BR-3
          disposition: addressed
          note: handleESC clears framed/frameOpen on ESC c (terminal_model.go:265); the reset row in TestTerminalModelFrameComplete goes red without it.
          round: 2
      recipe: fix-review
      reviewed: de62fe63fe6f1030609437c55db3b8b33251191d
      blocked: false
---

# Gate ledger — pair#438 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T01:55:02-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `doc-claim-matches-code` peerConfirmEvidence doc says the last-seen reading; the expiry now reports a ranked history
  peer_delivery.go:639 says the uncertain receipt reports the composer and turn as last seen. waitSummaryLocked replaces that with the longest waits, can drop the final reading (under 1s, or below third), and joins with the same separator the evidence uses internally. Other instances in this window: atlas/couch.md:355 does not mention the top-3 cap or the 1s floor.
- **BR-2** [Minor] `unmeasured-claim` Atlas and Plan say Codex frames repaints and queues mid-turn input; only Claude Code was measured
  atlas/couch.md:304 and the Plan's policy bullet both name Codex, but there is no Codex recording or fixture. Either measure it or scope the wording to Claude Code. The safe fallback (an unframed stream keeps the old rule) limits the impact.
- **BR-3** [Minor] `terminal-reset-clears-mode-state` RIS does not clear the framed and frameOpen flags
  handleESC resets visibility on ESC c but leaves the frame state. After a reset, or a hand-off to a program that does not frame, output in flight still counts as complete. Low risk.

## Round 2 — 2026-10-11T01:57:24-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — peerConfirmEvidence doc now says expiry ranks it; waitSummaryLocked doc states top-3/1s floor/fallback; joiner is " | " (no collision with evidence's ";"), pinned by the updated test; atlas:357-358 mentions cap and floor.
- BR-2 — addressed — atlas/couch.md:304-309 and the Plan bullet now say Claude Code was measured and Codex's queueing was not.
- BR-3 — addressed — handleESC clears framed/frameOpen on ESC c (terminal_model.go:265); the reset row in TestTerminalModelFrameComplete goes red without it.

## Open findings

(none — every finding has been disposed)
