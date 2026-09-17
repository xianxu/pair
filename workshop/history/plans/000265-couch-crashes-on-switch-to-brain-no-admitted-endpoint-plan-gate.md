---
gate: plan-quality
issue: 265
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-16T15:09:04-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: The non-fatal notice path re-enters terminalError through publishNotice's repaint
          detail: |-
            deliverChildInput's setNotice goes publishNotice (console.go:1945) -> repaint -> paintNow
            (console.go:1101-1121) -> presenter.UpdateChrome, and paintNow feeds any error to
            terminalError -> Stop. UpdateChrome returns a plain errors.New when p.selected == nil
            (presenter.go:782-783), which is NOT ErrNoDestination, so deliverPresenterInput cannot
            classify it. That window is real: showMenu nils selected via presenter.Panel
            (console_menu.go:220) and only then sets c.focus = FocusPanel (console_menu.go:225-228),
            and showMenu is reachable off the Run goroutine. The refusal the plan calls non-fatal can
            still exit couch. It also contradicts the ARCH-CONSTRAINTS claim that the path gets
            strictly cheaper: it adds a Snapshot + Compose + paintEndpoint round-trip per refused
            keystroke, raising pressure on the deliberately-deferred fatal ErrBackpressure path.
            No task tests the ARCH-ORDER row "actor | printable / ESC / paste | notice, non-fatal".
          family: recovery-path-reenters-escalation
          round: 1
        - id: PQ-2
          severity: Important
          title: Panel safety is tested for three hand-picked event kinds, not the closed enumeration
          detail: |-
            makeInputEvent (terminal/input.go:292-300) is a closed set: KeyPress, KeyRelease, four
            mouse kinds, Focus, Blur, Paste, default -> Reply. Tasks 3-5 cover three kinds by hand,
            which is the exact shape Task 6's own lessons entry names as the bug surface. Task 4's AST
            guard pins the door, not the panel rule, so a future arm calling deliverPresenterInput
            directly passes it. Add one table-driven test over every kind makeInputEvent can emit,
            driven through routeInputEvent with the panel focused, asserting terminalFailure stays nil
            and c.stop stays open (ARCH-PURPOSE: the class, not the three instances observed).
          family: closed-enumeration-needs-mechanical-guard
          round: 1
        - id: PQ-3
          severity: Minor
          title: Task 6 declares atlas/terminal.md but its git add omits it
          detail: |-
            Task 6's Files section lists atlas/terminal.md as new exported surface to map, while
            Step 5 stages only atlas/couch.md, workshop/lessons.md and the issue file.
          family: commit-set-matches-declared-files
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-16T15:12:47-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Notice removed from deliverChildInput; UpdateChrome typed in Task 2 and classified in paintNow at Step 3b; ARCH-CONSTRAINTS claim corrected.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Step 3c adds a table-driven test over makeInputEvent's full closed set (input.go:290-302), driven through routeInputEvent with the panel focused.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: atlas/terminal.md now appears in Task 6's git add alongside its Files declaration.
          round: 2
      blocked: false
content_hash: 5d589b312c096059d61c3c489afb880da85176e85e8b6227ac695053d228a104
---

# Gate ledger — pair#265 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-16T15:09:04-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `recovery-path-reenters-escalation` The non-fatal notice path re-enters terminalError through publishNotice's repaint
  deliverChildInput's setNotice goes publishNotice (console.go:1945) -> repaint -> paintNow
  (console.go:1101-1121) -> presenter.UpdateChrome, and paintNow feeds any error to
  terminalError -> Stop. UpdateChrome returns a plain errors.New when p.selected == nil
  (presenter.go:782-783), which is NOT ErrNoDestination, so deliverPresenterInput cannot
  classify it. That window is real: showMenu nils selected via presenter.Panel
  (console_menu.go:220) and only then sets c.focus = FocusPanel (console_menu.go:225-228),
  and showMenu is reachable off the Run goroutine. The refusal the plan calls non-fatal can
  still exit couch. It also contradicts the ARCH-CONSTRAINTS claim that the path gets
  strictly cheaper: it adds a Snapshot + Compose + paintEndpoint round-trip per refused
  keystroke, raising pressure on the deliberately-deferred fatal ErrBackpressure path.
  No task tests the ARCH-ORDER row "actor | printable / ESC / paste | notice, non-fatal".
- **PQ-2** [Important] `closed-enumeration-needs-mechanical-guard` Panel safety is tested for three hand-picked event kinds, not the closed enumeration
  makeInputEvent (terminal/input.go:292-300) is a closed set: KeyPress, KeyRelease, four
  mouse kinds, Focus, Blur, Paste, default -> Reply. Tasks 3-5 cover three kinds by hand,
  which is the exact shape Task 6's own lessons entry names as the bug surface. Task 4's AST
  guard pins the door, not the panel rule, so a future arm calling deliverPresenterInput
  directly passes it. Add one table-driven test over every kind makeInputEvent can emit,
  driven through routeInputEvent with the panel focused, asserting terminalFailure stays nil
  and c.stop stays open (ARCH-PURPOSE: the class, not the three instances observed).
- **PQ-3** [Minor] `commit-set-matches-declared-files` Task 6 declares atlas/terminal.md but its git add omits it
  Task 6's Files section lists atlas/terminal.md as new exported surface to map, while
  Step 5 stages only atlas/couch.md, workshop/lessons.md and the issue file.

## Round 2 — 2026-09-16T15:12:47-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Notice removed from deliverChildInput; UpdateChrome typed in Task 2 and classified in paintNow at Step 3b; ARCH-CONSTRAINTS claim corrected.
- PQ-2 — addressed — Step 3c adds a table-driven test over makeInputEvent's full closed set (input.go:290-302), driven through routeInputEvent with the panel focused.
- PQ-3 — addressed — atlas/terminal.md now appears in Task 6's git add alongside its Files declaration.

## Open findings

(none — every finding has been disposed)
