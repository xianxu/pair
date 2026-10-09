---
gate: boundary-review
issue: 417
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-09T11:15:52-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: sgrDimmer injects SGR inside a split CSI on dim transitions; test pins the corrupt output
          detail: 'dim.go:36-45 flushes the pending CSI tail and then emits ESC[22m (on to off), and while off it does not track sequences, so ESC[2m can land mid-CSI (off to on). The agent''s SGR is aborted and its remainder prints as literal text. TestSGRDimmerTransitions asserts this. Fix: emit 22m before the pending tail, frame-track while off, and only emit transition prefixes at sequence boundaries.'
          family: stream-rewrite-sequence-boundary
          round: 1
        - id: BR-2
          severity: Important
          title: Dim observer runs zellij exec with no timeout on wrap's signal goroutine and at startup
          detail: refreshDim (wrap.go:2911, 2924) calls exec.Command zellij list-panes with no context or timeout, before setWinsize and inside the loop that also handles SIGUSR1/SIGUSR2. A hung zellij stalls resizes and the capture/re-exec signals for a cosmetic feature (ARCH-CONSTRAINTS). Bound it with a short exec.CommandContext timeout and/or move it off the signal loop.
          family: optional-io-on-critical-path
          round: 1
        - id: BR-3
          severity: Important
          title: SIGWINCH handler ordering and child re-signal on a dim flip are untested; the planned signalChild seam is absent
          detail: Plan Task 5 promised a proxy test showing the dim is set before setWinsize and that a flip with an unchanged size signals the child. The handler calls syscall.Kill inline (wrap.go:2930), and only refreshDim is tested.
          family: plan-promised-test-missing
          round: 1
        - id: BR-4
          severity: Important
          title: README still describes Alt+Shift+Return as a fullscreen toggle
          detail: README.md:30, :148 and :191 should describe the normal, focus (centered, agent dimmed) and maximize cycle that the help text and atlas now document.
          family: readme-user-surface-sync
          round: 1
        - id: BR-5
          severity: Minor
          title: A floating terminal plus a different fullscreen tiled terminal plans embed+fullscreen without collapsing the existing fullscreen
          family: observed-state-convergence
          round: 1
        - id: BR-6
          severity: Minor
          title: Focus to Maximize leaves stale swap/order tokens in the return record until the collapse clears it
          family: record-hygiene
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-09T11:23:04-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Feed frames in both modes, never delays a byte, and switches only at a sequence boundary; TestSGRDimmerTransitions plus TestSGRDimmerChunkInvariance fail against the old dim.go.
          round: 2
        - id: BR-2
          disposition: addressed
          note: zellijFocusObserver uses ListPanesJSONContext with a 1s focusObserveTimeout (dim.go, layoutcmd.go execZellijContext).
          round: 2
        - id: BR-3
          disposition: addressed
          note: handleWinch plus the signalChild seam; TestHandleWinchOrdersDimBeforeResizeAndSignalsOnFlip pins observe, then resize, then a signal only on a flip.
          round: 2
        - id: BR-4
          disposition: addressed
          note: README.md:30, :148 and :191 now describe the focus to maximize to normal cycle.
          round: 2
        - id: BR-5
          disposition: addressed
          note: ObserveRightPaneMode refuses floating plus fullscreen; the new case in TestObserveRightPaneMode fails without the guard.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: 'Minor and still open: Focus to Maximize carries the swap/order tokens until StepClear. Harmless, because the collapse clears the record.'
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#417 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-09T11:15:52-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `stream-rewrite-sequence-boundary` sgrDimmer injects SGR inside a split CSI on dim transitions; test pins the corrupt output
  dim.go:36-45 flushes the pending CSI tail and then emits ESC[22m (on to off), and while off it does not track sequences, so ESC[2m can land mid-CSI (off to on). The agent's SGR is aborted and its remainder prints as literal text. TestSGRDimmerTransitions asserts this. Fix: emit 22m before the pending tail, frame-track while off, and only emit transition prefixes at sequence boundaries.
- **BR-2** [Important] `optional-io-on-critical-path` Dim observer runs zellij exec with no timeout on wrap's signal goroutine and at startup
  refreshDim (wrap.go:2911, 2924) calls exec.Command zellij list-panes with no context or timeout, before setWinsize and inside the loop that also handles SIGUSR1/SIGUSR2. A hung zellij stalls resizes and the capture/re-exec signals for a cosmetic feature (ARCH-CONSTRAINTS). Bound it with a short exec.CommandContext timeout and/or move it off the signal loop.
- **BR-3** [Important] `plan-promised-test-missing` SIGWINCH handler ordering and child re-signal on a dim flip are untested; the planned signalChild seam is absent
  Plan Task 5 promised a proxy test showing the dim is set before setWinsize and that a flip with an unchanged size signals the child. The handler calls syscall.Kill inline (wrap.go:2930), and only refreshDim is tested.
- **BR-4** [Important] `readme-user-surface-sync` README still describes Alt+Shift+Return as a fullscreen toggle
  README.md:30, :148 and :191 should describe the normal, focus (centered, agent dimmed) and maximize cycle that the help text and atlas now document.
- **BR-5** [Minor] `observed-state-convergence` A floating terminal plus a different fullscreen tiled terminal plans embed+fullscreen without collapsing the existing fullscreen
- **BR-6** [Minor] `record-hygiene` Focus to Maximize leaves stale swap/order tokens in the return record until the collapse clears it

## Round 2 — 2026-10-09T11:23:04-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Feed frames in both modes, never delays a byte, and switches only at a sequence boundary; TestSGRDimmerTransitions plus TestSGRDimmerChunkInvariance fail against the old dim.go.
- BR-2 — addressed — zellijFocusObserver uses ListPanesJSONContext with a 1s focusObserveTimeout (dim.go, layoutcmd.go execZellijContext).
- BR-3 — addressed — handleWinch plus the signalChild seam; TestHandleWinchOrdersDimBeforeResizeAndSignalsOnFlip pins observe, then resize, then a signal only on a flip.
- BR-4 — addressed — README.md:30, :148 and :191 now describe the focus to maximize to normal cycle.
- BR-5 — addressed — ObserveRightPaneMode refuses floating plus fullscreen; the new case in TestObserveRightPaneMode fails without the guard.
- BR-6 — not-addressed — Minor and still open: Focus to Maximize carries the swap/order tokens until StepClear. Harmless, because the collapse clears the record.

## Open findings

- **BR-6** [Minor] `record-hygiene` Focus to Maximize leaves stale swap/order tokens in the return record until the collapse clears it
