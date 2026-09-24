---
gate: boundary-review
issue: 292
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-20T17:58:28-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: README does not document the new browser-tab shortcuts and workflow
          detail: '`shortcut.go:212,234` adds Shift+Alt+B and Alt+B browser-tab actions, but `README.md:145-151` still documents Alt+B only as scrollback navigation. Add the operator-facing browser-tab usage and confirmation/lifecycle behavior before closing the boundary.'
          family: user-facing-surface-docs
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-20T18:06:59-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md now documents Alt+b, Shift+Alt+b, remote confirmation, Alt+w lifecycle, and Alt+h at lines 30-37 and 144-145.
          round: 2
      findings:
        - id: BR-2
          severity: Important
          title: browserController.Close is unbounded despite the M1 shutdown contract
          detail: cmd/internal/termcmd/browser.go:81 waits indefinitely on c.done, while the plan requires Close to be bounded at 3 seconds. Add a timeout and define the fallback behavior so pair term shutdown cannot hang on a blocked controller effect or child cleanup. ARCH-ORDER / ARCH-CONSTRAINTS.
          family: bounded-controller-shutdown
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-20T18:12:50-07:00"
      agent: codex
      dispose:
        - id: BR-2
          disposition: not-addressed
          note: browserController.Close now waits only 3 seconds, but removeTab and closeAll immediately call unbounded child.Close afterward (cmd/internal/termcmd/presentation.go:324-330, 495-498). The only regression test exercises waitBrowserController directly (browser_test.go:40-52), not a blocked controller or the complete shutdown path.
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-09-20T18:18:51-07:00"
      agent: codex
      dispose:
        - id: BR-2
          disposition: addressed
          note: browserController.Close now waits through a 3-second timeout at cmd/internal/termcmd/browser.go:97-107, with regression coverage in browser_test.go:40-53.
          round: 4
      findings:
        - id: BR-3
          severity: Critical
          title: The Core concepts inventory claims M2/M3 entities exist although they are absent at the review head
          detail: workshop/plans/000292-carbonyl-browser-tab-plan.md:57-169 lists record.go, cdp.go, browsercmd, and carbonylconformance as new entities, but those paths are absent. Split the inventory by milestone or revise the statuses before crossing M1. ARCH-PURPOSE / ARCH-ORDER.
          family: plan-core-concepts-drift
          round: 4
        - id: BR-4
          severity: Important
          title: Profile sweep error handling differs from the M1 plan
          detail: cmd/internal/termcmd/browser.go:228-230 aborts launch on ProfileStore.Sweep failure, while the plan specifies diagnostic logging and continued launch. Resolve the contract and add a regression test for sweep failure.
          family: sweep-error-policy
          round: 4
      boundary: M1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#292 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-20T17:58:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-facing-surface-docs` README does not document the new browser-tab shortcuts and workflow
  `shortcut.go:212,234` adds Shift+Alt+B and Alt+B browser-tab actions, but `README.md:145-151` still documents Alt+B only as scrollback navigation. Add the operator-facing browser-tab usage and confirmation/lifecycle behavior before closing the boundary.

## Round 2 — 2026-09-20T18:06:59-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — README.md now documents Alt+b, Shift+Alt+b, remote confirmation, Alt+w lifecycle, and Alt+h at lines 30-37 and 144-145.

### Raised

- **BR-2** [Important] `bounded-controller-shutdown` browserController.Close is unbounded despite the M1 shutdown contract
  cmd/internal/termcmd/browser.go:81 waits indefinitely on c.done, while the plan requires Close to be bounded at 3 seconds. Add a timeout and define the fallback behavior so pair term shutdown cannot hang on a blocked controller effect or child cleanup. ARCH-ORDER / ARCH-CONSTRAINTS.

## Round 3 — 2026-09-20T18:12:50-07:00 (codex) — BLOCKED

### Disposed

- BR-2 — not-addressed — browserController.Close now waits only 3 seconds, but removeTab and closeAll immediately call unbounded child.Close afterward (cmd/internal/termcmd/presentation.go:324-330, 495-498). The only regression test exercises waitBrowserController directly (browser_test.go:40-52), not a blocked controller or the complete shutdown path.

## Round 4 — 2026-09-20T18:18:51-07:00 (codex) — BLOCKED

### Disposed

- BR-2 — addressed — browserController.Close now waits through a 3-second timeout at cmd/internal/termcmd/browser.go:97-107, with regression coverage in browser_test.go:40-53.

### Raised

- **BR-3** [Critical] `plan-core-concepts-drift` The Core concepts inventory claims M2/M3 entities exist although they are absent at the review head
  workshop/plans/000292-carbonyl-browser-tab-plan.md:57-169 lists record.go, cdp.go, browsercmd, and carbonylconformance as new entities, but those paths are absent. Split the inventory by milestone or revise the statuses before crossing M1. ARCH-PURPOSE / ARCH-ORDER.
- **BR-4** [Important] `sweep-error-policy` Profile sweep error handling differs from the M1 plan
  cmd/internal/termcmd/browser.go:228-230 aborts launch on ProfileStore.Sweep failure, while the plan specifies diagnostic logging and continued launch. Resolve the contract and add a regression test for sweep failure.

## Open findings

- **BR-3** [Critical] `plan-core-concepts-drift` The Core concepts inventory claims M2/M3 entities exist although they are absent at the review head
- **BR-4** [Important] `sweep-error-policy` Profile sweep error handling differs from the M1 plan
