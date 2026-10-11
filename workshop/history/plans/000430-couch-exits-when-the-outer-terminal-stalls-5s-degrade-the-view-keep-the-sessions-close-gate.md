---
gate: boundary-review
issue: 430
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T18:40:45-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: EmitEffects drops non-writing effects and the rest of the batch while stalled, losing attention marks and directory updates
          detail: presenter.go:834 sends s=="" effects (inactive-slot bell, DirectoryEffect, policy-off) through writeUnlessStalled; a failed probe returns ok=false and the loop returns, so p.effects/delivered miss them and couchtty console.go:1259 never marks attention. Skip the outer terminal for empty writes, continue past dropped writes, and add a stalled-batch test.
          family: stall-drop-loses-control-plane-state
          round: 1
        - id: BR-2
          severity: Important
          title: A probe succeeding inside writeUnlessStalled clears stalled without scheduling the full repaint
          detail: After the outer terminal drains, a Copy or effect that probes first sets stalled=false, so armRetry no longer fires; the screen keeps the abandoned mid-frame bytes and unset modes until new child output. Drop without probing while stalled, or keep a repaint-due flag until a paint succeeds; test Copy-after-drain.
          family: stall-recovery-must-repaint
          round: 1
        - id: BR-3
          severity: Minor
          title: Spec's stall notice/diagnostic is reduced to an unconsumed Stalls() counter
          detail: Nothing in production calls Stalls(), so the rollout live check cannot distinguish survived stalls; add a Revisions entry and a follow-up.
          family: spec-diagnostic-deferred
          round: 1
        - id: BR-4
          severity: Minor
          title: stalled/modesKnown/previous form a flag constellation outside the ViewEvent state machine
          family: state-flag-constellation
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-10T18:57:23-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: s=="" skips the terminal, a stalled write returns nil and bookkeeping proceeds (presenter.go:846-852); the stalled-batch test asserts 2 delivered effects.
          round: 2
        - id: BR-2
          disposition: addressed
          note: writeUnlessStalled drops without probing; only a completed paint clears stalled (presenter.go:484-486); Copy-after-drain test asserts recovery begins with stallProbe.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Revisions entry plus Done-when line scope the diagnostic to Presenter.Stalls() and name the follow-up (trace or crash dir).
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Still loose fields; the Log explicitly defers moving parent-screen knowledge into one value; minor, non-blocking.
          round: 2
      findings:
        - id: BR-5
          severity: Minor
          title: A TitleEffect dropped during a stall is never re-emitted on recovery, so the outer title stays stale
          detail: '2nd instance in this family (after the alt-screen kitty-push window). Rule: recovery must restore everything the presenter knows the outer terminal should show (screen, modes, keyboard stack, alt push, title), which is the deferred single parent-screen value that stall() resets whole. Measured prevalence: 2 instances (alt push, title); bell, notification and clipboard are one-shot events, not state.'
          family: stall-recovery-must-repaint
          round: 2
      recipe: milestone-review
      reviewed: 2017402e181f25f85af9c992ff8edae5e6794ee9
      blocked: false
    - "n": 3
      timestamp: "2026-10-10T19:02:36-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: not-addressed
          note: Still loose fields (stalled/modesKnown/previous/history/alt*); deferred to the parent-screen refactor per Log; Minor, non-blocking.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: No title state is kept or re-emitted on recovery (presenter.go:827 path drops it while stalled); same deferred parent-screen value; Minor.
          round: 3
      recipe: milestone-review
      reviewed: eeaf310ea1b82ef695bd380dd86a7851f9b7b67b
      blocked: false
---

# Gate ledger — pair#430 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T18:40:45-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `stall-drop-loses-control-plane-state` EmitEffects drops non-writing effects and the rest of the batch while stalled, losing attention marks and directory updates
  presenter.go:834 sends s=="" effects (inactive-slot bell, DirectoryEffect, policy-off) through writeUnlessStalled; a failed probe returns ok=false and the loop returns, so p.effects/delivered miss them and couchtty console.go:1259 never marks attention. Skip the outer terminal for empty writes, continue past dropped writes, and add a stalled-batch test.
- **BR-2** [Important] `stall-recovery-must-repaint` A probe succeeding inside writeUnlessStalled clears stalled without scheduling the full repaint
  After the outer terminal drains, a Copy or effect that probes first sets stalled=false, so armRetry no longer fires; the screen keeps the abandoned mid-frame bytes and unset modes until new child output. Drop without probing while stalled, or keep a repaint-due flag until a paint succeeds; test Copy-after-drain.
- **BR-3** [Minor] `spec-diagnostic-deferred` Spec's stall notice/diagnostic is reduced to an unconsumed Stalls() counter
  Nothing in production calls Stalls(), so the rollout live check cannot distinguish survived stalls; add a Revisions entry and a follow-up.
- **BR-4** [Minor] `state-flag-constellation` stalled/modesKnown/previous form a flag constellation outside the ViewEvent state machine

## Round 2 — 2026-10-10T18:57:23-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — s=="" skips the terminal, a stalled write returns nil and bookkeeping proceeds (presenter.go:846-852); the stalled-batch test asserts 2 delivered effects.
- BR-2 — addressed — writeUnlessStalled drops without probing; only a completed paint clears stalled (presenter.go:484-486); Copy-after-drain test asserts recovery begins with stallProbe.
- BR-3 — addressed — Revisions entry plus Done-when line scope the diagnostic to Presenter.Stalls() and name the follow-up (trace or crash dir).
- BR-4 — not-addressed — Still loose fields; the Log explicitly defers moving parent-screen knowledge into one value; minor, non-blocking.

### Raised

- **BR-5** [Minor] `stall-recovery-must-repaint` A TitleEffect dropped during a stall is never re-emitted on recovery, so the outer title stays stale
  2nd instance in this family (after the alt-screen kitty-push window). Rule: recovery must restore everything the presenter knows the outer terminal should show (screen, modes, keyboard stack, alt push, title), which is the deferred single parent-screen value that stall() resets whole. Measured prevalence: 2 instances (alt push, title); bell, notification and clipboard are one-shot events, not state.

## Round 3 — 2026-10-10T19:02:36-07:00 (claude) — passed

### Disposed

- BR-4 — not-addressed — Still loose fields (stalled/modesKnown/previous/history/alt*); deferred to the parent-screen refactor per Log; Minor, non-blocking.
- BR-5 — not-addressed — No title state is kept or re-emitted on recovery (presenter.go:827 path drops it while stalled); same deferred parent-screen value; Minor.

## Open findings

- **BR-4** [Minor] `state-flag-constellation` stalled/modesKnown/previous form a flag constellation outside the ViewEvent state machine
- **BR-5** [Minor] `stall-recovery-must-repaint` A TitleEffect dropped during a stall is never re-emitted on recovery, so the outer title stays stale
